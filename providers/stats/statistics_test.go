package stats

import (
	// stdlib
	"context"
	"testing"

	// local
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/stretchr/testify/require"
	// other
)

func TestStatisticsInitialize(t *testing.T) {
	ctx := context.Background()
	logger.Registrate(ctx)
	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

	Registrate(ctx)
	v := Get(ctx)
	require.NotNil(t, v)
}
