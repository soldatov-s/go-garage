package envconfig

// import (
// 	"context"
// 	"testing"

// 	"github.com/soldatov-s/go-garage/providers/config"
// 	"github.com/stretchr/testify/require"
// )

// type testConfig struct {
// 	intValue int
// 	strValue string
// }

// func newTestConfig() testConfig {
// 	return testConfig{
// 		intValue: 1,
// 		strValue: "test",
// 	}
// }

// func TestRegistrate(t *testing.T) {
// 	ctx := context.Background()
// 	ctx = config.Registrate(ctx, newTestConfig())
// 	ctx, err := Registrate(ctx)
// 	require.Nil(t, err)
// 	require.NotNil(t, ctx)
// }
