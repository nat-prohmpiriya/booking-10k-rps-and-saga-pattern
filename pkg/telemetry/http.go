package telemetry

import (
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPClientConfig holds configuration for instrumented HTTP client
type HTTPClientConfig struct {
	Timeout         time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

// DefaultHTTPClientConfig returns default configuration
func DefaultHTTPClientConfig() *HTTPClientConfig {
	return &HTTPClientConfig{
		Timeout:         30 * time.Second,
		MaxIdleConns:    100,
		IdleConnTimeout: 90 * time.Second,
	}
}

// NewHTTPClient creates an HTTP client with OTel instrumentation
func NewHTTPClient(cfg *HTTPClientConfig) *http.Client {
	if cfg == nil {
		cfg = DefaultHTTPClientConfig()
	}

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableCompression:  false,
		DisableKeepAlives:   false,
		MaxIdleConnsPerHost: cfg.MaxIdleConns / 2,
	}

	return &http.Client{
		Timeout:   cfg.Timeout,
		Transport: otelhttp.NewTransport(transport),
	}
}

// NewHTTPTransport creates an HTTP transport with OTel instrumentation
func NewHTTPTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return otelhttp.NewTransport(base)
}

// WrapHTTPClient wraps an existing HTTP client with OTel instrumentation
func WrapHTTPClient(client *http.Client) *http.Client {
	if client == nil {
		return NewHTTPClient(nil)
	}

	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	client.Transport = otelhttp.NewTransport(transport)
	return client
}
