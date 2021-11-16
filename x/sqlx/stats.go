package sqlx

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
)

type EnityGateway interface {
	GetFullName() string
	GetConn() *sqlx.DB
}

type DBStatGateway interface {
	Stats() sql.DBStats
}

// nolint:funlen // long function
func StatsAsMetrics(e EnityGateway) (*base.MapMetricsOptions, error) {
	dbStats := e.GetConn().Stats()
	fullName := e.GetFullName()

	metrics := base.NewMapMetricsOptions()

	if _, err := metrics.AddMetricGauge(fullName,
		"open connection",
		"open connection right now",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.OpenConnections), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddMetricGauge(fullName,
		"max open connection",
		"max open connection",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.MaxOpenConnections), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddMetricGauge(fullName,
		"in use",
		"connection in use right now",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.InUse), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddMetricGauge(fullName,
		"wait duration",
		"wait duration",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.WaitDuration), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddMetricGauge(fullName,
		"max idle closed",
		"max idle closed",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.MaxIdleClosed), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddMetricGauge(fullName,
		"max life time closed",
		"max life time closed",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.MaxLifetimeClosed), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddMetricGauge(fullName,
		"idle",
		"idle",
		func(ctx context.Context) (float64, error) {
			return float64(dbStats.Idle), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	return metrics, nil
}
