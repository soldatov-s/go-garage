package ch

// import (
// 	"context"
// 	"database/sql"
// 	"fmt"
// 	"os"
// 	"testing"

// 	"github.com/ory/dockertest"
// 	"github.com/soldatov-s/go-garage/providers/logger"
// 	"github.com/stretchr/testify/require"
// )

// const (
// 	testConnectionName = "testconnection"
// )

// type invalidType struct{}

// func initConnOptions() *ConnectionOptions {
// 	connOptions := &ConnectionOptions{
// 		DSN: "tcp://127.0.0.1:9000",
// 		MigrateOptions: &MigrateOptions{
// 			MigrationsType: MigrationTypeGoCode,
// 		},
// 		Timeout: 1,
// 	}

// 	return connOptions
// }

// func initializeClickHouseProvider(t *testing.T) *Provider {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	prov := NewProvider(ctx)
// 	require.Nil(t, prov)

// 	require.NotNil(t, prov.Entitys)

// 	return prov
// }

// // TestClickHouseProviderAppendToQueue
// // TestClickHouseProviderAppendToQueueWithEmptyConnectionName
// // TestClickHouseProviderAppendToQueueWithInvalidItem
// func TestClickHouseProviderCreateEnityWithGettingConnection(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderCreateEnityWithGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderCreateEnityWithGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderCreateEnityWithGettingConnectionAndInvalidConnectionStruct(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderCreateEnityWithoutGettingConnection(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderCreateEnityWithoutGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderCreateEnityWithoutGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

// 	err := prov.CreateEnity(testConnectionName, &invalidType{}, nil)
// 	if err != nil {
// 		t.Log(err)
// 	}

// 	require.NotNil(t, err)

// 	err = prov.Shutdown()
// 	if err != nil {
// 		t.Log(err)
// 	}
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderGetEnity(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderGetEnityWithEmptyConnectionName(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderGetEnityWithInvalidConnectionStruct(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderGetEnityWithNilAsConnectionStruct(t *testing.T) {
// 	prov := initializeClickHouseProvider(t)

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
// 	require.Nil(t, err)
// }

// func TestClickHouseProviderInitialize(t *testing.T) {
// 	initializeClickHouseProvider(t)
// }

// func TestClickHouseProviderRegisterMigration(t *testing.T) {
// 	// Init docker pool
// 	t.Log("init docker pool")

// 	var db *sql.DB
// 	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
// 	pool, err := dockertest.NewPool("unix:///var/run/docker.sock")
// 	if err != nil {
// 		t.Fatalf("Could not connect to docker: %s", err)
// 	}

// 	t.Log("run db docker")

// 	resource, err := pool.RunWithOptions(
// 		&dockertest.RunOptions{
// 			Repository: "yandex/clickhouse-server",
// 			Tag:        "20.8.11.17",
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
// 		connOptions.DSN = fmt.Sprintf("tcp://%s:%s", resource.Container.NetworkSettings.IPAddress, "9000")
// 	} else {
// 		connOptions.DSN = fmt.Sprintf("tcp://%s:%s", "localhost", resource.GetPort("9000/tcp"))
// 	}
// 	t.Logf("try to connect to DSN %s", connOptions.DSN)

// 	if err = pool.Retry(func() error {
// 		var err1 error
// 		db, err1 = sql.Open("clickhouse",
// 			fmt.Sprintf("%s?database=default&read_timeout=10&write_timeout=20&compress=true&debug=true", connOptions.DSN))
// 		if err1 != nil {
// 			return err1
// 		}
// 		return db.Ping()
// 	}); err != nil {
// 		t.Fatalf("Could not connect to database: %s", err)
// 	}

// 	// Run subtests
// 	t.Run("Normal", testClickHouseProviderRegisterMigrationNormal(connOptions))
// 	t.Run("OnlySchema", testClickHouseProviderRegisterMigrationSchema(connOptions))
// 	t.Run("WithEmptyConnectionName", testClickHouseProviderRegisterMigrationWithEmptyConnectionName(connOptions))
// 	t.Run("WithInvalidMigration", testClickHouseProviderRegisterMigrationWithInvalidMigration(connOptions))
// }

// func testClickHouseProviderRegisterMigrationNormal(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateOptions.Action = actionUp

// 		prov := initializeClickHouseProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConn, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConn)

// 		migration := &MigrationInCode{
// 			Name: "1_initial.go",
// 			Down: func(tx *sql.Tx) error {
// 				if _, err1 := tx.Exec(`DROP TABLE test_table`); err1 != nil {
// 					return err1
// 				}
// 				return nil
// 			},
// 			Up: func(tx *sql.Tx) error {
// 				if _, err1 := tx.Exec(`CREATE TABLE IF NOT EXISTS test_table (ID UInt8 NOT NULL, NAME String NOT NULL) engine=Memory`); err1 != nil {
// 					return err1
// 				}
// 				return nil
// 			},
// 		}

// 		err1 := prov.RegisterMigration(testConnectionName, migration)
// 		if err1 != nil {
// 			t.Log(err1)
// 		}

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		// Check created table structure.
// 		type ts struct {
// 			ColumnName string `db:"name"`
// 		}

// 		result := []ts{}

// 		err2 := conn.Conn.Select(
// 			&result,
// 			"SELECT name FROM system.tables WHERE name = 'test_table';",
// 		)
// 		if err2 != nil {
// 			t.Log(err2)
// 		}

// 		require.Nil(t, err2)

// 		t.Logf("%+v", result)

// 		// We have two columns in test table.
// 		for _, col := range result {
// 			if col.ColumnName == "name" {
// 				t.Fatal("Column 'name' isn't text!")
// 			}
// 		}

// 		err = prov.Shutdown()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)
// 	}
// }

// func testClickHouseProviderRegisterMigrationSchema(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateOptions.Action = actionUp

// 		prov := initializeClickHouseProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConn, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConn)

// 		conn.SetDatabaseName("production")

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		// Check created table structure.
// 		type ts struct {
// 			SchemaName string `db:"name"`
// 		}

// 		result := []ts{}

// 		err2 := conn.Conn.Select(
// 			&result,
// 			"SELECT name FROM system.databases WHERE name  = 'production';",
// 		)
// 		if err2 != nil {
// 			t.Log(err2)
// 		}

// 		require.Nil(t, err2)

// 		t.Logf("%+v", result)
// 	}
// }

// func testClickHouseProviderRegisterMigrationWithEmptyConnectionName(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateOptions.Action = actionUp

// 		prov := initializeClickHouseProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConn, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConn)

// 		migration := &MigrationInCode{
// 			Name: "1_initial.go",
// 			Down: func(tx *sql.Tx) error {
// 				if _, err := tx.Exec(`DROP TABLE test_table`); err != nil {
// 					return err
// 				}
// 				return nil
// 			},
// 			Up: func(tx *sql.Tx) error {
// 				if _, err := tx.Exec(`CREATE TABLE test_table (ID UInt8 NOT NULL, NAME STRING NOT NULL) engine=Memory`); err != nil {
// 					return err
// 				}
// 				return nil
// 			},
// 		}

// 		err1 := prov.RegisterMigration("", migration)
// 		if err1 != nil {
// 			t.Log(err1)
// 		}

// 		require.NotNil(t, err1)
// 	}
// }

// func testClickHouseProviderRegisterMigrationWithInvalidMigration(connOptions *ConnectionOptions) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateOptions.Action = actionUp

// 		prov := initializeClickHouseProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConn, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConn)

// 		err1 := prov.RegisterMigration(testConnectionName, &invalidType{})
// 		if err1 != nil {
// 			t.Log(err1)
// 		}

// 		require.NotNil(t, err1)
// 	}
// }

// // TestClickHouseProviderWaitForFlush
// // TestClickHouseProviderWaitForFlushWithEmptyConnectionName
