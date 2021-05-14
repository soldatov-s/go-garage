package db

import (
	"errors"

	"github.com/soldatov-s/go-garage/x/helper"
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
	return errors.New("passed configuration is not *" + helper.ObjName(iface))
}

func ErrNotEnityPointer(iface interface{}) error {
	return errors.New("passed enity isn't *" + helper.ObjName(iface))
}

func ErrNotMutexPointer(iface interface{}) error {
	return errors.New("passed mutex isn't **" + helper.ObjName(iface))
}

func ErrNotMigrationPointer(iface interface{}) error {
	return errors.New("passed pointer is not *" + helper.ObjName(iface))
}

func ErrNotQueueItemPointer(iface interface{}) error {
	return errors.New("passed item isn't *" + helper.ObjName(iface))
}

func ErrNotBulker(iface interface{}) error {
	return errors.New("passed provider " + helper.ObjName(iface) + "is not bulker")
}

func ErrNotMigrator(iface interface{}) error {
	return errors.New("passed provider " + helper.ObjName(iface) + "is not migrator")
}
