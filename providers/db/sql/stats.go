package sql

import (
	"database/sql"

	"github.com/soldatov-s/go-garage/providers/base"
)

type EnityGateway interface {
	GetFullName() string
}

type DBStatGateway interface {
	Stats() sql.DBStats
}

func GetDBStats(e EnityGateway, db DBStatGateway, metrics base.MapMetricsOptions) {
	dbStats := db.Stats()
	fullName := e.GetFullName()

	metrics.AddNewMetricGauge(fullName,
		"open connection",
		"open connection right now",
		func() float64 {
			return float64(dbStats.OpenConnections)
		},
	)

	metrics.AddNewMetricGauge(fullName,
		"max open connection",
		"max open connection",
		func() float64 {
			return float64(dbStats.MaxOpenConnections)
		},
	)

	metrics.AddNewMetricGauge(fullName,
		"in use",
		"connection in use right now",
		func() float64 {
			return float64(dbStats.InUse)
		},
	)

	metrics.AddNewMetricGauge(fullName,
		"wait duration",
		"wait duration",
		func() float64 {
			return float64(dbStats.WaitDuration)
		},
	)

	metrics.AddNewMetricGauge(fullName,
		"max idle closed",
		"max idle closed",
		func() float64 {
			return float64(dbStats.MaxIdleClosed)
		},
	)

	metrics.AddNewMetricGauge(fullName,
		"max life time closed",
		"max life time closed",
		func() float64 {
			return float64(dbStats.MaxLifetimeClosed)
		},
	)

	metrics.AddNewMetricGauge(fullName,
		"idle",
		"idle",
		func() float64 {
			return float64(dbStats.Idle)
		},
	)
}
