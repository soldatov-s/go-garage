package envconfig

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/config"
)

func Registrate(ctx context.Context) (context.Context, error) {
	d := config.Get(ctx)
	return ctx, d.RegisterProvider(DefaultProviderName, NewProvider())
}

func RegistrateAndParse(ctx context.Context, cfg interface{}) (context.Context, error) {
	ctx = config.Registrate(ctx, cfg)

	d := config.Get(ctx)
	if err := d.RegisterProvider(DefaultProviderName, NewProvider()); err != nil {
		return nil, err
	}

	if err := config.Parse(ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}
