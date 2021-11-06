package base

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/soldatov-s/go-garage/utils"
)

// MetricOptions descrbes struct with options for metrics
type MetricOptions struct {
	// Metric is a metric
	Metric interface{}
	// MetricFunc is a func for update metric
	MetricFunc func(metric interface{})
}

func NewMetricOptionsGauge(fullName, postfix, help string, value float64) *MetricOptions {
	return &MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: fullName + preparePotfix(postfix),
				Help: utils.JoinStrings(" ", fullName, help),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(value)
		},
	}
}

func preparePotfix(postfix string) string {
	postfix = "_" + strings.ReplaceAll(postfix, " ", "_")
	return postfix
}

type MapMetricsOptions map[string]*MetricOptions

func (mmo MapMetricsOptions) Append(src MapMetricsOptions) {
	for k, m := range src {
		mmo[k] = m
	}
}

func (mmo MapMetricsOptions) Add(key string, metric *MetricOptions) {
	mmo[key] = metric
}

func (mmo MapMetricsOptions) AddNewMetricGauge(fullName, postfix, help string, value float64) {
	mmo[fullName+preparePotfix(postfix)] = NewMetricOptionsGauge(fullName, postfix, help, value)
}

// Checking function should accept no parameters and return
// boolean value which will be interpreted as dependency service readiness
// and a string which should provide error text if dependency service
// isn't ready and something else if dependency service is ready (for
// example, dependency service's version).
type CheckFunc func() (bool, string)

type MapCheckFunc map[string]CheckFunc

func (mcf MapCheckFunc) Append(src MapCheckFunc) {
	for k, m := range src {
		mcf[k] = m
	}
}

func (mcf MapCheckFunc) Add(key string, checkFunc CheckFunc) {
	mcf[key] = checkFunc
}
