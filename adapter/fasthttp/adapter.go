package fasthttp

import (
	"strings"

	"github.com/KittenzExe/hyproxia"
	"github.com/valyala/fasthttp"
)

// Adapter wraps a hyproxia.Proxy for fasthttp integration.
type Adapter struct {
	proxy      *hyproxia.Proxy
	pathPrefix string
	stripPath  bool
}

// Option configures the adapter.
type Option func(*Adapter)

// WithStripPrefix removes the path prefix before forwarding.
// e.g., /api/users -> /users when prefix is /api
func WithStripPrefix() Option {
	return func(a *Adapter) {
		a.stripPath = true
	}
}

// New creates a new fasthttp adapter for a specific path prefix.
// Example:
//
//	adapter := fasthttp.New("/api", proxy)
//	// or with options
//	adapter := fasthttp.New("/api", proxy, fasthttp.WithStripPrefix())
func New(pathPrefix string, proxy *hyproxia.Proxy, opts ...Option) *Adapter {
	a := &Adapter{
		proxy:      proxy,
		pathPrefix: strings.TrimSuffix(pathPrefix, "/"),
		stripPath:  false,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Handle processes the request if it matches the path prefix.
// Returns true if the request was handled, false otherwise.
//
// Example usage in a custom router:
//
//	apiAdapter := fasthttp.New("/api", apiProxy)
//	handler := func(ctx *fasthttp.RequestCtx) {
//	    if apiAdapter.Handle(ctx) {
//	        return
//	    }
//	    // Handle other routes
//	    ctx.WriteString("Hello from main app")
//	}
func (a *Adapter) Handle(ctx *fasthttp.RequestCtx) bool {
	path := string(ctx.Path())

	// Check if path matches prefix
	if !strings.HasPrefix(path, a.pathPrefix) {
		return false
	}

	// Check for exact match or path continuation (prefix/ or prefix at end)
	if len(path) > len(a.pathPrefix) && path[len(a.pathPrefix)] != '/' {
		return false
	}

	if a.stripPath {
		newPath := path[len(a.pathPrefix):]
		if newPath == "" {
			newPath = "/"
		}

		// Rebuild the full URI with query string
		queryString := ctx.QueryArgs().QueryString()
		if len(queryString) > 0 {
			newPath = newPath + "?" + string(queryString)
		}
		ctx.Request.SetRequestURI(newPath)
	}

	a.proxy.HandleRequest(ctx)
	return true
}

// RequestHandler returns a fasthttp.RequestHandler.
// Use this when you want the adapter to handle ALL requests to its prefix
// and return 404 for non-matching paths.
//
// Example:
//
//	fasthttp.ListenAndServe(":8080", adapter.RequestHandler())
func (a *Adapter) RequestHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if !a.Handle(ctx) {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			ctx.SetBodyString("Not Found")
		}
	}
}

// RouterAdapter manages multiple hyproxia proxies with path-based routing.
type RouterAdapter struct {
	adapters []*Adapter
	fallback fasthttp.RequestHandler
}

// NewRouter creates a new router adapter.
func NewRouter() *RouterAdapter {
	return &RouterAdapter{
		adapters: make([]*Adapter, 0),
	}
}

// Mount adds a proxy at the specified path prefix.
//
// Example:
//
//	router := fasthttp.NewRouter()
//	router.Mount("/api", apiProxy)
//	router.Mount("/auth", authProxy, fasthttp.WithStripPrefix())
func (r *RouterAdapter) Mount(pathPrefix string, proxy *hyproxia.Proxy, opts ...Option) {
	r.adapters = append(r.adapters, New(pathPrefix, proxy, opts...))
}

// SetFallback sets a handler for requests that don't match any proxy route.
//
// Example:
//
//	router.SetFallback(func(ctx *fasthttp.RequestCtx) {
//	    ctx.WriteString("Welcome to the main app!")
//	})
func (r *RouterAdapter) SetFallback(handler fasthttp.RequestHandler) {
	r.fallback = handler
}

// Handler returns a fasthttp.RequestHandler that routes to the appropriate proxy.
//
// Example:
//
//	router := fasthttp.NewRouter()
//	router.Mount("/api", apiProxy)
//	router.Mount("/auth", authProxy)
//	router.SetFallback(myAppHandler)
//	fasthttp.ListenAndServe(":8080", router.Handler())
func (r *RouterAdapter) Handler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		for _, adapter := range r.adapters {
			if adapter.Handle(ctx) {
				return
			}
		}

		if r.fallback != nil {
			r.fallback(ctx)
			return
		}

		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("Not Found")
	}
}
