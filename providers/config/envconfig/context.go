package envconfig

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/config"
)

func Registrate(ctx context.Context) (context.Context, error) {
	d, err := config.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return ctx, d.RegisterProvider(DefaultProviderName, NewProvider())
}
