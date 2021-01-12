package httpsrv

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistrate(t *testing.T) {
	ctx := context.Background()
	ctx = Registrate(ctx)

	require.NotNil(t, ctx)
}

func TestGet(t *testing.T) {
	ctx := context.Background()
	ctx = Registrate(ctx)

	require.NotNil(t, ctx)

	prov := Get(ctx)
	require.NotNil(t, prov)
}
