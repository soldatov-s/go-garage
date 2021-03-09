package gopcua

import (
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"
)

// Connection watcher goroutine entrypoint.
func (c *Enity) startWatcher() {
	c.log.Info().Msg("starting connection watcher")

	ticker := time.NewTicker(c.cfg.Timeout)

	// First start - manually.
	_ = c.watcher()

	// Then - every ticker tick.
	for range ticker.C {
		if c.watcher() {
			break
		}
	}

	ticker.Stop()
	c.log.Info().Msg("connection watcher stopped and connection to opc ua was shutted down")
	c.connWatcherStopped = true
}

func (c *Enity) shutdown() {
	if c.Conn == nil {
		return
	}
	c.log.Info().Msg("closing opc ua connection...")

	err := c.Conn.Close()
	if err != nil {
		c.log.Error().Err(err).Msg("failed to close opc ua connection")
	}

	c.Conn = nil
}

// Pinging connection if it's alive (or we think so).
func (c *Enity) ping() error {
	if c.Conn == nil {
		return nil
	}
	// TODO: create some code for ping opc ua service
	return nil
}

// Connection watcher itself.
func (c *Enity) watcher() bool {
	// If we're shutting down - stop connection watcher.
	if c.weAreShuttingDown {
		c.shutdown()
		return true
	}

	if err := c.ping(); err != nil {
		c.log.Error().Err(err).Msg("opc ua connection lost")
	}

	// If connection is nil - try to establish (or reestablish)
	// connection.
	if c.Conn == nil {
		c.log.Info().Msg("establishing connection to opc ua...")
		// Connect to opc ua.
		endpoints, err := opcua.GetEndpoints(c.cfg.DSN)
		if err != nil {
			c.log.Fatal().Err(err).Msg("failed to get endpoints")
		}

		ep := opcua.SelectEndpoint(endpoints, c.cfg.SecurityPolicy, ua.MessageSecurityModeFromString(c.cfg.SecurityMode))
		if ep == nil {
			c.log.Fatal().Err(err).Msg("failed to find suitable endpoint")
		}

		c.log.Info().Msgf("%s %s", ep.SecurityPolicyURI, ep.SecurityMode)

		opts := []opcua.Option{
			opcua.SecurityPolicy(c.cfg.SecurityPolicy),
			opcua.SecurityModeString(c.cfg.SecurityPolicy),
			opcua.CertificateFile(c.cfg.CertificateFile),
			opcua.PrivateKeyFile(c.cfg.PrivateKeyFile),
			opcua.AuthAnonymous(),
			opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		}

		opcUAConn := opcua.NewClient(c.cfg.DSN, opts...)
		err = opcUAConn.Connect(c.ctx)
		if err == nil {
			c.log.Info().Msg("opc ua connection established")
			c.Conn = opcUAConn

			return false
		}

		if !c.cfg.StartWatcher {
			c.log.Fatal().Err(err).Msgf("failed to connect to opc ua")
		}

		c.log.Error().Err(err).Msgf("failed to connect to opc ua, reconnect after %d seconds", c.cfg.Timeout)
	}

	return false
}
