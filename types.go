package hyproxia

import (
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// Config holds proxy configuration options.
type Config struct {
	// MaxConnsPerHost limits the maximum number of connections per host.
	// Default: 2048
	MaxConnsPerHost int

	// MaxIdleConnDuration is the maximum duration an idle connection is kept open.
	// Default: 60 seconds
	MaxIdleConnDuration time.Duration

	// ReadTimeout is the maximum duration for reading the entire request from upstream.
	// Default: 15 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration for writing the entire response to upstream.
	// Default: 15 seconds
	WriteTimeout time.Duration

	// MaxRetryAttempts is the maximum number of retry attempts for idempotent requests.
	// Default: 5
	MaxRetryAttempts int

	// ReadBufferSize is the size of the read buffer in bytes.
	// Default: 8192
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer in bytes.
	// Default: 8192
	WriteBufferSize int

	// MaxResponseBodySize is the maximum response body size in bytes.
	// Default: 100MB
	MaxResponseBodySize int

	// DisableHeaderNamesNormalizing disables header name normalization.
	// When true, header names are passed as-is.
	// Default: true
	DisableHeaderNamesNormalizing bool

	// DisablePathNormalizing disables path normalization.
	// When true, paths are passed as-is without cleaning.
	// Default: true
	DisablePathNormalizing bool

	// DNSCacheDuration is the duration for caching DNS lookups.
	// Default: 1 hour
	DNSCacheDuration time.Duration

	// DialConcurrency limits the number of concurrent dial operations.
	// Default: 1000
	DialConcurrency int

	// MaxRedirects is the maximum number of redirects to follow.
	// Set to 0 to disable redirect following.
	// Default: 3
	MaxRedirects int

	// ServerName is the server name sent in response headers.
	// Default: "hyproxia"
	ServerName string

	// MaxRequestBodySize is the maximum request body size for incoming requests in bytes.
	// Default: 10MB
	MaxRequestBodySize int

	// TCPKeepalive enables TCP keep-alive for incoming connections.
	// Default: true
	TCPKeepalive bool

	// TCPKeepalivePeriod is the interval between TCP keep-alive probes.
	// Default: 60 seconds
	TCPKeepalivePeriod time.Duration

	// DisableStartupMessage disables the startup message when the proxy starts.
	// Default: false
	DisableStartupMessage bool

	// EnableTracing enables all tracing code in the request hot path.
	// Default: false
	EnableTracing bool

	// Prefork enables multiple worker processes for improved performance on multi-core systems.
	// Default: false
	Prefork bool

	// PreforkProcesses is the number of child processes to spawn in prefork mode.
	// Default: runtime.NumCPU()
	PreforkProcesses int

	// PreforkGOMAXPROCS sets GOMAXPROCS for each child process in prefork mode.
	// Default: 2
	PreforkGOMAXPROCS int
}

// RouteType determines how a route is matched.
type RouteType int

const (
	// Path matches requests by URL path prefix.
	// Example: router.Route(Path, "/api/", proxy) -> proxyaddress.com/api/...
	Path RouteType = iota

	// Sub matches requests by subdomain.
	// Example: router.Route(Sub, "api", proxy) -> api.proxyaddress.com/...
	Sub
)

type route struct {
	routeType RouteType
	key       string // path prefix or subdomain
	proxy     *Proxy
}

// Router handles multiple proxy routes
type Router struct {
	config       Config
	routes       []route
	routesMu     sync.RWMutex
	server       *fasthttp.Server
	traceHandler func(*Trace)
}

// Proxy represents a reverse proxy instance
type Proxy struct {
	config       Config
	target       string
	client       *fasthttp.Client
	server       *fasthttp.Server
	urlPool      *sync.Pool
	tracePool    sync.Pool
	traceHandler func(*Trace)
	handle       func(*fasthttp.RequestCtx)
	workerID     int
	workerPID    int
}

// Timestamps for tracing request processing stages
// t0: handler start (request received)
// t1: request sent to upstream
// t2: response received from upstream
// t3: response sent to client
type traceTimestamps struct {
	t0 time.Time
	t1 time.Time
	t2 time.Time
	t3 time.Time
}

// Trace holds detailed information per single request
type Trace struct {
	ingestEndpoint             string
	outgoingEndpoint           string
	timeToRequestToUpstream    time.Duration
	timeToResponseFromUpstream time.Duration
	timeToResponseFromProxy    time.Duration
	timeToCompleteRequest      time.Duration
	proxyProcessingTime        time.Duration
	workerID                   int
	workerPID                  int
}
