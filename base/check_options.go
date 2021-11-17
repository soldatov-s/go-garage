package base

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

type CheckOptions struct {
	// Check name
	Name string
	// Checking function should accept no parameters and return
	// boolean value which will be interpreted as dependency service readiness
	// and a string which should provide error text if dependency service
	// isn't ready and something else if dependency service is ready (for
	// example, dependency service's version).
	CheckFunc func(ctx context.Context) error
}

type MapCheckOptions struct {
	mu      sync.RWMutex
	options map[string]*CheckOptions
}

func NewMapCheckOptions() *MapCheckOptions {
	return &MapCheckOptions{
		options: make(map[string]*CheckOptions),
	}
}

func (mcf *MapCheckOptions) Append(src *MapCheckOptions) error {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()

	for k, m := range src.options {
		if _, ok := mcf.options[k]; ok {
			return errors.Wrapf(ErrConflictName, "name: %s", k)
		}

		mcf.options[k] = m
	}

	return nil
}

func (mcf *MapCheckOptions) Add(options *CheckOptions) error {
	mcf.mu.Lock()
	defer mcf.mu.Unlock()

	if options == nil {
		return ErrOptionsIsNil
	}

	if options.Name == "" {
		return ErrEmptyOptionsName
	}

	if options.CheckFunc == nil {
		return ErrFuncIsNil
	}

	if _, ok := mcf.options[options.Name]; ok {
		return errors.Wrapf(ErrConflictName, "name: %s", options.Name)
	}

	mcf.options[options.Name] = options

	return nil
}
