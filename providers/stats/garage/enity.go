package garage

import (
	"context"
	"net/http"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/x/helper"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	cfg *Config

	// readyCheck is a map of functions for readyCheck-endpoint
	readyCheck      stats.MapCheckFunc
	readyCheckMutex sync.RWMutex

	// aliveCheck is a map functions for aliveCheck-endpoint
	aliveCheck      stats.MapCheckFunc
	aliveCheckMutex sync.RWMutex

	// metrics is a map functions for metrics-endpoint
	metrics      stats.MapMetricsOptions
	metricsMutex sync.RWMutex
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, collectorName, providerName, name string, cfg interface{}) (*Enity, error) {
	e, err := base.NewEnity(ctx, collectorName, providerName, name)
	if err != nil {
		return nil, errors.Wrap(err, "create base enity")
	}

	// Checking that passed config is OUR.
	config, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityOptions, "expected %q", helper.ObjName(Config{}))
	}

	prometheus.MustRegister(prometheus.NewBuildInfoCollector())

	return &Enity{Enity: e, cfg: config.SetDefault(), readyCheck: make(stats.MapCheckFunc), aliveCheck: make(stats.MapCheckFunc), metrics: make(stats.MapMetricsOptions)}, nil
}

func (e *Enity) errAnswer(ctx context.Context, w http.ResponseWriter, msg, code string) {
	answ := stats.ErrorAnswer{
		Body: stats.ErrorAnswerBody{
			Code: code,
			BaseAnswer: stats.BaseAnswer{
				StatusCode: http.StatusServiceUnavailable,
				Details:    msg,
			},
		},
	}

	w.WriteHeader(http.StatusFailedDependency)
	err := answ.WriteJSON(w)
	if err != nil {
		e.GetLogger(ctx).Err(err)
	}
}

func (e *Enity) readyCheckHandler(w http.ResponseWriter, r *http.Request) {
	defer e.readyCheckMutex.Unlock()
	e.readyCheckMutex.Lock()
	for key, f := range e.readyCheck {
		result, msg := f()
		if !result {
			e.errAnswer(r.Context(), w, msg, key)
			return
		}
	}

	answ := stats.ResultAnswer{Body: "ok"}
	err := answ.WriteJSON(w)
	if err != nil {
		e.GetLogger(r.Context()).Err(err)
	}
}

// RegisterReadyCheck should register a function for /health/ready
// endpoint.
func (e *Enity) RegisterReadyCheck(ctx context.Context, dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		e.GetLogger(ctx).Error().Msg("dependency name is empty")
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		e.GetLogger(ctx).Error().Msg("checkFunc is null")
		return stats.ErrCheckFuncIsNil
	}

	e.readyCheckMutex.Lock()
	e.readyCheck[dependencyName] = checkFunc
	e.readyCheckMutex.Unlock()

	return nil
}

func (e *Enity) aliveCheckHandler(w http.ResponseWriter, r *http.Request) {
	defer e.aliveCheckMutex.Unlock()
	e.aliveCheckMutex.Lock()
	for key, f := range e.aliveCheck {
		result, msg := f()
		if !result {
			e.errAnswer(r.Context(), w, msg, key)
			return
		}
	}

	answ := stats.ResultAnswer{Body: "ok"}
	err := answ.WriteJSON(w)
	if err != nil {
		e.GetLogger(r.Context()).Err(err)
	}
}

// RegisterAliveCheck should register a function for /health/alive
// endpoint.
func (e *Enity) RegisterAliveCheck(ctx context.Context, dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		e.GetLogger(ctx).Error().Msg("dependency name is empty")
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		e.GetLogger(ctx).Error().Msg("checkFunc is null")
		return stats.ErrCheckFuncIsNil
	}

	e.aliveCheckMutex.Lock()
	e.aliveCheck[dependencyName] = checkFunc
	e.aliveCheckMutex.Unlock()

	return nil
}

// RegisterMetric should register a metric of defined type. Passed
// metricName should be used only as internal identifier. Provider
// should provide instructions for using metricOptions as well as
// cast to appropriate type.
func (e *Enity) RegisterMetric(ctx context.Context, metricName string, options interface{}) error {
	if metricName == "" {
		e.GetLogger(ctx).Error().Msg("metric name is empty")
		return stats.ErrEmptyMetricName
	}

	metricOptions, ok := options.(*stats.MetricOptions)
	if !ok {
		return errors.Wrapf(stats.ErrInvalidMetricOptions, "expected %q", helper.ObjName(Config{}))
	}

	e.metricsMutex.Lock()
	e.metrics[metricName] = metricOptions
	prometheus.MustRegister(metricOptions.Metric.(prometheus.Collector))
	e.metricsMutex.Unlock()

	return nil
}

func (e *Enity) prometheusMiddleware(handler http.Handler) http.Handler {
	e.metricsMutex.Lock()
	for _, v := range e.metrics {
		v.MetricFunc(v.Metric)
	}
	e.metricsMutex.Unlock()

	return handler
}

// Start should start all things up.
func (e *Enity) Start(ctx context.Context) error {
	c, err := httpsrv.FromContext(ctx)
	if err != nil {
		return err
	}

	p, err := c.GetProvider(e.cfg.HTTPProviderName)
	if err != nil {
		return err
	}

	err = p.RegisterEndpoint(e.cfg.HTTPEnityName,
		"GET", stats.ReadyEndpoint, http.HandlerFunc(e.readyCheckHandler))
	if err != nil {
		return err
	}

	err = p.RegisterEndpoint(e.cfg.HTTPEnityName,
		"GET", stats.AliveEndpoint, http.HandlerFunc(e.aliveCheckHandler))
	if err != nil {
		return err
	}

	err = p.RegisterEndpoint(e.cfg.HTTPEnityName,
		"GET", stats.MetricsEndpoint, promhttp.Handler(), e.prometheusMiddleware)
	if err != nil {
		return err
	}

	return nil
}

func (e *Enity) Shutdown() error {
	return nil
}
