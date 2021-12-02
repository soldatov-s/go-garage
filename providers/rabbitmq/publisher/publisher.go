package rabbitmqpub

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	rabbitmqcon "github.com/soldatov-s/go-garage/providers/rabbitmq/connection"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"github.com/streadway/amqp"
)

const (
	ProviderName = "rabbitmq"
)

// Publisher is a RabbitPublisher
type Publisher struct {
	*base.MetricsStorage
	config      *Config
	conn        **rabbitmqcon.Connection
	isConnected bool
	name        string
	muConn      sync.Mutex

	okMessages  func(ctx context.Context) error
	badMessages func(ctx context.Context) error
}

func NewPublisher(ctx context.Context, config *Config, conn **rabbitmqcon.Connection) (*Publisher, error) {
	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	enity := &Publisher{
		MetricsStorage: base.NewMetricsStorage(),
		config:         config,
		conn:           conn,
		name:           stringsx.JoinStrings("_", config.ExchangeName, config.RoutingKey),
	}
	if err := enity.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	return enity, nil
}

func (p *Publisher) connect(_ context.Context) error {
	p.muConn.Lock()
	defer p.muConn.Unlock()
	if p.isConnected {
		return nil
	}

	conn := *p.conn
	if err := conn.Channel().ExchangeDeclare(p.config.ExchangeName, "direct", true,
		false, false,
		false, nil); err != nil {
		return errors.Wrap(err, "declare a exchange")
	}

	p.isConnected = true

	return nil
}

// SendMessage publish message to exchange
func (p *Publisher) SendMessage(ctx context.Context, message interface{}) error {
	logger := zerolog.Ctx(ctx)

	body, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "marshal message")
	}

	ampqMsg := buildMessage(body)

	logger.Debug().Msgf("send message: %s", string(body))

	if !p.isConnected {
		if err := p.connect(ctx); err != nil {
			logger.Err(err).Msg("connect publisher to rabbitMQ")
		}
	}

	// We try to send message twice. Between attempts we try to reconnect.
	if err := p.sendMessage(ctx, ampqMsg); err != nil {
		if errRetryPub := p.sendMessage(ctx, ampqMsg); err != nil {
			if errBadMsg := p.badMessages(ctx); errBadMsg != nil {
				return errors.Wrap(errBadMsg, "count bad messages")
			}
			return errors.Wrap(errRetryPub, "retry publish a message")
		}
	}

	if err := p.okMessages(ctx); err != nil {
		return errors.Wrap(err, "count ok messages")
	}

	return nil
}

func (p *Publisher) sendMessage(ctx context.Context, ampqMsg *amqp.Publishing) error {
	logger := zerolog.Ctx(ctx)
	if !p.isConnected {
		if err := p.connect(ctx); err != nil {
			logger.Err(err).Msg("connect publisher to rabbitMQ")
		}
	}

	conn := *p.conn
	if err := conn.Channel().Publish(
		p.config.ExchangeName,
		p.config.RoutingKey,
		false,
		false,
		*ampqMsg,
	); err != nil {
		p.muConn.Lock()
		p.isConnected = false
		p.muConn.Unlock()
		return errors.Wrap(err, "publish a message")
	}
	return nil
}

func buildMessage(body []byte) *amqp.Publishing {
	return &amqp.Publishing{
		ContentType: "text/plain",
		Body:        body,
	}
}

func (p *Publisher) buildMetrics(_ context.Context) error {
	fullName := p.name

	helpOKMessages := "ok send messages to exchange"
	okMessages, err := p.MetricsStorage.GetMetrics().AddIncCounter(fullName, "ok send messages", helpOKMessages)
	if err != nil {
		return errors.Wrap(err, "add inc metric")
	}

	p.okMessages = func(ctx context.Context) error {
		okMessages.Inc()
		return nil
	}

	helpBadMessages := "bad send messages to exchange"
	badMessages, err := p.MetricsStorage.GetMetrics().AddIncCounter(fullName, "bad send messages", helpBadMessages)
	if err != nil {
		return errors.Wrap(err, "add inc metric")
	}

	p.badMessages = func(ctx context.Context) error {
		badMessages.Inc()
		return nil
	}

	return nil
}
