package cache

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	providerswithmetrics "github.com/soldatov-s/go-garage/providers/base/providers/with_metrics"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/stats"
)

const (
	ProvidersName = "caches"
)

// Provider is an interface every cache provider should conform.
// Every provider can hold more than one connection.
type Provider interface {
	provider.IProvider
	provider.IEnityManager
	stats.IProviderMetrics

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

// Caches is a controlling structure for all caches and providers.
type Caches struct {
	*providerswithmetrics.BaseProvidersWithMetrics
}

func NewCaches(ctx context.Context) *Caches {
	return &Caches{
		BaseProvidersWithMetrics: providerswithmetrics.NewBaseProvidersWithMetrics(ctx, ProvidersName),
	}
}

// GetProvider returns requested caches provider. It'll return error if
// providers wasn't registered.
func (c *Caches) GetProvider(providerName string) (Provider, error) {
	if v, err := c.BaseProvidersWithMetrics.GetProvider(providerName); err != nil {
		return nil, err
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// GetAllMetrics collect all metrics for Databases
func (c *Caches) GetAllMetrics(out stats.MapMetricsOptions) (stats.MapMetricsOptions, error) {
	return c.BaseProvidersWithMetrics.GetAllMetrics(ProvidersName, out)
}

// GetAllAliveHandlers collect all aliveHandlers for Databases
func (c *Caches) GetAllAliveHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return c.BaseProvidersWithMetrics.GetAllAliveHandlers(ProvidersName, out)
}

// GetAllReadyHandlers collect all readyHandlers for Databases
func (c *Caches) GetAllReadyHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return c.BaseProvidersWithMetrics.GetAllReadyHandlers(ProvidersName, out)
}
