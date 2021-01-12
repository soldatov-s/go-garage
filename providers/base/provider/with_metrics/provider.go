package providerwithmetrics

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/stats"
)

type Entity interface {
	provider.Entity
	GetMetrics(string) map[string]*stats.MetricOptions
	GetAliveHandlers(string) map[string]stats.CheckFunc
	GetReadyHandlers(string) map[string]stats.CheckFunc
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
func (bp *Provider) GetMetrics(prefix string) (stats.MapMetricsOptions, error) {
	metrics := make(stats.MapMetricsOptions)
	bp.Entitys.Range(func(_, v interface{}) bool {
		metrics.Fill(v.(Entity).GetMetrics(prefix))
		return true
	})
	return metrics, nil
}

// GetAliveHandlers return array of the aliveHandlers from provider
func (bp *Provider) GetAliveHandlers(prefix string) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)
	bp.Entitys.Range(func(_, v interface{}) bool {
		handlers.Fill(v.(Entity).GetAliveHandlers(prefix))
		return true
	})
	return handlers, nil
}

// GetReadyHandlers return array of the readyHandlers from provider
func (bp *Provider) GetReadyHandlers(prefix string) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)
	bp.Entitys.Range(func(_, v interface{}) bool {
		handlers.Fill(v.(Entity).GetReadyHandlers(prefix))
		return true
	})
	return handlers, nil
}
