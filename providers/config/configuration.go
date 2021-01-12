package config

import (
	"sync"

	wraperr "github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/errors"
)

const (
	ProvidersName = "configurations"
)

// Provider is an interface that every configuration adapter
// should conform to.
type Provider interface {
	// Parse parses configuration. Passed structure is Configuration.
	// Adapter should return any occurred error and this error will be
	// considered as critical.
	// Pointer to configuration struct defined below will be passed here.
	Parse(structure interface{}) error
}

// Configuration holds everything that related to configuration.
type Configuration struct {
	Service interface{}
	// Internal things.
	// All available configuration providers.
	providers sync.Map
	// ProvidersOrder is an ordering thing for configuration
	// providers. They'll be executed in exact order as defined here.
	// When parsing configuration latest defined wins.
	ProvidersOrder []string
}

func NewConfiguration(cfg interface{}) *Configuration {
	return &Configuration{
		Service: cfg,
	}
}

// Parse executes configuration parsing. Providers will be launched in order
// specified in configuration file. If provider defined in configuration but
// wasn't added - error will be printed and next provider will be used (if
// any). For errors be printed logging provider should already be registered.
// If no configuration providers was registered prior to this function call
// application will immediately exit.
func (cfg *Configuration) Parse() error {
	for _, providerName := range cfg.ProvidersOrder {
		provider, found := cfg.providers.Load(providerName)
		if !found {
			return errors.ErrProviderNotRegistered
		}
		err := provider.(Provider).Parse(cfg.Service)
		if err != nil {
			return wraperr.Wrap(err, "failed to parse config")
		}
	}

	return nil
}

// RegisterProvider registers configuration adapter.
func (cfg *Configuration) RegisterProvider(providerName string, iface Provider) error {
	// We should not attempt to register providers with empty names.
	if providerName == "" {
		return errors.ErrEmptyProviderName
	}

	if _, found := cfg.providers.LoadOrStore(providerName, iface); found {
		return errors.ErrProviderAlreadyRegistered
	}

	cfg.SetProvidersOrder(providerName)

	return nil
}

func (cfg *Configuration) GetProvider(providerName string) (Provider, error) {
	if v, found := cfg.providers.Load(providerName); !found {
		return nil, errors.ErrProviderNotRegistered
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, errors.ErrBadTypeOfProvider
}

// ServiceConfig interface for service configuration
type ServiceConfig interface {
	// FillGowork fill gowork config from values of service configuration.
	FillGowork(cfg *Configuration) error
}

// SetProvidersOrder sets configuration providers execution order. It will
// overwrite Configuration.Gowork.ProvidersOrder slice completely.
func (cfg *Configuration) SetProvidersOrder(order ...string) {
	if len(cfg.ProvidersOrder) > 0 {
		cfg.ProvidersOrder = append(cfg.ProvidersOrder, order...)
	} else {
		cfg.ProvidersOrder = order
	}
}
