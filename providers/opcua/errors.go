package opcua

import (
	"errors"

	goworkerr "github.com/soldatov-s/go-garage/providers/errors"
)

var (
	ErrEmptyOPCUA              = errors.New("empty opcua")
	ErrEmptyConnectionName     = errors.New("empty connection name is not allowed")
	ErrConnectionPointerIsNil  = errors.New("nil passed as connection pointer")
	ErrConnectionDoesNotExists = errors.New("connection does not exist")
	ErrOPCUAConnNotEstablished = errors.New("opcua connection not established")
)

func ErrNotConfigPointer(iface interface{}) error {
	return errors.New("passed configuration is not *" + goworkerr.ObjName(iface))
}

func ErrNotEnityPointer(iface interface{}) error {
	return errors.New("passed enity isn't *" + goworkerr.ObjName(iface))
}
