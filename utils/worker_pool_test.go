/*
 *  Copyright 2020 the go-netty project
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package utils

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestWorkerPool_AddTask(t *testing.T) {
	var counter = int32(100)
	pool := NewWorkerPool(context.Background(), 10, 1, 10)
	func(n int32) {
		for i := int32(0); i < n; i++ {
			pool.AddTask(func() {
				atomic.AddInt32(&counter, -1)
			})
		}
	}(counter)

	pool.StopWait(0)

	if val := atomic.LoadInt32(&counter); val != 0 {
		t.Fatalf("unexpecteded counter: %d", val)
	}
}

func TestWorkerPool_RunTask(t *testing.T) {
	var counter = int32(100)
	pool := NewWorkerPool(context.Background(), 10, 1, 10)
	func(n int32) {
		for i := int32(0); i < n; i++ {
			pool.RunTask(func() {
				atomic.AddInt32(&counter, -1)
			})
		}
	}(counter)

	if val := atomic.LoadInt32(&counter); val != 0 {
		t.Fatalf("unexpecteded counter: %d", val)
	}
}

func TestWorkerPool_StopWait(t *testing.T) {
	var counter = int32(2)
	pool := NewWorkerPool(context.Background(), 10, 0, 0)
	pool.AddTask(func() {
		atomic.AddInt32(&counter, -1)
	})
	pool.RunTask(func() {
		atomic.AddInt32(&counter, -1)
	})
	pool.StopWait(time.Second)

	if val := atomic.LoadInt32(&counter); val != 0 {
		t.Fatalf("unexpecteded counter: %d", val)
	}
}

func TestWorkerPool_Stop(t *testing.T) {
	var counter = int32(100)
	pool := NewWorkerPool(context.Background(), 10, 1, 10)
	func(n int32) {
		for i := int32(0); i < n; i++ {
			pool.AddTask(func() {
				atomic.AddInt32(&counter, -1)
			})
		}
	}(counter)

	time.Sleep(time.Second * 6)
	rp := pool.(*workerPool)
	if rp.idleWorkers != len(rp.readyWorkers) {
		t.Fatalf("unexpecteded idle workers: %d != %d", rp.idleWorkers, len(rp.readyWorkers))
	}

	if val := atomic.LoadInt32(&counter); val != 0 {
		t.Fatalf("unexpecteded counter: %d", val)
	}
}
