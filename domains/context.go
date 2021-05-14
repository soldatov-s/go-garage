package domains

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	gogarage "github.com/soldatov-s/go-garage"
)

const (
	DomainsItem gogarage.GarageItem = "domains"
)

var (
	ErrNotFoundDomain    = errors.New("not found domain in context")
	ErrDuplicateDomain   = errors.New("duplicacte domain in context")
	ErrDuplicateDomains  = errors.New("duplicacte domains in context")
	ErrNotFoundInContext = errors.New("not found in context")
	ErrFailedTypeCast    = errors.New("failed type cast")
)

type Domains struct {
	sync.Map
}

func NewContext(ctx context.Context) (context.Context, error) {
	v, _ := FromContext(ctx)
	if v != nil {
		return nil, ErrDuplicateDomains
	}
	return context.WithValue(ctx, DomainsItem, &Domains{}), nil
}

func FromContext(ctx context.Context) (*Domains, error) {
	v := ctx.Value(DomainsItem)
	if v == nil {
		return nil, ErrNotFoundInContext
	}
	domains, ok := v.(*Domains)
	if !ok {
		return nil, ErrFailedTypeCast
	}
	return domains, nil
}

func FromContextByName(ctx context.Context, name string) (interface{}, error) {
	p, err := FromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "not found domains in context")
	}

	v, ok := p.Load(name)
	if !ok {
		return nil, ErrNotFoundDomain
	}

	return v, nil
}

func NewContextByName(ctx context.Context, name string, val interface{}) (context.Context, error) {
	var c context.Context
	_, err := FromContextByName(ctx, name)
	if err != nil {
		return nil, errors.Wrapf(ErrDuplicateDomain, "domain %q", name)
	}

	switch err {
	case ErrNotFoundDomain:
		p, _ := FromContext(ctx)
		p.Store(name, val)
	case ErrNotFoundInContext:
		c, _ = NewContext(ctx)
		p, _ := FromContext(c)
		p.Store(name, val)
	}

	return c, nil
}
