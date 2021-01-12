package stats

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/base/providers"
	"github.com/soldatov-s/go-garage/providers/errors"
)

const (
	ReadyEndpoint   = "/health/ready"
	AliveEndpoint   = "/health/alive"
	MetricsEndpoint = "/metrics"

	ProvidersName = "statistics"
)

// Checking function should accept no parameters and return
// boolean value which will be interpreted as dependency service readiness
// and a string which should provide error text if dependency service
// isn't ready and something else if dependency service is ready (for
// example, dependency service's version).
type CheckFunc func() (bool, string)

type MapCheckFunc map[string]CheckFunc

func (mcf MapCheckFunc) RegistrateHandler(prov Provider) error {
	for k, f := range mcf {
		err := prov.RegisterAliveCheck(k, f)
		if err != nil {
			return err
		}
	}
	return nil
}

type IProviderMetrics interface {
	// GetMetrics return map of the metrics from provider
	GetMetrics(prefix string) (MapMetricsOptions, error)
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAliveHandlers(prefix string) (MapCheckFunc, error)
	// GetReadyHandlers return array of the readyHandlers from provider
	GetReadyHandlers(prefix string) (MapCheckFunc, error)
}

type IProvidersMetrics interface {
	// GetMetrics return map of the metrics from provider
	GetMetrics(out MapMetricsOptions) (MapMetricsOptions, error)
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAllAliveHandlers(out MapCheckFunc) (MapCheckFunc, error)
	// GetReadyHandlers return array of the readyHandlers from provider
	GetReadyHandlers(out MapCheckFunc) (MapCheckFunc, error)
}

func (mcf MapCheckFunc) Fill(src MapCheckFunc) {
	for k, m := range src {
		mcf[k] = m
	}
}

// Provider is an interface which every statistics provider
// should conform.
// Every provider should be able to respond on alive checks, ready
// checks and metrics request. It is up to provider to decide protocol
// for all things, but HTTP is preferred.
type Provider interface {
	provider.IProvider
	provider.IEniter
	// RegisterAliveCheck should register a function for /health/alive
	// endpoint.
	// Checks registered via RegisterReadyCheck can appear in random order
	// in response.
	RegisterAliveCheck(dependencyName string, checkFunc CheckFunc) error
	// RegisterMetric should register a metric of defined type. Passed
	// metricName should be used only as internal identifier. Provider
	// should provide instructions for using metricOptions as well as
	// cast to appropriate type.
	RegisterMetric(metricName string, metricOptions interface{}) error
	// RegisterReadyCheck should register a function for /health/ready
	// endpoint.
	// Checks registered via RegisterReadyCheck can appear in random order
	// in response.
	RegisterReadyCheck(dependencyName string, checkFunc CheckFunc) error
}

// Statistics is a controlling structure for all statistics providers.
type Statistics struct {
	*providers.BaseProviders
}

func NewStatistics(ctx context.Context) *Statistics {
	return &Statistics{
		BaseProviders: providers.NewBaseProviders(ctx, ProvidersName),
	}
}

// RegisterAliveCheck registers check function for /health/alive endpoint
// in designated provider.
func (st *Statistics) RegisterAliveCheck(providerName, dependencyName string, checkFunc func() (bool, string)) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterAliveCheck(dependencyName, checkFunc)
}

// RegisterMetric registers metric in designated provider. It is up to
// provider to provide instructions on metrics registration.
func (st *Statistics) RegisterMetric(providerName, metricName string, metricOptions interface{}) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterMetric(metricName, metricOptions)
}

// RegisterReadyCheck registers check function for /health/ready endpoint
// in designated provider.
func (st *Statistics) RegisterReadyCheck(providerName, dependencyName string, checkFunc func() (bool, string)) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterReadyCheck(dependencyName, checkFunc)
}

// GetProvider returns provider interface to caller.
// Returns error if provider isn't registered.
func (st *Statistics) GetProvider(providerName string) (Provider, error) {
	if v, err := st.BaseProviders.GetProvider(providerName); err != nil {
		return nil, err
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// Start sends start signal to all Statistic providers.
func (st *Statistics) Start(metrics MapMetricsOptions, aliveHandlers, readyHandlers MapCheckFunc) error {
	var err error
	st.GetProviders().Range(func(k, v interface{}) bool {
		if err = st.StartProvider(k.(string), metrics, aliveHandlers, readyHandlers); err != nil {
			return false
		}

		return true
	})

	return nil
}

// Start sends start signal to selected Statistic provider.
func (st *Statistics) StartProvider(providerName string, metrics MapMetricsOptions, aliveHandlers, readyHandlers MapCheckFunc) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	if err := metrics.RegistrateMetric(prov); err != nil {
		return err
	}

	if err := aliveHandlers.RegistrateHandler(prov); err != nil {
		return err
	}

	if err := readyHandlers.RegistrateHandler(prov); err != nil {
		return err
	}

	return prov.Start()
}
