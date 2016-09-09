// Package pool manages a user-defined set of resources that can be shared among goroutines.
package pool

import (
	"errors"
	"io"
	"log"
	"sync"
)

// ErrPoolClosed is returned when an Acquire returns on a closed pool.
var ErrPoolClosed = errors.New("Pool has been closed.")

type Pool struct {
	m         sync.Mutex
	resources chan io.Closer
	factory   func() (io.Closer, error)
	closed    bool
}

// New creates a pool that manages resources.
// - Requires a function that can allocate a new resource, and the size of the pool.
func New(fn func() (io.Closer, error), size uint) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("Invalid size value: too small")
	}

	return &Pool{
		factory:   fn,
		resources: make(chan io.Closer, size),
	}, nil
}

// Acquire retrieves a resource from the pool.
func (p *Pool) Acquire() (io.Closer, error) {
	// Check for a free resource.
	select {
	case r, ok := <-p.resources:
		if !ok {
			return nil, ErrPoolClosed
		}
		return r, nil
	// Provide a new resource since there are none available.
	default:
		log.Println("Acquire:", "New resource")
		return p.factory()
	}
}

// Release places a new resource into the pool.
func (p *Pool) Release(r io.Closer) {
	// Secure this operation with `Close`.
	p.m.Lock()
	defer p.m.Unlock()

	// If the pool is closed, discard the resource.
	if p.closed {
		r.Close()
		return
	}

	select {
	// Attempt to place new resource on the queue.
	case p.resources <- r:
		log.Println("Release:", "In queue")
	// If the queue is already at max capacity, close the resource.
	default:
		log.Println("Release:", "Closing")
		r.Close()
	}
}

// Close shutsdown the pool and closes all existing resources.
func (p *Pool) Close() {
	p.m.Lock()
	defer p.m.Unlock()

	// If the pool is already closed, don't do anything.
	if p.closed {
		log.Println("Pool already closed.")
		return
	}

	// Close the pool
	log.Println("Closing pool..")
	p.closed = true

	// Close the channel before draining it of its resources. Deadlock will occur if this is not done.
	close(p.resources)

	// Close the resources
	for r := range p.resources {
		r.Close()
	}
}
