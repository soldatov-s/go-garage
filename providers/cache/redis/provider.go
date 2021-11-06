package redis

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/x/helper"
)

// ProviderName is default provider name
const ProviderName = "redis"

// Provider provides Redis database worker. This provider
// supports asynchronous database actions (like bulk inserting). Every
// connection will have own goroutine for queue processing.
type Provider struct {
	*base.ProviderWithMetrics
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) (*Provider, error) {
	p, err := base.NewProviderWithMetrics(ctx, cache.CollectorName, ProviderName)
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

	enity, err := NewEnity(ctx, cache.CollectorName, ProviderName, enityName, options)
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

// Get item from cache by key.
func (p *Provider) Get(ctx context.Context, enityName, key string, value interface{}) error {
	conn, err := p.getEnity(enityName)
	if err != nil {
		return errors.Wrap(err, "get enity")
	}

	if err := conn.Get(ctx, key, value); err != nil {
		return errors.Wrap(err, "get item from cache")
	}

	return nil
}

// Set item in cache by key.
func (p *Provider) Set(ctx context.Context, enityName, key string, value interface{}) error {
	conn, err := p.getEnity(enityName)
	if err != nil {
		return errors.Wrap(err, "get enity")
	}

	if err := conn.Set(ctx, key, value); err != nil {
		return errors.Wrap(err, "set item to cache")
	}

	return nil
}

// SetNX (Not eXist) item in cache by key.
func (p *Provider) SetNX(ctx context.Context, enityName, key string, value interface{}) error {
	conn, err := p.getEnity(enityName)
	if err != nil {
		return errors.Wrap(err, "get enity")
	}

	if err := conn.SetNX(ctx, key, value); err != nil {
		return errors.Wrap(err, "setNX item to cache")
	}

	return nil
}

// Delete item from cache by key.
func (p *Provider) Delete(ctx context.Context, enityName, key string) error {
	conn, err := p.getEnity(enityName)
	if err != nil {
		return errors.Wrap(err, "get enity")
	}

	return conn.Delete(ctx, key)
}

// Clear clear all items from selected connection.
func (p *Provider) Clear(ctx context.Context, enityName string) error {
	conn, err := p.getEnity(enityName)
	if err != nil {
		return errors.Wrap(err, "get enity")
	}

	if err := conn.Clear(ctx); err != nil {
		return errors.Wrap(err, "cleare items in cache")
	}

	return nil
}

// ClearAll clear all items from all connections.
func (p *Provider) ClearAll(ctx context.Context) error {
	var err error
	p.Entitys.Range(func(_, v interface{}) bool {
		err = v.(*Enity).Clear(ctx)
		return err == nil
	})

	return err
}
