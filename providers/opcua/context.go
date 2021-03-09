package opcua

import (
	"context"

	"github.com/soldatov-s/go-garage/providers"
)

func Registrate(ctx context.Context) context.Context {
	return providers.RegistrateByName(ctx, ProvidersName, NewOpcua(ctx))
}

func Get(ctx context.Context) *Opcua {
	v := providers.GetByName(ctx, ProvidersName)
	if v != nil {
		return providers.GetByName(ctx, ProvidersName).(*Opcua)
	}
	return nil
}

func RegistrateProvider(ctx context.Context, name string, p Provider) (context.Context, error) {
	ctx = Registrate(ctx)
	v := Get(ctx)
	if v == nil {
		return nil, ErrEmptyOPCUA
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

func RegistrateEnity(ctx context.Context, provider, enityName string, options interface{}) (context.Context, error) {
	v, err := GetProvider(ctx, provider)
	if err != nil {
		return nil, err
	}

	if err := v.CreateEnity(enityName, options); err != nil {
		return nil, err
	}

	return ctx, nil
}

func GetEnity(ctx context.Context, provider, enityName string) (interface{}, error) {
	v, err := GetProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	return v.GetEnity(enityName)
}
