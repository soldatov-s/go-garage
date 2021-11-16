package echo

import (
	"time"

	"github.com/labstack/echo/v4"
)

const (
	defaultBodyReadTimeout   = 10 * time.Second
	defaultBodyWriteTimeout  = 10 * time.Second
	defaultHeaderReadTimeout = 5 * time.Second
	defaultAddress           = "localhost:9000"
)

type Config struct {
	// Address is an address on which HTTP server will listen.
	Address string `envconfig:"optional"`
	// DisableHTTP2 disabled HTTP2 features (only HTTP 1.0/1.1 will work).
	DisableHTTP2 bool `envconfig:"optional"`
	// Debug enables internal echo debugging features.
	Debug bool `envconfig:"optional"`
	// HideBanner disables showing Echo banner on server's start.
	HideBanner bool `envconfig:"optional"`
	// HidePort disables showing address and port on which Echo will listen
	// for connections.
	HidePort bool `envconfig:"optional"`
	// Path to certificate for HTTPS
	CertFile string `envconfig:"optional"`
	// Path to key for HTTPS
	KeyFile string `envconfig:"optional"`

	// Timeouts
	// BodyReadTimeout sets body reading timeout in seconds. Defaults
	// to 10 seconds.
	BodyReadTimeout time.Duration `envconfig:"optional"`
	// BodyWriteTimeout sets body writing timeout in seconds. Defaults
	// to 10 seconds.
	BodyWriteTimeout time.Duration `envconfig:"optional"`
	// HeaderReadTimeout sets headers reading timeout in seconds. Defaults
	// to 5 seconds.
	HeaderReadTimeout time.Duration `envconfig:"optional"`

	// Middlewares for server endpoints
	Middlewares []echo.MiddlewareFunc `envconfig:"-"`
}

// Checks for necessity for default values.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if c.BodyReadTimeout == 0 {
		c.BodyReadTimeout = defaultBodyReadTimeout
	}

	if c.BodyWriteTimeout == 0 {
		c.BodyWriteTimeout = defaultBodyWriteTimeout
	}

	if c.HeaderReadTimeout == 0 {
		c.HeaderReadTimeout = defaultHeaderReadTimeout
	}

	if c.Address == "" {
		c.Address = defaultAddress
	}

	return &cfgCopy
}

func (c *Config) NewEcho() *echo.Echo {
	srv := echo.New()
	srv.Debug = c.Debug
	srv.DisableHTTP2 = c.DisableHTTP2
	srv.HideBanner = c.HideBanner
	srv.HidePort = c.HidePort

	srv.Server.ReadHeaderTimeout = c.HeaderReadTimeout
	srv.Server.ReadTimeout = c.BodyReadTimeout
	srv.Server.WriteTimeout = c.BodyWriteTimeout

	srv.Server.Addr = c.Address

	return srv
}
