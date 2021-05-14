package db

import (
	"errors"
)

var (
	ErrEmptyDatabases          = errors.New("empty databases")
	ErrEmptyConnectionName     = errors.New("empty connection name is not allowed")
	ErrConnectionPointerIsNil  = errors.New("nil passed as connection pointer")
	ErrConnectionDoesNotExists = errors.New("connection does not exist")
	ErrMutexPointerIsNil       = errors.New("nil passed as mutex pointer")
	ErrNotLockIDPointer        = errors.New("passed lockID is not int64")
	ErrDBConnNotEstablished    = errors.New("database connection not established")
	ErrNotBulker               = errors.New("passed provider is not bulker")
	ErrNotMigrator             = errors.New("passed provider is not migrator")
)
