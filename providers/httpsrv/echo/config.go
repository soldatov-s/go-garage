package echo

import (
	"time"

	"github.com/labstack/echo/v4"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
)

const (
	defaultBodyReadTimeout   = 10 * time.Second
	defaultBodyWriteTimeout  = 10 * time.Second
	defaultHeaderReadTimeout = 5 * time.Second
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
	Middlewares []interface{} `envconfig:"-"`
}

// Checks for necessity for default values.
func (sc *Config) Validate() error {
	if sc.BodyReadTimeout == 0 {
		sc.BodyReadTimeout = defaultBodyReadTimeout
	}

	if sc.BodyWriteTimeout == 0 {
		sc.BodyWriteTimeout = defaultBodyWriteTimeout
	}

	if sc.HeaderReadTimeout == 0 {
		sc.HeaderReadTimeout = defaultHeaderReadTimeout
	}

	if sc.Address == "" {
		return httpsrv.ErrBindAddressMissing
	}

	return nil
}

func (sc *Config) Copy() *Config {
	return &Config{
		Address:           sc.Address,
		DisableHTTP2:      sc.DisableHTTP2,
		Debug:             sc.Debug,
		HideBanner:        sc.HideBanner,
		HidePort:          sc.HidePort,
		CertFile:          sc.CertFile,
		KeyFile:           sc.KeyFile,
		BodyReadTimeout:   sc.BodyReadTimeout,
		BodyWriteTimeout:  sc.BodyWriteTimeout,
		HeaderReadTimeout: sc.HeaderReadTimeout,
	}
}

func (sc *Config) InitilizeEcho(e *echo.Echo) {
	e.Debug = sc.Debug
	e.DisableHTTP2 = sc.DisableHTTP2
	e.HideBanner = sc.HideBanner
	e.HidePort = sc.HidePort

	e.Server.ReadHeaderTimeout = sc.HeaderReadTimeout
	e.Server.ReadTimeout = sc.BodyReadTimeout
	e.Server.WriteTimeout = sc.BodyWriteTimeout

	e.Server.Addr = sc.Address
}
