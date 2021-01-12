package echo

// import (
// 	"context"
// 	"testing"
// 	"time"

// 	"github.com/soldatov-s/go-garage/providers/httpsrv"
// 	"github.com/soldatov-s/go-garage/providers/logger"
// 	"github.com/stretchr/testify/require"
// )

// const (
// 	testAPIVersion          = "1"
// 	testServerListenAddress = "127.0.0.1:60000"
// 	testServerName          = "testsrv"
// 	testCertPath            = "./testssl/certs/server.crt"
// 	testKeyPath             = "./testssl/certs/server.key"
// )

// type invalidType struct{}

// func initializeProvider(t *testing.T) *Provider {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	prov := NewProvider(ctx)
// 	require.Nil(t, prov)

// 	return prov
// }

// func TestInitialization(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p)
// }

// func TestCreateAPIVersionGroupWithoutCreatingServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p)

// 	err := p.CreateAPIVersionGroup(testServerName, testAPIVersion, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.NotNil(t, err)
// 	require.Equal(t, httpsrv.ErrServerNotFound, err)
// }

// func TestCreateAPIVersionGroupWithGettingGroup(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress})
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	group := &Group{}

// 	emptyGroup := &Group{}

// 	err1 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, &group)
// 	if err1 != nil {
// 		t.Log(err1.Error())
// 	}

// 	require.IsType(t, &Group{}, group)
// 	require.NotEqual(t, emptyGroup, group)
// 	require.Nil(t, err1)
// }

// func TestCreateAPIVersionGroupWithGettingGroupAndInvalidType(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	group := &invalidType{}

// 	emptyGroup := &Group{}

// 	err1 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, group)
// 	if err1 != nil {
// 		t.Log(err1.Error())
// 	}

// 	require.NotNil(t, err1)
// 	require.IsType(t, &invalidType{}, group)
// 	require.NotEqual(t, emptyGroup, group)
// }

// func TestCreateAPIVersionGroupWithoutGettingGroup(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	err1 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, nil)
// 	if err1 != nil {
// 		t.Log(err1.Error())
// 	}

// 	require.Nil(t, err)
// 	testServer, _ := p.Entitys.Load(testServerName)
// 	require.Len(t, testServer.(*Enity).apiGroups, 1)
// }

// func TestCreateAPIVersionGroupWithSameVersionAndWithGettingGroup(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	emptyGroup := &Group{}

// 	group1 := &Group{}

// 	err1 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, &group1)
// 	if err1 != nil {
// 		t.Log(err1.Error())
// 	}

// 	require.NotNil(t, group1)
// 	require.IsType(t, &Group{}, group1)
// 	require.NotEqual(t, emptyGroup, group1)
// 	require.Nil(t, err1)

// 	group2 := &Group{}

// 	err2 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, &group2)
// 	if err2 != nil {
// 		t.Log(err2.Error())
// 	}

// 	require.NotNil(t, group2)
// 	require.IsType(t, &Group{}, group2)
// 	require.NotEqual(t, emptyGroup, group2)
// 	require.Nil(t, err2)
// 	require.Equal(t, group1, group2)
// }

// func TestCreateAPIVersionGroupWithSameVersionAndWithoutGettingGroup(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	err1 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, nil)
// 	if err1 != nil {
// 		t.Log(err1.Error())
// 	}

// 	require.Nil(t, err1)

// 	err2 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, nil)
// 	if err2 != nil {
// 		t.Log(err2.Error())
// 	}

// 	require.Nil(t, err2)
// }

// func TestCreateServerWithGettingServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	srv := &Enity{}

// 	emptySrv := &Enity{}

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, &srv)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.IsType(t, &Enity{}, srv)
// 	require.NotEqual(t, emptySrv, srv)
// 	testServer, _ := p.Entitys.Load(testServerName)
// 	require.Same(t, srv, testServer)
// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)
// }

// func TestCreateServerWithoutGettingServerAndInvalidName(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity("", &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.NotNil(t, err)
// 	require.Len(t, p.Entitys, 0)
// }

// func TestCreateServerWithoutGettingServerAndInvalidOptionsAddress(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: ""}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.NotNil(t, err)
// 	require.Len(t, p.Entitys, 0)
// }

// func TestCreateServerWithoutGettingServerAndInvalidOptionsType(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &invalidType{}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.NotNil(t, err)
// 	require.Len(t, p.Entitys, 0)
// }

// func TestCreateServerWithGettingServerAndInvalidServerType(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	srv := &invalidType{}

// 	emptySrv := &Enity{}

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, srv)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.NotNil(t, err)
// 	require.Len(t, p.Entitys, 1)
// 	require.IsType(t, &invalidType{}, srv)
// 	require.NotEqual(t, emptySrv, srv)
// }

// func TestCreateServerWithoutGettingServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)
// }

// func TestCreateServerWithSameNameWithoutGettingServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	err1 := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	require.Nil(t, err1)
// 	require.Len(t, p.Entitys, 1)
// }

// func TestCreateServerWithSameNameWithGettingServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	srv1 := &Enity{}

// 	emptySrv := &Enity{}

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, &srv1)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)
// 	require.IsType(t, &Enity{}, srv1)
// 	require.NotEqual(t, emptySrv, srv1)

// 	srv2 := &Enity{}

// 	err1 := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, &srv2)

// 	require.Nil(t, err1)
// 	require.Len(t, p.Entitys, 1)
// 	require.IsType(t, &Enity{}, srv1)
// 	require.NotEqual(t, emptySrv, srv2)
// 	require.Equal(t, srv1, srv2)
// }

// func TestGetAPIVersionGroup(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	emptyGroup := &Group{}

// 	createdGroup := &Group{}

// 	err1 := p.CreateAPIVersionGroup(testServerName, testAPIVersion, &createdGroup)
// 	if err1 != nil {
// 		t.Log(err1.Error())
// 	}

// 	require.NotNil(t, createdGroup)
// 	require.IsType(t, &Group{}, createdGroup)
// 	require.NotEqual(t, emptyGroup, createdGroup)
// 	require.Nil(t, err)

// 	var requestedGroup *Group

// 	err2 := p.GetAPIVersionGroup(testServerName, testAPIVersion, &requestedGroup)
// 	if err2 != nil {
// 		t.Log(err2)
// 	}

// 	require.Nil(t, err2)
// 	require.NotNil(t, requestedGroup)
// 	require.IsType(t, &Group{}, requestedGroup)
// 	require.Exactly(t, createdGroup, requestedGroup)
// 	require.NotEqual(t, emptyGroup, requestedGroup)
// }

// func TestGetAPIVersionGroupWithoutCreatedServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)
// 	require.Len(t, p.Entitys, 0)

// 	emptyGroup := &Group{}

// 	group := &Group{}

// 	err := p.GetAPIVersionGroup(testServerName, testAPIVersion, &group)
// 	require.NotNil(t, err)
// 	require.Equal(t, emptyGroup, group)
// }

// func TestGetAPIVersionGroupWithoutCreatedGroup(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	emptyGroup := &Group{}

// 	group := &Group{}

// 	err1 := p.GetAPIVersionGroup(testServerName, testAPIVersion, &group)

// 	require.NotNil(t, err1)
// 	require.Equal(t, emptyGroup, group)
// }

// func TestGetServerViaCreateServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	emptyServer := &Enity{}

// 	server := &Enity{}

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, &server)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.NotNil(t, server)
// 	require.IsType(t, &Enity{}, server)
// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)
// 	require.NotEqual(t, emptyServer, server)
// }

// func TestGetServerViaGetServerWithCreatedServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName, &Options{Address: testServerListenAddress}, nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	emptyServer := &Enity{}

// 	enity, err := p.GetEnity(testServerName)

// 	require.Nil(t, err)
// 	require.NotEqual(t, emptyServer, enity)
// }

// func TestGetServerViaGetServerWithoutCreatedServer(t *testing.T) {
// 	p := initializeProvider(t)
// 	require.NotNil(t, p.Entitys)

// 	_, err := p.GetEnity(testServerName)
// 	require.NotNil(t, err)
// }

// func TestServerStartAndShutdown(t *testing.T) {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	p := NewProvider(ctx)
// 	require.Nil(t, p)

// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName,
// 		&Options{
// 			Address:    testServerListenAddress,
// 			HideBanner: true,
// 			HidePort:   true,
// 		},
// 		nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	go func() {
// 		err1 := p.Start()
// 		require.Nil(t, err1)
// 	}()

// 	iterations := 0

// 	for {
// 		iterations++

// 		testServer, _ := p.Entitys.Load(testServerName)
// 		if testServer.(*Enity).IsStarted() {
// 			break
// 		}

// 		if iterations > 10 {
// 			t.Fatal("HTTP server isn't up after 10 seconds")
// 		}

// 		time.Sleep(time.Second * 1)
// 	}

// 	err = p.Shutdown()

// 	require.Nil(t, err)
// }

// func TestHTTPSServerStartAndShutdown(t *testing.T) {
// 	ctx := context.Background()
// 	ctx = logger.Registrate(ctx)
// 	logger.Get(ctx).Initialize(&logger.Config{Level: logger.LoggerLevelInfo, NoColoredOutput: true, WithTrace: false})

// 	p := NewProvider(ctx)
// 	require.Nil(t, p)

// 	require.NotNil(t, p.Entitys)

// 	err := p.CreateEnity(testServerName,
// 		&Options{
// 			Address:    testServerListenAddress,
// 			HideBanner: true,
// 			HidePort:   true,
// 			CertFile:   testCertPath,
// 			KeyFile:    testKeyPath,
// 		},
// 		nil)
// 	if err != nil {
// 		t.Log(err.Error())
// 	}

// 	require.Nil(t, err)
// 	require.Len(t, p.Entitys, 1)

// 	go func() {
// 		err1 := p.Start()
// 		require.Nil(t, err1)
// 	}()

// 	iterations := 0

// 	for {
// 		iterations++

// 		testServer, _ := p.Entitys.Load(testServerName)
// 		if testServer.(*Enity).IsStarted() {
// 			break
// 		}

// 		if iterations > 10 {
// 			t.Fatal("HTTP server isn't up after 10 seconds")
// 		}

// 		time.Sleep(time.Second * 1)
// 	}

// 	err = p.Shutdown()
// 	require.Nil(t, err)
// }
