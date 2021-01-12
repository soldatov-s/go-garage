package meta

import (
	"context"

	gogarage "github.com/soldatov-s/go-garage"
)

const (
	AppItem gogarage.GarageItem = "app"
)

func CreateApp(ctx context.Context) (context.Context, *ApplicationInfo) {
	v := GetApp(ctx)
	if v != nil {
		return ctx, v
	}

	a := &ApplicationInfo{
		Name:        "unknown",
		Version:     "0.0.0",
		Description: "no description",
	}
	return context.WithValue(ctx, AppItem, a), a
}

func GetApp(ctx context.Context) *ApplicationInfo {
	v := ctx.Value(AppItem)
	if v != nil {
		return v.(*ApplicationInfo)
	}
	return nil
}

func FillAppInfo(ctx context.Context, name, builded, hash, version, description string) context.Context {
	a := GetApp(ctx)
	if a == nil {
		ctx, a = CreateApp(ctx)
	}
	a.Name = name
	a.Builded = builded
	a.Hash = hash
	a.Version = version
	a.Description = description

	return ctx
}
