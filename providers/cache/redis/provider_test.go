package redis

// import (
// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"testing"

// 	"github.com/go-redis/redis"
// 	"github.com/ory/dockertest"
// 	"github.com/soldatov-s/go-garage/providers/cache"
// 	"github.com/soldatov-s/go-garage/providers/logger"
// 	"github.com/stretchr/testify/require"
// )

// const (
// 	testConnectionName = "testconnection"
// 	testKey            = "testKey"
// 	testKeyPrefix      = "testtest_"
// )

// type invalidType struct{}
// type testValueType struct {
// 	ID   int
// 	Name string
// }

// func (c *testValueType) UnmarshalBinary(data []byte) error {
// 	return json.Unmarshal(data, c)
// }

// func (c *testValueType) MarshalBinary() ([]byte, error) {
// 	return json.Marshal(c)
// }

// func newTestValue() *testValueType {
// 	return &testValueType{
// 		ID:   123,
// 		Name: "Ivan",
// 	}
// }

// func initConnOptions() *ConnectionOptions {
// 	connOptions := &ConnectionOptions{
// 		DSN:     "redis://:@localhost:5432",
// 		Timeout: 1,
// 	}

// 	dsnFromEnv, found := os.LookupEnv("DSN")
// 	if found {
// 		connOptions.DSN = dsnFromEnv
// 	}

// 	return connOptions
// }

// func initializeRedisProvider(t *testing.T) *Provider {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	prov := NewProvider(ctx)
// 	require.Nil(t, prov)

// 	require.NotNil(t, prov.Entitys)

// 	return prov
// }

// func TestRedisProviderCreateEnityWithGettingConnection(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	emptyConn := &Enity{}

// 	connOptions := initConnOptions()

// 	conn := &Enity{}

// 	err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.Nil(t, err)
// 	require.IsType(t, &Enity{}, conn)
// 	require.NotEqual(t, emptyConn, conn)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderCreateEnityWithGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	emptyConn := &Enity{}

// 	connOptions := initConnOptions()

// 	conn := &Enity{}

// 	err := prov.CreateEnity("", connOptions, &conn)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.NotNil(t, err)
// 	require.IsType(t, &Enity{}, conn)
// 	require.Equal(t, emptyConn, conn)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderCreateEnityWithGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	emptyConn := &Enity{}

// 	conn := &Enity{}

// 	err := prov.CreateEnity(testConnectionName, &invalidType{}, &conn)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.NotNil(t, err)
// 	require.IsType(t, &Enity{}, conn)
// 	require.Equal(t, emptyConn, conn)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderCreateEnityWithGettingConnectionAndInvalidConnectionStruct(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	emptyConn := &Enity{}

// 	connOptions := initConnOptions()

// 	conn := &Enity{}

// 	err := prov.CreateEnity(testConnectionName, connOptions, &invalidType{})
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.NotNil(t, err)
// 	require.IsType(t, &Enity{}, conn)
// 	require.Equal(t, emptyConn, conn)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderCreateEnityWithoutGettingConnection(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	connOptions := initConnOptions()

// 	err := prov.CreateEnity(testConnectionName, connOptions, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.Nil(t, err)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderCreateEnityWithoutGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	connOptions := initConnOptions()

// 	err := prov.CreateEnity("", connOptions, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.NotNil(t, err)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderCreateEnityWithoutGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	err := prov.CreateEnity(testConnectionName, &invalidType{}, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.NotNil(t, err)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderGetEnity(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	connOptions := initConnOptions()

// 	err := prov.CreateEnity(testConnectionName, connOptions, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.Nil(t, err)

// 	emptyConn := &Enity{}

// 	conn := &Enity{}

// 	err1 := prov.GetEnity(testConnectionName, &conn)
// 	if err1 != nil {
// 		t.Log(err1)
// 	}

// 	require.Nil(t, err1)
// 	require.IsType(t, &Enity{}, conn)
// 	require.NotEqual(t, emptyConn, conn)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderGetEnityWithEmptyConnectionName(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	connOptions := initConnOptions()

// 	err := prov.CreateEnity(testConnectionName, connOptions, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.Nil(t, err)

// 	emptyConn := &Enity{}

// 	conn := &Enity{}

// 	err1 := prov.GetEnity("", &conn)
// 	if err1 != nil {
// 		t.Log(err1)
// 	}

// 	require.NotNil(t, err1)
// 	require.IsType(t, &Enity{}, conn)
// 	require.Equal(t, emptyConn, conn)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderGetEnityWithInvalidConnectionStruct(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	connOptions := initConnOptions()

// 	err := prov.CreateEnity(testConnectionName, connOptions, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.Nil(t, err)

// 	err1 := prov.GetEnity(testConnectionName, &invalidType{})
// 	if err1 != nil {
// 		t.Log(err1)
// 	}

// 	require.NotNil(t, err1)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderGetEnityWithNilAsConnectionStruct(t *testing.T) {
// 	prov := initializeRedisProvider(t)

// 	connOptions := initConnOptions()

// 	err := prov.CreateEnity(testConnectionName, connOptions, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.Nil(t, err)

// 	err1 := prov.GetEnity(testConnectionName, nil)
// 	if err1 != nil {
// 		t.Log(err1)
// 	}

// 	require.NotNil(t, err1)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// }

// func TestRedisProviderInitialize(t *testing.T) {
// 	initializeRedisProvider(t)
// }

// func TestRedisProviderCacheOperations(t *testing.T) {
// 	// Init docker pool
// 	t.Log("init docker pool")

// 	var rcl *redis.Client
// 	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
// 	pool, err := dockertest.NewPool("unix:///var/run/docker.sock")
// 	if err != nil {
// 		t.Fatalf("Could not connect to docker: %s", err)
// 	}

// 	t.Log("run db docker")

// 	resource, err := pool.RunWithOptions(
// 		&dockertest.RunOptions{
// 			Repository: "redis",
// 			Tag:        "alpine3.12",
// 			Env:        []string{"ALLOW_EMPTY_PASSWORD=yes"},
// 		})
// 	if err != nil {
// 		t.Fatalf("Could not start resource: %s", err)
// 	}

// 	defer func() {
// 		// When you're done, kill and remove the container
// 		err = pool.Purge(resource)
// 		if err != nil {
// 			t.Fatal(err)
// 		}
// 	}()

// 	connOptions := initConnOptions()

// 	if gitlab := os.Getenv("GITLAB"); gitlab != "" {
// 		connOptions.DSN = fmt.Sprintf("redis://%s:%s", resource.Container.NetworkSettings.IPAddress, "6379")
// 	} else {
// 		connOptions.DSN = fmt.Sprintf("redis://%s:%s", "localhost", resource.GetPort("6379/tcp"))
// 	}
// 	t.Logf("try to connect to DSN %s", connOptions.DSN)

// 	if err = pool.Retry(func() error {
// 		var err1 error
// 		opt, err1 := redis.ParseURL(connOptions.DSN)
// 		if err1 != nil {
// 			return err1
// 		}

// 		rcl = redis.NewClient(opt)
// 		return rcl.Ping().Err()
// 	}); err != nil {
// 		t.Fatalf("Could not connect to database: %s", err)
// 	}

// 	connOptions.KeyPrefix = testKeyPrefix

// 	// Run subtests
// 	t.Run("SetValue", testRedisProviderCacheOperatiosSetValue(connOptions))
// 	t.Run("GetValue", testRedisProviderCacheOperatiosGetValue(connOptions))
// 	t.Run("DeleteValue", testRedisProviderCacheOperatiosDeleteValue(connOptions))
// 	t.Run("Clear", testRedisProviderCacheOperatiosClear(connOptions))
// 	t.Run("Size", testRedisProviderCacheOperatiosSize(connOptions))
// }

// func testRedisProviderCacheOperatiosSetValue(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRedisProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConnect, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConnect)

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		err = prov.Set(testConnectionName, testKey, newTestValue())
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }

// func testRedisProviderCacheOperatiosGetValue(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRedisProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConnect, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConnect)

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		var outValue testValueType
// 		err = prov.Get(testConnectionName, testKey, &outValue)
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		require.Equal(t, *newTestValue(), outValue)
// 		t.Logf("Value from cache %+v", outValue)

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }

// func testRedisProviderCacheOperatiosDeleteValue(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRedisProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConnect, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConnect)

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		err = prov.Delete(testConnectionName, testKey)
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		var outValue testValueType
// 		err = prov.Get(testConnectionName, testKey, &outValue)
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Equal(t, cache.ErrNotFoundInCache, err)

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }

// func testRedisProviderCacheOperatiosClear(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRedisProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConnect, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConnect)

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		err = prov.Set(testConnectionName, testKey, newTestValue())
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		err = prov.ClearConnection(testConnectionName)
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		var outValue testValueType
// 		err = prov.Get(testConnectionName, testKey, &outValue)
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Equal(t, cache.ErrNotFoundInCache, err)

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }

// func testRedisProviderCacheOperatiosSize(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRedisProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConnect, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConnect)

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		err = prov.Set(testConnectionName, testKey+"1", newTestValue())
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		err = prov.Set(testConnectionName, testKey+"2", newTestValue())
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		size := conn.Size()
// 		require.Equal(t, 2, size)

// 		err = prov.Delete(testConnectionName, testKey+"1")
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		size = conn.Size()
// 		require.Equal(t, 1, size)

// 		err = prov.ClearConnection(testConnectionName)
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		size = conn.Size()
// 		require.Equal(t, 0, size)

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }
