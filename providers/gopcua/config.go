package gopcua

import (
	"time"

	"github.com/gopcua/opcua"
)

const (
	// Default DSN and connection parameters that will be passed to
	// database driver.
	defaultDSN       = "opc.tcp://localhost:4840"
	defaultTimeout   = 10 * time.Second
	defaulHandle     = 42
	defaultQueueSzie = 1024
)

const (
	none = "None"
)

// Config represents configuration structure for every
// connection.
type Config struct {
	// DSN is a connection string in form of DSN. Example:
	// opc.tcp://host:port.
	// Default: "opc.tcp://localhost:4840"
	DSN string `envconfig:"optional"`
	// SecurityPolicy sets the security policy uri for the secure channel.
	SecurityPolicy string `envconfig:"optional"`
	// SecurityModeString sets the security mode for the secure channel.
	// Valid values are "None", "Sign", and "SignAndEncrypt".
	SecurityMode string `envconfig:"optional"`
	// Certificate sets the client X509 certificate in the secure channel configuration
	// from the PEM or DER encoded file. It also detects and sets the ApplicationURI
	// from the URI within the certificate.
	CertificateFile string `envconfig:"optional"`
	// PrivateKeyFile sets the RSA private key in the secure channel configuration
	// from a PEM or DER encoded file.
	PrivateKeyFile string `envconfig:"optional"`
	// Subscription interval
	Interval time.Duration `envconfig:"optional"`
	// Queue size
	QueueSize int `envconfig:"optional"`
	// Timeout is a timeout in seconds for connection checking. Every
	// this count of seconds opcua connection will be checked for
	// aliveness and, if it dies, attempt to reestablish connection
	// will be made. Default timeout is 10 seconds.
	Timeout time.Duration `envconfig:"optional"`
	// Handle is a client gandle
	Handle uint32 `envconfig:"optional"`
	// StartWatcher indicates to connection controller that it should
	// also start asynchronous connection watcher.
	StartWatcher bool `envconfig:"optional"`
}

// SetDefault checks connection config. If required field is empty - it will
// be filled with some default value.
// Returns a copy of config.
func (c *Config) SetDefault() *Config {
	cfgCopy := *c

	if cfgCopy.DSN == "" {
		cfgCopy.DSN = defaultDSN
	}

	if cfgCopy.Interval == 0 {
		cfgCopy.Interval = opcua.DefaultSubscriptionInterval
	}

	if cfgCopy.SecurityMode == "" {
		cfgCopy.SecurityMode = none
	}

	if cfgCopy.SecurityPolicy == "" {
		cfgCopy.SecurityPolicy = none
	}

	if cfgCopy.Handle == 0 {
		cfgCopy.Handle = defaulHandle
	}

	if cfgCopy.QueueSize == 0 {
		cfgCopy.QueueSize = defaultQueueSzie
	}

	if cfgCopy.Timeout == 0 {
		cfgCopy.Timeout = defaultTimeout
	}

	return &cfgCopy
}
