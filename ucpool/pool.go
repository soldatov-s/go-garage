//
// Package ucpool created on base database/sql pakage
//
package ucpool

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/soldatov-s/go-garage/ucpool/driver"
)

// nowFunc returns the current time; it's overridden in tests.
var nowFunc = time.Now

// Pool is a handle representing a pool of zero or more
// underlying connections. It's safe for concurrent use by multiple
// goroutines.
//
// The ucpool package creates and frees connections automatically; it
// also maintains a free pool of idle connections. The pool size
// can be controlled with SetMaxIdleConns.
type Pool struct {
	// Atomic access only. At top of struct to prevent mis-alignment
	// on 32-bit platforms. Of type time.Duration.
	waitDuration int64 // Total time waited for new connections.

	connector driver.Connector
	// numClosed is an atomic counter which represents a total number of
	// closed connections. Stmt.openStmt checks it before cleaning closed
	// connections in Stmt.css.
	numClosed uint64

	mu           sync.Mutex // protects following fields
	freeConn     []*DriverConn
	connRequests map[uint64]chan connRequest
	nextRequest  uint64 // Next key to use in connRequests.
	numOpen      int    // number of opened and pending open connections
	// Used to signal the need for new connections
	// a goroutine running connectionOpener() reads on this chan and
	// maybeOpenNewConnections sends on the chan (one send per needed connection)
	// It is closed during p.Close(). The close tells the connectionOpener
	// goroutine to exit.
	openerCh          chan struct{}
	closed            bool
	dep               map[finalCloser]depSet
	lastPut           map[*DriverConn]string // stacktrace of last conn's put; debug only
	maxIdleCount      int                    // zero means defaultMaxIdleConns; negative means 0
	maxOpen           int                    // <= 0 means unlimited
	maxLifetime       time.Duration          // maximum amount of time a connection may be reused
	maxIdleTime       time.Duration          // maximum amount of time a connection may be idle before being closed
	cleanerCh         chan struct{}
	waitCount         int64 // Total number of connections waited for.
	maxIdleClosed     int64 // Total number of connections closed due to idle count.
	maxIdleTimeClosed int64 // Total number of connections closed due to idle time.
	maxLifetimeClosed int64 // Total number of connections closed due to max connection lifetime limit.

	stop func() // stop cancels the connection opener.
}

// connReuseStrategy determines how (*Pool).conn returns connections.
type connReuseStrategy uint8

const (
	// alwaysNewConn forces a new connection.
	alwaysNewConn connReuseStrategy = iota
	// cachedOrNewConn returns a cached connection, if available, else waits
	// for one to become available (if MaxOpenConns has been reached) or
	// creates a new connection.
	cachedOrNewConn
)

// DriverConn wraps a driver.Conn with a mutex, to
// be held during all calls into the Conn. (including any calls onto
// interfaces returned via that Conn, such as calls on Tx, Stmt,
// Result, Rows)
type DriverConn struct {
	p         *Pool
	createdAt time.Time

	sync.Mutex  // guards following
	ci          io.Closer
	needReset   bool // The connection session should be reset before use if true.
	closed      bool
	finalClosed bool // ci.Close has been called

	// guarded by p.mu
	inUse      bool
	returnedAt time.Time // Time the connection was created or returned.
	onPut      []func()  // code (with p.mu held) run when conn is next returned
	pmuClosed  bool      // same as closed, but guarded by p.mu, for removeClosedStmtLocked
}

// GetCi returns connection interface
func (dc *DriverConn) GetCi() interface{} {
	return dc.ci
}

func (dc *DriverConn) releaseConn(err error) {
	dc.p.putConn(dc, err, true)
}

func (dc *DriverConn) expired(timeout time.Duration) bool {
	if timeout <= 0 {
		return false
	}
	return dc.createdAt.Add(timeout).Before(nowFunc())
}

// resetSession checks if the driver connection needs the
// session to be reset and if required, resets it.
func (dc *DriverConn) resetSession(ctx context.Context) error {
	dc.Lock()
	defer dc.Unlock()

	if !dc.needReset {
		return nil
	}
	if cr, ok := dc.ci.(driver.SessionResetter); ok {
		return cr.ResetSession(ctx)
	}
	return nil
}

// validateConnection checks if the connection is valid and can
// still be used. It also marks the session for reset if required.
func (dc *DriverConn) validateConnection(needsReset bool) bool {
	dc.Lock()
	defer dc.Unlock()

	if needsReset {
		dc.needReset = true
	}
	if cv, ok := dc.ci.(driver.Validator); ok {
		return cv.IsValid()
	}
	return true
}

var ErrDuplicateClose = errors.New("sql: duplicate driverConn close")

// the dc.p's Mutex is held.
func (dc *DriverConn) closePoolLocked() func() error {
	dc.Lock()
	defer dc.Unlock()
	if dc.closed {
		return func() error { return ErrDuplicateClose }
	}
	dc.closed = true
	return dc.p.removeDepLocked(dc, dc)
}

func (dc *DriverConn) Close() error {
	dc.Lock()
	if dc.closed {
		dc.Unlock()
		return ErrDuplicateClose
	}
	dc.closed = true
	dc.Unlock() // not defer; removeDep finalClose calls may need to lock

	// And now updates that require holding dc.mu.Lock.
	dc.p.mu.Lock()
	dc.pmuClosed = true
	fn := dc.p.removeDepLocked(dc, dc)
	dc.p.mu.Unlock()
	return fn()
}

func (dc *DriverConn) finalClose() error {
	var err error

	withLock(dc, func() {
		dc.finalClosed = true
		err = dc.ci.Close()
		dc.ci = nil
	})

	dc.p.mu.Lock()
	dc.p.numOpen--
	dc.p.maybeOpenNewConnections()
	dc.p.mu.Unlock()

	atomic.AddUint64(&dc.p.numClosed, 1)
	return err
}

func (dc *DriverConn) WithLock(fn func()) {
	withLock(dc, fn)
}

// depSet is a finalCloser's outstanding dependencies
type depSet map[interface{}]bool // set of true bools

// The finalCloser interface is used by (*Pool).addDep and related
// dependency reference counting.
type finalCloser interface {
	// finalClose is called when the reference count of an object
	// goes to zero. (*Pool).mu is not held while calling it.
	finalClose() error
}

func (p *Pool) addDepLocked(x finalCloser, dep interface{}) {
	if p.dep == nil {
		p.dep = make(map[finalCloser]depSet)
	}
	xdep := p.dep[x]
	if xdep == nil {
		xdep = make(depSet)
		p.dep[x] = xdep
	}
	xdep[dep] = true
}

func (p *Pool) removeDepLocked(x finalCloser, dep interface{}) func() error {
	xdep, ok := p.dep[x]
	if !ok {
		panic(fmt.Sprintf("unpaired removeDep: no deps for %T", x))
	}

	l0 := len(xdep)
	delete(xdep, dep)

	switch len(xdep) {
	case l0:
		// Nothing removed. Shouldn't happen.
		panic(fmt.Sprintf("unpaired removeDep: no %T dep on %T", dep, x))
	case 0:
		// No more dependencies.
		delete(p.dep, x)
		return x.finalClose
	default:
		// Dependencies remain.
		return func() error { return nil }
	}
}

// This is the size of the connectionOpener request chan (Pool.openerCh).
// This value should be larger than the maximum typical value
// used for p.maxOpen. If maxOpen is significantly larger than
// connectionRequestQueueSize then it is possible for ALL calls into the *Pool
// to block until the connectionOpener can satisfy the backlog of requests.
var connectionRequestQueueSize = 1000000

// OpenPool opens a p, message broker and etc. using a Connector, allowing drivers to
// bypass a string based data source name.
//
// OpenPool may just validate its arguments without creating a connection
// to the p, message broker and etc. To verify that the data source name is valid, call
// Ping.
//
// The returned Pool is safe for concurrent use by multiple goroutines
// and maintains its own pool of idle connections. Thus, the OpenPool
// function should be called just once. It is rarely necessary to
// close a Pool.
func OpenPool(ctx context.Context, c driver.Connector) *Pool {
	ctx, cancel := context.WithCancel(ctx)
	p := &Pool{
		connector:    c,
		openerCh:     make(chan struct{}, connectionRequestQueueSize),
		lastPut:      make(map[*DriverConn]string),
		connRequests: make(map[uint64]chan connRequest),
		stop:         cancel,
	}

	go p.connectionOpener(ctx)

	return p
}

// Close closes the p, message broker and etc. `and prevents new queries from starting.`
// Close then waits for all queries that have started processing on the server
// to finish.
//
// It is rare to Close a Pool, as the Pool handle is meant to be
// long-lived and shared between many goroutines.
func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed { // Make Pool.Close idempotent
		p.mu.Unlock()
		return nil
	}
	if p.cleanerCh != nil {
		close(p.cleanerCh)
	}
	var err error
	fns := make([]func() error, 0, len(p.freeConn))
	for _, dc := range p.freeConn {
		fns = append(fns, dc.closePoolLocked())
	}
	p.freeConn = nil
	p.closed = true
	for _, req := range p.connRequests {
		close(req)
	}
	p.mu.Unlock()
	for _, fn := range fns {
		err1 := fn()
		if err1 != nil {
			err = err1
		}
	}
	p.stop()
	if c, ok := p.connector.(io.Closer); ok {
		err1 := c.Close()
		if err1 != nil {
			err = err1
		}
	}
	return err
}

const defaultMaxIdleConns = 2

func (p *Pool) maxIdleConnsLocked() int {
	n := p.maxIdleCount
	switch {
	case n == 0:
		return defaultMaxIdleConns
	case n < 0:
		return 0
	default:
		return n
	}
}

func (p *Pool) shortestIdleTimeLocked() time.Duration {
	if p.maxIdleTime <= 0 {
		return p.maxLifetime
	}
	if p.maxLifetime <= 0 {
		return p.maxIdleTime
	}

	min := p.maxIdleTime
	if min > p.maxLifetime {
		min = p.maxLifetime
	}
	return min
}

// SetMaxIdleConns sets the maximum number of connections in the idle
// connection pool.
//
// If MaxOpenConns is greater than 0 but less than the new MaxIdleConns,
// then the new MaxIdleConns will be reduced to match the MaxOpenConns limit.
//
// If n <= 0, no idle connections are retained.
//
// The default max idle connections is currently 2. This may change in
// a future release.
func (p *Pool) SetMaxIdleConns(n int) {
	p.mu.Lock()
	if n > 0 {
		p.maxIdleCount = n
	} else {
		// No idle connections.
		p.maxIdleCount = -1
	}
	// Make sure maxIdle doesn't exceed maxOpen
	if p.maxOpen > 0 && p.maxIdleConnsLocked() > p.maxOpen {
		p.maxIdleCount = p.maxOpen
	}
	var closing []*DriverConn
	maxIdle := p.maxIdleConnsLocked()
	if len(p.freeConn) > maxIdle {
		closing = p.freeConn[maxIdle:]
		p.freeConn = p.freeConn[:maxIdle]
	}
	p.maxIdleClosed += int64(len(closing))
	p.mu.Unlock()
	for _, c := range closing {
		c.Close()
	}
}

// SetMaxOpenConns sets the maximum number of open connections to
// the pool, message broker and etc..
//
// If MaxIdleConns is greater than 0 and the new MaxOpenConns is less than
// MaxIdleConns, then MaxIdleConns will be reduced to match the new
// MaxOpenConns limit.
//
// If n <= 0, then there is no limit on the number of open connections.
// The default is 0 (unlimited).
func (p *Pool) SetMaxOpenConns(n int) {
	p.mu.Lock()
	p.maxOpen = n
	if n < 0 {
		p.maxOpen = 0
	}
	syncMaxIdle := p.maxOpen > 0 && p.maxIdleConnsLocked() > p.maxOpen
	p.mu.Unlock()
	if syncMaxIdle {
		p.SetMaxIdleConns(n)
	}
}

// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
//
// Expired connections may be closed lazily before reuse.
//
// If d <= 0, connections are not closed due to a connection's age.
func (p *Pool) SetConnMaxLifetime(d time.Duration) {
	if d < 0 {
		d = 0
	}
	p.mu.Lock()
	// Wake cleaner up when lifetime is shortened.
	if d > 0 && d < p.maxLifetime && p.cleanerCh != nil {
		select {
		case p.cleanerCh <- struct{}{}:
		default:
		}
	}
	p.maxLifetime = d
	p.startCleanerLocked()
	p.mu.Unlock()
}

// SetConnMaxIdleTime sets the maximum amount of time a connection may be idle.
//
// Expired connections may be closed lazily before reuse.
//
// If d <= 0, connections are not closed due to a connection's idle time.
func (p *Pool) SetConnMaxIdleTime(d time.Duration) {
	if d < 0 {
		d = 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// Wake cleaner up when idle time is shortened.
	if d > 0 && d < p.maxIdleTime && p.cleanerCh != nil {
		select {
		case p.cleanerCh <- struct{}{}:
		default:
		}
	}
	p.maxIdleTime = d
	p.startCleanerLocked()
}

// startCleanerLocked starts connectionCleaner if needed.
func (p *Pool) startCleanerLocked() {
	if (p.maxLifetime > 0 || p.maxIdleTime > 0) && p.numOpen > 0 && p.cleanerCh == nil {
		p.cleanerCh = make(chan struct{}, 1)
		go p.connectionCleaner(p.shortestIdleTimeLocked())
	}
}

func (p *Pool) connectionCleaner(d time.Duration) {
	const minInterval = time.Second

	if d < minInterval {
		d = minInterval
	}
	t := time.NewTimer(d)

	for {
		select {
		case <-t.C:
		case <-p.cleanerCh: // maxLifetime was changed or pool was closed.
		}

		p.mu.Lock()

		d = p.shortestIdleTimeLocked()
		if p.closed || p.numOpen == 0 || d <= 0 {
			p.cleanerCh = nil
			p.mu.Unlock()
			return
		}

		closing := p.connectionCleanerRunLocked()
		p.mu.Unlock()
		for _, c := range closing {
			c.Close()
		}

		if d < minInterval {
			d = minInterval
		}
		t.Reset(d)
	}
}

func (p *Pool) connectionCleanerRunLocked() (closing []*DriverConn) {
	if p.maxLifetime > 0 {
		expiredSince := nowFunc().Add(-p.maxLifetime)
		for i := 0; i < len(p.freeConn); i++ {
			c := p.freeConn[i]
			if c.createdAt.Before(expiredSince) {
				closing = append(closing, c)
				last := len(p.freeConn) - 1
				p.freeConn[i] = p.freeConn[last]
				p.freeConn[last] = nil
				p.freeConn = p.freeConn[:last]
				i--
			}
		}
		p.maxLifetimeClosed += int64(len(closing))
	}

	if p.maxIdleTime > 0 {
		expiredSince := nowFunc().Add(-p.maxIdleTime)
		var expiredCount int64
		for i := 0; i < len(p.freeConn); i++ {
			c := p.freeConn[i]
			if p.maxIdleTime > 0 && c.returnedAt.Before(expiredSince) {
				closing = append(closing, c)
				expiredCount++
				last := len(p.freeConn) - 1
				p.freeConn[i] = p.freeConn[last]
				p.freeConn[last] = nil
				p.freeConn = p.freeConn[:last]
				i--
			}
		}
		p.maxIdleTimeClosed += expiredCount
	}
	return
}

// PoolStats contains connection statistics.
type PoolStats struct {
	MaxOpenConnections int // Maximum number of open connections.

	// Pool Status
	OpenConnections int // The number of established connections both in use and idle.
	InUse           int // The number of connections currently in use.
	Idle            int // The number of idle connections.

	// Counters
	WaitCount         int64         // The total number of connections waited for.
	WaitDuration      time.Duration // The total time blocked waiting for a new connection.
	MaxIdleClosed     int64         // The total number of connections closed due to SetMaxIdleConns.
	MaxIdleTimeClosed int64         // The total number of connections closed due to SetConnMaxIdleTime.
	MaxLifetimeClosed int64         // The total number of connections closed due to SetConnMaxLifetime.
}

// Stats returns connection statistics.
func (p *Pool) Stats() *PoolStats {
	wait := atomic.LoadInt64(&p.waitDuration)

	p.mu.Lock()
	defer p.mu.Unlock()

	stats := &PoolStats{
		MaxOpenConnections: p.maxOpen,

		Idle:            len(p.freeConn),
		OpenConnections: p.numOpen,
		InUse:           p.numOpen - len(p.freeConn),

		WaitCount:         p.waitCount,
		WaitDuration:      time.Duration(wait),
		MaxIdleClosed:     p.maxIdleClosed,
		MaxIdleTimeClosed: p.maxIdleTimeClosed,
		MaxLifetimeClosed: p.maxLifetimeClosed,
	}
	return stats
}

// Assumes p.mu is locked.
// If there are connRequests and the connection limit hasn't been reached,
// then tell the connectionOpener to open new connections.
func (p *Pool) maybeOpenNewConnections() {
	numRequests := len(p.connRequests)
	if p.maxOpen > 0 {
		numCanOpen := p.maxOpen - p.numOpen
		if numRequests > numCanOpen {
			numRequests = numCanOpen
		}
	}
	for numRequests > 0 {
		p.numOpen++ // optimistically
		numRequests--
		if p.closed {
			return
		}
		p.openerCh <- struct{}{}
	}
}

// Runs in a separate goroutine, opens new connections when requested.
func (p *Pool) connectionOpener(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.openerCh:
			p.openNewConnection(ctx)
		}
	}
}

// Open one new connection
func (p *Pool) openNewConnection(ctx context.Context) {
	// maybeOpenNewConnections has already executed p.numOpen++ before it sent
	// on p.openerCh. This function must execute p.numOpen-- if the
	// connection fails or is closed before returning.
	ci, err := p.connector.Connect(ctx)
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		if err == nil {
			ci.Close()
		}
		p.numOpen--
		return
	}
	if err != nil {
		p.numOpen--
		p.putConnPoolLocked(nil, err)
		p.maybeOpenNewConnections()
		return
	}
	dc := &DriverConn{
		p:          p,
		createdAt:  nowFunc(),
		returnedAt: nowFunc(),
		ci:         ci,
	}
	if p.putConnPoolLocked(dc, err) {
		p.addDepLocked(dc, dc)
	} else {
		p.numOpen--
		ci.Close()
	}
}

// connRequest represents one request for a new connection
// When there are no idle connections available, Pool.conn will create
// a new connRequest and put it on the p.connRequests list.
type connRequest struct {
	conn *DriverConn
	err  error
}

var errPoolClosed = errors.New("pool is closed")

// nextRequestKeyLocked returns the next connection request key.
// It is assumed that nextRequest will not overflow.
func (p *Pool) nextRequestKeyLocked() uint64 {
	next := p.nextRequest
	p.nextRequest++
	return next
}

// conn returns a newly-opened or cached *driverConn.
// nolint:funlen,gocyclo,gocognit // long function
func (p *Pool) conn(ctx context.Context, strategy connReuseStrategy) (*DriverConn, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, errPoolClosed
	}
	// Check if the context is expired.
	select {
	default:
	case <-ctx.Done():
		p.mu.Unlock()
		return nil, ctx.Err()
	}
	lifetime := p.maxLifetime

	// Prefer a free connection, if possible.
	numFree := len(p.freeConn)
	if strategy == cachedOrNewConn && numFree > 0 {
		conn := p.freeConn[0]
		copy(p.freeConn, p.freeConn[1:])
		p.freeConn = p.freeConn[:numFree-1]
		conn.inUse = true
		if conn.expired(lifetime) {
			p.maxLifetimeClosed++
			p.mu.Unlock()
			conn.Close()
			return nil, p.connector.GetErrBadConn()
		}
		p.mu.Unlock()

		// Reset the session if required.
		if err := conn.resetSession(ctx); p.connector.IsErrBadConn(err) {
			conn.Close()
			return nil, p.connector.GetErrBadConn()
		}

		return conn, nil
	}

	// Out of free connections or we were asked not to use one. If we're not
	// allowed to open any more connections, make a request and wait.
	// nolint:nestif // deeply nested
	if p.maxOpen > 0 && p.numOpen >= p.maxOpen {
		// Make the connRequest channel. It's buffered so that the
		// connectionOpener doesn't block while waiting for the req to be read.
		req := make(chan connRequest, 1)
		reqKey := p.nextRequestKeyLocked()
		p.connRequests[reqKey] = req
		p.waitCount++
		p.mu.Unlock()

		waitStart := nowFunc()

		// Timeout the connection request with the context.
		select {
		case <-ctx.Done():
			// Remove the connection request and ensure no value has been sent
			// on it after removing.
			p.mu.Lock()
			delete(p.connRequests, reqKey)
			p.mu.Unlock()

			atomic.AddInt64(&p.waitDuration, int64(time.Since(waitStart)))

			select {
			default:
			case ret, ok := <-req:
				if ok && ret.conn != nil {
					p.putConn(ret.conn, ret.err, false)
				}
			}
			return nil, ctx.Err()
		case ret, ok := <-req:
			atomic.AddInt64(&p.waitDuration, int64(time.Since(waitStart)))

			if !ok {
				return nil, errPoolClosed
			}
			// Only check if the connection is expired if the strategy is cachedOrNewConns.
			// If we require a new connection, just re-use the connection without looking
			// at the expiry time. If it is expired, it will be checked when it is placed
			// back into the connection pool.
			// This prioritizes giving a valid connection to a client over the exact connection
			// lifetime, which could expire exactly after this point anyway.
			if strategy == cachedOrNewConn && ret.err == nil && ret.conn.expired(lifetime) {
				p.mu.Lock()
				p.maxLifetimeClosed++
				p.mu.Unlock()
				ret.conn.Close()
				return nil, p.connector.GetErrBadConn()
			}
			if ret.conn == nil {
				return nil, ret.err
			}

			// Reset the session if required.
			if err := ret.conn.resetSession(ctx); p.connector.IsErrBadConn(err) {
				ret.conn.Close()
				return nil, p.connector.GetErrBadConn()
			}
			return ret.conn, ret.err
		}
	}

	p.numOpen++ // optimistically
	p.mu.Unlock()
	ci, err := p.connector.Connect(ctx)
	if err != nil {
		p.mu.Lock()
		p.numOpen-- // correct for earlier optimism
		p.maybeOpenNewConnections()
		p.mu.Unlock()
		return nil, err
	}
	p.mu.Lock()
	dc := &DriverConn{
		p:          p,
		createdAt:  nowFunc(),
		returnedAt: nowFunc(),
		ci:         ci,
		inUse:      true,
	}
	p.addDepLocked(dc, dc)
	p.mu.Unlock()
	return dc, nil
}

// putConnHook is a hook for testing.
var putConnHook func(*Pool, *DriverConn)

// debugGetPut determines whether getConn & putConn calls' stack traces
// are returned for more verbose crashes.
const debugGetPut = false

// putConn adds a connection to the free pool.
// err is optionally the last error that occurred on this connection.
// nolint:gocyclo // long function
func (p *Pool) putConn(dc *DriverConn, err error, resetSession bool) {
	if !p.connector.IsErrBadConn(err) {
		if !dc.validateConnection(resetSession) {
			err = p.connector.GetErrBadConn()
		}
	}
	p.mu.Lock()
	if !dc.inUse {
		p.mu.Unlock()
		if debugGetPut {
			// nolint:forbidigo // only for debug
			fmt.Printf("putConn(%v) DUPLICATE was: %s\n\nPREVIOUS was: %s", dc, stack(), p.lastPut[dc])
		}
		panic("sql: connection returned that was never out")
	}

	if !p.connector.IsErrBadConn(err) && dc.expired(p.maxLifetime) {
		p.maxLifetimeClosed++
		err = p.connector.GetErrBadConn()
	}
	if debugGetPut {
		p.lastPut[dc] = stack()
	}
	dc.inUse = false
	dc.returnedAt = nowFunc()

	for _, fn := range dc.onPut {
		fn()
	}
	dc.onPut = nil

	if p.connector.IsErrBadConn(err) {
		// Don't reuse bad connections.
		// Since the conn is considered bad and is being discarded, treat it
		// as closed. Don't decrement the open count here, finalClose will
		// take care of that.
		p.maybeOpenNewConnections()
		p.mu.Unlock()
		dc.Close()
		return
	}
	if putConnHook != nil {
		putConnHook(p, dc)
	}
	added := p.putConnPoolLocked(dc, nil)
	p.mu.Unlock()

	if !added {
		dc.Close()
		return
	}
}

// Satisfy a connRequest or put the driverConn in the idle pool and return true
// or return false.
// putConnPoolLocked will satisfy a connRequest if there is one, or it will
// return the *driverConn to the freeConn list if err == nil and the idle
// connection limit will not be exceeded.
// If err != nil, the value of dc is ignored.
// If err == nil, then dc must not equal nil.
// If a connRequest was fulfilled or the *driverConn was placed in the
// freeConn list, then true is returned, otherwise false is returned.
func (p *Pool) putConnPoolLocked(dc *DriverConn, err error) bool {
	if p.closed {
		return false
	}
	if p.maxOpen > 0 && p.numOpen > p.maxOpen {
		return false
	}
	if c := len(p.connRequests); c > 0 {
		var req chan connRequest
		var reqKey uint64
		for reqKey, req = range p.connRequests {
			break
		}
		delete(p.connRequests, reqKey) // Remove from pending requests.
		if err == nil {
			dc.inUse = true
		}
		req <- connRequest{
			conn: dc,
			err:  err,
		}
		return true
	} else if err == nil && !p.closed {
		if p.maxIdleConnsLocked() > len(p.freeConn) {
			p.freeConn = append(p.freeConn, dc)
			p.startCleanerLocked()
			return true
		}
		p.maxIdleClosed++
	}
	return false
}

// maxBadConnRetries is the number of maximum retries if the driver returns
// ErrBadConn to signal a broken connection before forcing a new
// connection to be opened.
const maxBadConnRetries = 2

// ErrConnDone is returned by any operation that is performed on a connection
// that has already been returned to the connection pool.
var ErrConnDone = errors.New("connection is already closed")

// Conn represents a single connection rather than a pool of
// connections. Prefer running queries from Pool unless there is a specific
// need for a continuous single connection.
//
// A Conn must call Close to return the connection to the pool
// and may do so concurrently with a running query.
//
// After a call to Close, all operations on the
// connection fail with ErrConnDone.
type Conn struct {
	pool *Pool
	// closemu prevents the connection from closing while there
	// is an active query. It is held for read during queries
	// and exclusively during close.
	closemu sync.RWMutex

	// dc is owned until close, at which point
	// it's returned to the connection pool.
	dc *DriverConn

	// done transitions from 0 to 1 exactly once, on close.
	// Once done, all operations fail with ErrConnDone.
	// Use atomic operations on value when checking value.
	done int32
}

// grabConn takes a context to implement stmtConnGrabber
// but the context is not used.
func (c *Conn) GrabConn(context.Context) (*DriverConn, ReleaseConn, error) {
	if atomic.LoadInt32(&c.done) != 0 {
		return nil, nil, ErrConnDone
	}
	c.closemu.RLock()
	return c.dc, c.closemuRUnlockCondReleaseConn, nil
}

type ReleaseConn func(error)

// Conn returns a single connection by either opening a new connection
// or returning an existing connection from the connection pool. Conn will
// block until either a connection is returned or ctx is canceled.
// Queries run on the same Conn will be run in the same session.
//
// Every Conn must be returned to the pool after use by
// calling Conn.Close.
func (p *Pool) Conn(ctx context.Context) (*Conn, error) {
	var dc *DriverConn
	var err error
	for i := 0; i < maxBadConnRetries; i++ {
		dc, err = p.conn(ctx, cachedOrNewConn)
		if !p.connector.IsErrBadConn(err) {
			break
		}
	}
	if p.connector.IsErrBadConn(err) {
		dc, err = p.conn(ctx, alwaysNewConn)
	}
	if err != nil {
		return nil, err
	}

	conn := &Conn{
		pool: p,
		dc:   dc,
	}
	return conn, nil
}

// closemuRUnlockCondReleaseConn read unlocks closemu
// as the operation is done with the dc.
func (c *Conn) closemuRUnlockCondReleaseConn(err error) {
	c.closemu.RUnlock()
	if c.pool.connector.IsErrBadConn(err) {
		c.close(err)
	}
}

func (c *Conn) close(err error) error {
	if !atomic.CompareAndSwapInt32(&c.done, 0, 1) {
		return ErrConnDone
	}

	// Lock around releasing the driver connection
	// to ensure all queries have been stopped before doing so.
	c.closemu.Lock()
	defer c.closemu.Unlock()

	c.dc.releaseConn(err)
	c.dc = nil
	c.pool = nil
	return err
}

// Close returns the connection to the connection pool.
// All operations after a Close will return with ErrConnDone.
// Close is safe to call concurrently with other operations and will
// block until all other operations finish. It may be useful to first
// cancel any used context and then call close directly after.
func (c *Conn) Close() error {
	return c.close(nil)
}

func stack() string {
	var buf [2 << 10]byte
	return string(buf[:runtime.Stack(buf[:], false)])
}

// withLock runs while holding lk.
func withLock(lk sync.Locker, fn func()) {
	lk.Lock()
	defer lk.Unlock() // in case fn panics
	fn()
}
