package meta

import (
	"context"

	"github.com/pkg/errors"
	gogarage "github.com/soldatov-s/go-garage"
)

const (
	AppItem gogarage.GarageItem = "app"
)

var (
	ErrNotFoundInContext = errors.New("not found in context")
	ErrFailedTypeCast    = errors.New("failed type cast")
	ErrDuplicateMeta     = errors.New("duplicacte meta in context")
)

func NewContext(ctx context.Context) (context.Context, error) {
	v, _ := FromContext(ctx)
	if v != nil {
		return nil, ErrDuplicateMeta
	}
	return context.WithValue(ctx, AppItem, DefaultAppInfo()), nil
}

func FromContext(ctx context.Context) (*AppInfo, error) {
	v := ctx.Value(AppItem)
	if v == nil {
		return nil, ErrNotFoundInContext
	}
	appInfo, ok := v.(*AppInfo)
	if !ok {
		return nil, ErrFailedTypeCast
	}
	return appInfo, nil
}

func NewContextByParams(ctx context.Context, cfg *Config) (context.Context, error) {
	v, _ := FromContext(ctx)
	if v != nil {
		return nil, ErrDuplicateMeta
	}
	return context.WithValue(ctx, AppItem, NewAppInfo(cfg)), nil
}
