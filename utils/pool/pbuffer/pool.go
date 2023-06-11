//go:build !pool_sanitize
// +build !pool_sanitize

package pbuffer

import (
	"bytes"

	"github.com/go-netty/go-netty/utils/pool"
)

// Pool contains logic of reusing byte slices of various size.
type Pool struct {
	pool *pool.Pool
}

// New creates new Pool that reuses slices which size is in logarithmic range
// [min, max].
//
// Note that it is a shortcut for Custom() constructor with Options provided by
// pool.WithLogSizeMapping() and pool.WithLogSizeRange(min, max) calls.
func New(min, max int) *Pool {
	return &Pool{pool.New(min, max)}
}

// New creates new Pool with given options.
func Custom(opts ...pool.Option) *Pool {
	return &Pool{pool.Custom(opts...)}
}

// Get returns probably reused *bytes.Buffer of bytes with at least capacity of c.
func (p *Pool) Get(c int) *bytes.Buffer {
	v, x := p.pool.Get(c)
	if v != nil {
		bts := v.(*bytes.Buffer)
		return bts
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
