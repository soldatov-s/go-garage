package pq

import (
	"context"
	"database/sql"
	"hash/crc32"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

const (
	lockIDSalt           = uint(1364987532)
	defaultLockID        = int64(12345678)
	defaultCheckInterval = 100 * time.Millisecond
	requestLock          = "select pg_try_advisory_lock($1)"
	requestSharedLock    = "select pg_try_advisory_lock_shared($1)"
	requestUnlock        = "select pg_advisory_unlock($1)"
	requestSharedUnlock  = "select pg_advisory_unlock_shared($1)"
	requestIsLocked      = "select count(*) from pg_locks where pid = pg_backend_pid() AND objid=$1"
)

type MutexConnector interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// Mutex provides a distributed mutex across multiple instances via PostgreSQL database
type Mutex struct {
	// Connection to database
	conn MutexConnector
	// Mutex ID
	lockID int64
	// Inteval between checking mutex state
	checkInterval time.Duration
	// The current state of the mutex
	locked bool
	// RWMutex to lock inside an instance
	mu sync.RWMutex
}

type MutexOption func(*Mutex)

func WithCheckInterval(checkInterval time.Duration) MutexOption {
	return func(c *Mutex) {
		c.checkInterval = checkInterval
	}
}

func WithLockID(lockID int64) MutexOption {
	return func(c *Mutex) {
		c.lockID = lockID
	}
}

// NewMutex creates new distributed postgresql mutex
func NewMutex(conn MutexConnector, opts ...MutexOption) (*Mutex, error) {
	mu := &Mutex{
		conn:          conn,
		lockID:        defaultLockID,
		checkInterval: defaultCheckInterval,
		locked:        false,
	}

	for _, opt := range opts {
		opt(mu)
	}

	return mu, nil
}

// GenerateLockID generate LockID by database name and other artifacts
func (m *Mutex) GenerateLockID(databaseName string, additionalNames ...string) {
	if len(additionalNames) > 0 {
		databaseName = strings.Join(append(additionalNames, databaseName), "\x00")
	}

	sum := crc32.ChecksumIEEE([]byte(databaseName))
	m.lockID = int64(sum * uint32(lockIDSalt))
}

// Lock sets lock item for database (PostgreSQL) locking. It is
// blocking call which will wait until database lock key will be deleted,
// pretty much like simple mutex.
func (m *Mutex) Lock() (err error) {
	m.mu.Lock()
	return m.LockContext(context.Background())
}

func (m *Mutex) LockContext(ctx context.Context) (err error) {
	m.mu.Lock()
	return m.commonLock(ctx, requestLock)
}

// Unlock deletes lock item for database (PostgreSQL) locking.
func (m *Mutex) Unlock() (err error) {
	m.mu.Unlock()
	return m.UnlockContext(context.Background())
}

func (m *Mutex) UnlockContext(ctx context.Context) (err error) {
	m.mu.Unlock()
	return m.commonUnlock(ctx, requestUnlock)
}

// RWLock sets rwlock item for database (PostgreSQL) locking. It is
// blocking call which will wait until database lock key will be deleted,
// pretty much like simple mutex.
func (m *Mutex) RWLock() error {
	m.mu.RLock()
	return m.RWLockContext(context.Background())
}

func (m *Mutex) RWLockContext(ctx context.Context) error {
	m.mu.RLock()
	return m.commonLock(ctx, requestSharedLock)
}

// RWUnlock deletes rwlock item for database (PostgreSQL) locking.
func (m *Mutex) RWUnlock() error {
	m.mu.RUnlock()
	return m.RWUnlockContext(context.Background())
}

func (m *Mutex) RWUnlockContext(ctx context.Context) error {
	m.mu.RUnlock()
	return m.commonUnlock(ctx, requestSharedUnlock)
}

func (m *Mutex) commonLock(ctx context.Context, request string) error {
	var result bool

	if err := m.conn.GetContext(ctx, &result, request, m.lockID); err != nil {
		return errors.Wrap(err, "get lock")
	}

	m.locked = true

	if !result {
		for {
			if err := m.conn.GetContext(ctx, &result, request, m.lockID); err != nil {
				return errors.Wrap(err, "get lock")
			}

			if result || !m.locked {
				return nil
			}

			time.Sleep(m.checkInterval)
		}
	}

	return nil
}

func (m *Mutex) commonUnlock(ctx context.Context, request string) error {
	if m.locked {
		if _, err := m.conn.ExecContext(ctx, request, m.lockID); err != nil {
			return errors.Wrap(err, "exec request")
		}

		m.locked = false
	}

	return nil
}

// IsLocked returns locked or not locked mutex
func (m *Mutex) IsLocked(ctx context.Context) bool {
	if m.locked {
		return true
	}

	var result int
	if err := m.conn.GetContext(ctx, &result, requestIsLocked, m.lockID); err != nil {
		return true
	}

	return result != 0
}
