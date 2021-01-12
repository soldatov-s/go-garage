package httpsrv

import (
	"context"
	"net/http"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/base/providers"
	"github.com/soldatov-s/go-garage/providers/errors"
)

const (
	ProvidersName = "httpsrv"
)

type MiddleWareFunc func(http.Handler) http.Handler

// Provider is an interface every provider should follow.
type Provider interface {
	provider.IProvider
	provider.IEnityManager

	// CreateAPIVersionGroup should be responsible for creating
	// routes grouping for specified API version. Version should be
	// a string without leading "v" or "version" or anything like that.
	// Provider should append "v" automatically.
	// Passed group should be a pointer to needed structure that might be
	// provided by HTTP server provider. Provider should not attempt to
	// replace pointer if nil is passed. Provider should check if passed
	// group is a pointer to valid structure.
	// It is up to provider to cast options and middlewares to needed type.
	CreateAPIVersionGroup(serverName string, apiVersion string, middlewares ...interface{}) error
	// GetAPIVersionGroup should return handler for passed version
	// API. Version should be a string without leading "v" or "version"
	// or anything like that. Provider should append "v" automatically.
	// Passed group should be a pointer to needed structure that might be
	// provided by HTTP server provider. Provider should not attempt to
	// replace pointer if nil is passed. Provider should check if passed
	// group is a pointer to valid structure.
	GetAPIVersionGroup(serverName string, apiVersion string) (interface{}, error)
	// Register endpoint on server
	RegisterEndpoint(serverName, method, endpoint string, handler http.Handler, middleware ...MiddleWareFunc) error
}

// HTTPServers structure controls all HTTP servers for gowork.
type HTTPServers struct {
	*providers.BaseProviders
}

func NewHTTPServers(ctx context.Context) *HTTPServers {
	return &HTTPServers{
		BaseProviders: providers.NewBaseProviders(ctx, ProvidersName),
	}
}

// CreateEnity creates new HTTP server using designated provider.
func (https *HTTPServers) CreateEnity(providerName, serverName string, options interface{}) error {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.CreateEnity(serverName, options)
}

// GetEnity returns HTTP server controlling structure from designated
// provider.
func (https *HTTPServers) GetEnity(providerName, serverName string) (interface{}, error) {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return prov.GetEnity(serverName)
}

// GetAPIVersionGroup returns HTTP server routing group using designated
// provider.
func (https *HTTPServers) GetAPIVersionGroup(providerName, serverName, apiVersion string) (interface{}, error) {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return prov.GetAPIVersionGroup(serverName, apiVersion)
}

// CreateAPIVersionGroup creates new API version group using designated
// provider.
func (https *HTTPServers) CreateAPIVersionGroup(providerName, serverName, apiVersion string, middlewares ...interface{}) error {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.CreateAPIVersionGroup(serverName, apiVersion, middlewares...)
}

// GetProvider returns provider.
// Returns error if provider isn't registered.
func (https *HTTPServers) GetProvider(providerName string) (Provider, error) {
	if v, err := https.BaseProviders.GetProvider(providerName); err != nil {
		return nil, err
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// RegisterEndpoint register endpoint on selected HTTP server.
func (https *HTTPServers) RegisterEndpoint(
	providerName, serverName, method, endpoint string,
	handler http.Handler,
	m ...MiddleWareFunc) error {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterEndpoint(serverName, method, endpoint, handler, m...)
}
