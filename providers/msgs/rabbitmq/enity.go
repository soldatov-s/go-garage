package rabbitmq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/soldatov-s/go-garage/x/helper"
	"github.com/streadway/amqp"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	*base.EntityWithMetrics
	cfg       *Config
	consumer  *Consumer
	publisher *Publisher
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, collectorName, providerName, name string, cfg interface{}) (*Enity, error) {
	e, err := base.NewEntityWithMetrics(ctx, collectorName, providerName, name)
	if err != nil {
		return nil, errors.Wrap(err, "create base enity")
	}

	// Checking that passed config is OUR.
	config, ok := cfg.(*Config)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityOptions, "expected %q", helper.ObjName(Config{}))
	}

	var consumer *Consumer
	if config.Consumer != nil {
		consumer = NewConsumer(config.Consumer)
	}

	var publisher *Publisher
	if config.Publisher != nil {
		publisher = NewPublisher(config.Publisher)
	}

	return &Enity{EntityWithMetrics: e, cfg: config.SetDefault(), consumer: consumer, publisher: publisher}, nil
}

func (e *Enity) connectConsumer(ctx context.Context) error {
	var err error

	e.consumer.Conn, err = amqp.Dial(e.cfg.DSN)
	if err != nil {
		e.GetLogger(ctx).Error().Msgf("consumer can't connect to rabbitMQ %s, error %s", e.cfg.DSN, err)
		return err
	}

	e.consumer.Channel, err = e.consumer.Conn.Channel()
	if err != nil {
		e.GetLogger(ctx).Error().Msgf("failed to open a channel: %s", err)
		return err
	}

	return nil
}

func (conn *Enity) connectPublisher(ctx context.Context) error {
	return conn.publisher.Connect(conn.cfg.DSN)
}

// Start starts connection workers and connection procedure itself.
func (conn *Enity) Start(ctx context.Context) error {
	if conn.ConnWatcherStopped {
		conn.ConnWatcherStopped = false
		// Connect publisher
		if conn.publisher != nil {
			if err := conn.connectPublisher(ctx); err != nil {
				return err
			}

			go conn.publishStatus(ctx)
		}
	}

	return nil
}

// Shutdown shutdowns close connection to database and connection watcher.
// This is a blocking call.
func (e *Enity) Shutdown(ctx context.Context) error {
	e.WeAreShuttingDown = true

	if e.consumer != nil && e.consumer.Conn != nil {
		e.GetLogger(ctx).Info().Msg("closing queue consumer connection...")

		err := e.consumer.Channel.Close()
		if err != nil {
			e.GetLogger(ctx).Error().Err(err).Msg("failed to close queue channel")
		}
		err = e.consumer.Conn.Close()
		if err != nil {
			e.GetLogger(ctx).Error().Err(err).Msg("failed to close queue connection")
		}

		e.consumer.Channel = nil
		e.consumer.Conn = nil

		close(e.consumer.shutdownConsumer)

		e.consumer.wg.Wait()
	}

	if e.publisher != nil {
		e.GetLogger(ctx).Info().Msg("closing queue publisher connection...")
		err := e.publisher.Shutdown()
		if err != nil {
			e.GetLogger(ctx).Error().Err(err).Msg("failed shutdown publisher")
		}
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established.
func (e *Enity) WaitForEstablishing(ctx context.Context) {
	for {
		if (e.consumer != nil && e.consumer.Conn != nil && e.publisher != nil && e.publisher.Conn != nil) ||
			(e.consumer == nil && e.publisher != nil && e.publisher.Conn != nil) ||
			(e.consumer != nil && e.consumer.Conn != nil && e.publisher == nil) {
			break
		}

		e.GetLogger(ctx).Debug().Msg("connection isn't ready - not yet established")
		time.Sleep(time.Millisecond * 100)
	}
}

func (e *Enity) publishStatus(ctx context.Context) {
	for {
		reason, ok := <-e.publisher.Channel.NotifyClose(make(chan *amqp.Error))
		if !ok {
			if e.WeAreShuttingDown {
				break
			}

			e.GetLogger(ctx).Error().Msgf("rabbitMQ channel unexpected closed %s", reason)
			err := e.connectPublisher(ctx)
			if err != nil {
				e.GetLogger(ctx).Error().Msgf("can't reconnect to rabbit %s", err)
				time.Sleep(10 * time.Second)
				continue
			}
		}
	}
}

func (e *Enity) runConsumer(ctx context.Context) (<-chan amqp.Delivery, error) {
	var err error

	err = e.connectConsumer(ctx)
	if err != nil {
		return nil, err
	}

	err = e.consumer.Channel.ExchangeDeclare(e.consumer.ExchangeName, "direct", true,
		false, false,
		false, nil)
	if err != nil {
		e.GetLogger(ctx).Error().Msgf("failed to declare a exchange: %s", err)
		return nil, err
	}

	_, err = e.consumer.Channel.QueueDeclare(
		e.consumer.RabbitQueue, // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	if err != nil {
		e.GetLogger(ctx).Error().Msgf("failed to declare a queue: %s", err)
		return nil, err
	}

	err = e.consumer.Channel.QueueBind(
		e.consumer.RabbitQueue,  // queue name
		e.consumer.RoutingKey,   // routing key
		e.consumer.ExchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		e.GetLogger(ctx).Error().Msgf("failed to bind to queue: %s", err)
		return nil, err
	}

	msg, consumeError := e.consumer.Channel.Consume(
		e.consumer.RabbitQueue,   // queue
		e.consumer.RabbitConsume, // consume
		false,                    // auto-ack
		false,                    // exclusive
		false,                    // no-local
		false,                    // no-wait
		nil,                      // args
	)
	if consumeError != nil {
		e.GetLogger(ctx).Error().Msgf("failed to consume message: %s", consumeError)
		return nil, consumeError
	}

	e.GetLogger(ctx).Info().Msgf("connected to Rabbitmq %s", e.cfg.DSN)
	return msg, nil
}

func (e *Enity) startConsume(ctx context.Context, options SubscribeOptions) {
	var msg <-chan amqp.Delivery
	var err error

	for {
		msg, err = e.runConsumer(ctx)
		if err != nil {
			e.GetLogger(ctx).Error().Msgf("consumer can't connect to rabbitMQ %s", err)
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	close(e.consumer.consumerStarted)

	e.consumer.wg.Add(1)
	defer e.consumer.wg.Done()

	for {
		select {
		case <-e.consumer.shutdownConsumer:
			e.GetLogger(ctx).Info().Msg("consumer shut down")
			options.Shutdown()
			return
		default:
			for d := range msg {
				e.GetLogger(ctx).Debug().Msgf("got new event %+v", string(d.Body))
				if err1 := options.Consume(d.Body); err1 != nil {
					e.GetLogger(ctx).Error().Msgf("can't consume. error: %s", err1)
					continue
				}
				_ = d.Ack(true)
			}
		}
	}
}

// SendMessage publish message to exchange
func (e *Enity) SendMessage(ctx context.Context, message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	e.GetLogger(ctx).Debug().Msgf("send message: %s", string(body))

	err = e.publisher.Channel.Publish(e.publisher.ExchangeName, e.publisher.RoutingKey, false,
		false, amqp.Publishing{ContentType: "text/plain", Body: body})
	if err != nil {
		for _, i := range e.cfg.BackoffPolicy {
			connErr := e.connectPublisher(ctx)
			if connErr != nil {
				e.GetLogger(ctx).Error().Msgf("error: %s. Trying to reconnect to rabbitMQ", connErr)
				time.Sleep(i * time.Second)
				continue
			}
			break
		}

		pubErr := e.publisher.Channel.Publish(e.publisher.ExchangeName, e.publisher.RoutingKey, false,
			false, amqp.Publishing{ContentType: "text/plain", Body: body})
		if pubErr != nil {
			e.GetLogger(ctx).Error().Msgf("failed to publish a message %s", pubErr)
			return pubErr
		}
	}

	return nil
}

// Subscribe to channel for receiving message
func (e *Enity) Subscribe(ctx context.Context, options SubscribeOptions) error {
	go e.subscribe(ctx, options)

	return nil
}

func (e *Enity) subscribe(ctx context.Context, options SubscribeOptions) {
	if e.consumer == nil {
		return
	}

	go e.startConsume(ctx, options)
	<-e.consumer.consumerStarted
	e.GetLogger(ctx).Info().Msgf("consumer started at %s queue %s", e.cfg.DSN, e.consumer.RabbitQueue)

	i := 0
	for {
		reason, ok := <-e.consumer.Channel.NotifyClose(make(chan *amqp.Error))
		if !ok {
			if e.WeAreShuttingDown {
				break
			}

			e.GetLogger(ctx).Error().Msgf("rabbitMQ channel unexpected closed %s", reason)
			if i == 0 {
				e.consumer.shutdownConsumer <- true
			}
			i++

			err := e.Ping(ctx)
			if err != nil {
				e.GetLogger(ctx).Error().Msgf("Can't reconnect to rabbit %s", err)
				time.Sleep(10 * time.Second)
				continue
			}

			i = 0
			e.consumer.consumerStarted = make(chan bool)
			go e.startConsume(ctx, options)
			<-e.consumer.consumerStarted
			e.GetLogger(ctx).Info().Msgf("Reconnected to rabbitMQ %s", e.cfg.DSN)
		}
	}
}

// Ping checks that rabbitMQ connections is live
func (e *Enity) Ping(ctx context.Context) (err error) {
	client, err := amqp.Dial(e.cfg.DSN)
	if err != nil {
		e.GetLogger(ctx).Error().Msgf("Can't rabbit to rabbitMQ %s, error %s", e.cfg.DSN, err)
		return
	}

	_ = client.Close()
	return
}

// GetMetrics return map of the metrics from database connection
func (e *Enity) GetMetrics(ctx context.Context) base.MapMetricsOptions {
	e.Metrics.AddNewMetricGauge(
		e.GetFullName(),
		"status",
		utils.JoinStrings(" ", "status link to", utils.RedactedDSN(e.cfg.DSN)),
		func() float64 {
			err := e.Ping(ctx)
			if err == nil {
				return 1
			}
			return 0
		},
	)

	return e.Metrics
}

// GetReadyHandlers return array of the readyHandlers from database connection
func (e *Enity) GetReadyHandlers(ctx context.Context) (base.MapCheckFunc, error) {
	e.ReadyHandlers[strings.ToUpper(e.GetFullName()+"_notfailed")] = func() (bool, string) {
		if e.consumer.Conn == nil && e.publisher.Conn == nil {
			return false, "Not connected"
		}

		if err := e.Ping(ctx); err != nil {
			return false, err.Error()
		}

		return true, ""
	}
	return e.ReadyHandlers, nil
}
