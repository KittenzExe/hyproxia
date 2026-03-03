package hyproxia

import "time"

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
		DisableStartupMessage:         false,
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
	def.DisableStartupMessage = custom.DisableStartupMessage

	return def
}
