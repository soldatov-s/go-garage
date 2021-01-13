package providerswithmetrics

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/providers"
	"github.com/soldatov-s/go-garage/providers/stats"
)

type BaseProvidersWithMetrics struct {
	*providers.BaseProviders
}

// Initialize initializes  controlling structure.
func NewBaseProvidersWithMetrics(ctx context.Context, name string) *BaseProvidersWithMetrics {
	return &BaseProvidersWithMetrics{
		BaseProviders: providers.NewBaseProviders(ctx, name),
	}
}

// GetMetrics get metrics for provider by providerName
func (bp *BaseProvidersWithMetrics) GetMetrics(providerName string) (stats.MapMetricsOptions, error) {
	prov, err := bp.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	metrics := make(stats.MapMetricsOptions)
	providerMetrics := prov.(stats.IProviderMetrics).GetMetrics(providerName)
	metrics.Fill(providerMetrics)
	return metrics, nil
}

// GetAllMetrics collect all metrics for Databases
// nolint : dupl
func (bp *BaseProvidersWithMetrics) GetAllMetrics(providersName string, out stats.MapMetricsOptions) (stats.MapMetricsOptions, error) {
	metrics := make(stats.MapMetricsOptions)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerMetrics := v.(stats.IProviderMetrics).GetMetrics(k.(string))

		for k, m := range providerMetrics {
			metrics[k] = m
			if out != nil {
				out[providersName+"_"+k] = m
			}
		}
		return true
	})

	return metrics, err
}

// GetAliveHandlers get aliveHandlers for provider by providerName
func (bp *BaseProvidersWithMetrics) GetAliveHandlers(providerName string) (stats.MapCheckFunc, error) {
	prov, err := bp.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	handlers := make(stats.MapCheckFunc)
	providerHandlers := prov.(stats.IProviderMetrics).GetAliveHandlers(providerName)
	handlers.Fill(providerHandlers)

	return handlers, nil
}

// GetAllAliveHandlers collect all aliveHandlers for Databases
// nolint : dupl
func (bp *BaseProvidersWithMetrics) GetAllAliveHandlers(providersName string, out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerAliveHandlers := v.(stats.IProviderMetrics).GetAliveHandlers(k.(string))

		for k, m := range providerAliveHandlers {
			handlers[k] = m
			if out != nil {
				out[providersName+"_"+k] = m
			}
		}

		return true
	})

	return handlers, err
}

// GetReadyHandlers get readyHandlers for provider by providerName
func (bp *BaseProvidersWithMetrics) GetReadyHandlers(providerName string) (stats.MapCheckFunc, error) {
	prov, err := bp.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	handlers := make(stats.MapCheckFunc)
	providerHandlers := prov.(stats.IProviderMetrics).GetReadyHandlers(providerName)
	handlers.Fill(providerHandlers)

	return handlers, nil
}

// GetAllReadyHandlers collect all readyHandlers for Databases
// nolint : dupl
func (bp *BaseProvidersWithMetrics) GetAllReadyHandlers(providersName string, out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerReadyHandlers := v.(stats.IProviderMetrics).GetReadyHandlers(k.(string))

		for k, m := range providerReadyHandlers {
			handlers[k] = m
			if out != nil {
				out[providersName+"_"+k] = m
			}
		}

		return true
	})

	return handlers, err
}
