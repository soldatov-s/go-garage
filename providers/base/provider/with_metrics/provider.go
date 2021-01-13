package providerwithmetrics

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/stats"
)

type Entity interface {
	provider.Entity
	GetMetrics(prefix string) stats.MapMetricsOptions
	GetAliveHandlers(prefix string) stats.MapCheckFunc
	GetReadyHandlers(prefix string) stats.MapCheckFunc
}

// Provider provides abstract worker.
type Provider struct {
	Entitys provider.MapEnity
	*provider.Provider
}

// NewProvider creates provider
func NewProvider(ctx context.Context, providersName, providerName string) *Provider {
	return &Provider{
		Provider: provider.NewProvider(ctx, providersName, providerName),
	}
}

// GetMetrics return map of the metrics from provider
func (bp *Provider) GetMetrics(prefix string) stats.MapMetricsOptions {
	metrics := make(stats.MapMetricsOptions)
	bp.Entitys.Range(func(_, v interface{}) bool {
		metrics.Fill(v.(Entity).GetMetrics(prefix))
		return true
	})
	return metrics
}

// GetAliveHandlers return array of the aliveHandlers from provider
func (bp *Provider) GetAliveHandlers(prefix string) stats.MapCheckFunc {
	handlers := make(stats.MapCheckFunc)
	bp.Entitys.Range(func(_, v interface{}) bool {
		handlers.Fill(v.(Entity).GetAliveHandlers(prefix))
		return true
	})
	return handlers
}

// GetReadyHandlers return array of the readyHandlers from provider
func (bp *Provider) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	handlers := make(stats.MapCheckFunc)
	bp.Entitys.Range(func(_, v interface{}) bool {
		handlers.Fill(v.(Entity).GetReadyHandlers(prefix))
		return true
	})
	return handlers
}

// Shutdown should shutdown all known entitys.
func (bp *Provider) Shutdown() error {
	var err error
	bp.Log.Info().Msg("shutdown provider" + bp.Name + "...")
	bp.Entitys.Range(func(k, v interface{}) bool {
		if err = v.(Entity).Shutdown(); err != nil {
			return false
		}

		return true
	})
	return err
}

// Start starts all known entitys.
func (bp *Provider) Start() error {
	var err error
	bp.Log.Info().Msg("start provider " + bp.Name + "...")
	bp.Entitys.Range(func(k, v interface{}) bool {
		if err = v.(Entity).Start(); err != nil {
			return false
		}

		return true
	})
	return err
}

// StartEnity starts connection things like watchers and queues.
func (bp *Provider) StartEnity(enityName string) error {
	v, ok := bp.Entitys.Load(enityName)
	if ok {
		return v.(Entity).Start()
	}

	return nil
}

// ShutdownEnity starts connection things like watchers and queues.
func (bp *Provider) ShutdownEnity(enityName string) error {
	v, ok := bp.Entitys.Load(enityName)
	if ok {
		return v.(Entity).Shutdown()
	}

	return nil
}
