package hyproxia

import (
	"strings"

	"github.com/valyala/fasthttp"
)

// NewPathRouter creates a new path-based router instance.
func NewPathRouter() *PathRouter {
	return &PathRouter{
		routes: make(map[string]*Proxy),
	}
}

// Route adds a proxy for a specific path prefix.
// Example:
//
//	router.Route("/api/", hyproxia.New("https://api.example.com"))
//	router.Route("/auth/", hyproxia.New("https://auth.example.com"))
func (r *PathRouter) Route(prefix string, proxy *Proxy) {
	// Ensure prefix starts with /
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	r.routes[prefix] = proxy
}

// HandleRequest routes requests to the appropriate proxy based on path.
func (r *PathRouter) HandleRequest(ctx *fasthttp.RequestCtx) {
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

// Listen starts the path router on the specified address.
func (r *PathRouter) Listen(addr string) error {
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServe(addr)
}

// ListenTLS starts the path router with TLS.
func (r *PathRouter) ListenTLS(addr, certFile, keyFile string) error {
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Close releases resources for all proxies.
func (r *PathRouter) Close() {
	for _, proxy := range r.routes {
		proxy.Close()
	}
}
