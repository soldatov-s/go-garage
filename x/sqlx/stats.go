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
	fullName := e.GetFullName()

	metrics := base.NewMapMetricsOptions()

	if _, err := metrics.AddGauge(fullName,
		"open connection",
		"open connection right now",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().OpenConnections), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddGauge(fullName,
		"max open connection",
		"max open connection",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().MaxOpenConnections), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddGauge(fullName,
		"in use",
		"connection in use right now",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().InUse), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddGauge(fullName,
		"wait duration",
		"wait duration",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().WaitDuration), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddGauge(fullName,
		"max idle closed",
		"max idle closed",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().MaxIdleClosed), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddGauge(fullName,
		"max life time closed",
		"max life time closed",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().MaxLifetimeClosed), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	if _, err := metrics.AddGauge(fullName,
		"idle",
		"idle",
		func(ctx context.Context) (float64, error) {
			return float64(e.GetConn().Stats().Idle), nil
		},
	); err != nil {
		return nil, errors.Wrap(err, "add gauge metric")
	}

	return metrics, nil
}
