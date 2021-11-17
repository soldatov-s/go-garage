package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/log"
	"github.com/soldatov-s/go-garage/x/httpx"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"golang.org/x/sync/errgroup"
)

const (
	ReadyEndpoint   = "/health/ready"
	AliveEndpoint   = "/health/alive"
	MetricsEndpoint = "/metrics"
)

var (
	ErrAppendMetrics            = errors.New("failed to append metrics")
	ErrAliveHandlers            = errors.New("failed to append alive handlers")
	ErrReadyHandlers            = errors.New("failed to append ready handlers")
	ErrNotFindStatsHTTP         = errors.New("not find http server for stats")
	ErrFailedTypeCastHTTPServer = errors.New("failed typecast to http server")
)

type HTTPServer interface {
	RegisterEndpoint(method, endpoint string, handler http.Handler, m ...httpx.MiddleWareFunc) error
}

type EnityMetricsGateway interface {
	GetMetrics() *base.MapMetricsOptions
}

type EnityAliveGateway interface {
	GetAliveHandlers() *base.MapCheckOptions
}

type EnityReadyGateway interface {
	GetReadyHandlers() *base.MapCheckOptions
}

type EnityGateway interface {
	Shutdown(ctx context.Context) error
	Start(ctx context.Context) error
	GetFullName() string
}

type ManagerDeps struct {
	Meta               *MetaDeps
	StatsHTTPEnityName string
	Logger             *log.Logger
	ErrorGroup         *errgroup.Group
}

type MetaDeps struct {
	Name        string
	Builded     string
	Hash        string
	Version     string
	Description string
}

type Meta struct {
	Name        string
	Builded     string
	Hash        string
	Version     string
	Description string
}

func NewMeta(deps *MetaDeps) *Meta {
	meta := &Meta{
		Name:        deps.Name,
		Builded:     deps.Name,
		Hash:        deps.Hash,
		Version:     deps.Version,
		Description: deps.Description,
	}

	if meta.Description == "" {
		meta.Description = "no description"
	}

	if meta.Name == "" {
		meta.Name = "unknown"
	}

	if meta.Version == "" {
		meta.Name = "0.0.0"
	}

	return meta
}

func (m *Meta) BuildInfo() string {
	return m.Version + ", builded: " + m.Builded + ", hash: " + m.Hash
}

type Manager struct {
	*base.MetricsStorage
	*base.ReadyCheckStorage
	*base.AliveCheckStorage
	meta               *Meta
	mu                 sync.Mutex
	enities            map[string]EnityGateway
	enitiesOrder       []string
	statsHTTPEnityName string
	register           prometheus.Registerer
	logger             *log.Logger
	signals            []os.Signal
	errorGroup         *errgroup.Group
}

type ManagerOption func(*Manager)

func WithCustomRegister(register prometheus.Registerer) ManagerOption {
	return func(c *Manager) {
		c.register = register
	}
}

func WithCustomSignalas(signals []os.Signal) ManagerOption {
	return func(c *Manager) {
		c.signals = signals
	}
}

func NewApp(deps *ManagerDeps, opts ...ManagerOption) *Manager {
	app := &Manager{
		MetricsStorage:     base.NewMetricsStorage(),
		AliveCheckStorage:  base.NewAliveCheckStorage(),
		ReadyCheckStorage:  base.NewReadyCheckStorage(),
		meta:               NewMeta(deps.Meta),
		enities:            make(map[string]EnityGateway),
		enitiesOrder:       make([]string, 0, 16),
		statsHTTPEnityName: deps.StatsHTTPEnityName,
		register:           prometheus.DefaultRegisterer,
		logger:             deps.Logger,
		signals:            defaultOSSignals(),
		errorGroup:         deps.ErrorGroup,
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

func defaultOSSignals() []os.Signal {
	return []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT}
}

func (a *Manager) Meta() *Meta {
	return a.meta
}

type ErrSignal struct {
	Signal os.Signal
}

func (e ErrSignal) Error() string {
	return fmt.Sprintf("got error signal %s", e.Signal.String())
}

func (a *Manager) OSSignalWaiter(ctx context.Context) error {
	logger := a.logger.Zerolog()
	closeSignal := make(chan os.Signal, 1)
	signal.Notify(closeSignal, a.signals...)

	a.errorGroup.Go(func() error {
		select {
		case s := <-closeSignal:
			logger.Info().Msgf("got os signal: %s", s.String())
			if err := a.Shutdown(ctx); err != nil {
				return errors.Wrap(err, "shutdown app")
			}
			return ErrSignal{Signal: s}
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	return nil
}

// Loop is application loop
func (a *Manager) Loop(ctx context.Context) error {
	logger := a.logger.Zerolog()
	if err := a.errorGroup.Wait(); err != nil {
		switch {
		case isExitSignal(err):
			logger.Info().Msg("exited by exit signal")
		default:
			return errors.Wrap(err, "exited with error")
		}
	}
	return nil
}

func isExitSignal(err error) bool {
	errSig := ErrSignal{}
	is := errors.As(err, &errSig)
	return is
}

func (a *Manager) Start(ctx context.Context) error {
	for _, k := range a.enitiesOrder {
		if err := a.enities[k].Start(ctx); err != nil {
			return errors.Wrapf(err, "start enity %q", k)
		}
	}

	if err := a.startStatistic(ctx); err != nil {
		return errors.Wrap(err, "start statistics")
	}

	return nil
}

func (a *Manager) Shutdown(ctx context.Context) error {
	reversedProviders := stringsx.ReverseStringSlice(a.enitiesOrder)
	for _, k := range reversedProviders {
		if err := a.enities[k].Shutdown(ctx); err != nil {
			return errors.Wrapf(err, "shutdown enity %q", k)
		}
	}
	return nil
}

func (a *Manager) Add(ctx context.Context, e EnityGateway) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.enities[e.GetFullName()]; ok {
		return base.ErrConflictName
	}

	a.enities[e.GetFullName()] = e
	a.enitiesOrder = append(a.enitiesOrder, e.GetFullName())

	if v, ok := e.(EnityMetricsGateway); ok {
		if err := a.MetricsStorage.GetMetrics().Append(v.GetMetrics()); err != nil {
			return ErrAppendMetrics
		}
	}

	if v, ok := e.(EnityAliveGateway); ok {
		if err := a.AliveCheckStorage.GetAliveHandlers().Append(v.GetAliveHandlers()); err != nil {
			return ErrAliveHandlers
		}
	}

	if v, ok := e.(EnityReadyGateway); ok {
		if err := a.ReadyCheckStorage.GetReadyHandlers().Append(v.GetReadyHandlers()); err != nil {
			return ErrAliveHandlers
		}
	}

	return nil
}

func (a *Manager) startStatistic(ctx context.Context) error {
	enity, ok := a.enities[a.statsHTTPEnityName]
	if !ok {
		return ErrNotFindStatsHTTP
	}

	httpSrv, ok := enity.(HTTPServer)
	if !ok {
		return ErrFailedTypeCastHTTPServer
	}

	// Registrate metrics
	if err := a.MetricsStorage.GetMetrics().Registrate(a.register); err != nil {
		return errors.Wrap(err, "registarte metrics")
	}

	if err := a.logger.GetMetrics().Registrate(a.register); err != nil {
		return errors.Wrap(err, "registarte metrics")
	}

	if err := httpSrv.RegisterEndpoint(
		http.MethodGet,
		MetricsEndpoint,
		promhttp.Handler(),
		func(h http.Handler) http.Handler {
			return a.PrometheusMiddleware(ctx, h)
		}); err != nil {
		return errors.Wrap(err, "registrate prometheus endpoint")
	}

	// Registrate alive
	if err := httpSrv.RegisterEndpoint(
		http.MethodGet,
		AliveEndpoint,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.AliveCheckHandler(ctx, w)
		}),
	); err != nil {
		return errors.Wrap(err, "registrate alive endpoint")
	}

	// Registrate ready
	if err := httpSrv.RegisterEndpoint(
		http.MethodGet,
		ReadyEndpoint,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.ReadyCheckHandler(ctx, w)
		}),
	); err != nil {
		return errors.Wrap(err, "registrate ready endpoint")
	}

	return nil
}
