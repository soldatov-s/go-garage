package log

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

type MetricErrorHook struct {
	metric prometheus.Counter
}

func NewMetricErrorHook(metric prometheus.Counter) *MetricErrorHook {
	return &MetricErrorHook{metric: metric}
}

func (h *MetricErrorHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	if level != zerolog.ErrorLevel {
		return
	}

	h.metric.Inc()
}

type MetricWarnHook struct {
	metric prometheus.Counter
}

func NewMetricWarnHook(metric prometheus.Counter) *MetricErrorHook {
	return &MetricErrorHook{metric: metric}
}

func (h *MetricWarnHook) Run(e *zerolog.Event, level zerolog.Level, message string) {
	if level != zerolog.WarnLevel {
		return
	}

	h.metric.Inc()
}
