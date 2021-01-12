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

	metrics := make(map[string]*stats.MetricOptions)

	providerMetrics, err := prov.(stats.IProviderMetrics).GetMetrics(providerName)
	if err != nil {
		return nil, err
	}

	for k, m := range providerMetrics {
		metrics[k] = m
	}

	return metrics, nil
}

// GetAllMetrics collect all metrics for Databases
// nolint : dupl
func (bp *BaseProvidersWithMetrics) GetAllMetrics(providersName string, out stats.MapMetricsOptions) (stats.MapMetricsOptions, error) {
	metrics := make(stats.MapMetricsOptions)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerMetrics, err1 := v.(stats.IProviderMetrics).GetMetrics(k.(string))
		if err1 != nil {
			err = err1
			return false
		}

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
	handlers := make(stats.MapCheckFunc)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerAliveHandlers, err1 := v.(stats.IProviderMetrics).GetAliveHandlers(providerName)
		if err1 != nil {
			err = err1
			return false
		}

		handlers.Fill(providerAliveHandlers)
		return true
	})

	return handlers, err
}

// GetAllAliveHandlers collect all aliveHandlers for Databases
// nolint : dupl
func (bp *BaseProvidersWithMetrics) GetAllAliveHandlers(providersName string, out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerAliveHandlers, err1 := v.(stats.IProviderMetrics).GetAliveHandlers(k.(string))
		if err1 != nil {
			err = err1
			return false
		}

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
	handlers := make(stats.MapCheckFunc)

	var err error
	bp.GetProviders().Range(func(_, v interface{}) bool {
		providerReadyHandlers, err1 := v.(stats.IProviderMetrics).GetReadyHandlers(providerName)
		if err1 != nil {
			err = err1
			return false
		}

		handlers.Fill(providerReadyHandlers)
		return true
	})

	return handlers, err
}

// GetAllReadyHandlers collect all readyHandlers for Databases
// nolint : dupl
func (bp *BaseProvidersWithMetrics) GetAllReadyHandlers(providersName string, out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	handlers := make(stats.MapCheckFunc)

	var err error
	bp.GetProviders().Range(func(k, v interface{}) bool {
		providerReadyHandlers, err1 := v.(stats.IProviderMetrics).GetReadyHandlers(k.(string))
		if err1 != nil {
			err = err1
			return false
		}

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
