package httpsrv

import (
	"errors"
)

var (
	ErrEmptyHTTPServers     = errors.New("empty http(s) servers")
	ErrEmptyServerName      = errors.New("empty server name is not allowed")
	ErrServerNotFound       = errors.New("requested server wasn't created yet")
	ErrServerNameMissing    = errors.New("server's name is empty")
	ErrGroupNotFound        = errors.New("requested API version group wasn't found")
	ErrBindAddressMissing   = errors.New("bind address wasn't specified")
	ErrServerAlreadyStarted = errors.New("server already started")
	ErrEmptyHTTPHandler     = errors.New("empty http handler")
	ErrUnknownHTTPMethod    = errors.New("unknown http method")
	ErrServerNotUp          = errors.New("http server isn't up after 10 seconds")
	ErrInvalidGroupPointer  = errors.New("passed pointer to group is not valid")
)
