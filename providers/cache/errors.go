package cache

import (
	"errors"

	goworkerr "github.com/soldatov-s/go-garage/providers/errors"
)

var (
	ErrEmptyCachees            = errors.New("empty caches")
	ErrConnectionPointerIsNil  = errors.New("nil passed as connection pointer")
	ErrConnectionDoesNotExists = errors.New("connection does not exist")
	ErrMutexPointerIsNil       = errors.New("nil passed as mutex pointer")
	ErrNotLockIDPointer        = errors.New("passed lockID is not int64")
	ErrDBConnNotEstablished    = errors.New("cache connection not established")
	ErrNotFoundInCache         = errors.New("not found in cache")
	ErrNotLockKey              = errors.New("passed lockID is not string")
)

func ErrInvalidConnectionOptionsPointer(iface interface{}) error {
	return errors.New("passed configuration is not *" +
		goworkerr.ObjName(iface))
}

func ErrNotConnectionPointer(iface interface{}) error {
	return errors.New("passed conn isn't **" +
		goworkerr.ObjName(iface))
}

func ErrNotMutexPointer(iface interface{}) error {
	return errors.New("passed mutex isn't **" +
		goworkerr.ObjName(iface))
}

func ErrNotMigrationPointer(iface interface{}) error {
	return errors.New("passed pointer is not *" +
		goworkerr.ObjName(iface))
}

func ErrNotQueueItemPointer(iface interface{}) error {
	return errors.New("passed item isn't *" +
		goworkerr.ObjName(iface))
}
