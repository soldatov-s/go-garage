package msgs

import (
	"errors"
)

var (
	ErrEmptyMsgs                      = errors.New("empty msgs")
	ErrConnectionPointerIsNil         = errors.New("nil passed as connection pointer")
	ErrConnectionDoesNotExists        = errors.New("connection does not exist")
	ErrMutexPointerIsNil              = errors.New("nil passed as mutex pointer")
	ErrNotLockIDPointer               = errors.New("passed lockID is not int64")
	ErrDBConnNotEstablished           = errors.New("database connection not established")
	ErrInvalidSubscribeOptionsPointer = errors.New("passed subscribe options is not SubscribeOptions interface")
)
