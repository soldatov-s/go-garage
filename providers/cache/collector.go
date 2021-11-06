package cache

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
)

const (
	CollectorName = "caches"
)

var (
	ErrEmptyCachees    = errors.New("empty caches")
	ErrNotFoundInCache = errors.New("not found in cache")
)

// ProviderGateway is an interface every cache provider should conform.
// Every provider can hold more than one connection.
type ProviderGateway interface {
	// Get item from cache by key.
	Get(connectionName, key string, value interface{}) error
	// Set item in cache by key.
	Set(connectionName, key string, value interface{}) error
	// SetNX (Not eXist) item in cache by key.
	SetNX(connectionName, key string, value interface{}) error
	// Delete item from cache by key.
	Delete(connectionName, key string) error
	// Clear all items from selected connection.
	Clear(connectionName string) error
	// Clear all items from all connections.
	ClearAll() error
}

// Collector is a controlling structure for all caches.
type Collector struct {
	*base.CollectorWithMetrics
}

func NewCollector(ctx context.Context) (*Collector, error) {
	c, err := base.NewCollectorWithMetrics(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "new collector with metrics")
	}
	return &Collector{c}, nil
}

// GetProvider returns requested caches provider. It'll return error if
// providers wasn't registered.
func (c *Collector) GetProvider(providerName string) (ProviderGateway, error) {
	p, err := c.CollectorWithMetrics.GetProvider(providerName)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	pg, ok := p.(ProviderGateway)
	if !ok {
		return nil, errors.Wrap(base.ErrBadTypeOfProvider, "expected ProviderGateway")
	}

	return pg, nil
}

// GetMetrics collect all metrics for Caches
func (c *Collector) GetMetrics(ctx context.Context) (base.MapMetricsOptions, error) {
	return c.CollectorWithMetrics.GetMetrics(ctx)
}

// GetAliveHandlers collect all aliveHandlers for Caches
func (c *Collector) GetAliveHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	return c.CollectorWithMetrics.GetAliveHandlers(ctx)
}

// GetReadyHandlers collect all readyHandlers for Caches
func (c *Collector) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	return c.CollectorWithMetrics.GetReadyHandlers(ctx)
}

func NewContext(ctx context.Context) (context.Context, error) {
	c, err := NewCollector(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "new collector %q", CollectorName)
	}
	return base.NewContextByName(ctx, CollectorName, c)
}

func FromContext(ctx context.Context) (*Collector, error) {
	v, err := base.FromContextByName(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "get from context by name")
	}
	c, ok := v.(*Collector)
	if !ok {
		return nil, base.ErrFailedTypeCast
	}
	return c, nil
}
