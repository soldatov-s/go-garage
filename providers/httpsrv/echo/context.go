package echo

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/x/helper"
)

func AddProvider(ctx context.Context) (context.Context, error) {
	ctx, err := db.NewContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "new context")
	}

	c, err := db.FromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get collector from context")
	}

	p, err := NewProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "new provider")
	}

	if err := c.AddProvider(ProviderName, p); err != nil {
		return nil, errors.Wrap(err, "add provider")
	}

	return ctx, nil
}

func GetProvider(ctx context.Context) (db.ProviderGateway, error) {
	c, err := db.FromContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get collector from context")
	}

	p, err := c.GetProvider(ProviderName)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	return p, nil
}

func AddEnity(ctx context.Context, enityName string, options interface{}) (context.Context, error) {
	p, err := GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	em, ok := p.(base.EntityGateway)
	if !ok {
		return nil, errors.Wrap(base.ErrBadTypeCast, "expected EnityGateway")
	}

	err = em.CreateEnity(ctx, enityName, options)
	if err != nil {
		return nil, errors.Wrap(err, "create enity")
	}

	return ctx, nil
}

func RegistrateEnity(ctx context.Context, enityName string, options interface{}) (context.Context, error) {
	var err error
	ctx, err = db.NewContext(ctx)
	if err != nil {
		switch {
		case errors.Is(err, base.ErrDuplicateCollector):
			// skip
		default:
			return nil, errors.Wrap(err, "add provider")
		}
	}

	ctx, err = AddProvider(ctx)
	if err != nil {
		switch {
		case errors.Is(err, base.ErrProviderAlreadyRegistered):
			// skip
		default:
			return nil, errors.Wrap(err, "add provider")
		}
	}

	ctx, err = AddEnity(ctx, enityName, options)
	if err != nil {
		return nil, errors.Wrap(err, "add enity")
	}

	return ctx, nil
}

func GetEnity(ctx context.Context, enityName string) (interface{}, error) {
	p, err := GetProvider(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get provider")
	}

	em, ok := p.(base.EntityGateway)
	if !ok {
		return nil, errors.Wrap(base.ErrBadTypeCast, "expected EnityGateway")
	}

	e, err := em.GetEnity(enityName)
	if err != nil {
		return nil, errors.Wrap(err, "get enity")
	}

	return e, nil
}

func GetEnityTypeCast(ctx context.Context, enityName string) (*Enity, error) {
	e, err := GetEnity(ctx, enityName)
	if err != nil {
		return nil, err
	}

	// Checking that enity type is *Enity
	enityPointer, ok := e.(*Enity)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityPointer, "expected %q", helper.ObjName(Enity{}))
	}

	return enityPointer, nil
}

func GetAPIVersionGroup(ctx context.Context, enityName, apiVersion string) (*Group, error) {
	enity, err := GetEnityTypeCast(ctx, enityName)
	if err != nil {
		return nil, err
	}

	return enity.GetAPIVersionGroup(apiVersion)
}
