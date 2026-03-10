package hyproxia

import (
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// NewRouter creates a new router instance.
// An optional config can be provided; otherwise, the DefaultConfig is used.
//
// Example:
//
//	// With default config
//	router := hyproxia.NewRouter()
//
//	// With custom config
//	router := hyproxia.NewRouter(hyproxia.Config{
//	    ReadTimeout:  30 * time.Second,
//	    WriteTimeout: 30 * time.Second,
//	})
//
//	defer router.Close()
//
//	router.Route(hyproxia.Path, "/api/", hyproxia.New("https://api.example.com"))
//	// or
//	router.Route(hyproxia.Sub, "api", hyproxia.New("https://api.example.com"))
//
//	router.Listen(":8080")
func NewRouter(config ...Config) *Router {
	cfg := DefaultConfig()
	if len(config) > 0 {
		cfg = mergeConfig(cfg, config[0])
	}

	return &Router{
		config: cfg,
		routes: make([]route, 0),
	}
}

// Route adds a proxy for a specific route type and key.
//
//   - Path routes match by URL path prefix:
//     router.Route(hyproxia.Path, "/api/", proxy) -> proxyaddress.com/api/...
//
//   - Sub routes match by subdomain:
//     router.Route(hyproxia.Sub, "api", proxy) -> api.proxyaddress.com/...
func (r *Router) Route(routeType RouteType, key string, proxy *Proxy) {
	switch routeType {
	case Path:
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
	case Sub:
		key = strings.ToLower(key)
	}

	r.routes = append(r.routes, route{
		routeType: routeType,
		key:       key,
		proxy:     proxy,
	})
}

// RemoveRoute removes a route matching the given type and key.
// Returns true if the route was found and removed, false otherwise.
func (r *Router) RemoveRoute(routeType RouteType, key string) bool {
	switch routeType {
	case Path:
		if !strings.HasPrefix(key, "/") {
			key = "/" + key
		}
	case Sub:
		key = strings.ToLower(key)
	}

	for i, rt := range r.routes {
		if rt.routeType == routeType && rt.key == key {
			rt.proxy.Close()
			r.routes = append(r.routes[:i], r.routes[i+1:]...)
			return true
		}
	}
	return false
}

// HandleRequest routes requests to the appropriate proxy.
func (r *Router) HandleRequest(ctx *fasthttp.RequestCtx) {
	var t0 time.Time
	if r.config.EnableTracing {
		t0 = time.Now()
	}

	path := string(ctx.Path())
	host := string(ctx.Host())

	// Remove port from host if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Extract subdomain (first part before the first dot)
	subdomain := ""
	if idx := strings.Index(host, "."); idx != -1 {
		subdomain = strings.ToLower(host[:idx])
	}

	var matchedProxy *Proxy
	var matchedKey string
	var matchedType RouteType
	matchedLen := 0

	for _, rt := range r.routes {
		switch rt.routeType {
		case Path:
			if strings.HasPrefix(path, rt.key) && len(rt.key) > matchedLen {
				matchedLen = len(rt.key)
				matchedProxy = rt.proxy
				matchedKey = rt.key
				matchedType = rt.routeType
			}
		case Sub:
			if subdomain == rt.key {
				matchedProxy = rt.proxy
				matchedKey = rt.key
				matchedType = rt.routeType
				matchedLen = len(path) + 1
			}
		}
	}

	if matchedProxy == nil {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Not Found")
		return
	}

	// Strip the matched path prefix before forwarding to the upstream proxy
	if matchedType == Path {
		stripped := strings.TrimPrefix(path, strings.TrimSuffix(matchedKey, "/"))
		if stripped == "" {
			stripped = "/"
		}

		uri := ctx.URI()
		queryString := uri.QueryString()
		if len(queryString) > 0 {
			ctx.Request.SetRequestURI(stripped + "?" + string(queryString))
		} else {
			ctx.Request.SetRequestURI(stripped)
		}
	}

	matchedProxy.HandleRequest(ctx)

	// Router-level tracing
	if r.config.EnableTracing && r.traceHandler != nil {
		t := &Trace{}
		total := time.Since(t0)
		t.ingestEndpoint = path
		t.outgoingEndpoint = matchedProxy.target + string(ctx.Request.RequestURI())
		t.timeToCompleteRequest = total
		r.traceHandler(t)
	}
}

// OnTrace sets a trace handler for all requests routed through this router.
// Requires EnableTracing: true in the router config.
func (r *Router) OnTrace(fn func(*Trace)) {
	if !r.config.EnableTracing {
		return
	}
	r.traceHandler = fn
}

// Listen starts the router on the specified address.
func (r *Router) Listen(addr string) error {
	if !r.config.DisableStartupMessage {
		startupMessage(addr)
	}
	r.server = &fasthttp.Server{
		Handler:                       r.HandleRequest,
		Name:                          r.config.ServerName,
		ReadTimeout:                   r.config.ReadTimeout,
		WriteTimeout:                  r.config.WriteTimeout,
		MaxRequestBodySize:            r.config.MaxRequestBodySize,
		DisableHeaderNamesNormalizing: r.config.DisableHeaderNamesNormalizing,
		TCPKeepalive:                  r.config.TCPKeepalive,
		TCPKeepalivePeriod:            r.config.TCPKeepalivePeriod,
	}
	return r.server.ListenAndServe(addr)
}

// ListenTLS starts the router with TLS.
func (r *Router) ListenTLS(addr, certFile, keyFile string) error {
	if !r.config.DisableStartupMessage {
		startupMessage(addr, true)
	}
	r.server = &fasthttp.Server{
		Handler:                       r.HandleRequest,
		Name:                          r.config.ServerName,
		ReadTimeout:                   r.config.ReadTimeout,
		WriteTimeout:                  r.config.WriteTimeout,
		MaxRequestBodySize:            r.config.MaxRequestBodySize,
		DisableHeaderNamesNormalizing: r.config.DisableHeaderNamesNormalizing,
		TCPKeepalive:                  r.config.TCPKeepalive,
		TCPKeepalivePeriod:            r.config.TCPKeepalivePeriod,
	}
	return r.server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Close releases resources for all proxies.
func (r *Router) Close() {
	for _, router := range r.routes {
		router.proxy.Close()
	}
}
