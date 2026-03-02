// hyproxia: A minimal, high-performance reverse proxy built on fasthttp
//
// Basic usage:
//
//	proxy := hyproxia.New("https://api.example.com")
//	defer proxy.Close()
//	proxy.Listen(":8080")

package hyproxia

import (
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// DefaultConfig returns sensible defaults
func DefaultConfig() Config {
	return Config{
		MaxConnsPerHost:               2048,
		MaxIdleConnDuration:           60 * time.Second,
		ReadTimeout:                   15 * time.Second,
		WriteTimeout:                  15 * time.Second,
		MaxRetryAttempts:              5,
		ReadBufferSize:                8192,
		WriteBufferSize:               8192,
		MaxResponseBodySize:           100 * 1024 * 1024, // 100MB
		DisableHeaderNamesNormalizing: true,
		DisablePathNormalizing:        true,
		DNSCacheDuration:              time.Hour,
		DialConcurrency:               1000,
		MaxRedirects:                  3,
		ServerName:                    "hyproxia",
		MaxRequestBodySize:            10 * 1024 * 1024, // 10MB
		TCPKeepalive:                  true,
		TCPKeepalivePeriod:            60 * time.Second,
	}
}

// mergeConfig applies non-zero values from custom config.
func mergeConfig(def, custom Config) Config {
	if custom.MaxConnsPerHost != 0 {
		def.MaxConnsPerHost = custom.MaxConnsPerHost
	}
	if custom.MaxIdleConnDuration != 0 {
		def.MaxIdleConnDuration = custom.MaxIdleConnDuration
	}
	if custom.ReadTimeout != 0 {
		def.ReadTimeout = custom.ReadTimeout
	}
	if custom.WriteTimeout != 0 {
		def.WriteTimeout = custom.WriteTimeout
	}
	if custom.MaxRetryAttempts != 0 {
		def.MaxRetryAttempts = custom.MaxRetryAttempts
	}
	if custom.ReadBufferSize != 0 {
		def.ReadBufferSize = custom.ReadBufferSize
	}
	if custom.WriteBufferSize != 0 {
		def.WriteBufferSize = custom.WriteBufferSize
	}
	if custom.MaxResponseBodySize != 0 {
		def.MaxResponseBodySize = custom.MaxResponseBodySize
	}
	if custom.DNSCacheDuration != 0 {
		def.DNSCacheDuration = custom.DNSCacheDuration
	}
	if custom.DialConcurrency != 0 {
		def.DialConcurrency = custom.DialConcurrency
	}
	if custom.MaxRedirects != 0 {
		def.MaxRedirects = custom.MaxRedirects
	}
	if custom.ServerName != "" {
		def.ServerName = custom.ServerName
	}
	if custom.MaxRequestBodySize != 0 {
		def.MaxRequestBodySize = custom.MaxRequestBodySize
	}
	if custom.TCPKeepalivePeriod != 0 {
		def.TCPKeepalivePeriod = custom.TCPKeepalivePeriod
	}
	// Booleans need explicit handling since false is valid
	def.DisableHeaderNamesNormalizing = custom.DisableHeaderNamesNormalizing
	def.DisablePathNormalizing = custom.DisablePathNormalizing
	def.TCPKeepalive = custom.TCPKeepalive

	return def
}

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

// NewRouter creates a new router instance.
func NewRouter() *Router {
	return &Router{
		routes: make(map[string]*Proxy),
	}
}

// Route adds a proxy for a specific path prefix.
// Example:
//
//	router.Route("/api/", hyproxia.New("https://api.example.com"))
//	router.Route("/auth/", hyproxia.New("https://auth.example.com"))
func (r *Router) Route(prefix string, proxy *Proxy) {
	// Ensure prefix starts with /
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	r.routes[prefix] = proxy
}

// Handler returns a fasthttp.RequestHandler for the router.
func (r *Router) Handler() fasthttp.RequestHandler {
	return r.HandleRequest
}

// HandleRequest routes requests to the appropriate proxy.
func (r *Router) HandleRequest(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())

	// Find longest matching prefix
	var matchedPrefix string
	var matchedProxy *Proxy

	for prefix, proxy := range r.routes {
		if strings.HasPrefix(path, prefix) && len(prefix) > len(matchedPrefix) {
			matchedPrefix = prefix
			matchedProxy = proxy
		}
	}

	if matchedProxy == nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Not Found")
		return
	}

	matchedProxy.HandleRequest(ctx)
}

// Listen starts the proxy server on the specified address.
// Example:
//
//	proxy.Listen(":8080")
func (p *Proxy) Listen(addr string) error {
	return p.server.ListenAndServe(addr)
}

// Listen starts the router on the specified address.
func (r *Router) Listen(addr string) error {
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServe(addr)
}

// ListenWithTLS starts the proxy server with TLS on the specified address.
// Example:
//
//	proxy.ListenWithTLS(":443", "cert.pem", "key.pem")
func (p *Proxy) ListenWithTLS(addr, certFile, keyFile string) error {
	return p.server.ListenAndServeTLS(addr, certFile, keyFile)
}

// ListenTLS starts the router with TLS.
func (r *Router) ListenTLS(addr, certFile, keyFile string) error {
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServeTLS(addr, certFile, keyFile)
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
	p.handleHTTP(ctx)
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

// doWithRedirects executes the request and follows redirects up to the maxRedirects
func (p *Proxy) doWithRedirects(req *fasthttp.Request, resp *fasthttp.Response, maxRedirects int) error {
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

		req.SetRequestURIBytes(location)
		resp.Reset()
	}

	return nil
}

// Close releases resources
func (p *Proxy) Close() {
	p.client.CloseIdleConnections()
}

// Close releases resources for all proxies.
func (r *Router) Close() {
	for _, proxy := range r.routes {
		proxy.Close()
	}
}
