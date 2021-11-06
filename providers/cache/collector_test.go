package cache

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
	ctx, err = NewContext(ctx)
	require.Nil(t, err)
	v, err := FromContext(ctx)
	require.Nil(t, err)
	require.NotNil(t, v)
}
