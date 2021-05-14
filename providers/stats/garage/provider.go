package garage

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/x/helper"
)

const (
	ProviderName = "stats"
)

type Provider struct {
	*base.Provider
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) (*Provider, error) {
	p, err := base.NewProvider(ctx, db.CollectorName, ProviderName)
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

// RegisterReadyCheck should register a function for /health/ready
// endpoint.
// nolint : dupl
func (p *Provider) RegisterReadyCheck(ctx context.Context, dependencyName string, checkFunc stats.CheckFunc) error {
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
		if err = conn.RegisterReadyCheck(ctx, dependencyName, checkFunc); err != nil {
			return false
		}

		return true
	})

	return err
}

// RegisterAliveCheck should register a function for /health/alive
// endpoint.
// nolint : dupl
func (p *Provider) RegisterAliveCheck(ctx context.Context, dependencyName string, checkFunc stats.CheckFunc) error {
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
		if err = conn.RegisterAliveCheck(ctx, dependencyName, checkFunc); err != nil {
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
func (p *Provider) RegisterMetric(ctx context.Context, metricName string, options interface{}) error {
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
		if err = conn.RegisterMetric(ctx, metricName, options); err != nil {
			return false
		}

		return true
	})
	return err
}
