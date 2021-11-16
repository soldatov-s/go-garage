package pq

import (
	"hash/crc32"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
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

var ErrDBConnNotEstablished = errors.New("connection not established")

// Mutex provides a distributed mutex across multiple instances via PostgreSQL database
type Mutex struct {
	// Connection to database
	conn *sqlx.DB
	// Mutex ID
	lockID int64
	// Inteval between checking mutex state
	checkInterval time.Duration
	// The current state of the mutex
	locked bool
	// RWMutex to lock inside an instance
	mu sync.RWMutex
}

// NewMutex creates new distributed postgresql mutex
func NewMutex(conn *sqlx.DB, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(conn, defaultLockID, checkInterval)
}

// NewMutexByID creates new distributed postgresql mutex by ID
func NewMutexByID(conn *sqlx.DB, lockID int64, checkInterval time.Duration) (*Mutex, error) {
	checkIntervalValue := checkInterval
	if checkIntervalValue == 0 {
		checkIntervalValue = defaultCheckInterval
	}

	return &Mutex{
		conn:          conn,
		lockID:        lockID,
		checkInterval: checkIntervalValue,
		locked:        false,
	}, nil
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
	return m.commonLock(requestLock)
}

// Unlock deletes lock item for database (PostgreSQL) locking.
func (m *Mutex) Unlock() (err error) {
	m.mu.Unlock()
	return m.commonUnlock(requestUnlock)
}

// RWLock sets rwlock item for database (PostgreSQL) locking. It is
// blocking call which will wait until database lock key will be deleted,
// pretty much like simple mutex.
func (m *Mutex) RWLock() (err error) {
	m.mu.RLock()
	return m.commonLock(requestSharedLock)
}

// RWUnlock deletes rwlock item for database (PostgreSQL) locking.
func (m *Mutex) RWUnlock() (err error) {
	m.mu.RUnlock()
	return m.commonUnlock(requestSharedUnlock)
}

// checkDBConn check that connection not nil and active
func (m *Mutex) checkDBConn() error {
	if m.conn == nil {
		return ErrDBConnNotEstablished
	}

	if m.conn.Ping() != nil {
		return ErrDBConnNotEstablished
	}

	return nil
}

func (m *Mutex) get(dest interface{}, query string) (err error) {
	err = m.checkDBConn()
	if err != nil {
		return err
	}

	return m.conn.Get(dest, query, m.lockID)
}

func (m *Mutex) commonLock(request string) error {
	var result bool

	if err := m.get(&result, request); err != nil {
		return errors.Wrap(err, "get lock")
	}

	m.locked = true

	if !result {
		for {
			if err := m.get(&result, request); err != nil {
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

func (m *Mutex) commonUnlock(request string) (err error) {
	if m.locked {
		err := m.checkDBConn()
		if err != nil {
			return err
		}

		_, err = m.conn.Exec(request, m.lockID)
		if err != nil {
			return err
		}

		m.locked = false
	}

	return nil
}

// IsLocked returns locked or not locked mutex
func (m *Mutex) IsLocked() bool {
	if m.locked {
		return true
	}

	var result int
	if err := m.get(&result, requestIsLocked); err != nil {
		return true
	}

	return result != 0
}
