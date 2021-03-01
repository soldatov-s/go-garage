package dbgraph

import (
	"context"

	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/errors"
)

func Registrate(ctx context.Context) (context.Context, error) {
	ctx = db.Registrate(ctx)
	d := db.Get(ctx)
	if d == nil {
		return nil, db.ErrEmptyDatabases
	}

	if _, err := d.GetProvider(defaultProviderName); err == nil {
		return ctx, nil
	} else if err != errors.ErrProviderNotRegistered {
		return nil, err
	}
	return ctx, d.RegisterProvider(defaultProviderName, NewProvider(ctx))
}

func Get(ctx context.Context) (db.Provider, error) {
	d := db.Get(ctx)
	if d == nil {
		return nil, db.ErrEmptyDatabases
	}

	return d.GetProvider(defaultProviderName)
}

func RegistrateEnity(ctx context.Context, enityName string, options interface{}) (context.Context, error) {
	ctx, err := Registrate(ctx)
	if err != nil {
		return nil, err
	}

	p, err := Get(ctx)
	if err != nil {
		return nil, err
	}

	err = p.CreateEnity(enityName, options)
	if err != nil {
		return nil, err
	}

	return ctx, nil
}

func GetEnity(ctx context.Context, enityName string) (interface{}, error) {
	p, err := Get(ctx)
	if err != nil {
		return nil, err
	}

	return p.GetEnity(enityName)
}

func GetEnityTypeCast(ctx context.Context, enityName string) (*Enity, error) {
	enity, err := GetEnity(ctx, enityName)
	if err != nil {
		return nil, err
	}

	// Checking that enity type is *Enity
	enityPointer, ok := enity.(*Enity)
	if !ok {
		return nil, errors.ErrInvalidEnityPointer(Enity{})
	}

	return enityPointer, nil
}
