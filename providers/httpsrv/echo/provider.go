package echo

import (
	"context"
	"net/http"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
)

const DefaultProviderName = "echo"

// Provider is a HTTP server provider using labstack's Echo
// HTTP framework.
type Provider struct {
	*provider.Provider
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) *Provider {
	return &Provider{
		Provider: provider.NewProvider(ctx, httpsrv.ProvidersName, DefaultProviderName),
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

// CreateAPIVersionGroup creates requested API version routes group.
// Passed apiVersion should be a string with API version without
// leading "v", "version" or anything like that.
func (p *Provider) CreateAPIVersionGroup(serverName, apiVersion string, middlewares ...interface{},
) error {
	enity, err := p.getEnity(serverName)
	if err != nil {
		return err
	}

	return enity.CreateAPIVersionGroup(apiVersion, middlewares...)
}

// GetAPIVersionGroup returns routes grouping for passed apiVersion.
// Passed apiVersion should be a string with API version without
// leading "v", "version" or anything like that.
func (p *Provider) GetAPIVersionGroup(serverName, apiVersion string) (interface{}, error) {
	enity, err := p.getEnity(serverName)
	if err != nil {
		return nil, err
	}

	return enity.GetAPIVersionGroup(apiVersion)
}

// RegisterEndpoint register endpoint on HTTP server.
func (p *Provider) RegisterEndpoint(
	serverName, method, endpoint string,
	handler http.Handler,
	middleware ...httpsrv.MiddleWareFunc) error {
	enity, err := p.getEnity(serverName)
	if err != nil {
		return err
	}

	return enity.RegisterEndpoint(method, endpoint, handler, middleware...)
}
