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
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/meta"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/x/helper"
	echoSwagger "github.com/soldatov-s/go-swagger/echo-swagger"
	"github.com/soldatov-s/go-swagger/swagger"
)

// Enity describes every HTTP server's structure and configuration.
type Enity struct {
	*base.Enity
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
func NewEnity(ctx context.Context, collectorName, providerName, name string, cfg interface{}) (*Enity, error) {
	e, err := base.NewEnity(ctx, collectorName, providerName, name)
	if err != nil {
		return nil, errors.Wrap(err, "create base enity")
	}

	srv := &Enity{
		Enity: e,
	}

	serverConfig, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityOptions, "expected %q", helper.ObjName(Config{}))
	}

	if err := serverConfig.Validate(); err != nil {
		return nil, err
	}

	e.GetLogger(ctx).Debug().Msg("creating new HTTP server")
	srv.address = serverConfig.Address
	srv.certFile = serverConfig.CertFile
	srv.keyFile = serverConfig.KeyFile

	echoSrv := echo.New()
	serverConfig.InitilizeEcho(echoSrv)
	echoSrv.GET("/_internal/waitForOnline", srv.waitForHTTPServerToBeUpHandler)

	echoSrv.Use(middleware.Recover())
	for _, mdl := range serverConfig.Middlewares {
		middlewareFunc, ok := mdl.(echo.MiddlewareFunc)
		if !ok {
			e.GetLogger(ctx).Error().Msgf("%+v is not echo.MiddlewareFunc", mdl)
			continue
		}

		echoSrv.Use(middlewareFunc)
	}

	srv.Server = echoSrv

	return srv, nil
}

func (e *Enity) buildSwagger(ctx context.Context, apiVer string) {
	metaData, err := meta.FromContext(ctx)
	if err != nil {
		e.GetLogger(ctx).Err(err).Msg("get meta")
		metaData = meta.DefaultAppInfo()
	}

	swaggerName := metaData.Name + " " + e.name + " API v" + apiVer
	// Build swagger
	err = echoSwagger.BuildSwagger(
		e.Server,
		"/swagger/*",
		e.address,
		swagger.NewSwagger().
			SetBasePath("/api/v"+apiVer+"/").
			SetInfo(swagger.NewInfo().
				SetTitle(swaggerName).
				SetDescription("This is a documentation of "+swaggerName).
				SetTermOfService("http://swagger.io/terms/").
				SetContact(swagger.NewContact()).
				SetLicense(swagger.NewLicense()).
				SetVersion(metaData.GetBuildInfo())),
		e.GetLogger(ctx),
	)

	if err != nil {
		e.GetLogger(ctx).Warn().Err(err).Msg("failed to build " + swaggerName)

		return
	}
}

// CreateAPIVersionGroup creates routing group for desired API version.
// If API group was already created it will return nil and error about
// that.
func (e *Enity) CreateAPIVersionGroup(ctx context.Context, apiVersion string, middlewares ...interface{}) error {
	if _, err := e.GetAPIVersionGroup(apiVersion); err == nil {
		e.GetLogger(ctx).Debug().Msg("api version group already created")
		return nil
	}

	e.GetLogger(ctx).Debug().Str("api version group", "v"+apiVersion).Msg("Creating new API version group")
	newGroup := e.Server.Group("/api/v" + apiVersion)
	createdGroup := &Group{Group: *newGroup}

	for _, mdl := range middlewares {
		middlewareFunc, ok := mdl.(echo.MiddlewareFunc)
		if !ok {
			e.GetLogger(ctx).Error().Msgf("this middleware cannot be used (not echo.MiddlewareFunc): %+v", mdl)
			continue
		}

		createdGroup.Use(middlewareFunc)
	}

	e.apiGroups.Store(apiVersion, createdGroup)

	return nil
}

// GetAPIVersionGroup will return created API version routing group.
// It will return nil and error if API group wasn't created.
func (e *Enity) GetAPIVersionGroup(apiVersion string) (*Group, error) {
	group, found := e.apiGroups.Load(apiVersion)

	if !found {
		return nil, httpsrv.ErrGroupNotFound
	}

	groupPointer, ok := group.(*Group)
	if !ok {
		return nil, errors.Wrapf(httpsrv.ErrInvalidGroupPointer, "expected %q", helper.ObjName(Config{}))
	}

	return groupPointer, nil
}

// IsStarted return server state
func (e *Enity) IsStarted() bool {
	e.serverStartedMutex.Lock()
	defer e.serverStartedMutex.Unlock()
	started := e.serverStarted
	return started
}

// isStarted checks that server started
func (e *Enity) isStarted(ctx context.Context) error {
	// Try to check if server is ready to process requests within 10
	// seconds after start.
	httpc := &http.Client{
		Timeout: time.Second * 1,
	}

	if e.certFile != "" && e.keyFile != "" {
		caCert, err := ioutil.ReadFile(e.certFile)
		if err != nil {
			e.GetLogger(ctx).Fatal().Msgf("failed reading server certificate: %s", err)
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
			e.GetLogger(ctx).Error().Int("seconds passed", checks).Msg("http server isn't up")
			return httpsrv.ErrServerNotUp
		}

		time.Sleep(time.Second * 1)

		testURL := e.address + "/_internal/waitForOnline"
		if e.certFile == "" || e.keyFile == "" {
			testURL = "http://" + testURL
		} else if e.certFile != "" && e.keyFile != "" {
			testURL = "https://" + testURL
		}

		resp, err := httpc.Get(testURL)
		if err != nil {
			e.GetLogger(ctx).Debug().Err(err).Msg("http error occurred, http server isn't ready, waiting...")
			continue
		}

		response, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			e.GetLogger(ctx).Debug().Err(err).Msg("failed to read response body, http server isn't ready, waiting...")
			continue
		}

		e.GetLogger(ctx).Debug().Str("status", resp.Status).Int("body length", len(response)).Msg("http response received")

		if resp.StatusCode == http.StatusOK {
			if len(response) == 0 {
				e.GetLogger(ctx).Debug().Msg("response is empty, http server isn't ready, waiting...")
				continue
			}

			e.GetLogger(ctx).Debug().Int("status code", resp.StatusCode).Msgf("Response: %+v", string(response))

			if len(response) == 17 {
				break
			}
		}
	}
	e.GetLogger(ctx).Info().Msg("http server is ready to process requests")
	e.serverStartedMutex.Lock()
	e.serverStarted = true
	e.serverStartedMutex.Unlock()

	return nil
}

// BuildSwagger build/rebuild swagger for all apiGroups
func (e *Enity) BuildSwagger(ctx context.Context) {
	e.apiGroups.Range(func(k, v interface{}) bool {
		e.buildSwagger(ctx, k.(string))
		return true
	})
}

// Start starts HTTP server listening.
func (e *Enity) Start(ctx context.Context) error {
	e.serverStartMutex.RLock()
	if e.serverStart {
		e.serverStartMutex.RUnlock()
		return httpsrv.ErrServerAlreadyStarted
	}
	e.serverStartMutex.RUnlock()

	// Build swagger
	e.BuildSwagger(ctx)

	// Start HTTP server.
	e.serverStartMutex.Lock()
	defer e.serverStartMutex.Unlock()
	e.serverStart = true

	go func() {
		e.GetLogger(ctx).Info().Str("address", e.Server.Server.Addr).Msg("starting server...")

		var err error
		if e.certFile != "" && e.keyFile != "" {
			err = e.Server.StartTLS(e.address, e.certFile, e.keyFile)
		} else {
			err = e.Server.Start(e.address)
		}

		if !strings.Contains(err.Error(), "server closed") {
			e.GetLogger(ctx).Error().Err(err).Msg("http server critical error occurred")
		}

		e.serverStartMutex.Lock()
		defer e.serverStartMutex.Unlock()
		e.serverStart = false
	}()

	return e.isStarted(ctx)
}

// Shutdown stops HTTP server listening.
func (e *Enity) Shutdown() error {
	err := e.Server.Shutdown(context.Background())
	if err != nil {
		return err
	}

	e.serverStartedMutex.Lock()
	defer e.serverStartedMutex.Unlock()
	e.serverStarted = false

	e.serverStartMutex.Lock()
	defer e.serverStartMutex.Unlock()
	e.serverStart = false

	return nil
}

func (e *Enity) waitForHTTPServerToBeUpHandler(ec echo.Context) error {
	response := map[string]string{
		"error": "None",
	}

	return ec.JSON(200, response)
}

func (e *Enity) RegisterEndpoint(method, endpoint string, handler http.Handler, m ...httpsrv.MiddleWareFunc) error {
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
		e.Server.GET(endpoint, echoHandler, echoMiddleware...)
	case echo.POST:
		e.Server.POST(endpoint, echoHandler, echoMiddleware...)
	case echo.PUT:
		e.Server.PUT(endpoint, echoHandler, echoMiddleware...)
	case echo.DELETE:
		e.Server.DELETE(endpoint, echoHandler, echoMiddleware...)
	case echo.PATCH:
		e.Server.PATCH(endpoint, echoHandler, echoMiddleware...)
	case echo.CONNECT:
		e.Server.CONNECT(endpoint, echoHandler, echoMiddleware...)
	case echo.OPTIONS:
		e.Server.OPTIONS(endpoint, echoHandler, echoMiddleware...)
	case echo.TRACE:
		e.Server.TRACE(endpoint, echoHandler, echoMiddleware...)
	case echo.HEAD:
		e.Server.HEAD(endpoint, echoHandler, echoMiddleware...)
	default:
		return httpsrv.ErrUnknownHTTPMethod
	}

	return nil
}
