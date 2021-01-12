package errors

import (
	"errors"
	"path/filepath"
	"reflect"
)

var (
	ErrEmptyProviderName     = errors.New("empty provider name isn't allowed")
	ErrProviderNotRegistered = errors.New("provider wasn't registered")
	ErrLoggerPointerIsNil    = errors.New("pointer to logger is nil")
	ErrEmptyConnectionName   = errors.New("empty connection name isn't allowed")
	ErrBadTypeOfProvider     = errors.New("bad type of provider")

	ErrEmptyEnityName     = errors.New("empty enity name is not allowed")
	ErrEnityDoesNotExists = errors.New("enity does not exist")
)

// ObjName return object name for passing it to errors messages
func ObjName(iface interface{}) string {
	return filepath.Base(reflect.TypeOf(iface).PkgPath()) +
		"." + reflect.TypeOf(iface).Name()
}

func ErrInvalidEnityOptions(iface interface{}) error {
	return errors.New("passed configuration is not *" +
		ObjName(iface))
}

func ErrInvalidEnityPointer(iface interface{}) error {
	return errors.New("passed pointer to server is not **" +
		ObjName(iface))
}

func ErrProviderAlreadyRegistered(name string) error {
	return errors.New("provider " + name + "already registered")
}
