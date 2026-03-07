// hyproxia: A minimal, high-performance reverse proxy built on fasthttp
//
// Basic usage:
//
//	proxy := hyproxia.New("https://api.example.com")
//	defer proxy.Close()
//	proxy.Listen(":8080")

package hyproxia

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// Version of current hyproxia
const version = "0.1.4"

// New creates a new proxy instance targeting the specified URL.
// An optional config can be provided; otherwise, the DefaultConfig is used.
// Example:
//
//	// With default config
//	proxy := hyproxia.New("https://api.example.com")
//
//	// With a custom config
//	proxy := hyproxia.New("https://api.example.com", hyproxia.Config{
//	    MaxConnsPerHost: 4096,
//	    ReadTimeout:     30 * time.Second,
//	})
func New(targetURL string, config ...Config) *Proxy {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = mergeConfig(cfg, config[0])
	}

	target := strings.TrimSuffix(targetURL, "/")

	client := &fasthttp.Client{
		NoDefaultUserAgentHeader:      true,
		MaxConnsPerHost:               cfg.MaxConnsPerHost,
		MaxIdleConnDuration:           cfg.MaxIdleConnDuration,
		MaxConnDuration:               0,
		ReadTimeout:                   cfg.ReadTimeout,
		WriteTimeout:                  cfg.WriteTimeout,
		MaxIdemponentCallAttempts:     cfg.MaxRetryAttempts,
		ReadBufferSize:                cfg.ReadBufferSize,
		WriteBufferSize:               cfg.WriteBufferSize,
		MaxResponseBodySize:           cfg.MaxResponseBodySize,
		DisableHeaderNamesNormalizing: cfg.DisableHeaderNamesNormalizing,
		DisablePathNormalizing:        cfg.DisablePathNormalizing,
		Dial: (&fasthttp.TCPDialer{
			Concurrency:      cfg.DialConcurrency,
			DNSCacheDuration: cfg.DNSCacheDuration,
		}).Dial,
	}

	p := &Proxy{
		config: cfg,
		target: target,
		client: client,
		urlPool: &sync.Pool{
			New: func() any {
				buf := make([]byte, 0, 512)
				return &buf
			},
		},
		tracePool: sync.Pool{New: func() any { return &Trace{} }},
	}

	if cfg.EnableTracing {
		p.handle = p.handleHTTPTracing
	} else {
		p.handle = p.handleHTTP
	}

	p.server = &fasthttp.Server{
		Handler:                       p.HandleRequest,
		Name:                          cfg.ServerName,
		ReadTimeout:                   cfg.ReadTimeout,
		WriteTimeout:                  cfg.WriteTimeout,
		MaxRequestBodySize:            cfg.MaxRequestBodySize,
		DisableHeaderNamesNormalizing: cfg.DisableHeaderNamesNormalizing,
		TCPKeepalive:                  cfg.TCPKeepalive,
		TCPKeepalivePeriod:            cfg.TCPKeepalivePeriod,
	}

	return p
}

// Listen starts the proxy server on the specified address.
// Example:
//
//	proxy.Listen(":8080")
func (p *Proxy) Listen(addr string) error {
	if !p.config.DisableStartupMessage {
		startupMessage(addr)
	}
	return p.server.ListenAndServe(addr)
}

// ListenWithTLS starts the proxy server with TLS on the specified address.
// Example:
//
//	proxy.ListenWithTLS(":443", "cert.pem", "key.pem")
func (p *Proxy) ListenWithTLS(addr, certFile, keyFile string) error {
	if !p.config.DisableStartupMessage {
		startupMessage(addr, true)
	}
	return p.server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Shutdown gracefully shuts down the server.
func (p *Proxy) Shutdown() error {
	return p.server.Shutdown()
}

// Handler returns a fasthttp.RequestHandler.
func (p *Proxy) Handler() fasthttp.RequestHandler {
	return p.HandleRequest
}

// HandleRequest processes incoming requests.
func (p *Proxy) HandleRequest(ctx *fasthttp.RequestCtx) {
	p.handle(ctx)
}

// buildTargetURL constructs the full upstream URL using the pool.
func (p *Proxy) buildTargetURL(ctx *fasthttp.RequestCtx) string {
	urlBytesPtr := p.urlPool.Get().(*[]byte)
	urlBytes := (*urlBytesPtr)[:0]

	urlBytes = append(urlBytes, p.target...)
	urlBytes = append(urlBytes, ctx.RequestURI()...)
	targetURL := string(urlBytes)

	*urlBytesPtr = urlBytes
	p.urlPool.Put(urlBytesPtr)

	return targetURL
}

// handleHTTP forwards the request to the target server and writes the response back to the client.
func (p *Proxy) handleHTTP(ctx *fasthttp.RequestCtx) {
	targetURL := p.buildTargetURL(ctx)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	ctx.Request.CopyTo(req)
	req.SetRequestURI(targetURL)

	if err := p.doWithRedirects(req, resp, p.config.MaxRedirects); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Bad Gateway")
		return
	}

	resp.CopyTo(&ctx.Response)
}

// handleHTTPTracing forwards the request to the target server and writes the response back to the client with tracing.
func (p *Proxy) handleHTTPTracing(ctx *fasthttp.RequestCtx) {
	var ts traceTimestamps
	if p.traceHandler != nil {
		ts = *newTraceTimestamps() // t0
	}

	targetURL := p.buildTargetURL(ctx)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	ctx.Request.CopyTo(req)
	req.SetRequestURI(targetURL)

	if p.traceHandler != nil {
		ts.t1 = time.Now() // t1
	}

	if err := p.doWithRedirects(req, resp, p.config.MaxRedirects); err != nil {
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Bad Gateway")
		return
	}

	if p.traceHandler != nil {
		ts.t2 = time.Now() // t2
	}

	resp.CopyTo(&ctx.Response)

	if p.traceHandler != nil {
		ts.t3 = time.Now() // t3
		trace := p.tracePool.Get().(*Trace)
		buildTrace(ts, ctx.LocalAddr().String(), targetURL, trace)
		p.traceHandler(trace)
		p.tracePool.Put(trace)
	}
}

// doWithRedirects executes the request and follows redirects up to the maxRedirects
func (p *Proxy) doWithRedirects(req *fasthttp.Request, resp *fasthttp.Response, maxRedirects int) error {
	// Parse the allowed host once from the configured target.
	allowedURI := fasthttp.AcquireURI()
	defer fasthttp.ReleaseURI(allowedURI)
	if err := allowedURI.Parse(nil, []byte(p.target)); err != nil {
		return err
	}
	allowedHost := allowedURI.Host()

	for i := 0; i <= maxRedirects; i++ {
		if err := p.client.Do(req, resp); err != nil {
			return err
		}

		statusCode := resp.StatusCode()
		if statusCode < 300 || statusCode >= 400 {
			return nil
		}

		location := resp.Header.Peek("Location")
		if len(location) == 0 {
			return nil
		}

		redirectURI := fasthttp.AcquireURI()
		parseErr := redirectURI.Parse(nil, location)
		redirectHost := redirectURI.Host()
		fasthttp.ReleaseURI(redirectURI)
		if parseErr != nil {
			return errors.New("invalid redirect location!")
		}
		if len(redirectHost) > 0 && !bytes.Equal(redirectHost, allowedHost) {
			return errors.New("redirect to disallowed host blocked!")
		}

		req.SetRequestURIBytes(location)
		resp.Reset()
	}

	return nil
}

// Close releases resources
func (p *Proxy) Close() {
	p.client.CloseIdleConnections()
}
