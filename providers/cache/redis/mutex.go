package redis

import (
	// stdlib
	"sync"
	"time"

	// local
	"github.com/soldatov-s/go-garage/providers/cache"

	// other
	"github.com/KromDaniel/rejonson"
	uuid "gitlab.com/pztrn/go-uuid"
)

const (
	defaultLockID        = "8d0d7b2641d711ea8f655f8773827ca0"
	defaultCheckInterval = 100 * time.Millisecond
	defaultExpire        = 15 * time.Second
)

// Mutex provides a distributed mutex across multiple instances via Redis
type Mutex struct {
	conn          **rejonson.Client
	lockKey       string
	lockValue     string
	checkInterval time.Duration
	expire        time.Duration
	locked        bool
	// RWMutex to lock inside an instance
	mu sync.RWMutex
}

// NewMutex creates new distributed redis mutex
func NewMutex(conn **rejonson.Client, expire, checkInterval time.Duration) (*Mutex, error) {
	return NewMutexByID(conn, defaultLockID, expire, checkInterval)
}

// NewMutexByID creates new distributed redis mutex by ID
func NewMutexByID(conn **rejonson.Client, lockKey interface{}, expire, checkInterval time.Duration) (*Mutex, error) {
	id, ok := lockKey.(string)
	if !ok {
		return nil, cache.ErrNotLockKey
	}

	checkIntervalValue := checkInterval
	if checkIntervalValue == 0 {
		checkIntervalValue = defaultCheckInterval
	}

	expireValue := expire
	if expireValue == 0 {
		expireValue = defaultExpire
	}

	return &Mutex{
		conn:          conn,
		lockKey:       id,
		checkInterval: checkIntervalValue,
		expire:        expireValue,
		locked:        false,
	}, nil
}

// Lock sets Redis-lock item. It is blocking call which will wait until
// redis lock key will be deleted, pretty much like simple mutex.
func (redisLock *Mutex) Lock() (err error) {
	redisLock.mu.Lock()
	return redisLock.commonLock()
}

// Unlock deletes Redis-lock item.
func (redisLock *Mutex) Unlock() (err error) {
	redisLock.mu.Unlock()
	return redisLock.commonUnlock()
}

// Extend attempts to extend the timeout of a Redis-lock.
func (redisLock *Mutex) Extend(timeout time.Duration) (err error) {
	return redisLock.commonExtend(timeout)
}

// checkDBConn check that connection not nil and active
func (redisLock *Mutex) checkDBConn() (conn *rejonson.Client, err error) {
	conn = *redisLock.conn

	if conn == nil {
		return nil, cache.ErrDBConnNotEstablished
	}

	_, err = conn.Ping().Result()
	if err != nil {
		return nil, cache.ErrDBConnNotEstablished
	}

	return conn, nil
}

func (redisLock *Mutex) commonLock() (err error) {
	var result bool

	conn, err := redisLock.checkDBConn()
	if err != nil {
		return err
	}

	newUUID, err := uuid.NewV4()
	if err != nil {
		return err
	}

	redisLock.lockValue = newUUID.String()

	result, err = conn.SetNX(redisLock.lockKey, redisLock.lockValue, redisLock.expire).Result()
	if err != nil {
		return err
	}

	if !result {
		redisLock.locked = true

		for {
			conn, err := redisLock.checkDBConn()
			if err != nil {
				return err
			}

			result, err = conn.SetNX(redisLock.lockKey, redisLock.lockValue, redisLock.expire).Result()
			if err != nil {
				return err
			}

			if result || !redisLock.locked {
				return nil
			}

			time.Sleep(redisLock.checkInterval)
		}
	}

	return nil
}

func (redisLock *Mutex) commonUnlock() (err error) {
	if redisLock.locked {
		conn, err := redisLock.checkDBConn()
		if err != nil {
			return err
		}

		cmdString, err := conn.Get(redisLock.lockKey).Result()
		if err != nil {
			return err
		}

		if redisLock.lockValue == cmdString {
			_, err = conn.Del(redisLock.lockKey).Result()
			if err != nil {
				return err
			}

			redisLock.locked = false
		}
	}

	return nil
}

func (redisLock *Mutex) commonExtend(timeout time.Duration) (err error) {
	if redisLock.locked {
		conn, err := redisLock.checkDBConn()
		if err != nil {
			return err
		}

		cmdString, err := conn.Get(redisLock.lockKey).Result()
		if err != nil {
			return err
		}

		if redisLock.lockValue == cmdString {
			_, err = conn.Expire(redisLock.lockKey, timeout).Result()
			if err != nil {
				return err
			}

			redisLock.locked = false
		}
	}

	return nil
}
