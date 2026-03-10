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

By default, the proxy will forward all incoming requests to the targeted URL while preserving the original request path and query parameters.

## Router Proxy

```go
func main() {
    router := hyproxia.NewRouter()
    defer router.Close()
    
    // hyproxia.Path proxies requests from /api onward. (localhost:8080/api/endpoint -> https://api.example.com/endpoint)
    router.Route(hyproxia.Path, "/api", hyproxia.New("https://api.example.com"))
    // hyproxia.Sub proxies requests from the root of the subdomain. (auth.localhost:8080/endpoint -> https://auth.example.com/endpoint)
    router.Route(hyproxia.Sub, "auth", hyproxia.New("https://auth.example.com"))

    router.Listen(":8080")
}
```

Both `hyproxia.Path` and `hyproxia.Sub` can be used to route requests based on path prefixes or subdomains, respectively. The router will match incoming requests against the registered routes and forward them to the appropriate backend proxy.

Both are also not limited to only 1 instance per backend, so you can have multiple path or subdomain routes pointing to the same backend if needed.


## TLS Support
```go
// Single proxy
proxy := hyproxia.New("https://api.example.com")
defer proxy.Close()
proxy.ListenWithTLS(":443", "cert.pem", "key.pem")

// Router proxy
router := hyproxia.NewRouter()
defer router.Close()
router.Route(hyproxia.Sub, "auth", hyproxia.New("https://auth.example.com"))
router.ListenWithTLS(":443", "cert.pem", "key.pem")
```

## Tracing

```go
// Single proxy
proxy := hyproxia.New("https://api.example.com", hyproxia.Config{
    EnableTracing: true,
})

proxy.OnTrace(func(t *hyproxia.Trace) {
    fmt.Printf("ingest=%s outgoing=%s prep=%s upstream=%s write=%s total=%s overhead=%s\n",
        t.IngestEndpoint(),
        t.OutgoingEndpoint(),
        t.PrepTime(),
        t.UpstreamLatency(),
        t.WriteTime(),
        t.TotalDuration(),
        t.ProxyOverhead(),

        // With Prefork (optionally):
        t.WorkerID(),
        t.WorkerPID(),
    )
})
```

Tracing is also supported in the router proxy, the exact same way as the single proxy.

## Prefork (Only supported on single proxy)

```go
proxy := hyproxia.New("https://api.example.com", hyproxia.Config{
    Prefork: true,
})
defer proxy.Close()
proxy.Listen(":8080")
```

Prefork can have other config settings such as `PreforkProcesses` to control the number of worker processes, and `PreforkGOMAXPROCS` to control the GOMAXPROCS setting for each worker.

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
| `DisableStartupMessage` | false | Disable startup message |
| `EnableTracing` | false | Enable detailed request tracing |
| `Prefork` | false | Enable prefork mode for multi-core performance |
| `PreforkProcesses` | Number of CPU cores | Number of worker processes in prefork mode |
| `PreforkGOMAXPROCS` | 2 | GOMAXPROCS setting for each worker in prefork mode |


## Adapters

See [hyproxia/adapter](adapter/adapter.md) for building adapters to integrate hyproxia with other Go web frameworks.

Frameworks supported:
- [fasthttp](adapter/fasthttp)
- [fiber](adapter/fiber)

## License

[MIT License](LICENSE)