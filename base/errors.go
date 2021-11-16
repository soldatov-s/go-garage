package base

import "github.com/pkg/errors"

var (
	ErrInvalidEnityOptions = errors.New("invalid passed enity options pointer")
	ErrNotConnected        = errors.New("not connected")
	ErrConflictName        = errors.New("conflict name")
)
