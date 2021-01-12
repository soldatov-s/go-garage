package config

import (
	"context"

	"github.com/soldatov-s/go-garage/providers"
)

func Registrate(ctx context.Context, cfg interface{}) context.Context {
	return providers.RegistrateByName(ctx, ProvidersName, NewConfiguration(cfg))
}

func Get(ctx context.Context) *Configuration {
	v := providers.GetByName(ctx, ProvidersName)
	if v != nil {
		return providers.GetByName(ctx, ProvidersName).(*Configuration)
	}
	return nil
}

func Parse(ctx context.Context) error {
	c := Get(ctx)
	return c.Parse()
}
