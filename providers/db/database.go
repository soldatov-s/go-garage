package db

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/x/helper"
)

const (
	CollectorName = "database"
)

// Provider is an interface every database provider should conform.
// Every provider can hold more than one connection.
type ProviderGateway interface{}

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

// Collector is a controlling structure for all providers.
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

// GetProvider returns requested provider. It'll return error if
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

// AppendToQueue appends passed queueItem to queue for designated provider
// and connection. Passed queue item should be valid for designated provider,
// otherwise error will be returned.
func (c *Collector) AppendToQueue(providerName, connectionName string, queueItem interface{}) error {
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	if _, ok := prov.(Bulker); !ok {
		return errors.Wrapf(ErrNotBulker, "passed %s", helper.ObjName(prov))
	}

	return prov.(Bulker).AppendToQueue(connectionName, queueItem)
}

// RegisterMigration registers migration for designated provider and
// connection.
func (c *Collector) RegisterMigration(providerName, connectionName string, migration interface{}) error {
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	if _, ok := prov.(Migrator); !ok {
		return errors.Wrapf(ErrNotMigrator, "passed %s", helper.ObjName(prov))
	}

	return prov.(Migrator).RegisterMigration(connectionName, migration)
}

// WaitForFlush block execution until underlying provider will flush
// queue. If something will go wrong here - error will be returned by
// provider.
func (c *Collector) WaitForFlush(providerName, connectionName string) error {
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	if _, ok := prov.(Bulker); !ok {
		return errors.Wrapf(ErrNotBulker, "passed %s", helper.ObjName(prov))
	}

	return prov.(Bulker).WaitForFlush(connectionName)
}

// GetMetrics collect all metrics for providers
func (c *Collector) GetMetrics(ctx context.Context) (base.MapMetricsOptions, error) {
	return c.CollectorWithMetrics.GetMetrics(ctx)
}

// GetAliveHandlers collect all aliveHandlers for providers
func (c *Collector) GetAliveHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	return c.CollectorWithMetrics.GetAliveHandlers(ctx)
}

// GetReadyHandlers collect all readyHandlers for providers
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
