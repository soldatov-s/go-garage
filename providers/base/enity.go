package base

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/utils"
)

var (
	ErrEmptyEnityName      = errors.New("empty enity name is not allowed")
	ErrInvalidEnityOptions = errors.New("invalid passed enity options pointer")
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	name          string
	collectorName string
	providerName  string

	// Shutdown flags.
	WeAreShuttingDown  bool
	ConnWatcherStopped bool
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, collectorName, providerName, name string) (*Enity, error) {
	if collectorName == "" {
		return nil, ErrEmptyCollectorName
	}

	if providerName == "" {
		return nil, ErrEmptyProviderName
	}

	if name == "" {
		return nil, ErrEmptyEnityName
	}

	e := &Enity{
		name:               name,
		providerName:       providerName,
		collectorName:      collectorName,
		ConnWatcherStopped: true,
	}

	logger := e.GetLogger(ctx)
	logger.Info().Msgf("initializing...")

	return e, nil
}

func (e *Enity) GetName() string {
	return e.name
}

func (e *Enity) GetFullName() string {
	return utils.JoinStrings("_", e.collectorName, e.providerName, e.name)
}

func (e *Enity) GetLogger(ctx context.Context) *zerolog.Logger {
	logger := zerolog.Ctx(ctx).With().Logger()
	logger = logger.With().Str("provider_type", e.collectorName).Str("provider_name", e.providerName).Str("enity_name", e.name).Logger()
	return &logger
}

type EntityWithMetrics struct {
	*Enity
	// Connection metrics
	Metrics MapMetricsOptions
	// Connection alive handlers
	AliveHandlers MapCheckFunc
	// Connection ready handlers
	ReadyHandlers MapCheckFunc
}

func NewEntityWithMetrics(ctx context.Context, collectorName, providerName, name string) (*EntityWithMetrics, error) {
	em := &EntityWithMetrics{}
	e, err := NewEnity(ctx, collectorName, providerName, name)
	if err != nil {
		return nil, errors.Wrap(err, "new enity")
	}
	em.Enity = e
	em.AliveHandlers = make(MapCheckFunc)
	em.Metrics = make(MapMetricsOptions)
	em.ReadyHandlers = make(MapCheckFunc)

	return em, nil
}

// GetAliveHandlers return array of the aliveHandlers from service
func (e *EntityWithMetrics) GetAliveHandlers(ctx context.Context) (MapCheckFunc, error) {
	return e.AliveHandlers, nil
}

// GetMetrics return map of the metrics from cache connection
func (e *EntityWithMetrics) GetMetrics(ctx context.Context) (MapMetricsOptions, error) {
	return e.Metrics, nil
}

// GetReadyHandlers return array of the readyHandlers from cache connection
func (e *EntityWithMetrics) GetReadyHandlers(ctx context.Context) (MapCheckFunc, error) {
	return e.ReadyHandlers, nil
}
