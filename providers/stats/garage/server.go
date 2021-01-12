package garage

import (
	"context"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/stats"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	ctx     context.Context
	log     zerolog.Logger
	name    string
	options *Config

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
func NewEnity(ctx context.Context, name string, opts interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	conn := &Enity{
		name: name,
		ctx:  logger.Registrate(ctx),
	}

	logger.Get(conn.ctx).GetLogger(stats.ProvidersName, nil).Info().Msgf("initializing enity " + name + "...")
	conn.log = logger.Get(conn.ctx).GetLogger(stats.ProvidersName, nil).With().Str("connection", name).Logger()

	// We should not attempt to establish connection if passed options
	// isn't OUR options.
	var ok bool
	conn.options, ok = opts.(*Config)
	if !ok {
		return nil, errors.ErrInvalidEnityOptions(Config{})
	}

	conn.options.Validate()

	conn.readyCheck = make(stats.MapCheckFunc)
	conn.aliveCheck = make(stats.MapCheckFunc)
	prometheus.MustRegister(prometheus.NewBuildInfoCollector())
	conn.metrics = make(stats.MapMetricsOptions)

	return conn, nil
}

func (e *Enity) errAnswer(w http.ResponseWriter, msg, code string) {
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
		e.log.Err(err)
	}
}

func (e *Enity) readyCheckHandler(w http.ResponseWriter, r *http.Request) {
	defer e.readyCheckMutex.Unlock()
	e.readyCheckMutex.Lock()
	for key, f := range e.readyCheck {
		result, msg := f()
		if !result {
			e.errAnswer(w, msg, key)
			return
		}
	}

	answ := stats.ResultAnswer{Body: "ok"}
	err := answ.WriteJSON(w)
	if err != nil {
		e.log.Err(err)
	}
}

// RegisterReadyCheck should register a function for /health/ready
// endpoint.
func (e *Enity) RegisterReadyCheck(dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		e.log.Error().Msg("dependency name is empty")
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		e.log.Error().Msg("checkFunc is null")
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
			e.errAnswer(w, msg, key)
			return
		}
	}

	answ := stats.ResultAnswer{Body: "ok"}
	err := answ.WriteJSON(w)
	if err != nil {
		e.log.Err(err)
	}
}

// RegisterAliveCheck should register a function for /health/alive
// endpoint.
func (e *Enity) RegisterAliveCheck(dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		e.log.Error().Msg("dependency name is empty")
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		e.log.Error().Msg("checkFunc is null")
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
func (e *Enity) RegisterMetric(metricName string, options interface{}) error {
	if metricName == "" {
		e.log.Error().Msg("metric name is empty")
		return stats.ErrEmptyMetricName
	}

	metricOptions, ok := options.(*stats.MetricOptions)
	if !ok {
		return stats.ErrInvalidMetricOptions(stats.MetricOptions{})
	}

	e.metricsMutex.Lock()
	e.metrics[metricName] = metricOptions
	prometheus.MustRegister(metricOptions.Metric.(prometheus.Collector))
	e.metricsMutex.Unlock()

	return nil
}

func (e *Enity) prometheusMiddleware(handler http.Handler) http.Handler {
	e.metricsMutex.Lock()
	for name, v := range e.metrics {
		v.MetricFunc(v.Metric)
		e.log.Debug().Msg("run metric " + name)
	}
	e.metricsMutex.Unlock()

	return handler
}

// Start should start all things up.
func (e *Enity) Start() error {
	p, err := httpsrv.GetProvider(e.ctx, e.options.HTTPProviderName)
	if err != nil {
		return err
	}
	err = p.RegisterEndpoint(e.options.HTTPEnityName,
		"GET", stats.ReadyEndpoint, http.HandlerFunc(e.readyCheckHandler))
	if err != nil {
		return err
	}

	err = p.RegisterEndpoint(e.options.HTTPEnityName,
		"GET", stats.AliveEndpoint, http.HandlerFunc(e.aliveCheckHandler))
	if err != nil {
		return err
	}

	err = p.RegisterEndpoint(e.options.HTTPEnityName,
		"GET", stats.MetricsEndpoint, promhttp.Handler(), e.prometheusMiddleware)
	if err != nil {
		return err
	}

	return nil
}

func (e *Enity) Shutdown() error {
	return nil
}
