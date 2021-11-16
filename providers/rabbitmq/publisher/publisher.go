package rabbitmqpub

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/base"
	"github.com/soldatov-s/go-garage/x/stringsx"
	"github.com/streadway/amqp"
)

const (
	ProviderName = "rabbitmq"
)

type Connector interface {
	Channel() *amqp.Channel
}

// Publisher is a RabbitPublisher
type Publisher struct {
	*base.MetricsStorage
	config      *Config
	conn        Connector
	isConnected bool
	name        string

	okMessages  func(ctx context.Context) error
	badMessages func(ctx context.Context) error
}

func NewPublisher(ctx context.Context, name string, config *Config, conn Connector) (*Publisher, error) {
	if config == nil {
		return nil, base.ErrInvalidEnityOptions
	}

	enity := &Publisher{
		MetricsStorage: base.NewMetricsStorage(),
		config:         config,
		conn:           conn,
		name:           name,
	}
	if err := enity.buildMetrics(ctx); err != nil {
		return nil, errors.Wrap(err, "build metrics")
	}

	return enity, nil
}

func (p *Publisher) connect(_ context.Context) error {
	if err := p.conn.Channel().ExchangeDeclare(p.config.ExchangeName, "direct", true,
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

	logger.Debug().Msgf("send message: %s", string(body))

	if !p.isConnected {
		if err := p.connect(ctx); err != nil {
			logger.Err(err).Msg("connect publisher to rabbitMQ")
		}
	}

	if err := p.conn.Channel().Publish(p.config.ExchangeName, p.config.RoutingKey, false,
		false, amqp.Publishing{ContentType: "text/plain", Body: body}); err != nil {
		p.isConnected = false
		if errBadMsg := p.badMessages(ctx); errBadMsg != nil {
			return errors.Wrap(errBadMsg, "count bad messages")
		}
		return errors.Wrap(err, "publish a message")
	}

	if err := p.okMessages(ctx); err != nil {
		return errors.Wrap(err, "count ok messages")
	}

	return nil
}

func (p *Publisher) buildMetrics(_ context.Context) error {
	fullName := stringsx.JoinStrings("_", p.name, p.config.ExchangeName, p.config.RoutingKey)

	helpOKMessages := "ok send messages to exchange"
	okMessages, err := p.MetricsStorage.GetMetrics().AddMetricIncCounter(fullName, "ok send messages", helpOKMessages)
	if err != nil {
		return errors.Wrap(err, "add inc metric")
	}

	p.okMessages = func(ctx context.Context) error {
		okMessages.Inc()
		return nil
	}

	helpBadMessages := "bad send messages to exchange"
	badMessages, err := p.MetricsStorage.GetMetrics().AddMetricIncCounter(fullName, "bad send messages", helpBadMessages)
	if err != nil {
		return errors.Wrap(err, "add inc metric")
	}

	p.badMessages = func(ctx context.Context) error {
		badMessages.Inc()
		return nil
	}

	return nil
}
