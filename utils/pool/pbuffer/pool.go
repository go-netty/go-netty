//go:build !pool_sanitize
// +build !pool_sanitize

package pbuffer

import (
	"bytes"

	"github.com/go-netty/go-netty/utils/pool"
)

// Pool contains logic of reusing byte slices of various size.
type Pool struct {
	pool *pool.Pool[*bytes.Buffer]
}

// New creates new Pool that reuses slices which size
func New(max int) *Pool {
	return &Pool{pool.New[*bytes.Buffer](max)}
}

// Get returns probably reused *bytes.Buffer of bytes with at least capacity of c.
func (p *Pool) Get(c int) *bytes.Buffer {
	v, x := p.pool.Get(c)
	if v != nil {
		return v
	}
	return bytes.NewBuffer(make([]byte, 0, x))
}

// Put returns given *bytes.Buffer to reuse pool.
// It does not reuse bytes whose size is not power of two or is out of pool
// min/max range.
func (p *Pool) Put(bts *bytes.Buffer) {
	bts.Reset()
	p.pool.Put(bts, bts.Cap())
}
