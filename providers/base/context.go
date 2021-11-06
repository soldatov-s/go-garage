package base

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	gogarage "github.com/soldatov-s/go-garage"
)

const (
	ProvidersItem gogarage.GarageItem = "providers"
)

var (
	ErrDuplicateCollectors = errors.New("duplicacte collectors in context")
	ErrDuplicateCollector  = errors.New("duplicacte collector")
	ErrNotFoundInContext   = errors.New("not found in context")
	ErrFailedTypeCast      = errors.New("failed type cast")
	ErrNotFoundCollectors  = errors.New("not found providers in context")
	ErrNotFoundCollector   = errors.New("not found provider")
)

type Collectors struct {
	sync.Map
}

func NewContext(ctx context.Context) (context.Context, error) {
	v, _ := FromContext(ctx)
	if v != nil {
		return nil, ErrDuplicateCollectors
	}

	return context.WithValue(ctx, ProvidersItem, &Collectors{}), nil
}

func NewContextByNameWithMetrics(ctx context.Context, name string) (context.Context, error) {
	c, err := NewCollectorWithMetrics(ctx, name)
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateCollector):
			// skip
		default:
			return nil, errors.Wrapf(err, "%q: new collector with metrics", name)
		}
	}

	return NewContextByName(ctx, name, c)
}

func FromContext(ctx context.Context) (*Collectors, error) {
	v := ctx.Value(ProvidersItem)
	if v == nil {
		return nil, ErrNotFoundInContext
	}
	providers, ok := v.(*Collectors)
	if !ok {
		return nil, ErrFailedTypeCast
	}
	return providers, nil
}

func FromContextByName(ctx context.Context, name string) (interface{}, error) {
	p, err := FromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "not found providers in context")
	}

	v, ok := p.Load(name)
	if !ok {
		return nil, ErrNotFoundCollector
	}

	return v, nil
}

func FromContextByNameWithMetrics(ctx context.Context, name string) (*CollectorWithMetrics, error) {
	v, err := FromContextByName(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(err, "%q: get from context by name", name)
	}
	c, ok := v.(*CollectorWithMetrics)
	if !ok {
		return nil, ErrFailedTypeCast
	}
	return c, nil
}

func NewContextByName(ctx context.Context, name string, val interface{}) (context.Context, error) {
	var c context.Context
	_, err := FromContextByName(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(ErrDuplicateCollector, "provider %q", name)
	}

	switch err {
	case ErrNotFoundCollector:
		p, _ := FromContext(ctx)
		p.Store(name, val)
	case ErrNotFoundInContext:
		c, _ = NewContext(ctx)
		p, _ := FromContext(c)
		p.Store(name, val)
	}

	return c, nil
}

type ContextHelperGateway interface {
	NewProvider(ctx context.Context) (interface{}, error)
	GetCollectorName() string
	GetProviderName() string
}

func AddProvider(ctx context.Context, helper ContextHelperGateway) (context.Context, error) {
	ctx, err := NewContextByNameWithMetrics(ctx, helper.GetCollectorName())
	if err != nil {
		return nil, errors.Wrap(err, "add collector")
	}

	collector, err := FromContextByNameWithMetrics(ctx, helper.GetCollectorName())
	if err != nil {
		return nil, errors.Wrap(err, "get collector from context")
	}

	provider, err := helper.NewProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "new provider")
	}

	if err := collector.AddProvider(helper.GetProviderName(), provider); err != nil {
		return nil, errors.Wrap(err, "add provider to collector")
	}

	return ctx, nil
}

func GetProvider(ctx context.Context, helper ContextHelperGateway) (interface{}, error) {
	collector, err := FromContextByNameWithMetrics(ctx, helper.GetCollectorName())
	if err != nil {
		return nil, errors.Wrap(err, "get collector from context")
	}

	provider, err := collector.GetProvider(helper.GetProviderName())
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	return provider, nil
}

func AddEnity(ctx context.Context, helper ContextHelperGateway, enityName string, options interface{}) (context.Context, error) {
	var err error
	ctx, err = NewContextByNameWithMetrics(ctx, helper.GetCollectorName())
	if err != nil {
		return nil, errors.Wrap(err, "add collector")
	}

	ctx, err = AddProvider(ctx, helper)
	if err != nil {
		return nil, errors.Wrap(err, "add provider")
	}

	p, err := GetProvider(ctx, helper)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	em, ok := p.(EntityGateway)
	if !ok {
		return nil, errors.Wrap(ErrBadTypeCast, "expected EnityGateway")
	}

	err = em.CreateEnity(ctx, enityName, options)
	if err != nil {
		return nil, errors.Wrap(err, "create enity")
	}

	return ctx, nil
}

func GetEnity(ctx context.Context, helper ContextHelperGateway, enityName string) (interface{}, error) {
	p, err := GetProvider(ctx, helper)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	em, ok := p.(EntityGateway)
	if !ok {
		return nil, errors.Wrap(ErrBadTypeCast, "expected EnityGateway")
	}

	e, err := em.GetEnity(enityName)
	if err != nil {
		return nil, errors.Wrap(err, "get enity")
	}

	return e, nil
}
