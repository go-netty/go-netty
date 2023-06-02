package utils

import (
	"runtime"
	"sync/atomic"
)

type SpinLocker int32

func (s *SpinLocker) Lock() {
	const maxBackoff = 16
	var backoff = 1
	for !atomic.CompareAndSwapInt32((*int32)(s), 0, 1) {
		// Leverage the exponential backoff algorithm, see https://en.wikipedia.org/wiki/Exponential_backoff.
		for i := 0; i < backoff; i++ {
			runtime.Gosched()
		}
		if backoff < maxBackoff {
			backoff <<= 1
		}
	}
}

func (s *SpinLocker) Unlock() {
	atomic.StoreInt32((*int32)(s), 0)
}
