package stats

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
)

const (
	ReadyEndpoint   = "/health/ready"
	AliveEndpoint   = "/health/alive"
	MetricsEndpoint = "/metrics"

	CollectorName = "statistics"
)

// Checking function should accept no parameters and return
// boolean value which will be interpreted as dependency service readiness
// and a string which should provide error text if dependency service
// isn't ready and something else if dependency service is ready (for
// example, dependency service's version).
type CheckFunc func() (bool, string)

type MapCheckFunc map[string]CheckFunc

func (mcf MapCheckFunc) RegistrateHandler(prov ProviderGateway) error {
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
	GetMetrics(prefix string) MapMetricsOptions
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAliveHandlers(prefix string) MapCheckFunc
	// GetReadyHandlers return array of the readyHandlers from provider
	GetReadyHandlers(prefix string) MapCheckFunc
}

type IProvidersMetrics interface {
	// GetMetrics return map of the metrics from provider
	GetAllMetrics(out MapMetricsOptions) (MapMetricsOptions, error)
	// GetAliveHandlers return array of the aliveHandlers from provider
	GetAllAliveHandlers(out MapCheckFunc) (MapCheckFunc, error)
	// GetReadyHandlers return array of the readyHandlers from provider
	GetAllReadyHandlers(out MapCheckFunc) (MapCheckFunc, error)
}

func (mcf MapCheckFunc) Fill(src MapCheckFunc) {
	for k, m := range src {
		mcf[k] = m
	}
}

// ProviderGateway is an interface which every statistics provider
// should conform.
// Every provider should be able to respond on alive checks, ready
// checks and metrics request. It is up to provider to decide protocol
// for all things, but HTTP is preferred.
type ProviderGateway interface {
	Start(ctx context.Context) error
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

// Collector is a controlling structure for all caches.
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

// GetProvider returns requested caches provider. It'll return error if
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

// RegisterAliveCheck registers check function for /health/alive endpoint
// in designated provider.
func (st *Collector) RegisterAliveCheck(providerName, dependencyName string, checkFunc func() (bool, string)) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterAliveCheck(dependencyName, checkFunc)
}

// RegisterMetric registers metric in designated provider. It is up to
// provider to provide instructions on metrics registration.
func (st *Collector) RegisterMetric(providerName, metricName string, metricOptions interface{}) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterMetric(metricName, metricOptions)
}

// RegisterReadyCheck registers check function for /health/ready endpoint
// in designated provider.
func (st *Collector) RegisterReadyCheck(providerName, dependencyName string, checkFunc func() (bool, string)) error {
	prov, err := st.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterReadyCheck(dependencyName, checkFunc)
}

// Start sends start signal to all Statistic providers.
func (st *Collector) Start(ctx context.Context, metrics MapMetricsOptions, aliveHandlers, readyHandlers MapCheckFunc) error {
	var err error
	st.GetProviders().Range(func(k, v interface{}) bool {
		if err = st.StartProvider(ctx, k.(string), metrics, aliveHandlers, readyHandlers); err != nil {
			return false
		}

		return true
	})

	return nil
}

// Start sends start signal to selected Statistic provider.
func (st *Collector) StartProvider(ctx context.Context, providerName string, metrics MapMetricsOptions, aliveHandlers, readyHandlers MapCheckFunc) error {
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

	return prov.Start(ctx)
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
