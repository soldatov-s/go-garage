package config

import (
	"sync"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
)

const (
	CollectorName = "configurations"
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

// Collector holds everything that related to configuration.
type Collector struct {
	Service interface{}
	// Internal things.
	// All available configuration providers.
	providers sync.Map
	// ProvidersOrder is an ordering thing for configuration
	// providers. They'll be executed in exact order as defined here.
	// When parsing configuration latest defined wins.
	ProvidersOrder []string
}

func NewCollector(cfg interface{}) *Collector {
	return &Collector{
		Service: cfg,
	}
}

// Parse executes configuration parsing. Providers will be launched in order
// specified in configuration file. If provider defined in configuration but
// wasn't added - error will be printed and next provider will be used (if
// any). For errors be printed logging provider should already be registered.
// If no configuration providers was registered prior to this function call
// application will immediately exit.
func (cfg *Collector) Parse() error {
	for _, providerName := range cfg.ProvidersOrder {
		provider, found := cfg.providers.Load(providerName)
		if !found {
			return base.ErrProviderNotRegistered
		}
		err := provider.(Provider).Parse(cfg.Service)
		if err != nil {
			return errors.Wrap(err, "failed to parse config")
		}
	}

	return nil
}

// RegisterProvider registers configuration adapter.
func (cfg *Collector) RegisterProvider(providerName string, iface Provider) error {
	// We should not attempt to register providers with empty names.
	if providerName == "" {
		return base.ErrEmptyProviderName
	}

	if _, found := cfg.providers.LoadOrStore(providerName, iface); found {
		return errors.Wrapf(base.ErrProviderAlreadyRegistered, "providerName %q", providerName)
	}

	cfg.SetProvidersOrder(providerName)

	return nil
}

func (cfg *Collector) GetProvider(providerName string) (Provider, error) {
	if v, found := cfg.providers.Load(providerName); !found {
		return nil, base.ErrProviderNotRegistered
	} else if prov, ok := v.(Provider); ok {
		return prov, nil
	}

	return nil, base.ErrBadTypeOfProvider
}

// ServiceConfig interface for service configuration
type ServiceConfig interface {
	// FillGowork fill gowork config from values of service configuration.
	FillGowork(cfg *Collector) error
}

// SetProvidersOrder sets configuration providers execution order. It will
// overwrite Configuration.Gowork.ProvidersOrder slice completely.
func (cfg *Collector) SetProvidersOrder(order ...string) {
	if len(cfg.ProvidersOrder) > 0 {
		cfg.ProvidersOrder = append(cfg.ProvidersOrder, order...)
	} else {
		cfg.ProvidersOrder = order
	}
}
