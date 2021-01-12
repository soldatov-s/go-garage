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
func (p *Provider) RegisterReadyCheck(dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		return stats.ErrCheckFuncIsNil
	}

	var (
		err  error
		conn *Enity
	)
	p.Entitys.Range(func(k, _ interface{}) bool {
		conn, err = p.getEnity(k.(string))
		if err != nil {
			return false
		}
		if err = conn.RegisterReadyCheck(dependencyName, checkFunc); err != nil {
			return false
		}

		return true
	})

	return err
}

// RegisterAliveCheck should register a function for /health/alive
// endpoint.
func (p *Provider) RegisterAliveCheck(dependencyName string, checkFunc stats.CheckFunc) error {
	if dependencyName == "" {
		return stats.ErrEmptyDependencyName
	}

	if checkFunc == nil {
		return stats.ErrCheckFuncIsNil
	}

	var (
		err  error
		conn *Enity
	)
	p.Entitys.Range(func(k, _ interface{}) bool {
		conn, err = p.getEnity(k.(string))
		if err != nil {
			return false
		}
		if err = conn.RegisterAliveCheck(dependencyName, checkFunc); err != nil {
			return false
		}

		return true
	})

	return err
}

// RegisterMetric should register a metric of defined type. Passed
// metricName should be used only as internal identifier. Provider
// should provide instructions for using metricOptions as well as
// cast to appropriate type.
func (p *Provider) RegisterMetric(metricName string, options interface{}) error {
	if metricName == "" {
		return stats.ErrEmptyMetricName
	}

	var (
		err  error
		conn *Enity
	)
	p.Entitys.Range(func(k, _ interface{}) bool {
		conn, err = p.getEnity(k.(string))
		if err != nil {
			return false
		}
		if err = conn.RegisterMetric(metricName, options); err != nil {
			return false
		}

		return true
	})
	return err
}
