package pool

import (
	"sync"

	"github.com/go-netty/go-netty/utils/pool/internal/pmath"
)

// Pool contains logic of reusing objects distinguishable by size in generic
// way.
type Pool struct {
	pool map[int]*sync.Pool
	size func(int) int
}

// New creates new Pool that reuses objects which size is in logarithmic range
// [min, max].
//
// Note that it is a shortcut for Custom() constructor with Options provided by
// WithLogSizeMapping() and WithLogSizeRange(min, max) calls.
func New(min, max int) *Pool {
	return Custom(
		WithLogSizeMapping(),
		WithLogSizeRange(min, max),
	)
}

// Custom creates new Pool with given options.
func Custom(opts ...Option) *Pool {
	p := &Pool{
		pool: make(map[int]*sync.Pool),
		size: pmath.Identity,
	}

	c := (*poolConfig)(p)
	for _, opt := range opts {
		opt(c)
	}

	return p
}

// Get pulls object whose generic size is at least of given size.
// It also returns a real size of x for further pass to Put() even if x is nil.
// Note that size could be ceiled to the next power of two.
func (p *Pool) Get(size int) (interface{}, int) {
	n := p.size(size)
	if pool := p.pool[n]; pool != nil {
		return pool.Get(), n
	}
	return nil, size
}

// Put takes x and its size for future reuse.
func (p *Pool) Put(x interface{}, size int) {
	if pool := p.pool[size]; pool != nil {
		pool.Put(x)
	}
}

type poolConfig Pool

// AddSize adds size n to the map.
func (p *poolConfig) AddSize(n int) {
	p.pool[n] = new(sync.Pool)
}

// SetSizeMapping sets up incoming size mapping function.
func (p *poolConfig) SetSizeMapping(size func(int) int) {
	p.size = size
}
