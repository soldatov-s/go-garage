package ch

import (
	"context"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/providers/base"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/x/helper"
)

const ProviderName = "ch"

// Provider provides PostgreSQL database worker. This provider
// supports asynchronous database actions (like bulk inserting). Every
// connection will have own goroutine for queue processing.
type Provider struct {
	*base.ProviderWithMetrics
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) (*Provider, error) {
	p, err := base.NewProviderWithMetrics(ctx, db.CollectorName, ProviderName)
	if err != nil {
		return nil, errors.Wrap(err, "new provider with metrics")
	}
	return &Provider{p}, nil
}

// CreateEnity should create enity using passed parameters.
func (p *Provider) CreateEnity(ctx context.Context, enityName string, options interface{}) error {
	if _, err := p.GetEnity(enityName); err == nil {
		return base.ErrDuplicateEnity
	}

	enity, err := NewEnity(ctx, db.CollectorName, ProviderName, enityName, options)
	if err != nil {
		return errors.Wrap(err, "create enity")
	}

	p.Entitys.Store(enityName, enity)
	return nil
}

// getEnity should return pointer to enity structure to caller.
func (p *Provider) getEnity(enityName string) (*Enity, error) {
	enity, err := p.Provider.GetEnity(enityName)
	if err != nil {
		return nil, errors.Wrap(err, "get enity from base provider")
	}

	// Checking that enity type is *Enity
	enityPointer, ok := enity.(*Enity)
	if !ok {
		return nil, errors.Wrapf(base.ErrInvalidEnityPointer, "expect %q", helper.ObjName(Enity{}))
	}

	return enityPointer, nil
}

// GetEnity should return pointer to connection structure to caller.
func (p *Provider) GetEnity(enityName string) (interface{}, error) {
	return p.getEnity(enityName)
}

// AppendToQueue adds passed item into processing queue.
func (p *Provider) AppendToQueue(connectionName string, item interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	itemPointer, ok := item.(*QueueItem)
	if !ok {
		return errors.Wrapf(base.ErrInvalidPointer, "expect %q", helper.ObjName(QueueItem{}))
	}

	conn.AppendToQueue(itemPointer)

	return nil
}

// RegisterMigration registers migration for specified connection.
// It is up to provider to provide instructions about working with
// migrations and how to put them into migration interface. It is
// recommended to use separate structure.
func (p *Provider) RegisterMigration(connectionName string, migration interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	migrationStruct, ok := migration.(*MigrationInCode)
	if !ok {
		return errors.Wrapf(base.ErrInvalidPointer, "expect %q", helper.ObjName(MigrationInCode{}))
	}

	conn.RegisterMigration(migrationStruct)

	return nil
}

// WaitForFlush blocks execution until queue will be empty.
func (p *Provider) WaitForFlush(ctx context.Context, connectionName string) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	waitChan := make(chan bool)
	item := &QueueItem{IsWaitForFlush: true, WaitForFlush: waitChan}
	conn.AppendToQueue(item)
	<-waitChan
	p.GetLogger(ctx).Debug().Msg("data flushed to database")

	return nil
}
