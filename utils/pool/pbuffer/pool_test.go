package pbuffer

import (
	"bytes"
	"reflect"
	"testing"
	"unsafe"
)

func TestPoolGet(t *testing.T) {
	for _, test := range []struct {
		min      int
		max      int
		cap      int
		exactCap int
	}{
		{
			max:      64,
			cap:      24,
			exactCap: 32,
		},
		{
			max:      10,
			cap:      24,
			exactCap: 24,
		},
	} {
		t.Run("", func(t *testing.T) {
			p := New(test.max)
			act := p.Get(test.cap)
			if c := act.Cap(); c < test.cap {
				t.Errorf(
					"Get(%d) retured %d-cap *bytes.Buffer; want at least %[1]d",
					test.cap, c,
				)
			}
			if c := act.Cap(); test.exactCap != 0 && c < test.exactCap {
				t.Errorf(
					"Get(%d) retured %d-cap *bytes.Buffer; want exact %d",
					test.cap, c, test.exactCap,
				)
			}
		})
	}
}

func TestPoolPut(t *testing.T) {
	p := New(32)

	miss := bytes.NewBuffer(make([]byte, 5, 33))
	p.Put(miss) // Should not reuse.

	hit := bytes.NewBuffer(make([]byte, 8, 8))
	p.Put(hit) // Should reuse.

	b := p.Get(5)
	if data(b.Bytes()) == data(miss.Bytes()) {
		t.Fatalf("unexpected reuse")
	}
	if data(b.Bytes()) != data(hit.Bytes()) {
		t.Fatalf("want reuse")
	}
}

func data(p []byte) uintptr {
	hdr := (*reflect.SliceHeader)(unsafe.Pointer(&p))
	return hdr.Data
}
