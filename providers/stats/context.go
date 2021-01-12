package stats

import (
	"context"

	"github.com/soldatov-s/go-garage/providers"
)

func Registrate(ctx context.Context) context.Context {
	return providers.RegistrateByName(ctx, ProvidersName, NewStatistics(ctx))
}

func Get(ctx context.Context) *Statistics {
	v := providers.GetByName(ctx, ProvidersName)
	if v != nil {
		return providers.GetByName(ctx, ProvidersName).(*Statistics)
	}
	return nil
}

func RegistrateProvider(ctx context.Context, name string, p Provider) (context.Context, error) {
	ctx = Registrate(ctx)
	v := Get(ctx)
	if v == nil {
		return nil, ErrEmptyStatistics
	}

	if err := v.RegisterProvider(name, p); err != nil {
		return nil, err
	}

	return ctx, nil
}

func GetProvider(ctx context.Context, name string) (Provider, error) {
	v := Get(ctx)
	if v != nil {
		return v.GetProvider(name)
	}

	return nil, nil
}
