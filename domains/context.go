package domains

import (
	"context"
	"sync"

	gogarage "github.com/soldatov-s/go-garage"
)

const (
	DomainsItem gogarage.GarageItem = "domains"
)

type Domains struct {
	sync.Map
}

func Create(ctx context.Context) (context.Context, *Domains) {
	p := &Domains{}
	return context.WithValue(ctx, DomainsItem, p), p
}

func Get(ctx context.Context) *Domains {
	v := ctx.Value(DomainsItem)
	if v != nil {
		return v.(*Domains)
	}
	return nil
}

func GetByName(ctx context.Context, name string) interface{} {
	p := Get(ctx)
	if p != nil {
		v, _ := ctx.Value(DomainsItem).(*Domains).Load(name)
		return v
	}

	return nil
}

func RegistrateByName(ctx context.Context, name string, val interface{}) context.Context {
	if v := GetByName(ctx, name); v != nil {
		return ctx
	}

	if v := Get(ctx); v != nil {
		v.Store(name, val)
		return ctx
	}

	c, v := Create(ctx)
	v.Store(name, val)
	return c
}
