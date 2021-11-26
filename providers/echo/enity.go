package echo

import (
	"context"
	"net/http"
	"strconv"
	"time"

	oapiMiddleware "github.com/deepmap/oapi-codegen/pkg/middleware"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/swagger"
	"github.com/soldatov-s/go-garage/x/httpx"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "echo"

var (
	ErrNoServersInSpec   = errors.New("no servers in spec")
	ErrIPsInSpec         = errors.New("no ips in spec")
	ErrPortsInSpec       = errors.New("no ports in spec")
	ErrEmptyHTTPHandler  = errors.New("empty http handler")
	ErrUnknownHTTPMethod = errors.New("unknown http method")
)

// Enity describes every HTTP server's structure and configuration.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	config *Config
	// Server is an echo.Echo thing that represents HTTP server.
	server *echo.Echo

	prometheusMiddleware func(next echo.HandlerFunc) echo.HandlerFunc
}

func DefaultMiddlewares() []echo.MiddlewareFunc {
	return []echo.MiddlewareFunc{
		middleware.Recover(),
		CORSDefault(),
	}
}

// Create configures structure and creates new echo HTTP server.
func NewEnity(ctx context.Context, name string, config *Config, middlewares ...echo.MiddlewareFunc) (*Enity, error) {
	deps := &base.EnityDeps{
		ProviderName: ProviderName,
		Name:         name,
	}
	baseEnity := base.NewEnity(deps)

	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	enity := &Enity{
		MetricsStorage: base.NewMetricsStorage(),
		Enity:          baseEnity,
		config:         config.SetDefault(),
	}
	server := config.NewEcho()
	if err := enity.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	server.Use(enity.prometheusMiddleware)
	server.Use(middlewares...)

	enity.server = server

	return enity, nil
}

func (e *Enity) GetConfig() *Config {
	return e.config
}

func (e *Enity) GetServer() *echo.Echo {
	return e.server
}

type GetSwagger func() (swagger *openapi3.T, err error)

// CreateAPIVersionGroup creates routing group for desired API version.
// If API group was already created it will return nil and error about
// that.
func (e *Enity) APIGroup(ctx context.Context,
	version, buildInfo string,
	swaggerGet GetSwagger,
	middlewares ...echo.MiddlewareFunc) (*echo.Group, error) {
	e.GetLogger(ctx).Debug().Str("api version group", "v"+version).Msg("creating new API group")
	validateSwaggerSpec, err := buildValidateSwaggerSpec(swaggerGet, version)
	if err != nil {
		return nil, errors.Wrap(err, "build validate swagger spec")
	}

	guiSwaggerSpec, err := swaggerGet()
	if err != nil {
		return nil, errors.Wrap(err, "get swagger spec")
	}

	apiGroup := e.server.Group("/api/v" + version)
	apiGroup.Use(middlewares...)
	apiGroup.Use(oapiMiddleware.OapiRequestValidator(validateSwaggerSpec))

	guiSwaggerSpec.Info.Version = buildInfo
	swaggerDoc := swagger.NewDoc(guiSwaggerSpec)

	swaggerName := e.config.Address + "/api/v" + version + "/"
	swagger.Register(swaggerName, swaggerDoc)

	e.server.GET("/swagger/api/v"+version+"/*",
		swagger.EchoHandler(
			ctx,
			swagger.Fill("doc.json", swaggerName), // The url pointing to API definition"
		),
	)

	return apiGroup, nil
}

func buildValidateSwaggerSpec(swaggerGet GetSwagger, version string) (*openapi3.T, error) {
	validateSwaggerSpec, err := swaggerGet()
	if err != nil {
		return nil, errors.Wrap(err, "get swagger spec")
	}

	if len(validateSwaggerSpec.Servers) == 0 {
		return nil, ErrNoServersInSpec
	}

	ips, ok := validateSwaggerSpec.Servers[0].Variables["ip"]
	if !ok {
		return nil, ErrIPsInSpec
	}

	ports, ok := validateSwaggerSpec.Servers[0].Variables["port"]
	if !ok {
		return nil, ErrPortsInSpec
	}

	validateSwaggerSpec.Servers = make(openapi3.Servers, 0)
	for _, ip := range ips.Enum {
		for _, port := range ports.Enum {
			srv := openapi3.Server{
				URL: "http://" + ip + ":" + port + "/api/v" + version,
			}
			validateSwaggerSpec.AddServer(&srv)
		}
	}

	return validateSwaggerSpec, nil
}

// Start starts HTTP server listening.
func (e *Enity) Start(ctx context.Context, errGroup *errgroup.Group) error {
	logger := e.GetLogger(ctx)

	errGroup.Go(func() error {
		logger.Info().Str("address", e.config.Address).Msg("starting server...")

		var err error
		if e.config.CertFile != "" && e.config.KeyFile != "" {
			err = e.server.StartTLS(e.config.Address, e.config.CertFile, e.config.KeyFile)
		} else {
			err = e.server.Start(e.config.Address)
		}

		if err != nil {
			return errors.Wrap(err, "start http server")
		}

		return nil
	})

	return nil
}

// Shutdown stops HTTP server listening.
func (e *Enity) Shutdown(ctx context.Context) error {
	if err := e.server.Shutdown(ctx); err != nil {
		return errors.Wrap(err, "shutdown http server")
	}

	return nil
}

// nolint:gocyclo // long switch
func (e *Enity) RegisterEndpoint(method, endpoint string, handler http.Handler, m ...httpx.MiddleWareFunc) error {
	if handler == nil {
		return ErrEmptyHTTPHandler
	}

	echoHandler := echo.WrapHandler(handler)
	echoMiddleware := make([]echo.MiddlewareFunc, len(m))
	for i, v := range m {
		echoMiddleware[i] = echo.WrapMiddleware(v)
	}

	switch method {
	case echo.GET:
		e.server.GET(endpoint, echoHandler, echoMiddleware...)
	case echo.POST:
		e.server.POST(endpoint, echoHandler, echoMiddleware...)
	case echo.PUT:
		e.server.PUT(endpoint, echoHandler, echoMiddleware...)
	case echo.DELETE:
		e.server.DELETE(endpoint, echoHandler, echoMiddleware...)
	case echo.PATCH:
		e.server.PATCH(endpoint, echoHandler, echoMiddleware...)
	case echo.CONNECT:
		e.server.CONNECT(endpoint, echoHandler, echoMiddleware...)
	case echo.OPTIONS:
		e.server.OPTIONS(endpoint, echoHandler, echoMiddleware...)
	case echo.TRACE:
		e.server.TRACE(endpoint, echoHandler, echoMiddleware...)
	case echo.HEAD:
		e.server.HEAD(endpoint, echoHandler, echoMiddleware...)
	default:
		return ErrUnknownHTTPMethod
	}

	return nil
}

// nolint:funlen // long function
func (e *Enity) buildMetrics(_ context.Context) error {
	fullName := e.GetFullName()

	reqCntHelp := "How many HTTP requests processed, partitioned by status code and HTTP method."
	reqCntArgs := []string{"code", "method", "host", "url"}
	reqCnt, err := e.MetricsStorage.GetMetrics().AddCounterVec(fullName, "requests total", reqCntHelp, reqCntArgs)
	if err != nil {
		return errors.Wrap(err, "add counter vek")
	}

	reqDurHelp := "The HTTP request latencies in seconds."
	reqDurArgs := []string{"code", "method", "url"}
	reqDur, err := e.MetricsStorage.GetMetrics().AddHistogramVec(fullName, "request duration seconds", reqDurHelp, reqDurArgs)
	if err != nil {
		return errors.Wrap(err, "add histogram vek")
	}

	resSzHelp := "The HTTP response sizes in bytes."
	resSzArgs := []string{"code", "method", "url"}
	resSz, err := e.MetricsStorage.GetMetrics().AddHistogramVec(fullName, "response size bytes", resSzHelp, resSzArgs)
	if err != nil {
		return errors.Wrap(err, "add histogram vek")
	}

	reqSzHelp := "The HTTP request sizes in bytes."
	reqSzArgs := []string{"code", "method", "url"}
	reqSz, err := e.MetricsStorage.GetMetrics().AddHistogramVec(fullName, "request size bytes", reqSzHelp, reqSzArgs)
	if err != nil {
		return errors.Wrap(err, "add histogram vek")
	}

	e.prometheusMiddleware = func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Path() == "/metrics" {
				return next(c)
			}

			start := time.Now()
			requestSize := computeApproximateRequestSize(c.Request())

			err := next(c)

			status := c.Response().Status
			if err != nil {
				var httpError *echo.HTTPError
				if errors.As(err, &httpError) {
					status = httpError.Code
				}
				if status == 0 || status == http.StatusOK {
					status = http.StatusInternalServerError
				}
			}

			elapsed := float64(time.Since(start)) / float64(time.Second)

			url := c.Path()
			statusStr := strconv.Itoa(status)
			reqDur.WithLabelValues(statusStr, c.Request().Method, url).Observe(elapsed)
			reqCnt.WithLabelValues(statusStr, c.Request().Method, c.Request().Host, url).Inc()
			reqSz.WithLabelValues(statusStr, c.Request().Method, url).Observe(float64(requestSize))

			responseSize := float64(c.Response().Size)
			resSz.WithLabelValues(statusStr, c.Request().Method, url).Observe(responseSize)

			return err
		}
	}

	return nil
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}
