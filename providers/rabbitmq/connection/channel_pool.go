package rabbitmqcon

import (
	"errors"
	"io"
	"log"
	"sync"
)

var (
	ErrSizeTooSmall = errors.New("size is too small")
	ErrPoolClosed   = errors.New("pool closed")
)

type ChannelPool struct {
	m       sync.Mutex                // Ensure the thread safety of closed when accessing multiple goroutines
	res     chan io.Closer            // Connecting stored Chan
	factory func() (io.Closer, error) // factory method of new connection
	closed  bool                      // connection pool close flag
}

func New(fn func() (io.Closer, error), size uint) (*ChannelPool, error) {
	if size <= 0 {
		return nil, ErrSizeTooSmall
	}
	return &ChannelPool{
		factory: fn,
		res:     make(chan io.Closer, size),
	}, nil
}

// Get a resource from the resource pool
func (p *ChannelPool) Get() (io.Closer, error) {
	select {
	case r, ok := <-p.res:
		log.Println("acquire: shared resources")
		if !ok {
			return nil, ErrPoolClosed
		}
		return r, nil
	default:
		log.Println("acquire: newly generated resource")
		return p.factory()
	}
}

// Close the resource pool and release the resources
func (p *ChannelPool) Close() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return
	}

	p.closed = true

	// Turn off the channel and stop writing
	close(p.res)

	// Shut down the resources in the channel
	for r := range p.res {
		r.Close()
	}
}

func (p *ChannelPool) Release(r io.Closer) {
	// Ensure that the operation and the operation of the close method are safe
	p.m.Lock()
	defer p.m.Unlock()

	// If the resource pool is closed, the resource that has not been released can be released
	if p.closed {
		r.Close()
		return
	}

	select {
	case p.res <- r:
		log.Println("resources are released into the pool")
	default:
		log.Println("The resource pool is full. Release this resource.")
		r.Close()
	}
}
