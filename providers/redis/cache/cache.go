package rediscache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
)

var ErrNotFoundInCache = errors.New("not found in cache")

type Connector interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	JSONGet(ctx context.Context, key string, args ...interface{}) (*redis.StringCmd, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	JSONSet(ctx context.Context, key, path, json string, args ...interface{}) (*redis.StatusCmd, error)
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Pipeline() *redis.Pipeline
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
}

type Cache struct {
	*base.MetricsStorage
	config *Config
	conn   Connector
	name   string
}

func NewCache(ctx context.Context, name string, config *Config, conn Connector) (*Cache, error) {
	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	cache := &Cache{
		MetricsStorage: base.NewMetricsStorage(),
		config:         config.SetDefault(),
		conn:           conn,
		name:           name,
	}

	if err := cache.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	return cache, nil
}

func (c *Cache) keyPrefix() string {
	prefix := c.config.KeyPrefix
	if c.config.GlobalKeyPrefix != "" {
		prefix = c.config.GlobalKeyPrefix + ":" + prefix
	}
	return prefix
}

func (c *Cache) buildKey(key string) string {
	return c.keyPrefix() + ":" + key
}

// Get item from cache by key.
func (c *Cache) Get(ctx context.Context, key string, value interface{}) error {
	cmdString := c.conn.Get(ctx, c.buildKey(key))
	if _, err := cmdString.Result(); err != nil {
		return ErrNotFoundInCache
	}

	if err := cmdString.Scan(value); err != nil {
		return errors.Wrap(err, "unmarshal value")
	}

	return nil
}

// JSONGet item from cache by key.
func (c *Cache) JSONGet(ctx context.Context, key, path string, value interface{}) error {
	cmdString, err := c.conn.JSONGet(ctx, c.buildKey(key), path)
	if err != nil {
		return errors.Wrap(err, "JSONGet")
	}

	if _, err := cmdString.Result(); err != nil {
		switch {
		case errors.Is(err, redis.Nil):
			return ErrNotFoundInCache
		default:
			return errors.Wrap(err, "JSONGet")
		}
	}

	if err := cmdString.Scan(value); err != nil {
		return errors.Wrap(err, "unmarshal value")
	}

	return nil
}

// Set item in cache by key.
func (c *Cache) Set(ctx context.Context, key string, value interface{}) error {
	if _, err := c.conn.Set(ctx, c.buildKey(key), value, c.config.ClearTime).Result(); err != nil {
		return errors.Wrap(err, "set key")
	}

	return nil
}

// SetNX (Not eXist) item in cache by key.
func (c *Cache) SetNX(ctx context.Context, key string, value interface{}) error {
	if _, err := c.conn.SetNX(ctx, c.buildKey(key), value, c.config.ClearTime).Result(); err != nil {
		return errors.Wrap(err, "setNX key")
	}

	return nil
}

// JSONSet item in cache by key.
func (c *Cache) JSONSet(ctx context.Context, key, path, json string) error {
	cmd, err := c.conn.JSONSet(ctx, c.buildKey(key), path, json)
	if err != nil {
		return errors.Wrap(err, "JSONSet key")
	}

	if _, err := cmd.Result(); err != nil {
		return errors.Wrap(err, "JSONSet key")
	}

	if c.config.ClearTime > 0 {
		_, err := c.conn.Expire(ctx, c.buildKey(key), c.config.ClearTime).Result()
		if err != nil {
			return errors.Wrap(err, "expire key")
		}
	}

	return nil
}

// JSONSetNX item in cache by key.
func (c *Cache) JSONSetNX(ctx context.Context, key, path, json string) error {
	cmd, err := c.conn.JSONSet(ctx, c.buildKey(key), path, json, "NX")
	if err != nil {
		return errors.Wrap(err, "JSONSetNX key")
	}

	if _, err := cmd.Result(); err != nil {
		return errors.Wrap(err, "JSONSetNX key")
	}

	if c.config.ClearTime > 0 {
		_, err := c.conn.Expire(ctx, c.buildKey(key), c.config.ClearTime).Result()
		if err != nil {
			return errors.Wrap(err, "expire key")
		}
	}

	return nil
}

// Delete item from cache by key.
func (c *Cache) Delete(ctx context.Context, key string) error {
	if _, err := c.conn.Del(ctx, c.buildKey(key)).Result(); err != nil {
		return errors.Wrap(err, "del key")
	}

	return nil
}

// Clear clear all items from selected connection.
func (c *Cache) Clear(ctx context.Context) error {
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.conn.Scan(ctx, cursor, c.keyPrefix()+"*", 10).Result()
		if err != nil {
			return errors.Wrap(err, "scan")
		}

		if len(keys) == 0 {
			continue
		}

		pipe := c.conn.Pipeline()
		err = pipe.Del(ctx, keys...).Err()
		if err != nil {
			return errors.Wrap(err, "del")
		}

		_, err = pipe.Exec(ctx)
		if err != nil {
			return errors.Wrap(err, "exec")
		}
		if cursor == 0 {
			break
		}
	}

	return nil
}

// Size return count of item in cache
func (c *Cache) Size(ctx context.Context) (int, error) {
	length := 0
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = c.conn.Scan(ctx, cursor, c.keyPrefix()+"*", 10).Result()
		if err != nil {
			return -1, errors.Wrap(err, "scan keys")
		}

		length += len(keys)

		if cursor == 0 {
			break
		}
	}

	return length, nil
}

// GetMetrics return map of the metrics from cache connection
func (c *Cache) buildMetrics(_ context.Context) error {
	fullName := c.name

	sizeMetricFunc := func(ctx context.Context) (float64, error) {
		logger := zerolog.Ctx(ctx)
		if c.conn != nil {
			size, err := c.Size(ctx)
			if err != nil {
				logger.Err(err).Msg("cache size")
			}
			return float64(size), nil
		}
		return 0, nil
	}

	if _, err := c.MetricsStorage.GetMetrics().AddGauge(fullName, "cache size", "cache size", sizeMetricFunc); err != nil {
		return errors.Wrap(err, "add gauge metric")
	}

	return nil
}
