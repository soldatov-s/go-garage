package echo

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/httpsrv"
	"github.com/soldatov-s/go-garage/x/helper"
)

const ProviderName = "echo"

// Provider is a HTTP server provider using labstack's Echo
// HTTP framework.
type Provider struct {
	*base.Provider
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) (*Provider, error) {
	p, err := base.NewProvider(ctx, httpsrv.CollectorName, ProviderName)
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

	enity, err := NewEnity(ctx, httpsrv.CollectorName, ProviderName, enityName, options)
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
func (p *Provider) GetEnity(connectionName string) (interface{}, error) {
	return p.getEnity(connectionName)
}

// CreateAPIVersionGroup creates requested API version routes group.
// Passed apiVersion should be a string with API version without
// leading "v", "version" or anything like that.
func (p *Provider) CreateAPIVersionGroup(ctx context.Context, serverName, apiVersion string, middlewares ...interface{},
) error {
	enity, err := p.getEnity(serverName)
	if err != nil {
		return err
	}

	return enity.CreateAPIVersionGroup(ctx, apiVersion, middlewares...)
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
