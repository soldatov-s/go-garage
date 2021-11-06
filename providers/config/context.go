package config

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
)

func NewContext(ctx context.Context, cfg interface{}) (context.Context, error) {
	return base.NewContextByName(ctx, CollectorName, NewCollector(cfg))
}

func FromContext(ctx context.Context) (*Collector, error) {
	v, err := base.FromContextByName(ctx, CollectorName)
	if err != nil {
		return nil, errors.Wrap(err, "get from context by name")
	}
	c, ok := v.(*Collector)
	if !ok {
		return nil, base.ErrFailedTypeCast
	}
	return c, nil
}

func Parse(ctx context.Context) error {
	c, err := FromContext(ctx)
	if err != nil {
		return err
	}

	return c.Parse()
}
