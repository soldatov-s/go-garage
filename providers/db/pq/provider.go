package pq

import (
	"context"
	"time"

	providerwithmetrics "github.com/soldatov-s/go-garage/providers/base/provider/with_metrics"
	"github.com/soldatov-s/go-garage/providers/db"
	"github.com/soldatov-s/go-garage/providers/errors"
)

const defaultProviderName = "pq"

// Provider provides PostgreSQL database worker. This provider
// supports asynchronous database actions (like bulk inserting). Every
// connection will have own goroutine for queue processing.
type Provider struct {
	*providerwithmetrics.Provider
}

// Initialize should initialize provider. If asynchronous mode
// supported by provider (e.g. for batch inserting using transactions)
// queue processor should also be started here.
func NewProvider(ctx context.Context) *Provider {
	return &Provider{
		Provider: providerwithmetrics.NewProvider(ctx, db.ProvidersName, defaultProviderName),
	}
}

// AppendToQueue adds passed item into processing queue.
func (p *Provider) AppendToQueue(connectionName string, item interface{}) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	itemPointer, ok := item.(*QueueItem)
	if !ok {
		return db.ErrNotQueueItemPointer(QueueItem{})
	}

	conn.AppendToQueue(itemPointer)

	return nil
}

// CreateEnity should create enity using passed parameters.
func (p *Provider) CreateEnity(enityName string, options interface{}) error {
	if _, err := p.GetEnity(enityName); err == nil {
		p.Log.Debug().Str("enity name", enityName).Msg("enity already created")
		return nil
	}

	enity, err := NewEnity(p.GetContext(), enityName, options)
	if err != nil {
		return err
	}

	p.Entitys.Store(enityName, enity)

	return nil
}

// GetEnity should return pointer to connection structure to caller.
func (p *Provider) getEnity(enityName string) (*Enity, error) {
	if enityName == "" {
		return nil, errors.ErrEmptyEnityName
	}

	enity, found := p.Entitys.Load(enityName)
	if !found {
		return nil, errors.ErrEnityDoesNotExists
	}

	// Checking that enity type is *Enity
	enityPointer, ok := enity.(*Enity)
	if !ok {
		return nil, errors.ErrInvalidEnityPointer(Enity{})
	}

	return enityPointer, nil
}

// GetEnity should return pointer to connection structure to caller.
func (p *Provider) GetEnity(connectionName string) (interface{}, error) {
	return p.getEnity(connectionName)
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
		return db.ErrNotMigrationPointer(MigrationInCode{})
	}

	conn.RegisterMigration(migrationStruct)

	return nil
}

// WaitForFlush blocks execution until queue will be empty.
func (p *Provider) WaitForFlush(connectionName string) error {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return err
	}

	waitChan := make(chan bool)
	item := &QueueItem{IsWaitForFlush: true, WaitForFlush: waitChan}
	conn.AppendToQueue(item)
	<-waitChan
	p.Log.Debug().Msg("data flushed to database")

	return nil
}

// NewMutex creates new distributed mutex
func (p *Provider) NewMutex(connectionName string, checkInterval time.Duration) (*Mutex, error) {
	return p.NewMutexByID(connectionName, defaultLockID, checkInterval)
}

// NewMutexByID creates new distributed postgresql mutex by ID
func (p *Provider) NewMutexByID(connectionName string, lockID int64, checkInterval time.Duration) (*Mutex, error) {
	conn, err := p.getEnity(connectionName)
	if err != nil {
		return nil, err
	}

	return conn.NewMutexByID(lockID, checkInterval)
}
