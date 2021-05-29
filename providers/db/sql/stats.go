package sql

import (
	"github.com/jmoiron/sqlx"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/utils"
)

type EnityGateway interface {
	GetFullName() string
}

func GetDBStats(e EnityGateway, db *sqlx.DB, metrics base.MapMetricsOptions) {
	dbStats := db.Stats()

	// nolint : dupl
	metrics[e.GetFullName()+"_open_connection"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_open_connection",
				Help: utils.JoinStrings(" ", e.GetFullName(), "open connection right now"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.OpenConnections))
		},
	}

	// nolint : dupl
	metrics[e.GetFullName()+"_max_open_connection"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_max_open_connection",
				Help: utils.JoinStrings(" ", e.GetFullName(), "max open connection"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxOpenConnections))
		},
	}

	// nolint : dupl
	metrics[e.GetFullName()+"_in_use"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_in_use",
				Help: utils.JoinStrings(" ", e.GetFullName(), "connection in use right now"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.InUse))
		},
	}

	// nolint : dupl
	metrics[e.GetFullName()+"_wait_duration"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_wait_duration",
				Help: utils.JoinStrings(" ", e.GetFullName(), "wait duration"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.WaitDuration))
		},
	}

	// nolint : dupl
	metrics[e.GetFullName()+"_max_idle_closed"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_max_idle_closed",
				Help: utils.JoinStrings(" ", e.GetFullName(), "max idle closed"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxIdleClosed))
		},
	}

	// nolint : dupl
	metrics[e.GetFullName()+"_max_life_time_closed"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_max_life_time_closed",
				Help: utils.JoinStrings(" ", e.GetFullName(), "max life time closed"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.MaxLifetimeClosed))
		},
	}

	// nolint : dupl
	metrics[e.GetFullName()+"_idle"] = &base.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: e.GetFullName() + "_idle",
				Help: utils.JoinStrings(" ", e.GetFullName(), "idle"),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(float64(dbStats.Idle))
		},
	}
}
