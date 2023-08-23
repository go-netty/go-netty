// Package pbytes contains tools for pooling byte pool.
// Note that by default it reuse slices with capacity from 128 to 65536 bytes.
package pbytes

// DefaultPool is used by pacakge level functions.
var DefaultPool = New(65536)

// Get returns probably reused slice of bytes with capacity of c.
// Get is a wrapper around DefaultPool.Get().
func Get(c int) *[]byte { return DefaultPool.Get(c) }

// Put returns given slice to reuse pool.
// Put is a wrapper around DefaultPool.Put().
func Put(p *[]byte) { DefaultPool.Put(p) }
