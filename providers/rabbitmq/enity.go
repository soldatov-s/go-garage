package rabbitmq

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/base"
	rabbitmqcon "github.com/soldatov-s/go-garage/providers/rabbitmq/connection"
	rabbitmqconsum "github.com/soldatov-s/go-garage/providers/rabbitmq/consumer"
	rabbitmqpub "github.com/soldatov-s/go-garage/providers/rabbitmq/publisher"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"github.com/streadway/amqp"
	"golang.org/x/sync/errgroup"
)

const ProviderName = "rabbitmq"

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.Enity
	*base.MetricsStorage
	*base.ReadyCheckStorage
	config     *Config
	conn       *rabbitmqcon.Connection
	consumers  map[string]*rabbitmqconsum.Consumer
	publishers map[string]*rabbitmqpub.Publisher
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, config *Config) (*Enity, error) {
	deps := &base.EnityDeps{
		ProviderName: ProviderName,
		Name:         name,
	}
	baseEnity := base.NewEnity(deps)

	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	e := &Enity{
		MetricsStorage:    base.NewMetricsStorage(),
		ReadyCheckStorage: base.NewReadyCheckStorage(),
		Enity:             baseEnity,
		config:            config.SetDefault(),
		consumers:         make(map[string]*rabbitmqconsum.Consumer),
		publishers:        make(map[string]*rabbitmqpub.Publisher),
	}

	if err := e.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	if err := e.buildReadyHandlers(ctx); err != nil {
		return nil, errors.Wrap(err, "build ready handlers")
	}

	return e, nil
}

func (e *Enity) GetConn() *rabbitmqcon.Connection {
	return e.conn
}

func (e *Enity) GetConfig() *Config {
	return e.config
}

func (e *Enity) AddConsumer(ctx context.Context, config *rabbitmqconsum.Config) (*rabbitmqconsum.Consumer, error) {
	name := stringsx.JoinStrings("_", config.ExchangeName, config.RabbitQueue, config.RabbitConsume, config.RoutingKey)
	if _, ok := e.consumers[name]; ok {
		return nil, errors.Wrapf(base.ErrConflictName, "name is %q", name)
	}

	consumer, err := rabbitmqconsum.NewConsumer(ctx, e.GetFullName(), config, &e.conn)
	if err != nil {
		return nil, errors.Wrap(err, "new consumer")
	}

	e.consumers[name] = consumer
	if err := e.MetricsStorage.GetMetrics().Append(consumer.GetMetrics()); err != nil {
		return nil, errors.Wrap(err, "append metrics")
	}

	return consumer, nil
}

func (e *Enity) AddPublisher(ctx context.Context, config *rabbitmqpub.Config) (*rabbitmqpub.Publisher, error) {
	name := stringsx.JoinStrings("_", config.ExchangeName, config.RoutingKey)
	if _, ok := e.consumers[name]; ok {
		return nil, errors.Wrapf(base.ErrConflictName, "name is %q", name)
	}

	publisher, err := rabbitmqpub.NewPublisher(ctx, e.GetFullName()+"_"+name, config, &e.conn)
	if err != nil {
		return nil, errors.Wrap(err, "new consumer")
	}

	e.publishers[name] = publisher
	if err := e.MetricsStorage.GetMetrics().Append(publisher.GetMetrics()); err != nil {
		return nil, errors.Wrap(err, "append metrics")
	}

	return publisher, nil
}

// Ping checks that rabbitMQ connections is live
func (e *Enity) Ping(ctx context.Context) error {
	client, err := amqp.Dial(e.config.DSN)
	if err != nil {
		return errors.Wrap(err, "ampq dial")
	}

	if err := client.Close(); err != nil {
		return errors.Wrap(err, "close client")
	}
	return nil
}

func (e *Enity) Start(ctx context.Context, errorGroup *errgroup.Group) error {
	logger := e.GetLogger(ctx)

	if e.conn != nil {
		return nil
	}
	logger.Info().Msg("establishing connection...")
	var err error

	e.conn, err = rabbitmqcon.NewConnection(e.config.DSN, e.config.BackoffPolicy)
	if err != nil {
		return errors.Wrap(err, "create connection")
	}

	if err := e.conn.Connect(ctx, errorGroup); err != nil {
		return errors.Wrap(err, "connect")
	}

	// Connection watcher will be started in any case, but only if
	// it wasn't launched before.
	if e.IsWatcherStopped() {
		e.SetWatcher(false)
		errorGroup.Go(func() error {
			return e.startWatcher(ctx)
		})
	}

	return nil
}

// Connection watcher goroutine entrypoint.
func (e *Enity) startWatcher(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("starting connection watcher")

	for {
		select {
		case <-ctx.Done():
			e.GetLogger(ctx).Info().Msg("connection watcher stopped")
			e.SetWatcher(true)
			return ctx.Err()
		default:
			if err := e.Ping(ctx); err != nil {
				e.GetLogger(ctx).Error().Err(err).Msg("connection lost")
			}
		}
		time.Sleep(e.config.Timeout)
	}
}

// Shutdown shutdowns queue worker and connection watcher. Later will also
// close connection to database. This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.GetLogger(ctx).Info().Msg("shutting down")
	e.SetShuttingDown(true)

	if e.config.StartWatcher {
		for {
			if e.IsWatcherStopped() {
				break
			}
			time.Sleep(time.Millisecond * 500)
		}
	} else if err := e.shutdown(ctx); err != nil {
		return errors.Wrapf(err, "shutdown %q", e.GetFullName())
	}

	e.GetLogger(ctx).Info().Msg("shutted down")
	return nil
}

func (e *Enity) shutdown(ctx context.Context) error {
	if e.conn == nil {
		return nil
	}
	e.GetLogger(ctx).Info().Msg("closing connection...")

	if err := e.conn.Close(ctx); err != nil {
		return errors.Wrap(err, "close connection")
	}

	e.conn = nil

	return nil
}

func (e *Enity) buildMetrics(_ context.Context) error {
	fullName := e.GetFullName()
	redactedDSN, err := stringsx.RedactedDSN(e.config.DSN)
	if err != nil {
		return errors.Wrap(err, "redacted dsn")
	}
	help := stringsx.JoinStrings(" ", "status link to", redactedDSN)
	metricFunc := func(ctx context.Context) (float64, error) {
		err := e.Ping(ctx)
		if err == nil {
			return 1, nil
		}
		return 0, nil
	}
	if _, err := e.MetricsStorage.GetMetrics().AddGauge(fullName, "status", help, metricFunc); err != nil {
		return errors.Wrap(err, "add gauge metric")
	}

	return nil
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) buildReadyHandlers(_ context.Context) error {
	checkOptions := &base.CheckOptions{
		Name: strings.ToUpper(e.GetFullName() + "_notfailed"),
		CheckFunc: func(ctx context.Context) error {
			if e.conn == nil {
				return base.ErrNotConnected
			}

			if err := e.Ping(ctx); err != nil {
				return errors.Wrap(err, "ping")
			}

			return nil
		},
	}
	if err := e.ReadyCheckStorage.GetReadyHandlers().Add(checkOptions); err != nil {
		return errors.Wrap(err, "add ready handler")
	}
	return nil
}
