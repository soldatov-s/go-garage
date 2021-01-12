package logger

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/meta"
	"github.com/soldatov-s/go-garage/providers"
)

func Registrate(ctx context.Context) context.Context {
	v := Get(ctx)
	if v != nil {
		return ctx
	}
	return providers.RegistrateByName(ctx, ProvidersName, NewLogger())
}

func RegistrateAndInitilize(ctx context.Context, cfg *Config) context.Context {
	ctx = Registrate(ctx)
	Get(ctx).Initialize(cfg)

	return ctx
}

func Get(ctx context.Context) *Logger {
	v := providers.GetByName(ctx, ProvidersName)
	if v != nil {
		return providers.GetByName(ctx, ProvidersName).(*Logger)
	}
	return nil
}

// GetPackageLogger return logger for package
func GetPackageLogger(ctx context.Context, emptyStruct interface{}) zerolog.Logger {
	log := Get(ctx)
	if log == nil {
		Registrate(ctx)
	}

	a := meta.Get(ctx)
	l := log.GetLogger(a.Name, nil)
	return Initialize(l, emptyStruct)
}
