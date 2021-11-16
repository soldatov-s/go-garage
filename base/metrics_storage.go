package base

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
)

type MetricsStorage struct {
	metrics *MapMetricsOptions
}

func NewMetricsStorage() *MetricsStorage {
	return &MetricsStorage{
		metrics: NewMapMetricsOptions(),
	}
}

func (s *MetricsStorage) GetMetrics() *MapMetricsOptions {
	return s.metrics
}

func (s *MetricsStorage) PrometheusMiddleware(ctx context.Context, handler http.Handler) http.Handler {
	logger := zerolog.Ctx(ctx)
	for _, v := range s.GetMetrics().options {
		if v.Func == nil {
			continue
		}
		if err := v.Func(ctx, v.Metric); err != nil {
			logger.Err(err).Msg("handle metric")
		}
	}

	return handler
}
