package garage

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/stats"
	"github.com/soldatov-s/go-garage/x/helper"
)

type contextHelper struct {
	collectorName string
	providerName  string
}

var _contextHelper = &contextHelper{
	collectorName: stats.CollectorName,
	providerName:  ProviderName,
}

func (ch *contextHelper) NewProvider(ctx context.Context) (interface{}, error) {
	provider, err := NewProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "new provider")
	}

	return provider, nil
}

func (ch *contextHelper) GetCollectorName() string {
	return ch.collectorName
}

func (ch *contextHelper) GetProviderName() string {
	return ch.providerName
}

func AddProvider(ctx context.Context) (context.Context, error) {
	ctx, err := base.AddProvider(ctx, _contextHelper)
	if err != nil {
		return nil, errors.Wrap(err, "base add provider")
	}

	return ctx, nil
}

func GetProvider(ctx context.Context) (interface{}, error) {
	provider, err := base.GetProvider(ctx, _contextHelper)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	return provider, nil
}

func AddEnity(ctx context.Context, enityName string, options interface{}) (context.Context, error) {
	ctx, err := base.AddEnity(ctx, _contextHelper, enityName, options)
	if err != nil {
		return nil, errors.Wrap(err, "base add enity")
	}

	return ctx, nil
}

func GetEnity(ctx context.Context, enityName string) (interface{}, error) {
	e, err := base.GetEnity(ctx, _contextHelper, enityName)
	if err != nil {
		return nil, errors.Wrap(err, "base get enity")
	}

	return e, nil
}

func GetEnityTypeCast(ctx context.Context, enityName string) (*Enity, error) {
	e, err := GetEnity(ctx, enityName)
	if err != nil {
		return nil, errors.Wrap(err, "get enity")
	}

	// Checking that enity type is *Enity
	enityPointer, ok := e.(*Enity)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityPointer, "expected %q", helper.ObjName(Enity{}))
	}

	return enityPointer, nil
}
