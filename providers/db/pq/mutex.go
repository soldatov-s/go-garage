package pq

import (
	// stdlib
	"hash/crc32"
	"strings"
	"sync"
	"time"

	// local
	"github.com/soldatov-s/go-garage/providers/db"

	// other
	"github.com/jmoiron/sqlx"
)

const (
	lockIDSalt           = uint(1364987532)
	defaultLockID        = int64(12345678)
	defaultCheckInterval = 100 * time.Millisecond
	requestLock          = "select pg_try_advisory_lock($1)"
	requestSharedLock    = "select pg_try_advisory_lock_shared($1)"
	requestUnlock        = "select pg_advisory_unlock($1)"
	requestSharedUnlock  = "select pg_advisory_unlock_shared($1)"
)

// Mutex provides a distributed mutex across multiple instances via PostgreSQL database
type Mutex struct {
	// Connection to database
	conn **sqlx.DB
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
func NewMutex(conn **sqlx.DB, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(conn, defaultLockID, checkInterval)
}

// NewMutexByID creates new distributed postgresql mutex by ID
func NewMutexByID(conn **sqlx.DB, lockID interface{}, checkInterval time.Duration) (*Mutex, error) {
	id, ok := lockID.(int64)
	if !ok {
		return nil, db.ErrNotLockIDPointer
	}

	checkIntervalValue := checkInterval
	if checkIntervalValue == 0 {
		checkIntervalValue = defaultCheckInterval
	}

	return &Mutex{
		conn:          conn,
		lockID:        id,
		checkInterval: checkIntervalValue,
		locked:        false,
	}, nil
}

// GenerateLockID generate LockID by database name and other artifacts
func (psqlLock *Mutex) GenerateLockID(databaseName string, additionalNames ...string) {
	if len(additionalNames) > 0 {
		databaseName = strings.Join(append(additionalNames, databaseName), "\x00")
	}

	sum := crc32.ChecksumIEEE([]byte(databaseName))
	psqlLock.lockID = int64(sum * uint32(lockIDSalt))
}

// Lock sets lock item for database (PostgreSQL) locking. It is
// blocking call which will wait until database lock key will be deleted,
// pretty much like simple mutex.
func (psqlLock *Mutex) Lock() (err error) {
	psqlLock.mu.Lock()
	return psqlLock.commonLock(requestLock)
}

// Unlock deletes lock item for database (PostgreSQL) locking.
func (psqlLock *Mutex) Unlock() (err error) {
	psqlLock.mu.Unlock()
	return psqlLock.commonUnlock(requestUnlock)
}

// RWLock sets rwlock item for database (PostgreSQL) locking. It is
// blocking call which will wait until database lock key will be deleted,
// pretty much like simple mutex.
func (psqlLock *Mutex) RWLock() (err error) {
	psqlLock.mu.RLock()
	return psqlLock.commonLock(requestSharedLock)
}

// RWUnlock deletes rwlock item for database (PostgreSQL) locking.
func (psqlLock *Mutex) RWUnlock() (err error) {
	psqlLock.mu.RUnlock()
	return psqlLock.commonUnlock(requestSharedUnlock)
}

// checkDBConn check that connection not nil and active
func (psqlLock *Mutex) checkDBConn() (conn *sqlx.DB, err error) {
	conn = *psqlLock.conn

	if conn == nil {
		return nil, db.ErrDBConnNotEstablished
	}

	if conn.Ping() != nil {
		return nil, db.ErrDBConnNotEstablished
	}

	return conn, nil
}

func (psqlLock *Mutex) commonLock(request string) (err error) {
	var result bool

	conn, err := psqlLock.checkDBConn()
	if err != nil {
		return err
	}

	err = conn.Get(&result, request, psqlLock.lockID)
	if err != nil {
		return err
	}

	psqlLock.locked = true

	if !result {
		for {
			conn, err := psqlLock.checkDBConn()
			if err != nil {
				return err
			}

			err = conn.Get(&result, request, psqlLock.lockID)
			if err != nil {
				return err
			}

			if result || !psqlLock.locked {
				return nil
			}

			time.Sleep(psqlLock.checkInterval)
		}
	}

	return nil
}

func (psqlLock *Mutex) commonUnlock(request string) (err error) {
	if psqlLock.locked {
		conn, err := psqlLock.checkDBConn()
		if err != nil {
			return err
		}

		_, err = conn.Exec(request, psqlLock.lockID)
		if err != nil {
			return err
		}

		psqlLock.locked = false
	}

	return nil
}
