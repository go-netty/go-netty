/*
Copyright 2014 Workiva, LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package queue

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// containsKind checks if a specified kind in the slice of kinds.
func containsKind(kinds []reflect.Kind, kind reflect.Kind) bool {
	for i := 0; i < len(kinds); i++ {
		if kind == kinds[i] {
			return true
		}
	}

	return false
}

// isNil checks if a specified object is nil or not, without Failing.
func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	isNilableKind := containsKind(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice, reflect.UnsafePointer},
		kind)

	if isNilableKind && value.IsNil() {
		return true
	}

	return false
}

type TestingT interface {
	Errorf(format string, args ...interface{})
}

func Equal(t TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
	if expected != actual {
		t.Errorf("Not equal: expected: %v, actual: %v %s", expected, actual, fmt.Sprint(msgAndArgs...))
		return false
	}
	return true
}

func Nil(t TestingT, object interface{}, msgAndArgs ...interface{}) bool {
	if isNil(object) {
		return true
	}

	t.Errorf("Expected nil, but got: %#v %s", object, fmt.Sprint(msgAndArgs...))
	return false
}

func NotNil(t TestingT, object interface{}, msgAndArgs ...interface{}) bool {
	if !isNil(object) {
		return true
	}
	t.Errorf("Expected value not to be nil %s", fmt.Sprint(msgAndArgs...))
	return false
}

func True(t TestingT, value bool, msgAndArgs ...interface{}) bool {
	if !value {
		t.Errorf("Should be true %s", fmt.Sprint(msgAndArgs...))
		return false
	}
	return true
}

func False(t TestingT, value bool, msgAndArgs ...interface{}) bool {
	if value {
		t.Errorf("Should be false %s", fmt.Sprint(msgAndArgs...))
		return false
	}
	return true
}

func TestRingInsert(t *testing.T) {
	rb := NewRingBuffer(5)
	Equal(t, uint64(8), rb.Cap())

	err := rb.Put(5)
	if !Nil(t, err) {
		return
	}

	result, err := rb.Get()
	if !Nil(t, err) {
		return
	}

	Equal(t, 5, result)
}

func TestRingMultipleInserts(t *testing.T) {
	rb := NewRingBuffer(5)

	err := rb.Put(1)
	if !Nil(t, err) {
		return
	}

	err = rb.Put(2)
	if !Nil(t, err) {
		return
	}

	result, err := rb.Get()
	if !Nil(t, err) {
		return
	}

	Equal(t, 1, result)

	result, err = rb.Get()
	if Nil(t, err) {
		return
	}

	Equal(t, 2, result)
}

func TestIntertwinedGetAndPut(t *testing.T) {
	rb := NewRingBuffer(5)
	err := rb.Put(1)
	if !Nil(t, err) {
		return
	}

	result, err := rb.Get()
	if !Nil(t, err) {
		return
	}

	Equal(t, 1, result)

	err = rb.Put(2)
	if !Nil(t, err) {
		return
	}

	result, err = rb.Get()
	if !Nil(t, err) {
		return
	}

	Equal(t, 2, result)
}

func TestPutToFull(t *testing.T) {
	rb := NewRingBuffer(3)

	for i := 0; i < 4; i++ {
		err := rb.Put(i)
		if !Nil(t, err) {
			return
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		err := rb.Put(4)
		Nil(t, err)
		wg.Done()
	}()

	go func() {
		defer wg.Done()
		result, err := rb.Get()
		if !Nil(t, err) {
			return
		}

		Equal(t, 0, result)
	}()

	wg.Wait()
}

func TestOffer(t *testing.T) {
	rb := NewRingBuffer(2)

	err := rb.Offer("foo")
	Nil(t, err)
	err = rb.Offer("bar")
	Nil(t, err)
	err = rb.Offer("baz")
	NotNil(t, err)

	item, err := rb.Get()
	Nil(t, err)
	Equal(t, "foo", item)
	item, err = rb.Get()
	Nil(t, err)
	Equal(t, "bar", item)
}

func TestRingGetEmpty(t *testing.T) {
	rb := NewRingBuffer(3)

	var wg sync.WaitGroup
	wg.Add(1)

	// want to kick off this consumer to ensure it blocks
	go func() {
		wg.Done()
		result, err := rb.Get()
		Nil(t, err)
		Equal(t, 0, result)
		wg.Done()
	}()

	wg.Wait()
	wg.Add(2)

	go func() {
		defer wg.Done()
		err := rb.Put(0)
		Nil(t, err)
	}()

	wg.Wait()
}

func TestRingPollEmpty(t *testing.T) {
	rb := NewRingBuffer(3)

	_, err := rb.Poll(1)
	Equal(t, ErrTimeout, err)
}

func TestRingPoll(t *testing.T) {
	rb := NewRingBuffer(10)

	// should be able to Poll() before anything is present, without breaking future Puts
	rb.Poll(time.Millisecond)

	rb.Put(`test`)
	result, err := rb.Poll(0)
	if !Nil(t, err) {
		return
	}

	Equal(t, `test`, result)
	Equal(t, uint64(0), rb.Len())

	rb.Put(`1`)
	rb.Put(`2`)

	result, err = rb.Poll(time.Millisecond)
	if !Nil(t, err) {
		return
	}

	Equal(t, `1`, result)
	Equal(t, uint64(1), rb.Len())

	result, err = rb.Poll(time.Millisecond)
	if !Nil(t, err) {
		return
	}

	Equal(t, `2`, result)

	before := time.Now()
	_, err = rb.Poll(5 * time.Millisecond)
	if ed := time.Since(before); ed > time.Millisecond*10 {
		t.Errorf("poll wait to long: %v", ed)
	}
	Equal(t, ErrTimeout, err)
}

func TestRingLen(t *testing.T) {
	rb := NewRingBuffer(4)
	Equal(t, uint64(0), rb.Len())

	rb.Put(1)
	Equal(t, uint64(1), rb.Len())

	rb.Get()
	Equal(t, uint64(0), rb.Len())

	for i := 0; i < 4; i++ {
		rb.Put(1)
	}
	Equal(t, uint64(4), rb.Len())

	rb.Get()
	Equal(t, uint64(3), rb.Len())
}

func TestDisposeOnGet(t *testing.T) {
	numThreads := 8
	var wg sync.WaitGroup
	wg.Add(numThreads)
	rb := NewRingBuffer(4)
	var spunUp sync.WaitGroup
	spunUp.Add(numThreads)

	for i := 0; i < numThreads; i++ {
		go func() {
			spunUp.Done()
			defer wg.Done()
			_, err := rb.Get()
			NotNil(t, err)
		}()
	}

	spunUp.Wait()
	rb.Dispose()

	wg.Wait()
	True(t, rb.IsDisposed())
}

func TestDisposeOnPut(t *testing.T) {
	numThreads := 8
	var wg sync.WaitGroup
	wg.Add(numThreads)
	rb := NewRingBuffer(4)
	var spunUp sync.WaitGroup
	spunUp.Add(numThreads)

	// fill up the queue
	for i := 0; i < 4; i++ {
		rb.Put(i)
	}

	// it's now full
	for i := 0; i < numThreads; i++ {
		go func(i int) {
			spunUp.Done()
			defer wg.Done()
			err := rb.Put(i)
			NotNil(t, err)
		}(i)
	}

	spunUp.Wait()

	rb.Dispose()

	wg.Wait()

	True(t, rb.IsDisposed())
}

func BenchmarkRBLifeCycle(b *testing.B) {
	rb := NewRingBuffer(64)

	counter := uint64(0)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			_, err := rb.Get()
			Nil(b, err)

			if atomic.AddUint64(&counter, 1) == uint64(b.N) {
				return
			}
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rb.Put(i)
	}

	wg.Wait()
}

func BenchmarkRBLifeCycleContention(b *testing.B) {
	rb := NewRingBuffer(64)

	var wwg sync.WaitGroup
	var rwg sync.WaitGroup
	wwg.Add(10)
	rwg.Add(10)

	for i := 0; i < 10; i++ {
		go func() {
			for {
				_, err := rb.Get()
				if err == ErrDisposed {
					rwg.Done()
					return
				} else {
					Nil(b, err)
				}
			}
		}()
	}

	b.ResetTimer()

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < b.N; j++ {
				rb.Put(j)
			}
			wwg.Done()
		}()
	}

	wwg.Wait()
	rb.Dispose()
	rwg.Wait()
}

func BenchmarkRBPut(b *testing.B) {
	rb := NewRingBuffer(uint64(b.N))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := rb.Offer(i)
		if err != nil {
			b.Log(err)
			b.Fail()
		}
	}
}

func BenchmarkRBGet(b *testing.B) {
	rb := NewRingBuffer(uint64(b.N))

	for i := 0; i < b.N; i++ {
		rb.Offer(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		rb.Get()
	}
}

func BenchmarkRBAllocation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		NewRingBuffer(1024)
	}
}
