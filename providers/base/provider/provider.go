package provider

import (
	"context"
	"sync"

	"github.com/rs/zerolog"
	"github.com/soldatov-s/go-garage/providers/logger"
)

type IProvider interface {
	// Start start all connections
	Start() error
	// Shutdown should shutdown all connections.
	Shutdown() error
}

type IEniter interface {
	// CreateEnity should create connection using passed parameters.
	// It is up to provider to cast passed options into required data
	// type and provide instructions for configuring.
	// Passed conn should be a pointer to needed structure that might be
	// provided by database provider. Provider should not attempt to
	// replace pointer if nil is passed. Provider should check if passed
	// conn is a pointer to valid structure and return error if it is
	// invalid AND continue with connection creating.
	CreateEnity(connectionName string, options interface{}) error
	// GetEnity should return pointer to connection structure to
	// caller. It is up to caller to cast returned interface{} into
	// required data type.
	// Passed conn should be a pointer to needed structure that might be
	// provided by database provider. Provider should not attempt to
	// replace pointer if nil is passed. Provider should check if passed
	// conn is a pointer to valid structure and return error if it is
	// invalid.
	GetEnity(connectionName string) (interface{}, error)
}

type IEnityManager interface {
	IEniter
	// StartEnity starts connection things.
	StartEnity(connectionName string) error
	// ShutdownEnity shutdown connection things.
	ShutdownEnity(connectionName string) error
}

type Entity interface {
	Shutdown() error
	Start() error
}

type MapEnity struct {
	sync.Map
}

// Provider provides abstract worker.
type Provider struct {
	Entitys MapEnity
	Log     *zerolog.Logger
	Logger  *logger.Logger
	ctx     context.Context
	Name    string
}

// NewProvider creates provider
func NewProvider(ctx context.Context, providersName, providerName string) *Provider {
	bp := &Provider{
		ctx:  logger.Registrate(ctx),
		Name: providerName,
	}
	bp.Log = logger.Get(bp.ctx).GetLogger(providersName, nil)
	bp.Log.Info().Msg("initializing " + providerName + " provider...")

	return bp
}

// Shutdown should shutdown all known entitys.
func (bp *Provider) Shutdown() error {
	var err error
	bp.Log.Info().Msg("shutdown provider" + bp.Name + "...")
	bp.Entitys.Range(func(k, v interface{}) bool {
		if err = v.(Entity).Shutdown(); err != nil {
			return false
		}

		return true
	})
	return err
}

// Start starts all known entitys.
func (bp *Provider) Start() error {
	var err error
	bp.Log.Info().Msg("start provider " + bp.Name + "...")
	bp.Entitys.Range(func(k, v interface{}) bool {
		if err = v.(Entity).Start(); err != nil {
			return false
		}

		return true
	})
	return err
}

// StartEnity starts connection things like watchers and queues.
func (bp *Provider) StartEnity(enityName string) error {
	v, ok := bp.Entitys.Load(enityName)
	if ok {
		return v.(Entity).Start()
	}

	return nil
}

// ShutdownEnity starts connection things like watchers and queues.
func (bp *Provider) ShutdownEnity(enityName string) error {
	v, ok := bp.Entitys.Load(enityName)
	if ok {
		return v.(Entity).Shutdown()
	}

	return nil
}

func (bp *Provider) GetContext() context.Context {
	return bp.ctx
}
