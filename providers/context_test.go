package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	providerName = "test"
)

type testProvider struct {
	name string
}

func newTestProvider() *testProvider {
	return &testProvider{
		name: providerName,
	}
}

func TestCreateProviders(t *testing.T) {
	ctx := context.Background()
	ctx, p := Create(ctx)

	require.NotNil(t, ctx)
	require.NotNil(t, p)
}

func TestGetProviders(t *testing.T) {
	ctx := context.Background()
	ctx, p := Create(ctx)

	require.NotNil(t, ctx)
	require.NotNil(t, p)

	p1 := Get(ctx)
	require.NotNil(t, p)

	require.Equal(t, p, p1)
}

func TestGetProvidersByName(t *testing.T) {
	ctx := context.Background()
	ctx, p := Create(ctx)

	require.NotNil(t, ctx)
	require.NotNil(t, p)

	p1 := GetByName(ctx, providerName)
	require.Nil(t, p1)

	p.Store(providerName, newTestProvider)

	p2 := GetByName(ctx, providerName)
	require.NotNil(t, p2)
}

func TestRegistrateProvidersByName(t *testing.T) {
	ctx := context.Background()

	ctx = RegistrateByName(ctx, providerName, newTestProvider())
	require.NotNil(t, ctx)

	p1 := GetByName(ctx, "test")
	require.NotNil(t, p1)

	require.Equal(t, providerName, p1.(*testProvider).name)
}
