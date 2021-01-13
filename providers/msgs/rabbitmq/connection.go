package rabbitmq

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
	"github.com/soldatov-s/go-garage/providers/msgs"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/utils"
	"github.com/streadway/amqp"
)

// Enity is a connection controlling structure. It controls
// connection, asynchronous queue and everything that related to
// specified connection.
type Enity struct {
	ctx     context.Context
	log     zerolog.Logger
	name    string
	options *Config

	consumer  *Consumer
	publisher *Publisher

	// Shutdown flags.
	weAreShuttingDown  bool
	connWatcherStopped bool

	// Metrics
	stats.Service
}

// NewEnity create new enity.
func NewEnity(ctx context.Context, name string, opts interface{}) (*Enity, error) {
	if name == "" {
		return nil, errors.ErrEmptyEnityName
	}

	conn := &Enity{
		name: name,
		ctx:  logger.Registrate(ctx),
	}

	logger.Get(conn.ctx).GetLogger(msgs.ProvidersName, nil).Info().Msgf("initializing enity " + name + "...")
	conn.log = logger.Get(conn.ctx).GetLogger(msgs.ProvidersName, nil).With().Str("connection", name).Logger()

	// We should not attempt to establish connection if passed options
	// isn't OUR options.
	var ok bool
	conn.options, ok = opts.(*Config)
	if !ok {
		return nil, errors.ErrInvalidEnityOptions(Config{})
	}

	conn.options.Validate()

	if conn.options.Consumer != nil {
		conn.consumer = NewConsumer(conn.options.Consumer)
	}

	if conn.options.Publisher != nil {
		conn.publisher = NewPublisher(conn.options.Publisher)
	}

	conn.name = name

	conn.connWatcherStopped = true

	return conn, nil
}

func (conn *Enity) connectConsumer() error {
	var err error

	conn.consumer.Conn, err = amqp.Dial(conn.options.DSN)
	if err != nil {
		conn.log.Error().Msgf("Consumer can't connect to rabbitMQ %s, error %s", conn.options.DSN, err)
		return err
	}

	conn.consumer.Channel, err = conn.consumer.Conn.Channel()
	if err != nil {
		conn.log.Error().Msgf("Failed to open a channel: %s", err)
		return err
	}

	return nil
}

func (conn *Enity) connectPublisher() error {
	return conn.publisher.Connect(conn.options.DSN)
}

// Start starts connection workers and connection procedure itself.
func (conn *Enity) Start() error {
	if conn.connWatcherStopped {
		conn.connWatcherStopped = false
		// Connect publisher
		if conn.publisher != nil {
			if err := conn.connectPublisher(); err != nil {
				return err
			}

			go conn.publishStatus()
		}
	}

	return nil
}

// Shutdown shutdowns close connection to database and connection watcher.
// This is a blocking call.
func (conn *Enity) Shutdown() error {
	conn.weAreShuttingDown = true

	if conn.consumer != nil && conn.consumer.Conn != nil {
		conn.log.Info().Msg("Closing queue consumer connection...")

		err := conn.consumer.Channel.Close()
		if err != nil {
			conn.log.Error().Err(err).Msg("Failed to close queue channel")
		}
		err = conn.consumer.Conn.Close()
		if err != nil {
			conn.log.Error().Err(err).Msg("Failed to close queue connection")
		}

		conn.consumer.Channel = nil
		conn.consumer.Conn = nil

		close(conn.consumer.shutdownConsumer)

		conn.consumer.wg.Wait()
	}

	if conn.publisher != nil {
		conn.log.Info().Msg("Closing queue publisher connection...")
		err := conn.publisher.Shutdown()
		if err != nil {
			conn.log.Error().Err(err).Msg("failed shutdown publisher")
		}
	}

	return nil
}

// WaitForEstablishing will block execution until connection will be
// successfully established.
func (conn *Enity) WaitForEstablishing() {
	for {
		if (conn.consumer != nil && conn.consumer.Conn != nil && conn.publisher != nil && conn.publisher.Conn != nil) ||
			(conn.consumer == nil && conn.publisher != nil && conn.publisher.Conn != nil) ||
			(conn.consumer != nil && conn.consumer.Conn != nil && conn.publisher == nil) {
			break
		}

		conn.log.Debug().Msg("Connection isn't ready - not yet established")
		time.Sleep(time.Millisecond * 100)
	}
}

func (conn *Enity) publishStatus() {
	for {
		reason, ok := <-conn.publisher.Channel.NotifyClose(make(chan *amqp.Error))
		if !ok {
			if conn.weAreShuttingDown {
				break
			}

			conn.log.Error().Msgf("RabbitMQ channel unexpected closed %s", reason)
			err := conn.connectPublisher()
			if err != nil {
				conn.log.Error().Msgf("Can't reconnect to rabbit %s", err)
				time.Sleep(10 * time.Second)
				continue
			}
		}
	}
}

func (conn *Enity) runConsumer() (<-chan amqp.Delivery, error) {
	var err error

	err = conn.connectConsumer()
	if err != nil {
		return nil, err
	}

	err = conn.consumer.Channel.ExchangeDeclare(conn.consumer.ExchangeName, "direct", true,
		false, false,
		false, nil)
	if err != nil {
		conn.log.Error().Msgf("Failed to declare a exchange: %s", err)
		return nil, err
	}

	_, err = conn.consumer.Channel.QueueDeclare(
		conn.consumer.RabbitQueue, // name
		true,                      // durable
		false,                     // delete when unused
		false,                     // exclusive
		false,                     // no-wait
		nil,                       // arguments
	)
	if err != nil {
		conn.log.Error().Msgf("Failed to declare a queue: %s", err)
		return nil, err
	}

	err = conn.consumer.Channel.QueueBind(
		conn.consumer.RabbitQueue,  // queue name
		conn.consumer.RoutingKey,   // routing key
		conn.consumer.ExchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		conn.log.Error().Msgf("Failed to bind to queue: %s", err)
		return nil, err
	}

	msg, consumeError := conn.consumer.Channel.Consume(
		conn.consumer.RabbitQueue,   // queue
		conn.consumer.RabbitConsume, // consume
		false,                       // auto-ack
		false,                       // exclusive
		false,                       // no-local
		false,                       // no-wait
		nil,                         // args
	)
	if consumeError != nil {
		conn.log.Error().Msgf("Failed to consume message: %s", consumeError)
		return nil, consumeError
	}

	conn.log.Info().Msgf("Connected to Rabbitmq %s", conn.options.DSN)
	return msg, nil
}

func (conn *Enity) startConsume(options *SubscribeOptions) {
	var msg <-chan amqp.Delivery
	var err error

	for {
		msg, err = conn.runConsumer()
		if err != nil {
			conn.log.Error().Msgf("Consumer can't connect to rabbitMQ %s", err)
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}

	close(conn.consumer.consumerStarted)

	conn.consumer.wg.Add(1)
	defer conn.consumer.wg.Done()

	for {
		select {
		case <-conn.consumer.shutdownConsumer:
			conn.log.Info().Msg("Consumer shut down")
			if options.Shutdownhndl != nil {
				options.Shutdownhndl()
			}
			return
		default:
			for d := range msg {
				conn.log.Debug().Msgf("Got new event %+v", string(d.Body))
				if options.ConsumeHndl != nil {
					if err1 := options.ConsumeHndl(d.Body); err1 != nil {
						conn.log.Error().Msgf("Can't consume. error: %s", err1)
						continue
					}
					_ = d.Ack(true)
				}
			}
		}
	}
}

// SendMessage publish message to exchange
func (conn *Enity) SendMessage(message interface{}) error {
	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	conn.log.Debug().Msgf("send message: %s", string(body))

	err = conn.publisher.Channel.Publish(conn.publisher.ExchangeName, conn.publisher.RoutingKey, false,
		false, amqp.Publishing{ContentType: "text/plain", Body: body})
	if err != nil {
		for _, i := range conn.options.BackoffPolicy {
			connErr := conn.connectPublisher()
			if connErr != nil {
				conn.log.Error().Msgf("Error: %s. Trying to reconnect to rabbitMQ", connErr)
				time.Sleep(i * time.Second)
				continue
			}
			break
		}

		pubErr := conn.publisher.Channel.Publish(conn.publisher.ExchangeName, conn.publisher.RoutingKey, false,
			false, amqp.Publishing{ContentType: "text/plain", Body: body})
		if pubErr != nil {
			conn.log.Error().Msgf("Failed to publish a message %s", pubErr)
			return pubErr
		}
	}

	return nil
}

// Subscribe to channel for receiving message
func (conn *Enity) Subscribe(options *SubscribeOptions) error {
	go conn.subscribe(options)

	return nil
}

func (conn *Enity) subscribe(options *SubscribeOptions) {
	if conn.consumer == nil {
		return
	}

	go conn.startConsume(options)
	<-conn.consumer.consumerStarted
	conn.log.Info().Msgf("Consumer started at %s queue %s", conn.options.DSN, conn.consumer.RabbitQueue)

	i := 0
	for {
		reason, ok := <-conn.consumer.Channel.NotifyClose(make(chan *amqp.Error))
		if !ok {
			if conn.weAreShuttingDown {
				break
			}

			conn.log.Error().Msgf("RabbitMQ channel unexpected closed %s", reason)
			if i == 0 {
				conn.consumer.shutdownConsumer <- true
			}
			i++

			err := conn.Ping()
			if err != nil {
				conn.log.Error().Msgf("Can't reconnect to rabbit %s", err)
				time.Sleep(10 * time.Second)
				continue
			}

			i = 0
			conn.consumer.consumerStarted = make(chan bool)
			go conn.startConsume(options)
			<-conn.consumer.consumerStarted
			conn.log.Info().Msgf("Reconnected to rabbitMQ %s", conn.options.DSN)
		}
	}
}

// Ping checks that rabbitMQ connections is live
func (conn *Enity) Ping() (err error) {
	client, err := amqp.Dial(conn.options.DSN)
	if err != nil {
		conn.log.Error().Msgf("Can't rabbit to rabbitMQ %s, error %s", conn.options.DSN, err)
		return
	}

	_ = client.Close()
	return
}

// GetMetrics return map of the metrics from cache connection
func (conn *Enity) GetMetrics(prefix string) stats.MapMetricsOptions {
	_ = conn.Service.GetMetrics(prefix)
	conn.Metrics[prefix+"_"+conn.name+"_status"] = &stats.MetricOptions{
		Metric: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: prefix + "_" + conn.name + "_status",
				Help: prefix + " " + conn.name + " status link to " + utils.RedactedDSN(conn.options.DSN),
			}),
		MetricFunc: func(m interface{}) {
			(m.(prometheus.Gauge)).Set(0)
			err := conn.Ping()
			if err == nil {
				(m.(prometheus.Gauge)).Set(1)
			}
		},
	}
	return conn.Metrics
}

// GetReadyHandlers return array of the readyHandlers from cache connection
func (conn *Enity) GetReadyHandlers(prefix string) stats.MapCheckFunc {
	_ = conn.Service.GetReadyHandlers(prefix)
	conn.ReadyHandlers[strings.ToUpper(prefix+"_"+conn.name+"_notfailed")] = func() (bool, string) {
		if conn.consumer.Conn == nil && conn.publisher.Conn == nil {
			return false, "Not connected"
		}

		if err := conn.Ping(); err != nil {
			return false, err.Error()
		}

		return true, ""
	}

	return conn.ReadyHandlers
}
