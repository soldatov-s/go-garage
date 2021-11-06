package httpclient

import (
	//
	"net"
	"net/http"
	"time"
)

const (
	defaultClientTimeout               = 3 * time.Second
	defaultClientDialerTimeout         = 5 * time.Second
	defaultClientTLSHandshakeTimeout   = 5 * time.Second
	defaultClientExpectContinueTimeout = 2 * time.Second
	defaultClientIdleConnTimeout       = 5 * time.Second
	defaultClientResponseHeaderTimeout = 5 * time.Second
)

type Client struct {
	*http.Client
}

func NewNetTransport(cfg *ClientConfig) *http.Transport {
	HTTPClientDialerTimeout := defaultClientDialerTimeout
	if cfg.DialerTimeout > 0 {
		HTTPClientDialerTimeout = cfg.DialerTimeout
	}

	dialer := &net.Dialer{
		Timeout: HTTPClientDialerTimeout,
	}

	HTTPClientTLSHandshakeTimeout := defaultClientTLSHandshakeTimeout
	if cfg.TLSHandshakeTimeout > 0 {
		HTTPClientTLSHandshakeTimeout = cfg.TLSHandshakeTimeout
	}

	HTTPClientExpectContinueTimeout := defaultClientExpectContinueTimeout
	if cfg.ExpectContinueTimeout > 0 {
		HTTPClientExpectContinueTimeout = cfg.ExpectContinueTimeout
	}

	HTTPClientIdleConnTimeout := defaultClientIdleConnTimeout
	if cfg.IdleConnTimeout > 0 {
		HTTPClientIdleConnTimeout = cfg.IdleConnTimeout
	}

	HTTPClientResponseHeaderTimeout := defaultClientResponseHeaderTimeout
	if cfg.ResponseHeaderTimeout > 0 {
		HTTPClientResponseHeaderTimeout = cfg.ResponseHeaderTimeout
	}

	return &http.Transport{
		Dial:                  dialer.Dial,
		TLSHandshakeTimeout:   HTTPClientTLSHandshakeTimeout,
		ExpectContinueTimeout: HTTPClientExpectContinueTimeout,
		IdleConnTimeout:       HTTPClientIdleConnTimeout,
		ResponseHeaderTimeout: HTTPClientResponseHeaderTimeout,
	}
}

// NewClient creates http-client with predifineted timeouts
func NewClient(cfg *ClientConfig, netTransport http.RoundTripper) *Client {
	HTTPClientTimeout := defaultClientTimeout
	if cfg.Timeout > 0 {
		HTTPClientTimeout = cfg.Timeout
	}

	return &Client{
		Client: &http.Client{
			Transport: netTransport,
			Timeout:   HTTPClientTimeout,
		},
	}
}
