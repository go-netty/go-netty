//go:build !pool_sanitize
// +build !pool_sanitize

package pbytes

import (
	"crypto/rand"
	"reflect"
	"strconv"
	"testing"
	"unsafe"
)

func TestPoolGet(t *testing.T) {
	for _, test := range []struct {
		max      int
		cap      int
		exactCap int
	}{
		{
			max:      64,
			cap:      10,
			exactCap: 16,
		},
		{
			max:      10,
			cap:      20,
			exactCap: 32,
		},
	} {
		t.Run("", func(t *testing.T) {
			p := New(test.max)
			act := p.Get(test.cap)
			if c := cap(*act); test.exactCap != 0 && c != test.exactCap {
				t.Errorf(
					"Get(%d) retured %d-cap slice; want exact %d",
					test.cap, c, test.exactCap,
				)
			}
		})
	}
}

func TestPoolPut(t *testing.T) {
	p := New(32)

	miss := make([]byte, 5, 5)
	rand.Read(miss)
	p.Put(&miss) // Should not reuse.

	hit := make([]byte, 8, 8)
	rand.Read(hit)
	p.Put(&hit) // Should reuse.

	b := p.Get(5)
	if data(*b) == data(miss) {
		t.Fatalf("unexpected reuse")
	}
	if data(*b) != data(hit) {
		t.Fatalf("want reuse")
	}
}

func data(p []byte) uintptr {
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	return hdr.Data
}

func BenchmarkPool(b *testing.B) {
	for _, size := range []int{
		1 << 4,
		1 << 5,
		1 << 6,
		1 << 7,
		1 << 8,
		1 << 9,
		1 << 10,
		1 << 11,
		1 << 12,
		1 << 13,
		1 << 14,
		1 << 15,
		1 << 16,
	} {
		b.Run(strconv.Itoa(size)+"(pool)", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					p := Get(size)
					Put(p)
				}
			})
		})
		b.Run(strconv.Itoa(size)+"(make)", func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_ = make([]byte, size)
				}
			})
		})
	}
}
