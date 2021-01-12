package httpsrv

import (
	"context"
	"testing"

	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/stretchr/testify/require"
)

func TestHTTPServersInitialize(t *testing.T) {
	ctx := context.Background()
	logger.Registrate(ctx)
	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

	Registrate(ctx)
	v := Get(ctx)
	require.NotNil(t, v)
}
