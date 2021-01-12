package rabbitmq

// import (
// 	//

// 	"context"
// 	"encoding/json"
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"

//
// 	"github.com/soldatov-s/go-garage/providers/logger"

//

// 	"github.com/ory/dockertest"
// 	"github.com/streadway/amqp"
// 	"github.com/stretchr/testify/require"
// )

// const (
// 	testConnectionName = "testconnection"
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

// func initConnOptionsPublisher() *ConnectionOptions {
// 	connOptions := &ConnectionOptions{
// 		DSN: "amqp://guest:guest@rabbitmq:5672",
// 		PublisherOptions: &PublisherOptions{
// 			RabbitBaseOptions: RabbitBaseOptions{
// 				ExchangeName: "test.events",
// 				RoutingKey:   "TEST_EVENT",
// 			},
// 		},
// 		BackoffPolicy: []time.Duration{
// 			2 * time.Second,
// 			4 * time.Second,
// 			8 * time.Second,
// 		},
// 	}

// 	dsnFromEnv, found := os.LookupEnv("DSN")
// 	if found {
// 		connOptions.DSN = dsnFromEnv
// 	}

// 	return connOptions
// }

// func initConnOptionsSubscribe() *ConnectionOptions {
// 	connOptions := &ConnectionOptions{
// 		DSN: "amqp://guest:guest@rabbitmq:5672",
// 		ConsumerOptions: &ConsumerOptions{
// 			RabbitBaseOptions: RabbitBaseOptions{
// 				ExchangeName: "test.events",
// 				RoutingKey:   "TEST_EVENT",
// 			},
// 			RabbitQueue:   "queue",
// 			RabbitConsume: "test",
// 		},
// 		BackoffPolicy: []time.Duration{
// 			2 * time.Second,
// 			4 * time.Second,
// 			8 * time.Second,
// 		},
// 	}

// 	dsnFromEnv, found := os.LookupEnv("DSN")
// 	if found {
// 		connOptions.DSN = dsnFromEnv
// 	}

// 	return connOptions
// }

// func initConnOptions() *ConnectionOptions {
// 	connOptions := &ConnectionOptions{
// 		DSN: "amqp://guest:guest@rabbitmq:5672",
// 		ConsumerOptions: &ConsumerOptions{
// 			RabbitBaseOptions: RabbitBaseOptions{
// 				ExchangeName: "test.events",
// 				RoutingKey:   "TEST_EVENT",
// 			},
// 			RabbitQueue:   "queue",
// 			RabbitConsume: "test",
// 		},
// 		PublisherOptions: &PublisherOptions{
// 			RabbitBaseOptions: RabbitBaseOptions{
// 				ExchangeName: "test.events",
// 				RoutingKey:   "TEST_EVENT",
// 			},
// 		},
// 		BackoffPolicy: []time.Duration{
// 			2 * time.Second,
// 			4 * time.Second,
// 			8 * time.Second,
// 		},
// 	}

// 	dsnFromEnv, found := os.LookupEnv("DSN")
// 	if found {
// 		connOptions.DSN = dsnFromEnv
// 	}

// 	return connOptions
// }

// func initializeRabbitmqProvider(t *testing.T) *Provider {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	prov := NewProvider(ctx)
// 	require.Nil(t, prov)

// 	require.NotNil(t, prov.Entitys)

// 	return prov
// }

// func TestRabbitmqsProviderCreateConnectionWithGettingConnection(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

// 	emptyConn := &Enity{}

// 	connOptions := initConnOptions()

// 	var conn *Enity

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

// func TestRabbitmqProviderCreateConnectionWithGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderCreateConnectionWithGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderCreateConnectionWithGettingConnectionAndInvalidConnectionStruct(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderCreateConnectionWithoutGettingConnection(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderCreateConnectionWithoutGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderCreateConnectionWithoutGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderGetConnection(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderGetConnectionWithEmptyConnectionName(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderGetConnectionWithInvalidConnectionStruct(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderGetConnectionWithNilAsConnectionStruct(t *testing.T) {
// 	prov := initializeRabbitmqProvider(t)

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

// func TestRabbitmqProviderInitialize(t *testing.T) {
// 	initializeRabbitmqProvider(t)
// }

// func TestRabbitmqProviderMsgsOperations(t *testing.T) {
// 	// Init docker pool
// 	t.Log("init docker pool")

// 	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
// 	pool, err := dockertest.NewPool("unix:///var/run/docker.sock")
// 	if err != nil {
// 		t.Fatalf("Could not connect to docker: %s", err)
// 	}

// 	t.Log("run db docker")

// 	resource, err := pool.RunWithOptions(
// 		&dockertest.RunOptions{
// 			Repository: "rabbitmq",
// 			Tag:        "3.8.5-management-alpine",
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
// 	connOptionsPublisher := initConnOptionsPublisher()
// 	connOptionsSubscribe := initConnOptionsSubscribe()

// 	if gitlab := os.Getenv("GITLAB"); gitlab != "" {
// 		connOptions.DSN = fmt.Sprintf("amqp://%s:%s", resource.Container.NetworkSettings.IPAddress, "5672")
// 		connOptionsPublisher.DSN = connOptions.DSN
// 		connOptionsSubscribe.DSN = connOptions.DSN
// 	} else {
// 		connOptions.DSN = fmt.Sprintf("amqp://%s:%s", "localhost", resource.GetPort("5672/tcp"))
// 		connOptionsPublisher.DSN = connOptions.DSN
// 		connOptionsSubscribe.DSN = connOptions.DSN
// 	}
// 	t.Logf("try to connect to DSN %s", connOptions.DSN)

// 	if err = pool.Retry(func() error {
// 		client, err1 := amqp.Dial(connOptions.DSN)
// 		if err1 != nil {
// 			return err1
// 		}

// 		_ = client.Close()

// 		return nil
// 	}); err != nil {
// 		t.Fatalf("Could not connect to database: %s", err)
// 	}

// 	// Run subtests
// 	t.Run("SendMessage", testRabbitmqProviderMsgsSendMessage(connOptionsPublisher))
// 	t.Run("Subscribe", testRabbitmqProviderMsgsSubscribe(connOptionsSubscribe))
// 	t.Run("SendMessageSubscribe", testRabbitmqProviderMsgsSendMessageSubscribe(connOptions))
// }

// func testRabbitmqProviderMsgsSendMessage(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRabbitmqProvider(t)

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

// 		err = prov.SendMessage(testConnectionName, newTestValue())
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

// func SubscribeHelper(t *testing.T) CosumeHandler {
// 	return func(data []byte) error {
// 		var request testValueType
// 		if err := json.Unmarshal(data, &request); err != nil {
// 			return err
// 		}

// 		require.Equal(t, *newTestValue(), request)
// 		t.Logf("received data %+v", &request)

// 		return nil
// 	}
// }

// func testRabbitmqProviderMsgsSubscribe(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRabbitmqProvider(t)

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

// 		t.Log("This message should appear when connection is established")

// 		err = prov.Subscribe(testConnectionName, &SubscribeOptions{ConsumeHndl: SubscribeHelper(t), Shutdownhndl: nil})
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }

// func testRabbitmqProviderMsgsSendMessageSubscribe(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		prov := initializeRabbitmqProvider(t)

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

// 		t.Log("This message should appear when connection is established")

// 		err = prov.Subscribe(testConnectionName, &SubscribeOptions{ConsumeHndl: SubscribeHelper(t), Shutdownhndl: nil})
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		conn.WaitForEstablishing()

// 		err = prov.SendMessage(testConnectionName, newTestValue())
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		time.Sleep(3 * time.Second)

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 	}
// }
