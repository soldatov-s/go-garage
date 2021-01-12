package garage

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/stats"
)

const (
	DefaultProviderName = "stats"
)

type Provider struct {
	*provider.Provider
}

// Initialize initializes provider.
func NewProvider(ctx context.Context) *Provider {
	return &Provider{
		Provider: provider.NewProvider(ctx, stats.ProvidersName, DefaultProviderName),
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

// RegisterReadyCheck should register a function for /health/ready
// endpoint.
func (p *Provider) RegisterReadyCheck(connectionName, dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		return stats.ErrCheckFuncIsNil
	}

	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.RegisterReadyCheck(dependencyName, checkFunc)
}

// RegisterAliveCheck should register a function for /health/alive
// endpoint.
func (p *Provider) RegisterAliveCheck(connectionName, dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		return stats.ErrCheckFuncIsNil
	}

	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.RegisterAliveCheck(dependencyName, checkFunc)
}

// RegisterMetric should register a metric of defined type. Passed
// metricName should be used only as internal identifier. Provider
// should provide instructions for using metricOptions as well as
// cast to appropriate type.
func (p *Provider) RegisterMetric(connectionName, metricName string, options interface{}) error {
	if metricName == "" {
		return stats.ErrEmptyMetricName
	}

	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	return conn.RegisterMetric(metricName, options)
}
