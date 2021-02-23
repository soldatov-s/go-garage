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
	d := &Domains{}
	return context.WithValue(ctx, DomainsItem, d), d
}

func Get(ctx context.Context) *Domains {
	if v := ctx.Value(DomainsItem); v != nil {
		return v.(*Domains)
	}
	return nil
}

func GetByName(ctx context.Context, name string) interface{} {
	if d := Get(ctx); d != nil {
		v, _ := d.Load(name)
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
