package base

import "github.com/pkg/errors"

var (
	ErrInvalidEnityOptions = errors.New("invalid passed enity options pointer")
	ErrNotConnected        = errors.New("not connected")
	ErrConflictName        = errors.New("conflict name")
	ErrEmptyOptionsName    = errors.New("empty options name")
	ErrOptionsIsNil        = errors.New("pointer to options is nil")
	ErrFuncIsNil           = errors.New("pointer to function is nil")
)
