package providers

import (
	"context"
	"sync"

	gogarage "github.com/soldatov-s/go-garage"
)

const (
	ProvidersItem gogarage.GarageItem = "providers"
)

type Providers struct {
	sync.Map
}

func Create(ctx context.Context) (context.Context, *Providers) {
	p := &Providers{}
	return context.WithValue(ctx, ProvidersItem, p), p
}

func Get(ctx context.Context) *Providers {
	v := ctx.Value(ProvidersItem)
	if v != nil {
		return v.(*Providers)
	}
	return nil
}

func GetByName(ctx context.Context, name string) interface{} {
	p := Get(ctx)
	if p != nil {
		v, _ := ctx.Value(ProvidersItem).(*Providers).Load(name)
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
