package base

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/log"
)

var (
	ErrEmptyCollectorName        = errors.New("empty collector name is not allowed")
	ErrProviderAlreadyRegistered = errors.New("provider already registered")
)

type ProviderGateway interface {
	// Start start all connections
	Start() error
	// Shutdown should shutdown all connections.
	Shutdown() error
}

type CollectorGateway interface {
	AddProvider(providerName string, iface interface{}) error
	GetProviders() *sync.Map
	Start(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type Collector struct {
	providers sync.Map
	name      string
}

var _ CollectorGateway = &Collector{}

// Initialize initializes  controlling structure.
func NewCollector(ctx context.Context, name string) (*Collector, error) {
	if name == "" {
		return nil, ErrEmptyCollectorName
	}

	c := &Collector{name: name}
	log.FromContext(ctx).GetLogger(c.name, &log.Field{Name: "collector", Value: name}).Info().Msg("initializing...")
	return c, nil
}

// AddProvider registers provider. If provider with passed
// name already registered return error.
func (c *Collector) AddProvider(providerName string, iface interface{}) error {
	// We should not attempt to register providers with empty names.
	if providerName == "" {
		return ErrEmptyProviderName
	}

	if _, found := c.providers.LoadOrStore(providerName, iface); found {
		return errors.Wrapf(ErrProviderAlreadyRegistered, "providerName %q", providerName)
	}

	return nil
}

func (c *Collector) GetProviders() *sync.Map {
	return &c.providers
}

func (c *Collector) getLogger(ctx context.Context) *zerolog.Logger {
	return log.FromContext(ctx).GetLogger(c.name)
}

func (c *Collector) Start(ctx context.Context) error {
	c.getLogger(ctx).Info().Msg("start")
	var err error
	c.providers.Range(func(k, v interface{}) bool {
		err = v.(ProviderGateway).Start()
		return err == nil
	})
	return err
}

// Shutdown shutdowns all connections.
func (c *Collector) Shutdown(ctx context.Context) error {
	c.getLogger(ctx).Info().Msg("shutdown")
	var err error
	c.providers.Range(func(k, v interface{}) bool {
		err = v.(ProviderGateway).Shutdown()
		return err == nil
	})
	return err
}

type EnityStarter interface {
	// StartEnity starts connection things.
	StartEnity(ctx context.Context, enityName string) error
}

// Start sends start signal to selected enity.
func (c *Collector) StartEnity(ctx context.Context, providerName, serverName string) error {
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	return prov.(EnityStarter).StartEnity(ctx, serverName)
}

type EnityShutdowner interface {
	// ShutdownEnity shutdown connection things.
	ShutdownEnity(ctx context.Context, enityName string) error
}

// Shutdown sends Shutdown signal to selected enity.
func (c *Collector) ShutdownEnity(ctx context.Context, providerName, serverName string) error {
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	return prov.(EnityShutdowner).ShutdownEnity(ctx, serverName)
}

// GetProvider returns requested database provider.
// Return error if providers wasn't registered.
func (c *Collector) GetProvider(providerName string) (interface{}, error) {
	if providerName == "" {
		return nil, ErrEmptyProviderName
	}

	v, found := c.providers.Load(providerName)
	if !found {
		return nil, ErrProviderNotRegistered
	}

	return v, nil
}

type EnityCreator interface {
	// CreateEnity should create connection using passed parameters.
	// It is up to provider to cast passed options into required data
	// type and provide instructions for configuring.
	// Passed conn should be a pointer to needed structure that might be
	// provided by database provider. Provider should not attempt to
	// replace pointer if nil is passed. Provider should check if passed
	// conn is a pointer to valid structure and return error if it is
	// invalid AND continue with connection creating.
	CreateEnity(ctx context.Context, connectionName string, options interface{}) error
}

// CreateEnity creates new connection for specified provider. Provider
// and connection names should not be empty, otherwise error will be returned.
func (c *Collector) CreateEnity(ctx context.Context, providerName, enityName string, options interface{}) error {
	// Get provider.
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return errors.Wrap(err, "get provider")
	}

	return prov.(EnityCreator).CreateEnity(ctx, enityName, options)
}

type EnityGetter interface {
	// GetEnity should return pointer to connection structure to
	// caller. It is up to caller to cast returned interface{} into
	// required data type.
	// Passed conn should be a pointer to needed structure that might be
	// provided by database provider. Provider should not attempt to
	// replace pointer if nil is passed. Provider should check if passed
	// conn is a pointer to valid structure and return error if it is
	// invalid.
	GetEnity(ctx context.Context, enityName string) (interface{}, error)
}

// GetEnity obtains pointer to connection from specified provider.
// Provider and connection names should not be empty, otherwise error will
// be returned.
func (c *Collector) GetEnity(ctx context.Context, providerName, enityName string) (interface{}, error) {
	// Get provider.
	prov, err := c.GetProvider(providerName)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	return prov.(EnityGetter).GetEnity(ctx, enityName)
}

type ProviderMetricsGateway interface {
	// GetMetrics return map of the metrics from provider
	GetMetrics(ctx context.Context) MapMetricsOptions
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAliveHandlers(ctx context.Context) MapCheckFunc
	// GetReadyHandlers return array of the readyHandlers from provider
	GetReadyHandlers(ctx context.Context) MapCheckFunc
}

type CollectorMetricsGateway interface {
	// GetMetrics return map of the metrics from provider
	GetMetrics(ctx context.Context) (MapMetricsOptions, error)
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAliveHandlers(ctx context.Context) (MapCheckFunc, error)
	// GetReadyHandlers return array of the readyHandlers from provider
	GetReadyHandlers(ctx context.Context) (MapCheckFunc, error)
}

type CollectorWithMetricsGateway interface {
	CollectorGateway
	CollectorMetricsGateway
}

type CollectorWithMetrics struct {
	*Collector
}

// Initialize initializes  controlling structure.
func NewCollectorWithMetrics(ctx context.Context, name string) (*CollectorWithMetrics, error) {
	c, err := NewCollector(ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "new collector")
	}

	return &CollectorWithMetrics{c}, nil
}

// GetMetrics return map of the metrics for all providers
func (c *CollectorWithMetrics) GetMetrics(ctx context.Context) (MapMetricsOptions, error) {
	metrics := make(MapMetricsOptions)
	var err error
	c.providers.Range(func(_, v interface{}) bool {
		var m MapMetricsOptions
		m, err = v.(EntityWithMetricsGateway).GetMetrics(ctx)
		if err != nil {
			return false
		}
		metrics.Add(m)
		return true
	})

	return metrics, err
}

// GetAliveHandlers collect all aliveHandlers for all providers
func (c *CollectorWithMetrics) GetAliveHandlers(ctx context.Context) (MapCheckFunc, error) {
	handlers := make(MapCheckFunc)
	var err error
	c.providers.Range(func(_, v interface{}) bool {
		var h MapCheckFunc
		h, err = v.(EntityWithMetricsGateway).GetAliveHandlers(ctx)
		if err != nil {
			return false
		}
		handlers.Add(h)
		return true
	})
	return handlers, err
}

// GetReadyHandlers collect all readyHandlers for Databases
func (c *CollectorWithMetrics) GetReadyHandlers(ctx context.Context) (MapCheckFunc, error) {
	handlers := make(MapCheckFunc)
	var err error
	c.providers.Range(func(_, v interface{}) bool {
		var h MapCheckFunc
		h, err = v.(EntityWithMetricsGateway).GetReadyHandlers(ctx)
		if err != nil {
			return false
		}
		handlers.Add(h)
		return true
	})
	return handlers, err
}
