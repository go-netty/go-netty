# pbytes
The initial code cloned from https://github.com/gobwas/pool

## Problem

> https://research.swtch.com/interfaces

If you try to assign a `[]byte` to an `interface{}`, the compiler will generate additional conversion code `runtime.convTslice` and cause memory allocation.

```go
func main() {
	buffer := make([]byte, 0, 64)
	var o interface{} = buffer // CALL    runtime.convTslice(SB)
	fmt.Println(o)
}
```

`convTslice` source code https://go.dev/src/runtime/iface.go
```go
func convTslice(val []byte) (x unsafe.Pointer) {
	// Note: this must work for any element type, not just byte.
	if (*slice)(unsafe.Pointer(&val)).array == nil {
		x = unsafe.Pointer(&zeroVal[0])
	} else {
		x = mallocgc(unsafe.Sizeof(val), sliceType, true)
		*(*[]byte)(x) = val
	}
	return
}
```

`(*sync.Pool).Put` source code https://go.dev/src/sync/pool.go

```go
// Put adds x to the pool
func (p *Pool) Put(x any) {
    ...
}
```

```go
var buf = make([]byte, 0, 16)
// equals:
// var x interface{} = buf // CALL    runtime.convTslice(SB)
// pool.Put(x)
pool.Put(buf) 
```

## Resolved

Don't put the `[]byte` into `sync.Pool`, to resolve the problems, put `*[]byte` instead.