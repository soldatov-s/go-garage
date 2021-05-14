package log

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	gogarage "github.com/soldatov-s/go-garage"
	"github.com/soldatov-s/go-garage/meta"
)

const (
	LoggerItem gogarage.GarageItem = "logger"
)

var (
	ErrDuplicateLogger = errors.New("duplicacte logger in context")
	ErrFailedTypeCast  = errors.New("failed type cast")
	ErrNotFoundLogger  = errors.New("not found logger in context")
)

func NewContext(ctx context.Context) (context.Context, error) {
	v := FromContext(ctx)
	if v != nil {
		return nil, ErrDuplicateLogger
	}

	return context.WithValue(ctx, LoggerItem, NewLogger(DefaultConfig())), nil
}

func NewContextByConfig(ctx context.Context, cfg *Config) (context.Context, error) {
	v := FromContext(ctx)
	if v != nil {
		return nil, ErrDuplicateLogger
	}

	return context.WithValue(ctx, LoggerItem, NewLogger(cfg)), nil
}

func FromContext(ctx context.Context) *Logger {
	v := ctx.Value(LoggerItem)
	if v == nil {
		return NewLogger(DefaultConfig())
	}

	logger, ok := v.(*Logger)
	if !ok {
		return NewLogger(DefaultConfig())
	}

	return logger
}

// PackageLoggerFromContext return logger for package
func PackageLoggerFromContext(ctx context.Context, emptyStruct interface{}) (*zerolog.Logger, error) {
	log := FromContext(ctx)

	a, err := meta.FromContext(ctx)
	var l *zerolog.Logger
	switch {
	case errors.Is(err, meta.ErrNotFoundInContext):
		l = log.GetLogger("logger", nil)
	default:
		l = log.GetLogger(a.Name, nil)
	}

	return BuildPackageLogger(l, emptyStruct), nil
}
