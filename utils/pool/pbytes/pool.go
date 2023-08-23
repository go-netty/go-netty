//go:build !pool_sanitize
// +build !pool_sanitize

package pbytes

import "github.com/go-netty/go-netty/utils/pool"

// Pool contains logic of reusing byte slices of various size.
type Pool struct {
	pool *pool.Pool[*[]byte]
}

// New creates new Pool that reuses slices which size
func New(max int) *Pool {
	return &Pool{pool.New[*[]byte](max)}
}

// Get returns probably reused slice of bytes with capacity of c.
func (p *Pool) Get(c int) *[]byte {
	v, x := p.pool.Get(c)
	if v != nil {
		return v
	}

	bts := make([]byte, 0, x)
	return &bts
}

// Put returns given slice to reuse pool.
// It does not reuse bytes whose size is not power of two or is out of pool
// min/max range.
func (p *Pool) Put(bts *[]byte) {
	p.pool.Put(bts, cap(*bts))
}
