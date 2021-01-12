package rabbitmq

import (
	"context"

	providerwithmetrics "github.com/soldatov-s/go-garage/providers/base/provider/with_metrics"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/msgs"
)

// DefaultProviderName is default provider name
const DefaultProviderName = "rabbitmq"

// Provider provides Rabbitmq worker. This provider
// supports asynchronous database actions (like bulk inserting). Every
// connection will have own goroutine for queue processing.
type Provider struct {
	*providerwithmetrics.Provider
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) *Provider {
	return &Provider{
		Provider: providerwithmetrics.NewProvider(ctx, msgs.ProvidersName, DefaultProviderName),
	}
}

// CreateEnity should create enity using passed parameters.
func (p *Provider) CreateEnity(enityName string, options interface{}) error {
	if _, err := p.GetEnity(enityName); err == nil {
		p.Log.Debug().Str("enity name", enityName).Msg("enity already created")
		return nil
	}

	enity, err := NewEnity(p.GetContext(), enityName, options)
	if err != nil {
		return err
	}

	p.Entitys.Store(enityName, enity)

	return nil
}

// GetEnity should return pointer to connection structure to caller.
func (p *Provider) getEnity(enityName string) (*Enity, error) {
	if enityName == "" {
		return nil, errors.ErrEmptyEnityName
	}

	enity, found := p.Entitys.Load(enityName)
	if !found {
		return nil, errors.ErrEnityDoesNotExists
	}

	// Checking that enity type is *Enity
	enityPointer, ok := enity.(*Enity)
	if !ok {
		return nil, errors.ErrInvalidEnityPointer(Enity{})
	}

	return enityPointer, nil
}

// GetEnity should return pointer to connection structure to caller.
func (p *Provider) GetEnity(connectionName string) (interface{}, error) {
	return p.getEnity(connectionName)
}

// SendMessage send message.
func (p *Provider) SendMessage(connectionName string, message interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.SendMessage(message)
}

// Subscribe subscribe for reciving messages.
func (p *Provider) Subscribe(connectionName string, options interface{}) error {
	subscribeOptions, ok := options.(*SubscribeOptions)
	if !ok {
		return msgs.ErrInvalidSubscribeOptionsPointer(SubscribeOptions{})
	}

	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.Subscribe(subscribeOptions)
}
