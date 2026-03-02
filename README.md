# hyproxia

A minimal, high-performance reverse proxy library for Go built on [fasthttp](https://github.com/valyala/fasthttp).

## Features

- **High Performance** - Built on fasthttp
- **Automatic Redirect Handling** - Configurable redirect following
- **TLS Support** - Built-in HTTPS listener
- **Optional Route-Based Proxying** - Route different paths to different backends
- **Easy Integration** - Works with existing fasthttp or fiber applications

## Installation

```bash
go get github.com/KittenzExe/hyproxia
```

## Quick Start

```go
package main

import "github.com/KittenzExe/hyproxia"

func main() {
    proxy := hyproxia.New("https://api.example.com")
    defer proxy.Close()
    proxy.Listen(":8080")
}
```

## Route-Based Proxying

Route different paths to different backends:

```go
package main

import "github.com/KittenzExe/hyproxia"

func main() {
    router := hyproxia.NewRouter()
    defer router.Close()

    router.Route("/api/", hyproxia.New("https://api.example.com"))
    router.Route("/auth/", hyproxia.New("https://auth.example.com"))
    router.Route("/static/", hyproxia.New("https://cdn.example.com"))

    router.Listen(":8080")
}
```

## TLS Support

```go
// Single proxy
proxy := hyproxia.New("https://api.example.com")
defer proxy.Close()
proxy.ListenWithTLS(":443", "cert.pem", "key.pem")

// Router
router := hyproxia.NewRouter()
defer router.Close()
router.Route("/api/", hyproxia.New("https://api.example.com"))
router.ListenTLS(":443", "cert.pem", "key.pem")
```

## Custom Configuration

```go
package main

import (
    "time"
    "github.com/KittenzExe/hyproxia"
)

func main() {
    proxy := hyproxia.New("https://api.example.com", hyproxia.Config{
        MaxConnsPerHost:     4096,
        ReadTimeout:         30 * time.Second,
        WriteTimeout:        30 * time.Second,
        MaxRetryAttempts:    3,
        MaxRedirects:        5,
        ServerName:          "my-proxy",
    })
    defer proxy.Close()
    proxy.Listen(":8080")
}
```

### Configuration Options

| Option | Default | Description |
|--------|---------|-------------|
| `MaxConnsPerHost` | 2048 | Maximum connections per upstream host |
| `MaxIdleConnDuration` | 60s | Maximum idle connection lifetime |
| `ReadTimeout` | 15s | Request read timeout |
| `WriteTimeout` | 15s | Response write timeout |
| `MaxRetryAttempts` | 5 | Retry attempts for idempotent requests |
| `ReadBufferSize` | 8192 | Read buffer size in bytes |
| `WriteBufferSize` | 8192 | Write buffer size in bytes |
| `MaxResponseBodySize` | 100MB | Maximum response body size |
| `MaxRequestBodySize` | 10MB | Maximum request body size |
| `DisableHeaderNamesNormalizing` | true | Pass header names as-is |
| `DisablePathNormalizing` | true | Pass paths without cleaning |
| `DNSCacheDuration` | 1h | DNS lookup cache duration |
| `DialConcurrency` | 1000 | Maximum concurrent dial operations |
| `MaxRedirects` | 3 | Maximum redirects to follow (0 to disable) |
| `ServerName` | "hyproxia" | Server name in response headers |
| `TCPKeepalive` | true | Enable TCP keep-alive |
| `TCPKeepalivePeriod` | 60s | TCP keep-alive probe interval |


## Integrations

### Using [fasthttp](https;//github.com/valyala/fasthttp)

```go
package main

import (
    "strings"

    "github.com/KittenzExe/hyproxia"
    "github.com/valyala/fasthttp"
)

func main() {
    apiProxy := hyproxia.New("https://api.example.com")
    defer apiProxy.Close()

    handler := func(ctx *fasthttp.RequestCtx) {
        path := string(ctx.Path())

        switch {
        case path == "/health":
            ctx.SetStatusCode(200)
            ctx.SetBodyString("OK")

        case strings.HasPrefix(path, "/api/"):
            apiProxy.HandleRequest(ctx)

        default:
            ctx.SetStatusCode(404)
            ctx.SetBodyString("Not Found")
        }
    }

    fasthttp.ListenAndServe(":8080", handler)
}
```

### Using [Fiber](https://github.com/gofiber/fiber)

```go
package main

import (
    "github.com/KittenzExe/hyproxia"
    "github.com/gofiber/fiber/v2"
)

func main() {
    app := fiber.New()

    apiProxy := hyproxia.New("https://api.example.com")
    defer apiProxy.Close()

    // Your Fiber routes
    app.Get("/health", func(c *fiber.Ctx) error {
        return c.SendString("OK")
    })

    // Proxy routes using All() to catch all methods
    app.All("/api/*", func(c *fiber.Ctx) error {
        apiProxy.HandleRequest(c.Context())
        return nil
    })

    app.Listen(":8080")
}
```

## License

[MIT License](LICENSE)