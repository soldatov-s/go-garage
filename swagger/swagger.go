package swagger

import (
	"errors"
	"sync"
)

var ErrNotYetRegistered = errors.New("not yet registered swag")

var (
	swaggerMu sync.RWMutex
	swag      map[string]Swagger
)

// Swagger is a interface to read swagger document.
type Swagger interface {
	ReadDoc() string
}

// Register registers swagger for given name.
func Register(name string, swagger Swagger) {
	if swagger == nil {
		panic("swagger is nil")
	}

	if swag == nil {
		swag = make(map[string]Swagger)
	}
	swaggerMu.Lock()
	defer swaggerMu.Unlock()

	if _, ok := swag[name]; ok {
		return
	}

	swag[name] = swagger
}

// ReadDoc reads swagger document.
func ReadDoc(name string) (string, error) {
	if swag != nil {
		swaggerMu.RLock()
		defer swaggerMu.RUnlock()
		return swag[name].ReadDoc(), nil
	}
	return "", ErrNotYetRegistered
}
