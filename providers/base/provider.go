package base

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/log"
)

var (
	ErrEmptyProviderName     = errors.New("empty provider name isn't allowed")
	ErrProviderNotRegistered = errors.New("provider wasn't registered")
	ErrLoggerPointerIsNil    = errors.New("pointer to logger is nil")
	ErrEmptyConnectionName   = errors.New("empty connection name isn't allowed")
	ErrBadTypeOfProvider     = errors.New("bad type of provider")

	ErrEnityDoesNotExists  = errors.New("enity does not exist")
	ErrInvalidEnityPointer = errors.New("invalid passed enity pointer")
	ErrInvalidPointer      = errors.New("invalid passed pointer")
	ErrDuplicateEnity      = errors.New("duplicate enity")
	ErrBadTypeCast         = errors.New("bad type cast")
)

type EntityGateway interface {
	GetEnity(enityName string) (interface{}, error)
	CreateEnity(ctx context.Context, enityName string, options interface{}) error
	Shutdown(ctx context.Context) error
	Start(ctx context.Context) error
}

// Provider provides abstract worker.
type Provider struct {
	Entitys       sync.Map
	name          string
	collectorName string
}

// NewProvider creates provider
func NewProvider(ctx context.Context, collectorName, name string) (*Provider, error) {
	if collectorName == "" {
		return nil, ErrEmptyCollectorName
	}

	if name == "" {
		return nil, ErrEmptyProviderName
	}

	p := &Provider{name: name, collectorName: collectorName}
	logger := log.FromContext(ctx)
	logger.GetLogger(
		collectorName+"_"+name,
		&log.Field{Name: "collector", Value: collectorName},
		&log.Field{Name: "provider", Value: name}).
		Info().Msg("initializing ...")
	return p, nil
}

func (p *Provider) GetLogger(ctx context.Context) *zerolog.Logger {
	return log.FromContext(ctx).GetLogger(p.collectorName + "_" + p.name)
}

func (p *Provider) GetEnity(enityName string) (interface{}, error) {
	if enityName == "" {
		return nil, ErrEmptyEnityName
	}

	enity, found := p.Entitys.Load(enityName)
	if !found {
		return nil, ErrEnityDoesNotExists
	}

	return enity, nil
}

// Shutdown should shutdown all known entitys.
func (p *Provider) Shutdown(ctx context.Context) error {
	p.GetLogger(ctx).Info().Msg("shutdown provider...")
	var err error
	p.Entitys.Range(func(k, v interface{}) bool {
		err = v.(EntityGateway).Shutdown(ctx)
		return err == nil
	})
	return err
}

// Start starts all known entitys.
func (p *Provider) Start(ctx context.Context) error {
	p.GetLogger(ctx).Info().Msg("start provider...")
	var err error
	p.Entitys.Range(func(k, v interface{}) bool {
		err = v.(EntityGateway).Start(ctx)
		return err == nil
	})
	return err
}

// StartEnity starts connection things like watchers and queues.
func (p *Provider) StartEnity(ctx context.Context, enityName string) error {
	v, ok := p.Entitys.Load(enityName)
	if ok {
		return v.(EntityGateway).Start(ctx)
	}

	return nil
}

// ShutdownEnity starts connection things like watchers and queues.
func (p *Provider) ShutdownEnity(ctx context.Context, enityName string) error {
	v, ok := p.Entitys.Load(enityName)
	if ok {
		return v.(EntityGateway).Shutdown(ctx)
	}

	return nil
}

type EntityWithMetricsGateway interface {
	EntityGateway
	GetMetrics(ctx context.Context) (MapMetricsOptions, error)
	GetAliveHandlers(ctx context.Context) (MapCheckFunc, error)
	GetReadyHandlers(ctx context.Context) (MapCheckFunc, error)
}

// ProviderWithMetrics provides abstract worker.
type ProviderWithMetrics struct {
	*Provider
}

// NewProvider creates provider
func NewProviderWithMetrics(ctx context.Context, collectorName, name string) (*ProviderWithMetrics, error) {
	p, err := NewProvider(ctx, collectorName, name)
	if err != nil {
		return nil, errors.Wrap(err, "new provider")
	}
	return &ProviderWithMetrics{p}, nil
}

// GetMetrics return map of the metrics from provider
func (p *ProviderWithMetrics) GetMetrics(ctx context.Context) (MapMetricsOptions, error) {
	metrics := make(MapMetricsOptions)
	var err error
	p.Entitys.Range(func(_, v interface{}) bool {
		var m MapMetricsOptions
		m, err = v.(EntityWithMetricsGateway).GetMetrics(ctx)
		if err != nil {
			return false
		}
		metrics.Append(m)
		return true
	})

	return metrics, err
}

// GetAliveHandlers return array of the aliveHandlers from provider
func (p *ProviderWithMetrics) GetAliveHandlers(ctx context.Context) (MapCheckFunc, error) {
	handlers := make(MapCheckFunc)
	var err error
	p.Entitys.Range(func(_, v interface{}) bool {
		var h MapCheckFunc
		h, err = v.(EntityWithMetricsGateway).GetAliveHandlers(ctx)
		if err != nil {
			return false
		}
		handlers.Append(h)
		return true
	})
	return handlers, err
}

// GetReadyHandlers return array of the readyHandlers from provider
func (p *ProviderWithMetrics) GetReadyHandlers(ctx context.Context) (MapCheckFunc, error) {
	handlers := make(MapCheckFunc)
	var err error
	p.Entitys.Range(func(_, v interface{}) bool {
		var h MapCheckFunc
		h, err = v.(EntityWithMetricsGateway).GetReadyHandlers(ctx)
		if err != nil {
			return false
		}
		handlers.Append(h)
		return true
	})
	return handlers, err
}
