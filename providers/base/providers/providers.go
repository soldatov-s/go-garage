package providers

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/base/provider"
	"github.com/soldatov-s/go-garage/providers/errors"
	"github.com/soldatov-s/go-garage/providers/logger"
)

type IBaseProviders interface {
	RegisterProvider(providerName string, iface interface{}) error
	Logger() *zerolog.Logger
	GetProviders() *sync.Map
	Start() error
	Shutdown() error
}

type BaseProviders struct {
	IBaseProviders
	logger    *zerolog.Logger
	providers sync.Map
	ctx       context.Context
}

// RegisterProvider registers provider. If provider with passed
// name already registered - application will exit.
func (bp *BaseProviders) RegisterProvider(providerName string, iface interface{}) error {
	// We should not attempt to register providers with empty names.
	if providerName == "" {
		return errors.ErrEmptyProviderName
	}

	if _, found := bp.providers.LoadOrStore(providerName, iface); found {
		return errors.ErrProviderAlreadyRegistered(providerName)
	}

	return nil
}

// Initialize initializes  controlling structure.
func NewBaseProviders(ctx context.Context, name string) *BaseProviders {
	bp := &BaseProviders{
		ctx: logger.Registrate(ctx),
	}
	l := logger.Get(bp.ctx)
	bp.logger = l.GetLogger(name, nil)
	bp.logger.Info().Msg("initializing " + name + " control...")

	return bp
}

func (bp *BaseProviders) Logger() *zerolog.Logger {
	return bp.logger
}

func (bp *BaseProviders) GetProviders() *sync.Map {
	return &bp.providers
}

func (bp *BaseProviders) Start() error {
	var err error
	bp.providers.Range(func(k, v interface{}) bool {
		err = v.(provider.IProvider).Start()
		return err == nil
	})

	return err
}

// Shutdown shutdowns all connections.
func (bp *BaseProviders) Shutdown() error {
	var err error
	bp.providers.Range(func(k, v interface{}) bool {
		bp.logger.Info().Str("provider", k.(string)).Msg("shutting down provider and it's connections")
		err = v.(provider.IProvider).Shutdown()
		return err == nil
	})

	return err
}

// Start sends start signal to selected enity.
func (bp *BaseProviders) StartEnity(providerName, serverName string) error {
	prov, err := bp.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.(provider.IEnityManager).StartEnity(serverName)
}

// Shutdown sends Shutdown signal to selected enity.
func (bp *BaseProviders) ShutdownEnity(providerName, serverName string) error {
	prov, err := bp.GetProvider(providerName)
	if err != nil {
		return err
	}

	return prov.(provider.IEnityManager).ShutdownEnity(serverName)
}

// GetProvider returns requested database provider.
// Return error if providers wasn't registered.
func (bp *BaseProviders) GetProvider(providerName string) (interface{}, error) {
	v, found := bp.providers.Load(providerName)
	if !found {
		return nil, errors.ErrProviderNotRegistered
	}

	return v, nil
}

// Checks for provider name and connection name for not breaking our
// rules.
func (bp *BaseProviders) CheckProviderAndEnityNames(providerName, connectionName string) error {
	// We should not allow calls with empty provider name.
	if providerName == "" {
		return errors.ErrEmptyProviderName
	}

	// We should not allow calls with empty connection name.
	if connectionName == "" {
		return errors.ErrEmptyConnectionName
	}

	return nil
}

// CreateEnity creates new connection for specified provider. Provider
// and connection names should not be empty, otherwise error will be returned.
func (bp *BaseProviders) CreateEnity(providerName, enityName string, options interface{}) error {
	// Check if provider and connection names isn't breaking rules.
	err := bp.CheckProviderAndEnityNames(providerName, enityName)
	if err != nil {
		return err
	}

	// Get provider.
	prov, err1 := bp.GetProvider(providerName)
	if err1 != nil {
		return err1
	}

	return prov.(provider.IEnityManager).CreateEnity(enityName, options)
}

// GetEnity obtains pointer to connection from specified provider.
// Provider and connection names should not be empty, otherwise error will
// be returned.
func (bp *BaseProviders) GetEnity(providerName, enityName string) (interface{}, error) {
	// Check if provider and connection names isn't breaking rules.
	err := bp.CheckProviderAndEnityNames(providerName, enityName)
	if err != nil {
		return nil, err
	}

	// Get provider.
	prov, err := bp.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	return prov.(provider.IEnityManager).GetEnity(enityName)
}
