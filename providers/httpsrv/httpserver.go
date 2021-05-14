package httpsrv

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
)

const (
	CollectorName = "httpsrv"
)

type MiddleWareFunc func(http.Handler) http.Handler

// Provider is an interface every provider should follow.
type ProviderGateway interface {
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

// Collector structure controls all HTTP servers for gowork.
type Collector struct {
	*base.Collector
}

func NewCollector(ctx context.Context) (*Collector, error) {
	c, err := base.NewCollector(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "new collector with metrics")
	}
	return &Collector{c}, nil
}

// GetAPIVersionGroup returns HTTP server routing group using designated
// provider.
func (https *Collector) GetAPIVersionGroup(providerName, serverName, apiVersion string) (interface{}, error) {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return prov.GetAPIVersionGroup(serverName, apiVersion)
}

// CreateAPIVersionGroup creates new API version group using designated
// provider.
func (https *Collector) CreateAPIVersionGroup(providerName, serverName, apiVersion string, middlewares ...interface{}) error {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.CreateAPIVersionGroup(serverName, apiVersion, middlewares...)
}

// GetProvider returns requested caches provider. It'll return error if
// providers wasn't registered.
func (c *Collector) GetProvider(providerName string) (ProviderGateway, error) {
	p, err := c.Collector.GetProvider(providerName)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	pg, ok := p.(ProviderGateway)
	if !ok {
		return nil, errors.Wrap(base.ErrBadTypeOfProvider, "expected ProviderGateway")
	}

	return pg, nil
}

// RegisterEndpoint register endpoint on selected HTTP server.
func (https *Collector) RegisterEndpoint(
	providerName, serverName, method, endpoint string,
	handler http.Handler,
	m ...MiddleWareFunc) error {
	prov, err := https.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.RegisterEndpoint(serverName, method, endpoint, handler, m...)
}

func NewContext(ctx context.Context) (context.Context, error) {
	c, err := NewCollector(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "new collector %q", CollectorName)
	}
	return base.NewContextByName(ctx, CollectorName, c)
}

func FromContext(ctx context.Context) (*Collector, error) {
	v, err := base.FromContextByName(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "get from context by name")
	}
	c, ok := v.(*Collector)
	if !ok {
		return nil, base.ErrFailedTypeCast
	}
	return c, nil
}
