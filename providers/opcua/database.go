package opcua

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	providerswithmetrics "github.com/soldatov-s/go-garage/providers/base/providers/with_metrics"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/stats"
)

const (
	ProvidersName = "opcua"
)

// Provider is an interface every database provider should conform.
// Every provider can hold more than one connection.
type Provider interface {
	provider.IProvider
	provider.IEnityManager
	stats.IProviderMetrics
}

type Bulker interface {
	// AppendToQueue should add passed item into processing queue, if provider
	// able to be asynchronous (e.g. for batching database inserts or updates,
	// aka transactions). It's up to provider to cast passed item into
	// required data type. Provider should return error if something goes
	// wrong, e.g. if AddToQueue() used on provider that doesn't support
	// transactions or queues.
	AppendToQueue(connectionName string, item interface{}) error
	// WaitForFlush blocks execution until queue will be empty.
	WaitForFlush(connectionName string) error
}

type Migrator interface {
	// RegisterMigration registers migration for specified connection.
	// It is up to provider to provide instructions about working with
	// migrations and how to put them into migration interface. It is
	// recommended to use separate structure.
	RegisterMigration(connectionName string, migration interface{}) error
}

// Databases is a controlling structure for all databases and providers.
type Databases struct {
	*providerswithmetrics.BaseProvidersWithMetrics
}

func NewDatabases(ctx context.Context) *Databases {
	return &Databases{
		BaseProvidersWithMetrics: providerswithmetrics.NewBaseProvidersWithMetrics(ctx, ProvidersName),
	}
}

// GetProvider returns requested database provider.
// Return error if providers wasn't registered.
func (d *Databases) GetProvider(providerName string) (Provider, error) {
	if v, err := d.BaseProviders.GetProvider(providerName); err != nil {
		return nil, err
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// GetAllMetrics collect all metrics for Databases
func (d *Databases) GetAllMetrics(out stats.MapMetricsOptions) (stats.MapMetricsOptions, error) {
	return d.BaseProvidersWithMetrics.GetAllMetrics(ProvidersName, out)
}

// GetAllAliveHandlers collect all aliveHandlers for Databases
func (d *Databases) GetAllAliveHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return d.BaseProvidersWithMetrics.GetAllAliveHandlers(ProvidersName, out)
}

// GetAllReadyHandlers collect all readyHandlers for Databases
func (d *Databases) GetAllReadyHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return d.BaseProvidersWithMetrics.GetAllReadyHandlers(ProvidersName, out)
}
