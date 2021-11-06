package base

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testProviderName = "test"
)

type testProvider struct {
	name string
}

func newTestProvider() *testProvider {
	return &testProvider{
		name: testProviderName,
	}
}

func TestNewContext(t *testing.T) {
	ctx, err := NewContext(context.Background())
	require.Nil(t, err)
	require.NotNil(t, ctx)
}

func TestFromContext(t *testing.T) {
	ctx, err := NewContext(context.Background())
	require.Nil(t, err)
	require.NotNil(t, ctx)

	p, err := FromContext(ctx)
	require.Nil(t, err)
	require.NotNil(t, p)
}

func TestFromContextByName(t *testing.T) {
	p := newTestProvider()
	ctx, err := NewContextByName(context.Background(), testProviderName, p)
	require.Nil(t, err)
	require.NotNil(t, ctx)

	p1, err := FromContextByName(ctx, testProviderName)
	require.Nil(t, err)
	require.NotNil(t, p)
	require.Equal(t, p, p1)
}

func TestNewContextByName(t *testing.T) {
	ctx, err := NewContextByName(context.Background(), testProviderName, newTestProvider)
	require.Nil(t, err)
	require.NotNil(t, ctx)
}
