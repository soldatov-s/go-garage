package echo

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/meta"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/providers/logger"
	echoSwagger "github.com/soldatov-s/go-swagger/echo-swagger"
	"github.com/soldatov-s/go-swagger/swagger"
)

// Enity describes every HTTP server's structure and configuration.
type Enity struct {
	ctx      context.Context
	log      zerolog.Logger
	address  string
	certFile string
	keyFile  string
	name     string

	// Server is an echo.Echo thing that represents HTTP server.
	Server *echo.Echo

	// API groups for this server
	apiGroups sync.Map

	// Start server
	serverStart      bool
	serverStartMutex sync.RWMutex

	// Is server already started?
	serverStarted      bool
	serverStartedMutex sync.RWMutex
}

// Create configures structure and creates new echo HTTP server.
func NewEnity(ctx context.Context, serverName string, options interface{}) (*Enity, error) {
	if serverName == "" {
		return nil, errors.ErrEmptyEnityName
	}

	srv := &Enity{
		name: serverName,
		ctx:  logger.Registrate(ctx),
	}

	logger.Get(srv.ctx).GetLogger(httpsrv.ProvidersName, nil).Info().Msgf("initializing enity " + serverName + "...")
	srv.log = logger.Get(srv.ctx).GetLogger(httpsrv.ProvidersName, nil).With().Str("server name", serverName).Logger()

	serverConfig, ok := options.(*Config)
	if !ok {
		return nil, errors.ErrInvalidEnityOptions(Config{})
	}

	if err := serverConfig.Validate(); err != nil {
		return nil, err
	}

	srv.log.Debug().Msg("creating new HTTP server")
	srv.address = serverConfig.Address
	srv.certFile = serverConfig.CertFile
	srv.keyFile = serverConfig.KeyFile

	e := echo.New()
	serverConfig.InitilizeEcho(e)
	e.GET("/_internal/waitForOnline", srv.waitForHTTPServerToBeUpHandler)

	e.Use(middleware.Recover())
	for _, mdl := range serverConfig.Middlewares {
		middlewareFunc, ok := mdl.(echo.MiddlewareFunc)
		if !ok {
			srv.log.Error().Msgf("%+v is not echo.MiddlewareFunc", mdl)
			continue
		}

		e.Use(middlewareFunc)
	}

	srv.Server = e

	return srv, nil
}

func (srv *Enity) buildSwagger(apiVer string) {
	var a *meta.ApplicationInfo
	srv.ctx, a = meta.CreateApp(srv.ctx)

	swaggerName := a.Name + " " + srv.name + " API v" + apiVer
	// Build swagger
	err := echoSwagger.BuildSwagger(
		srv.Server,
		"/swagger/*",
		srv.address,
		swagger.NewSwagger().
			SetBasePath("/api/v"+apiVer+"/").
			SetInfo(swagger.NewInfo().
				SetTitle(swaggerName).
				SetDescription("This is a documentation of "+swaggerName).
				SetTermOfService("http://swagger.io/terms/").
				SetContact(swagger.NewContact()).
				SetLicense(swagger.NewLicense()).
				SetVersion(a.GetBuildInfo())),
		&srv.log,
	)

	if err != nil {
		srv.log.Warn().Err(err).Msg("failed to build " + swaggerName)

		return
	}
}

// CreateAPIVersionGroup creates routing group for desired API version.
// If API group was already created it will return nil and error about
// that.
func (srv *Enity) CreateAPIVersionGroup(apiVersion string, middlewares ...interface{}) error {
	if _, err := srv.GetAPIVersionGroup(apiVersion); err == nil {
		srv.log.Debug().Msg("api version group already created")
		return nil
	}

	srv.log.Debug().Str("api version group", "v"+apiVersion).Msg("Creating new API version group")
	newGroup := srv.Server.Group("/api/v" + apiVersion)
	createdGroup := &Group{Group: *newGroup}

	for _, mdl := range middlewares {
		middlewareFunc, ok := mdl.(echo.MiddlewareFunc)
		if !ok {
			srv.log.Error().Msgf("this middleware cannot be used (not echo.MiddlewareFunc): %+v", mdl)
			continue
		}

		createdGroup.Use(middlewareFunc)
	}

	srv.apiGroups.Store(apiVersion, createdGroup)

	return nil
}

// GetAPIVersionGroup will return created API version routing group.
// It will return nil and error if API group wasn't created.
func (srv *Enity) GetAPIVersionGroup(apiVersion string) (*Group, error) {
	group, found := srv.apiGroups.Load(apiVersion)

	if !found {
		return nil, httpsrv.ErrGroupNotFound
	}

	groupPointer, ok := group.(*Group)
	if !ok {
		return nil, httpsrv.ErrInvalidGroupPointer(Group{})
	}

	return groupPointer, nil
}

// IsStarted return server state
func (srv *Enity) IsStarted() bool {
	srv.serverStartedMutex.Lock()
	defer srv.serverStartedMutex.Unlock()
	started := srv.serverStarted
	return started
}

// isStarted checks that server started
func (srv *Enity) isStarted() error {
	// Try to check if server is ready to process requests within 10
	// seconds after start.
	httpc := &http.Client{
		Timeout: time.Second * 1,
	}

	if srv.certFile != "" && srv.keyFile != "" {
		caCert, err := ioutil.ReadFile(srv.certFile)
		if err != nil {
			srv.log.Fatal().Msgf("failed reading server certificate: %s", err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// Create TLS configuration with the certificate of the server
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS13,
			RootCAs:    caCertPool,
		}
		httpc.Transport = &http.Transport{
			TLSClientConfig: tlsConfig,
		}
	}

	checks := 0

	for {
		checks++

		if checks >= 10 {
			srv.log.Error().Int("seconds passed", checks).Msg("http server isn't up")
			return httpsrv.ErrServerNotUp
		}

		time.Sleep(time.Second * 1)

		testURL := srv.address + "/_internal/waitForOnline"
		if srv.certFile == "" || srv.keyFile == "" {
			testURL = "http://" + testURL
		} else if srv.certFile != "" && srv.keyFile != "" {
			testURL = "https://" + testURL
		}

		resp, err := httpc.Get(testURL)
		if err != nil {
			srv.log.Debug().Err(err).Msg("http error occurred, http server isn't ready, waiting...")
			continue
		}

		response, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			srv.log.Debug().Err(err).Msg("failed to read response body, http server isn't ready, waiting...")
			continue
		}

		srv.log.Debug().Str("status", resp.Status).Int("body length", len(response)).Msg("http response received")

		if resp.StatusCode == http.StatusOK {
			if len(response) == 0 {
				srv.log.Debug().Msg("response is empty, http server isn't ready, waiting...")
				continue
			}

			srv.log.Debug().Int("status code", resp.StatusCode).Msgf("Response: %+v", string(response))

			if len(response) == 17 {
				break
			}
		}
	}
	srv.log.Info().Msg("http server is ready to process requests")
	srv.serverStartedMutex.Lock()
	srv.serverStarted = true
	srv.serverStartedMutex.Unlock()

	return nil
}

// BuildSwagger build/rebuild swagger for all apiGroups
func (srv *Enity) BuildSwagger() {
	srv.apiGroups.Range(func(k, v interface{}) bool {
		srv.buildSwagger(k.(string))
		return true
	})
}

// Start starts HTTP server listening.
func (srv *Enity) Start() error {
	srv.serverStartMutex.RLock()
	if srv.serverStart {
		srv.serverStartMutex.RUnlock()
		return httpsrv.ErrServerAlreadyStarted
	}
	srv.serverStartMutex.RUnlock()

	// Build swagger
	srv.BuildSwagger()

	// Start HTTP server.
	srv.serverStartMutex.Lock()
	defer srv.serverStartMutex.Unlock()
	srv.serverStart = true

	go func() {
		srv.log.Info().Str("address", srv.Server.Server.Addr).Msg("starting server...")

		var err error
		if srv.certFile != "" && srv.keyFile != "" {
			err = srv.Server.StartTLS(srv.address, srv.certFile, srv.keyFile)
		} else {
			err = srv.Server.Start(srv.address)
		}

		if !strings.Contains(err.Error(), "server closed") {
			srv.log.Error().Err(err).Msg("http server critical error occurred")
		}

		srv.serverStartMutex.Lock()
		defer srv.serverStartMutex.Unlock()
		srv.serverStart = false
	}()

	return srv.isStarted()
}

// Shutdown stops HTTP server listening.
func (srv *Enity) Shutdown() error {
	err := srv.Server.Shutdown(context.Background())
	if err != nil {
		return err
	}

	srv.serverStartedMutex.Lock()
	defer srv.serverStartedMutex.Unlock()
	srv.serverStarted = false

	srv.serverStartMutex.Lock()
	defer srv.serverStartMutex.Unlock()
	srv.serverStart = false

	return nil
}

func (srv *Enity) waitForHTTPServerToBeUpHandler(ec echo.Context) error {
	response := map[string]string{
		"error": "None",
	}

	return ec.JSON(200, response)
}

func (srv *Enity) RegisterEndpoint(method, endpoint string, handler http.Handler, m ...httpsrv.MiddleWareFunc) error {
	if handler == nil {
		return httpsrv.ErrEmptyHTTPHandler
	}

	echoHandler := echo.WrapHandler(handler)
	echoMiddleware := make([]echo.MiddlewareFunc, len(m))
	for i, v := range m {
		echoMiddleware[i] = echo.WrapMiddleware(v)
	}

	switch method {
	case echo.GET:
		srv.Server.GET(endpoint, echoHandler, echoMiddleware...)
	case echo.POST:
		srv.Server.POST(endpoint, echoHandler, echoMiddleware...)
	case echo.PUT:
		srv.Server.PUT(endpoint, echoHandler, echoMiddleware...)
	case echo.DELETE:
		srv.Server.DELETE(endpoint, echoHandler, echoMiddleware...)
	case echo.PATCH:
		srv.Server.PATCH(endpoint, echoHandler, echoMiddleware...)
	case echo.CONNECT:
		srv.Server.CONNECT(endpoint, echoHandler, echoMiddleware...)
	case echo.OPTIONS:
		srv.Server.OPTIONS(endpoint, echoHandler, echoMiddleware...)
	case echo.TRACE:
		srv.Server.TRACE(endpoint, echoHandler, echoMiddleware...)
	case echo.HEAD:
		srv.Server.HEAD(endpoint, echoHandler, echoMiddleware...)
	default:
		return httpsrv.ErrUnknownHTTPMethod
	}

	return nil
}
