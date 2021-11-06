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
