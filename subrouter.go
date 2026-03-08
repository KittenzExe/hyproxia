package hyproxia

import (
	"strings"

	"github.com/valyala/fasthttp"
)

// NewSubRouter creates a new subdomain-based router instance.
func NewSubRouter() *SubRouter {
	return &SubRouter{
		routes: make(map[string]*Proxy),
	}
}

// Route adds a proxy for a specific subdomain.
// Example:
//
//	router := hyproxia.NewSubRouter()
//	router.Route("api", hyproxia.New("https://api-backend.example.com"))
//	router.Route("auth", hyproxia.New("https://auth-backend.example.com"))
func (r *SubRouter) Route(subdomain string, proxy *Proxy) {
	r.routes[strings.ToLower(subdomain)] = proxy
}

// RemoveRoute removes a route for the specified subdomain.
// Returns true if the route was found and removed, false otherwise.
func (r *SubRouter) RemoveRoute(subdomain string) bool {
	subdomain = strings.ToLower(subdomain)
	if proxy, ok := r.routes[subdomain]; ok {
		proxy.Close()
		delete(r.routes, subdomain)
		return true
	}
	return false
}

// HandleRequest routes requests to the appropriate proxy based on subdomain.
func (r *SubRouter) HandleRequest(ctx *fasthttp.RequestCtx) {
	host := string(ctx.Host())

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	// Extract subdomain (first part before the first dot)
	subdomain := ""
	if idx := strings.Index(host, "."); idx != -1 {
		subdomain = strings.ToLower(host[:idx])
	}

	proxy, ok := r.routes[subdomain]
	if !ok {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Not Found")
		return
	}

	proxy.HandleRequest(ctx)
}

// Listen starts the subdomain router on the specified address.
func (r *SubRouter) Listen(addr string) error {
	startupMessage(addr)
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServe(addr)
}

// ListenTLS starts the subdomain router with TLS.
func (r *SubRouter) ListenTLS(addr, certFile, keyFile string) error {
	startupMessage(addr, true)
	server := &fasthttp.Server{Handler: r.HandleRequest}
	return server.ListenAndServeTLS(addr, certFile, keyFile)
}

// Close releases resources for all proxies.
func (r *SubRouter) Close() {
	for _, proxy := range r.routes {
		proxy.Close()
	}
}
