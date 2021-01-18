package redis

import (
	"context"

	providerwithmetrics "github.com/soldatov-s/go-garage/providers/base/provider/with_metrics"
	"github.com/soldatov-s/go-garage/providers/cache"
	"github.com/soldatov-s/go-garage/providers/errors"
)

// DefaultProviderName is default provider name
const DefaultProviderName = "redis"

// Provider provides Redis database worker. This provider
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
		Provider: providerwithmetrics.NewProvider(ctx, cache.ProvidersName, DefaultProviderName),
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

// Get item from cache by key.
func (p *Provider) Get(connectionName, key string, value interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.Get(key, value)
}

// Set item in cache by key.
func (p *Provider) Set(connectionName, key string, value interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.Set(key, value)
}

// SetNX (Not eXist) item in cache by key.
func (p *Provider) SetNX(connectionName, key string, value interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.SetNX(key, value)
}

// Delete item from cache by key.
func (p *Provider) Delete(connectionName, key string) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.Delete(key)
}

// Clear clear all items from selected connection.
func (p *Provider) Clear(connectionName string) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.Clear()
}

// ClearAll clear all items from all connections.
func (p *Provider) ClearAll() error {
	var err error
	p.Entitys.Range(func(_, v interface{}) bool {
		if err = v.(*Enity).Clear(); err != nil {
			return false
		}
		return true
	})

	return err
}
