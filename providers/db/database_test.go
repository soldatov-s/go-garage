package db

import (
	"context"
	"testing"

	"github.com/soldatov-s/go-garage/log"
	"github.com/stretchr/testify/require"
)

func TestCacheInitialize(t *testing.T) {
	ctx := context.Background()
	ctx, err := log.NewContextByConfig(ctx, &log.Config{Level: log.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})
	require.Nil(t, err)
}
