package redis

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/rejson"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	// Metrics
	stats.Service

	Conn *rejson.Client

	ctx  context.Context
	log  zerolog.Logger
	name string
	cfg  *Config

	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, cfg interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	// Checking that passed config is OUR.
	if _, ok := cfg.(*Config); !ok {
		return nil, cache.ErrNotConfigPointer(Config{})
	}

	conn := &Enity{
		name:               name,
		ctx:                logger.Registrate(ctx),
		log:                logger.Get(ctx).GetLogger(cache.ProvidersName, nil).With().Str("connection", name).Logger(),
		cfg:                cfg.(*Config).SetDefault(),
		connWatcherStopped: true,
	}

	conn.log.Info().Msgf("initializing enity " + name + "...")

	return conn, nil
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (c *Enity) Shutdown() error {
	c.log.Info().Msg("shutting down database connection watcher and queue worker")
	c.weAreShuttingDown = true

	if c.cfg.StartWatcher {
		for {
			if c.connWatcherStopped {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else {
		c.shutdown()
	}
	c.log.Info().Msg("connection shutted down")

	return nil
}

// SetConnPoolLifetime sets connection lifetime.
func (c *Enity) SetConnPoolLifetime(connMaxLifetime time.Duration) {
	// First - set passed data in connection options.
	c.cfg.MaxConnectionLifetime = connMaxLifetime

	// If connection already established - tweak it.
	if c.Conn != nil {
		c.Conn.Options().MaxConnAge = connMaxLifetime
	}
}

// SetConnPoolLimits sets pool limits for connections counts.
func (c *Enity) SetConnPoolLimits(minIdleConnections, maxOpenedConnections int) {
	// First - set passed data in connection options.
	c.cfg.MinIdleConnections = minIdleConnections
	c.cfg.MaxOpenedConnections = maxOpenedConnections

	// If connection already established - tweak it.
	if c.Conn != nil {
		c.Conn.Options().MinIdleConns = minIdleConnections
		c.Conn.Options().PoolSize = maxOpenedConnections
	}
}

// SetPoolLimits sets connection pool limits.
func (c *Enity) SetPoolLimits(maxIdleConnections, maxOpenedConnections int, connMaxLifetime time.Duration) {
	c.SetConnPoolLimits(maxIdleConnections, maxOpenedConnections)
	c.SetConnPoolLifetime(connMaxLifetime)
}

// Start starts connection workers and connection procedure itself.
func (c *Enity) Start() error {
	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if c.connWatcherStopped {
		if c.cfg.StartWatcher {
			c.connWatcherStopped = false
			go c.startWatcher()
		} else {
			// Manually start the connection once to establish connection
			_ = c.watcher()
		}
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established.
func (c *Enity) WaitForEstablishing() {
	for {
		if c.Conn != nil {
			break
		}

		c.log.Debug().Msg("connection isn't ready - not yet established")
		time.Sleep(time.Millisecond * 100)
	}
}

// Get item from cache by key.
func (c *Enity) Get(key string, value interface{}) error {
	cmdString := c.Conn.Get(c.Conn.Context(), c.cfg.KeyPrefix+key)
	_, err := cmdString.Result()

	if err != nil {
		return cache.ErrNotFoundInCache
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("get value by key %s from cache", key)

	return nil
}

// JSONGet item from cache by key.
func (c *Enity) JSONGet(key, path string, value interface{}) error {
	cmdString := c.Conn.JSONGet(c.Conn.Context(), c.cfg.KeyPrefix+key, path)
	_, err := cmdString.Result()

	if err != nil {
		return cache.ErrNotFoundInCache
	}

	err = cmdString.Scan(value)
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("jsonget value by key %s from cache, path %s", key, path)

	return nil
}

// Set item in cache by key.
func (c *Enity) Set(key string, value interface{}) error {
	_, err := c.Conn.Set(c.Conn.Context(), c.cfg.KeyPrefix+key, value, c.cfg.ClearTime).Result()
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("set key %s in cache, value %+v", key, value)

	return nil
}

// SetNX (Not eXist) item in cache by key.
func (c *Enity) SetNX(key string, value interface{}) error {
	_, err := c.Conn.SetNX(c.Conn.Context(), c.cfg.KeyPrefix+key, value, c.cfg.ClearTime).Result()
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("setnx key %s in cache, value %+v", key, value)

	return nil
}

// JSONSet item in cache by key.
func (c *Enity) JSONSet(key, path, json string) error {
	_, err := c.Conn.JSONSet(c.Conn.Context(), c.cfg.KeyPrefix+key, path, json).Result()
	if err != nil {
		return err
	}

	if c.cfg.ClearTime > 0 {
		_, err = c.Conn.Expire(c.Conn.Context(), c.cfg.KeyPrefix+key, c.cfg.ClearTime).Result()
		if err != nil {
			return err
		}
	}

	c.log.Debug().Msgf("jsonset key %s in cache, path %s, json %s", key, path, json)

	return nil
}

// JSONSetNX item in cache by key.
func (c *Enity) JSONSetNX(key, path, json string) error {
	_, err := c.Conn.JSONSet(c.Conn.Context(), c.cfg.KeyPrefix+key, path, json, "NX").Result()
	if err != nil {
		return err
	}

	if c.cfg.ClearTime > 0 {
		_, err = c.Conn.Expire(c.Conn.Context(), c.cfg.KeyPrefix+key, c.cfg.ClearTime).Result()
		if err != nil {
			return err
		}
	}

	c.log.Debug().Msgf("jsonsetnx key %s in cache, path %s, json %s", key, path, json)

	return nil
}

// Delete item from cache by key.
func (c *Enity) Delete(key string) error {
	_, err := c.Conn.Del(c.Conn.Context(), c.cfg.KeyPrefix+key).Result()
	if err != nil {
		return err
	}

	c.log.Debug().Msgf("delete key %s from cache", key)

	return nil
}

// Size return count of item in cache
func (c *Enity) Size() int {
	length := 0
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.Conn.Scan(c.Conn.Context(), cursor, c.cfg.KeyPrefix+"*", 10).Result()
		if err != nil {
			return -1
		}

		length += len(keys)

		if cursor == 0 {
			break
		}
	}

	c.log.Debug().Msgf("cache size is %d", length)

	return length
}

// Clear clear all items from selected connection.
func (c *Enity) Clear() error {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.Conn.Scan(c.Conn.Context(), cursor, c.cfg.KeyPrefix+"*", 10).Result()
		if err != nil {
			return err
		}

		pipe := c.Conn.Pipeline()
		err = pipe.Del(c.Conn.Context(), keys...).Err()
		if err != nil {
			return err
		}

		_, err = pipe.Exec(c.Conn.Context())
		if err != nil {
			return err
		}
		if cursor == 0 {
			break
		}
	}

	c.log.Debug().Msgf("delete all keys from cache")

	return nil
}

// NewMutex create new redis mutex
func (c *Enity) NewMutex(expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutex(c.Conn, expire, checkInterval)
}

// NewMutexByID create new redis mutex with selected id
func (c *Enity) NewMutexByID(lockID string, expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(c.Conn, lockID, expire, checkInterval)
}

// GetMetrics return map of the metrics from cache connection
func (c *Enity) GetMetrics(prefix string) stats.MapMetricsOptions {
	_ = c.Service.GetMetrics(prefix)
	c.Metrics[prefix+"_"+c.name+"_status"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_status",
				Help: prefix + " " + c.name + " status link to " + utils.RedactedDSN(c.cfg.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			if c.Conn != nil {
				err := c.ping()
				if err == nil {
					(m.(prometheus.Gauge)).Set(1)
				}
			}
		},
	}

	c.Metrics[prefix+"_"+c.name+"_cache_size"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + c.name + "_cache_size",
				Help: prefix + " " + c.name + " cache size",
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(c.Size()))
		},
	}

	return c.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (c *Enity) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	_ = c.Service.GetReadyHandlers(prefix)
	c.ReadyHandlers[strings.ToUpper(prefix+"_"+c.name+"_notfailed")] = func() (bool, string) {
		if c.Conn == nil {
			return false, "not connected"
		}

		if err := c.ping(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return c.ReadyHandlers
}
