package db

import (
	"errors"

	goworkerr "github.com/soldatov-s/go-garage/providers/errors"
)

var (
	ErrEmptyDatabases          = errors.New("empty databases")
	ErrEmptyConnectionName     = errors.New("empty connection name is not allowed")
	ErrConnectionPointerIsNil  = errors.New("nil passed as connection pointer")
	ErrConnectionDoesNotExists = errors.New("connection does not exist")
	ErrMutexPointerIsNil       = errors.New("nil passed as mutex pointer")
	ErrNotLockIDPointer        = errors.New("passed lockID is not int64")
	ErrDBConnNotEstablished    = errors.New("database connection not established")
)

func ErrNotConfigPointer(iface interface{}) error {
	return errors.New("passed configuration is not *" + goworkerr.ObjName(iface))
}

func ErrNotEnityPointer(iface interface{}) error {
	return errors.New("passed enity isn't *" + goworkerr.ObjName(iface))
}

func ErrNotMutexPointer(iface interface{}) error {
	return errors.New("passed mutex isn't **" + goworkerr.ObjName(iface))
}

func ErrNotMigrationPointer(iface interface{}) error {
	return errors.New("passed pointer is not *" + goworkerr.ObjName(iface))
}

func ErrNotQueueItemPointer(iface interface{}) error {
	return errors.New("passed item isn't *" + goworkerr.ObjName(iface))
}

func ErrNotBulker(iface interface{}) error {
	return errors.New("passed provider " + goworkerr.ObjName(iface) + "is not bulker")
}

func ErrNotMigrator(iface interface{}) error {
	return errors.New("passed provider " + goworkerr.ObjName(iface) + "is not migrator")
}
