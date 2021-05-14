package httpsrv

import (
	"context"
	"testing"

	"github.com/soldatov-s/go-garage/log"
	"github.com/stretchr/testify/require"
)

func TestHTTPServerInitialize(t *testing.T) {
	ctx := context.Background()
	ctx, err := log.NewContextByConfig(ctx, &log.Config{Level: log.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})
	require.Nil(t, err)
	ctx, err = NewContext(ctx)
	require.Nil(t, err)
	v, err := FromContext(ctx)
	require.Nil(t, err)
	require.NotNil(t, v)
}

// func TestRegistrate(t *testing.T) {
// 	ctx := context.Background()
// 	ctx = Registrate(ctx)

// 	require.NotNil(t, ctx)
// }

// func TestGet(t *testing.T) {
// 	ctx := context.Background()
// 	ctx = Registrate(ctx)

// 	require.NotNil(t, ctx)

// 	prov := Get(ctx)
// 	require.NotNil(t, prov)
// }
