package db

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	providerswithmetrics "github.com/soldatov-s/go-garage/providers/base/providers/with_metrics"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/stats"
)

const (
	ProvidersName = "database"
)

// Provider is an interface every database provider should conform.
// Every provider can hold more than one connection.
type Provider interface {
	provider.IProvider
	provider.IEnityManager
	stats.IProviderMetrics

	// AppendToQueue should add passed item into processing queue, if provider
	// able to be asynchronous (e.g. for batching database inserts or updates,
	// aka transactions). It's up to provider to cast passed item into
	// required data type. Provider should return error if something goes
	// wrong, e.g. if AddToQueue() used on provider that doesn't support
	// transactions or queues.
	AppendToQueue(connectionName string, item interface{}) error
	// RegisterMigration registers migration for specified connection.
	// It is up to provider to provide instructions about working with
	// migrations and how to put them into migration interface. It is
	// recommended to use separate structure.
	RegisterMigration(connectionName string, migration interface{}) error
	// WaitForFlush blocks execution until queue will be empty.
	WaitForFlush(connectionName string) error
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

// AppendToQueue appends passed queueItem to queue for designated provider
// and connection. Passed queue item should be valid for designated provider,
// otherwise error will be returned.
func (d *Databases) AppendToQueue(providerName, connectionName string, queueItem interface{}) error {
	prov, err := d.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.AppendToQueue(connectionName, queueItem)
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

// RegisterMigration registers migration for designated provider and
// connection.
func (d *Databases) RegisterMigration(providerName, connectionName string, migration interface{}) error {
	prov, err := d.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterMigration(connectionName, migration)
}

// WaitForFlush block execution until underlying provider will flush
// queue. If something will go wrong here - error will be returned by
// provider.
func (d *Databases) WaitForFlush(providerName, connectionName string) error {
	prov, err := d.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.WaitForFlush(connectionName)
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
