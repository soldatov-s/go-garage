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

// Opcua is a controlling structure for all Opcua and providers.
type Opcua struct {
	*providerswithmetrics.BaseProvidersWithMetrics
}

func NewOpcua(ctx context.Context) *Opcua {
	return &Opcua{
		BaseProvidersWithMetrics: providerswithmetrics.NewBaseProvidersWithMetrics(ctx, ProvidersName),
	}
}

// GetProvider returns requested database provider.
// Return error if providers wasn't registered.
func (d *Opcua) GetProvider(providerName string) (Provider, error) {
	if v, err := d.BaseProviders.GetProvider(providerName); err != nil {
		return nil, err
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// GetAllMetrics collect all metrics for Opcua
func (d *Opcua) GetAllMetrics(out stats.MapMetricsOptions) (stats.MapMetricsOptions, error) {
	return d.BaseProvidersWithMetrics.GetAllMetrics(ProvidersName, out)
}

// GetAllAliveHandlers collect all aliveHandlers for Opcua
func (d *Opcua) GetAllAliveHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return d.BaseProvidersWithMetrics.GetAllAliveHandlers(ProvidersName, out)
}

// GetAllReadyHandlers collect all readyHandlers for Opcua
func (d *Opcua) GetAllReadyHandlers(out stats.MapCheckFunc) (stats.MapCheckFunc, error) {
	return d.BaseProvidersWithMetrics.GetAllReadyHandlers(ProvidersName, out)
}
