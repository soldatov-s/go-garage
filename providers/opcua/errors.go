package opcua

import (
	"errors"
)

var (
	ErrEmptyOPCUA              = errors.New("empty opcua")
	ErrEmptyConnectionName     = errors.New("empty connection name is not allowed")
	ErrConnectionPointerIsNil  = errors.New("nil passed as connection pointer")
	ErrConnectionDoesNotExists = errors.New("connection does not exist")
	ErrOPCUAConnNotEstablished = errors.New("opcua connection not established")
)
