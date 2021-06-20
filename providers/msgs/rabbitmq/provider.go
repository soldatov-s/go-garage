package rabbitmq

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/x/helper"
)

// DefaultProviderName is default provider name
const ProviderName = "rabbitmq"

// Provider provides Rabbitmq worker. This provider
// supports asynchronous database actions (like bulk inserting). Every
// connection will have own goroutine for queue processing.
// Provider provides PostgreSQL database worker. This provider
// supports asynchronous database actions (like bulk inserting). Every
// connection will have own goroutine for queue processing.
type Provider struct {
	*base.ProviderWithMetrics
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) (*Provider, error) {
	p, err := base.NewProviderWithMetrics(ctx, db.CollectorName, ProviderName)
	if err != nil {
		return nil, errors.Wrap(err, "new provider with metrics")
	}
	return &Provider{p}, nil
}

// CreateEnity should create enity using passed parameters.
func (p *Provider) CreateEnity(ctx context.Context, enityName string, options interface{}) error {
	if _, err := p.GetEnity(enityName); err == nil {
		return base.ErrDuplicateEnity
	}

	enity, err := NewEnity(ctx, db.CollectorName, ProviderName, enityName, options)
	if err != nil {
		return errors.Wrap(err, "create enity")
	}

	p.Entitys.Store(enityName, enity)
	return nil
}

// getEnity should return pointer to enity structure to caller.
func (p *Provider) getEnity(enityName string) (*Enity, error) {
	enity, err := p.Provider.GetEnity(enityName)
	if err != nil {
		return nil, errors.Wrap(err, "get enity from base provider")
	}

	// Checking that enity type is *Enity
	enityPointer, ok := enity.(*Enity)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityPointer, "expect %q", helper.ObjName(Enity{}))
	}

	return enityPointer, nil
}

// GetEnity should return pointer to connection structure to caller.
func (p *Provider) GetEnity(enityName string) (interface{}, error) {
	return p.getEnity(enityName)
}
