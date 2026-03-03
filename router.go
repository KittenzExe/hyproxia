package hyproxia

import (
	"strings"

	"github.com/valyala/fasthttp"
)

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

// Listen starts the router on the specified address.
func (r *Router) Listen(addr string) error {
	startupMessage(addr)
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServe(addr)
}

// ListenTLS starts the router with TLS.
func (r *Router) ListenTLS(addr, certFile, keyFile string) error {
	startupMessage(addr, true)
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Close releases resources for all proxies.
func (r *Router) Close() {
	for _, proxy := range r.routes {
		proxy.Close()
	}
}
