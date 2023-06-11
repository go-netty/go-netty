package pbuffer

import "bytes"

// DefaultPool is used by pacakge level functions.
var DefaultPool = New(128, 65536)

// Get returns probably reused *bytes.Buffer of bytes with at least capacity of c
// Get is a wrapper around DefaultPool.Get().
func Get(c int) *bytes.Buffer { return DefaultPool.Get(c) }

// Put returns given *bytes.Buffer to reuse pool.
// Put is a wrapper around DefaultPool.Put().
func Put(p *bytes.Buffer) { DefaultPool.Put(p) }
