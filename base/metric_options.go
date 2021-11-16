package base

import (
	"context"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soldatov-s/go-garage/x/stringsx"
)

var (
	ErrEmptyOptionsName         = errors.New("empty options name")
	ErrFuncIsNil                = errors.New("pointer to function is nil")
	ErrOptionsIsNil             = errors.New("pointer to options is nil")
	ErrEmptyMetricName          = errors.New("empty metric name")
	ErrInvalidMetricOptions     = errors.New("passed metric options is not valid")
	ErrIsNotPrometheusCollector = errors.New("it is not a prometheus collector")
	ErrConflict                 = errors.New("conflict name")
	ErrFailedTypecastMetric     = errors.New("failed typecast metric")
	ErrInvalidCollector         = errors.New("invalid collector")
)

type MetricGateway interface {
	prometheus.Collector
}

type MetricFunc func(ctx context.Context, metric MetricGateway) error

// MetricOptions descrbes struct with options for metrics
type MetricOptions struct {
	// Metric name
	Name string
	// Metric is a metric
	Metric MetricGateway
	// Func is a func for update metric
	Func MetricFunc
}

func NewMetricOptions(name string, metric MetricGateway, f MetricFunc) *MetricOptions {
	return &MetricOptions{
		Name:   name,
		Metric: metric,
		Func:   f,
	}
}

type GaugeFunc func(ctx context.Context) (float64, error)

func NewMetricOptionsGauge(fullName, postfix, help string, f GaugeFunc) *MetricOptions {
	name := fullName + preparePotfix(postfix)
	gauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: name,
			Help: stringsx.JoinStrings(" ", fullName, help),
		})

	metricFunc := func(ctx context.Context, m MetricGateway) error {
		g, ok := m.(prometheus.Gauge)
		if !ok {
			return ErrFailedTypecastMetric
		}
		v, err := f(ctx)
		if err != nil {
			return errors.Wrap(err, "metric handler")
		}
		g.Set(v)

		return nil
	}
	return NewMetricOptions(name, gauge, metricFunc)
}

func NewIncCounter(fullName, postfix, help string) *MetricOptions {
	name := fullName + preparePotfix(postfix)
	counter := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: name,
			Help: stringsx.JoinStrings(" ", fullName, help),
		})

	metricFunc := func(ctx context.Context, m MetricGateway) error {
		c, ok := m.(prometheus.Counter)
		if !ok {
			return ErrFailedTypecastMetric
		}
		c.Inc()

		return nil
	}

	return NewMetricOptions(name, counter, metricFunc)
}

func NewHistogramVec(fullName, postfix, help string, args []string) *MetricOptions {
	name := fullName + preparePotfix(postfix)
	histogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Subsystem: fullName,
			Name:      name,
			Help:      help,
		},
		args,
	)

	return NewMetricOptions(name, histogram, nil)
}

func NewCounterVec(fullName, postfix, help string, args []string) *MetricOptions {
	name := fullName + preparePotfix(postfix)
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: fullName,
			Name:      name,
			Help:      help,
		},
		args,
	)

	return NewMetricOptions(name, counter, nil)
}

func preparePotfix(postfix string) string {
	postfix = "_" + strings.ReplaceAll(postfix, " ", "_")
	return postfix
}

type MapMetricsOptions struct {
	mu      sync.Mutex
	options map[string]*MetricOptions
}

func NewMapMetricsOptions() *MapMetricsOptions {
	return &MapMetricsOptions{
		options: make(map[string]*MetricOptions),
	}
}

func (mmo *MapMetricsOptions) Append(src *MapMetricsOptions) error {
	mmo.mu.Lock()
	defer mmo.mu.Unlock()

	for k, m := range src.options {
		if _, ok := mmo.options[k]; ok {
			return errors.Wrapf(ErrConflict, "name: %s", k)
		}

		mmo.options[k] = m
	}

	return nil
}

func (mmo *MapMetricsOptions) Add(options *MetricOptions) error {
	mmo.mu.Lock()
	defer mmo.mu.Unlock()

	if options == nil {
		return ErrOptionsIsNil
	}

	if options.Name == "" {
		return ErrEmptyMetricName
	}

	if options.Func == nil {
		return ErrFuncIsNil
	}

	if _, ok := mmo.options[options.Name]; ok {
		return errors.Wrapf(ErrConflict, "name: %s", options.Name)
	}

	mmo.options[options.Name] = options

	return nil
}

func (mmo *MapMetricsOptions) AddMetricGauge(fullName, postfix, help string, f GaugeFunc) (prometheus.Gauge, error) {
	metricOpts := NewMetricOptionsGauge(fullName, postfix, help, f)
	if err := mmo.Add(metricOpts); err != nil {
		return nil, errors.Wrap(err, "add to metrics map")
	}

	gauge, ok := metricOpts.Metric.(prometheus.Gauge)
	if !ok {
		return nil, ErrFailedTypecastMetric
	}

	return gauge, nil
}

func (mmo *MapMetricsOptions) AddHistogramVec(fullName, postfix, help string, args []string) (*prometheus.HistogramVec, error) {
	metricOpts := NewHistogramVec(fullName, postfix, help, args)
	if err := mmo.Add(metricOpts); err != nil {
		return nil, errors.Wrap(err, "add to metrics map")
	}

	histogram, ok := metricOpts.Metric.(*prometheus.HistogramVec)
	if !ok {
		return nil, ErrFailedTypecastMetric
	}

	return histogram, nil
}

func (mmo *MapMetricsOptions) AddCounterVec(fullName, postfix, help string, args []string) (*prometheus.CounterVec, error) {
	metricOpts := NewCounterVec(fullName, postfix, help, args)
	if err := mmo.Add(metricOpts); err != nil {
		return nil, errors.Wrap(err, "add to metrics map")
	}

	counter, ok := metricOpts.Metric.(*prometheus.CounterVec)
	if !ok {
		return nil, ErrFailedTypecastMetric
	}

	return counter, nil
}

func (mmo *MapMetricsOptions) AddMetricIncCounter(fullName, postfix, help string) (prometheus.Counter, error) {
	metricOpts := NewIncCounter(fullName, postfix, help)
	if err := mmo.Add(metricOpts); err != nil {
		return nil, errors.Wrap(err, "add to metrics map")
	}

	counter, ok := metricOpts.Metric.(prometheus.Counter)
	if !ok {
		return nil, ErrFailedTypecastMetric
	}

	return counter, nil
}

func (mmo *MapMetricsOptions) Registrate(register prometheus.Registerer) error {
	for _, v := range mmo.options {
		c, ok := v.Metric.(prometheus.Collector)
		if !ok {
			return ErrInvalidCollector
		}
		if err := register.Register(c); err != nil {
			return errors.Wrap(err, "registrate metric")
		}
	}

	return nil
}

type CheckOptions struct {
	// Check name
	Name string
	// Checking function should accept no parameters and return
	// boolean value which will be interpreted as dependency service readiness
	// and a string which should provide error text if dependency service
	// isn't ready and something else if dependency service is ready (for
	// example, dependency service's version).
	CheckFunc func(ctx context.Context) error
}

type MapCheckOptions struct {
	mu      sync.RWMutex
	options map[string]*CheckOptions
}

func NewMapCheckOptions() *MapCheckOptions {
	return &MapCheckOptions{
		options: make(map[string]*CheckOptions),
	}
}

func (mcf *MapCheckOptions) Append(src *MapCheckOptions) error {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()

	for k, m := range src.options {
		if _, ok := mcf.options[k]; ok {
			return errors.Wrapf(ErrConflict, "name: %s", k)
		}

		mcf.options[k] = m
	}

	return nil
}

func (mcf *MapCheckOptions) Add(options *CheckOptions) error {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()

	if options == nil {
		return ErrOptionsIsNil
	}

	if options.Name == "" {
		return ErrEmptyOptionsName
	}

	if options.CheckFunc == nil {
		return ErrFuncIsNil
	}

	if _, ok := mcf.options[options.Name]; ok {
		return errors.Wrapf(ErrConflict, "name: %s", options.Name)
	}

	mcf.options[options.Name] = options

	return nil
}
