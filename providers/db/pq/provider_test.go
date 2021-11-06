package pq

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

// func initConnOptions() *Config {
// 	connOptions := &Config{
// 		DSN: "postgres://postgres:postgres@localhost:5432/test",
// 		MigrateConfig: &MigrateConfig{
// 			MigrationsType: MigrationTypeGoCode,
// 		},
// 		Timeout: 1,
// 	}

// 	return connOptions
// }

// func initializePostgreSQLProvider(t *testing.T) *Provider {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	prov := NewProvider(ctx)
// 	require.Nil(t, prov)

// 	require.NotNil(t, prov.Entitys)

// 	return prov
// }

// // TestPostgreSQLProviderAppendToQueue
// // TestPostgreSQLProviderAppendToQueueWithEmptyConnectionName
// // TestPostgreSQLProviderAppendToQueueWithInvalidItem
// func TestPostgreSQLProviderCreateConnectionWithGettingConnection(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderCreateConnectionWithGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderCreateConnectionWithGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderCreateConnectionWithGettingConnectionAndInvalidConnectionStruct(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderCreateConnectionWithoutGettingConnection(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderCreateConnectionWithoutGettingConnectionAndEmptyConnectionName(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderCreateConnectionWithoutGettingConnectionAndInvalidOptions(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderGetConnection(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderGetConnectionWithEmptyConnectionName(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderGetConnectionWithInvalidConnectionStruct(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderGetConnectionWithNilAsConnectionStruct(t *testing.T) {
// 	prov := initializePostgreSQLProvider(t)

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

// func TestPostgreSQLProviderInitialize(t *testing.T) {
// 	initializePostgreSQLProvider(t)
// }

// func TestPostgreSQLProviderRegisterMigration(t *testing.T) {
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
// 			Repository: "postgres",
// 			Tag:        "12.3-alpine",
// 			Env:        []string{"POSTGRES_USER=postgres", "POSTGRES_PASSWORD=postgres", "POSTGRES_DB=test"},
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
// 		connOptions.DSN = fmt.Sprintf("postgres://postgres:postgres@%s:%s/%s", resource.Container.NetworkSettings.IPAddress, "5432", "test")
// 	} else {
// 		connOptions.DSN = fmt.Sprintf("postgres://postgres:postgres@%s:%s/%s", "localhost", resource.GetPort("5432/tcp"), "test")
// 	}
// 	t.Logf("try to connect to DSN %s", connOptions.DSN)

// 	if err = pool.Retry(func() error {
// 		var err1 error
// 		db, err1 = sql.Open("postgres",
// 			fmt.Sprintf("%s?sslmode=disable", connOptions.DSN))
// 		if err1 != nil {
// 			return err1
// 		}
// 		return db.Ping()
// 	}); err != nil {
// 		t.Fatalf("Could not connect to database: %s", err)
// 	}

// 	// Run subtests
// 	t.Run("Normal", testPostgreSQLProviderRegisterMigrationNormal(connOptions))
// 	t.Run("OnlySchema", testPostgreSQLProviderRegisterMigrationSchema(connOptions))
// 	t.Run("WithEmptyConnectionName", testPostgreSQLProviderRegisterMigrationWithEmptyConnectionName(connOptions))
// 	t.Run("WithInvalidMigration", testPostgreSQLProviderRegisterMigrationWithInvalidMigration(connOptions))
// }

// func testPostgreSQLProviderRegisterMigrationNormal(connOptions *Config) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateConfig.Action = actionUp

// 		prov := initializePostgreSQLProvider(t)

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
// 				if _, err1 := tx.Exec(`CREATE TABLE IF NOT EXISTS test_table (ID INTEGER NOT NULL, NAME TEXT NOT NULL)`); err1 != nil {
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
// 			ColumnName string `db:"column_name"`
// 			DataType   string `db:"data_type"`
// 		}

// 		result := []ts{}

// 		err2 := conn.Conn.Select(
// 			&result,
// 			"SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'test_table';",
// 		)
// 		if err2 != nil {
// 			t.Log(err2)
// 		}

// 		require.Nil(t, err2)

// 		t.Logf("%+v", result)

// 		// We have two columns in test table.
// 		for _, col := range result {
// 			if col.ColumnName == "id" && col.DataType != "integer" {
// 				t.Fatal("Column 'id' isn't an integer!")
// 			}

// 			if col.ColumnName == "name" && col.DataType != "text" {
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

// func testPostgreSQLProviderRegisterMigrationSchema(connOptions *Config) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateConfig.Action = actionUp

// 		prov := initializePostgreSQLProvider(t)

// 		var conn *Enity

// 		err := prov.CreateEnity(testConnectionName, connOptions, &conn)
// 		if err != nil {
// 			t.Log(err)
// 		}

// 		require.Nil(t, err)
// 		require.NotNil(t, conn.options)
// 		testConn, _ := prov.Entitys.Load(testConnectionName)
// 		require.Same(t, conn, testConn)

// 		conn.SetSchemaName("production")

// 		err = prov.Start()
// 		if err != nil {
// 			t.Log(err)
// 		}
// 		require.Nil(t, err)

// 		conn.WaitForEstablishing()
// 		t.Log("This message should appear when connection is established")

// 		// Check created table structure.
// 		type ts struct {
// 			SchemaName string `db:"schema_name"`
// 		}

// 		result := []ts{}

// 		err2 := conn.Conn.Select(
// 			&result,
// 			"SELECT schema_name FROM information_schema.schemata WHERE schema_name = 'production';",
// 		)
// 		if err2 != nil {
// 			t.Log(err2)
// 		}

// 		require.Nil(t, err2)

// 		t.Logf("%+v", result)
// 	}
// }

// func testPostgreSQLProviderRegisterMigrationWithEmptyConnectionName(connOptions *Config) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateConfig.Action = actionUp

// 		prov := initializePostgreSQLProvider(t)

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
// 				if _, err := tx.Exec(`CREATE TABLE test_table (ID INTEGER NOT NULL, NAME TEXT NOT NULL)`); err != nil {
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

// func testPostgreSQLProviderRegisterMigrationWithInvalidMigration(connOptions *Config) func(t *testing.T) {
// 	return func(t *testing.T) {
// 		connOptions.MigrateConfig.Action = actionUp

// 		prov := initializePostgreSQLProvider(t)

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

// // TestPostgreSQLProviderWaitForFlush
// // TestPostgreSQLProviderWaitForFlushWithEmptyConnectionName
