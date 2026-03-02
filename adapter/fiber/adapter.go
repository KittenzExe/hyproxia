package fiber

import (
	"strings"

	"github.com/KittenzExe/hyproxia"
	"github.com/gofiber/fiber/v2"
)

// Adapter wraps a hyproxia.Proxy for Fiber integration.
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

// New creates a new Fiber adapter for a specific path prefix.
// Example:
//
//	adapter := fiber.New("/api", proxy)
//	// or with options
//	adapter := fiber.New("/api", proxy, fiber.WithStripPrefix())
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
// Example usage in a custom handler:
//
//	apiAdapter := fiberadapter.New("/api", apiProxy)
//	app.Use(func(c *fiber.Ctx) error {
//	    if apiAdapter.Handle(c) {
//	        return nil
//	    }
//	    return c.Next()
//	})
func (a *Adapter) Handle(c *fiber.Ctx) bool {
	path := c.Path()

	// Check if path matches prefix
	if !strings.HasPrefix(path, a.pathPrefix) {
		return false
	}

	// Check for exact match or path continuation (prefix/ or prefix at end)
	if len(path) > len(a.pathPrefix) && path[len(a.pathPrefix)] != '/' {
		return false
	}

	// Access the underlying fasthttp context
	ctx := c.Context()

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

// Handler returns a Fiber handler that proxies matching requests.
// Non-matching requests will call c.Next().
//
// Example:
//
//	app.Use(adapter.Handler())
func (a *Adapter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if a.Handle(c) {
			return nil
		}
		return c.Next()
	}
}

// StrictHandler returns a Fiber handler that proxies matching requests
// and returns 404 for non-matching paths (does not call Next).
//
// Example:
//
//	app.Use("/api", adapter.StrictHandler())
func (a *Adapter) StrictHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if a.Handle(c) {
			return nil
		}
		return c.SendStatus(fiber.StatusNotFound)
	}
}

// RouterAdapter manages multiple hyproxia proxies with path-based routing.
type RouterAdapter struct {
	adapters []*Adapter
	fallback fiber.Handler
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
//	router := fiberadapter.NewRouter()
//	router.Mount("/api", apiProxy)
//	router.Mount("/auth", authProxy, fiberadapter.WithStripPrefix())
func (r *RouterAdapter) Mount(pathPrefix string, proxy *hyproxia.Proxy, opts ...Option) {
	r.adapters = append(r.adapters, New(pathPrefix, proxy, opts...))
}

// SetFallback sets a handler for requests that don't match any proxy route.
//
// Example:
//
//	router.SetFallback(func(c *fiber.Ctx) error {
//	    return c.SendString("Welcome to the main app!")
//	})
func (r *RouterAdapter) SetFallback(handler fiber.Handler) {
	r.fallback = handler
}

// Handler returns a Fiber handler that routes to the appropriate proxy.
//
// Example:
//
//	router := fiberadapter.NewRouter()
//	router.Mount("/api", apiProxy)
//	router.Mount("/auth", authProxy)
//	router.SetFallback(myAppHandler)
//	app.Use(router.Handler())
func (r *RouterAdapter) Handler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		for _, adapter := range r.adapters {
			if adapter.Handle(c) {
				return nil
			}
		}

		if r.fallback != nil {
			return r.fallback(c)
		}

		return c.SendStatus(fiber.StatusNotFound)
	}
}

// Middleware returns a Fiber middleware that proxies matching requests
// and passes non-matching requests to the next handler.
//
// Example:
//
//	app.Use(router.Middleware())
//	app.Get("/health", healthHandler)
func (r *RouterAdapter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		for _, adapter := range r.adapters {
			if adapter.Handle(c) {
				return nil
			}
		}

		return c.Next()
	}
}
