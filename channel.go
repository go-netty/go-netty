/*
 * Copyright 2019 the go-netty project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package netty

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/utils"
)

// Channel is defines a server-side-channel & client-side-channel
type Channel interface {
	// ID channel id
	ID() int64

	// Write a message through the Pipeline
	Write(Message) error

	// Trigger user event
	Trigger(event Event)

	// Close through the Pipeline
	Close(err error)

	// IsActive return true if the Channel is active and so connected
	IsActive() bool

	// Writev to write [][]byte for optimize syscall
	Writev([][]byte) (int64, error)

	// LocalAddr local address
	LocalAddr() string

	// RemoteAddr remote address
	RemoteAddr() string

	// Transport get transport of channel
	Transport() transport.Transport

	// Pipeline get pipeline of channel
	Pipeline() Pipeline

	// Attachment get attachment
	Attachment() Attachment

	// SetAttachment set attachment
	SetAttachment(Attachment)

	// Context channel context
	Context() context.Context

	// Start send & write routines.
	serveChannel()
}

// NewChannel create a ChannelFactory
func NewChannel() ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor) Channel {
		return newChannelWith(ctx, pipeline, transport, executor, id, 0)
	}
}

// NewAsyncWriteChannel create an async write ChannelFactory
func NewAsyncWriteChannel(capacity int) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor) Channel {
		return newChannelWith(ctx, pipeline, transport, executor, id, capacity)
	}
}

// newChannelWith internal method for NewChannel & NewBufferedChannel
func newChannelWith(ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor, id int64, capacity int) Channel {
	childCtx, cancel := context.WithCancel(ctx)

	var (
		sendQueue    *utils.RingBuffer
		writeBuffers net.Buffers
		writeIndexes []int
	)

	// enable async write
	if capacity > 0 {
		sendQueue = utils.NewRingBuffer(uint64(capacity))
		writeBuffers = make(net.Buffers, 0, (capacity/5)*2+1)
		writeIndexes = make([]int, 0, capacity/5+1)
	}

	return &channel{
		id:           id,
		ctx:          childCtx,
		cancel:       cancel,
		pipeline:     pipeline,
		transport:    transport,
		executor:     executor,
		sendQueue:    sendQueue,
		writeBuffers: writeBuffers,
		writeIndexes: writeIndexes,
	}
}

const idle = 0
const running = 1

// implement of Channel
type channel struct {
	id           int64
	ctx          context.Context
	cancel       context.CancelFunc
	transport    transport.Transport
	executor     Executor
	pipeline     Pipeline
	attachment   Attachment
	sendQueue    *utils.RingBuffer
	writeBuffers net.Buffers
	writeIndexes []int
	activeWait   sync.WaitGroup
	closed       int32
	running      int32
	closeErr     error
	writeLock    sync.Mutex // for sync write
}

// ID get channel id
func (c *channel) ID() int64 {
	return c.id
}

// Write a message through the Pipeline
func (c *channel) Write(message Message) error {
	if !c.IsActive() {
		select {
		case <-c.ctx.Done():
			return c.closeErr
		}
	}

	c.invokeMethod(func() {
		c.pipeline.FireChannelWrite(message)
	})
	return nil
}

// Trigger trigger event
func (c *channel) Trigger(event Event) {
	c.invokeMethod(func() {
		c.pipeline.FireChannelEvent(event)
	})
}

// Close through the Pipeline
func (c *channel) Close(err error) {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.closeErr = err
		c.transport.Close()
		c.cancel()

		if nil != c.sendQueue {
			c.sendQueue.Dispose()
		}

		c.invokeMethod(func() {
			c.pipeline.FireChannelInactive(err)
		})
	}
}

// Writev to write [][]byte for optimize syscall
func (c *channel) Writev(p [][]byte) (n int64, err error) {
	if !c.IsActive() {
		select {
		case <-c.ctx.Done():
			return 0, c.closeErr
		}
	}

	// enable async write
	if nil != c.sendQueue {
		// put packet to send queue
		if err = c.sendQueue.Put(p); nil != err {
			return 0, err
		}
		// try send
		if atomic.CompareAndSwapInt32(&c.running, idle, running) {
			c.executor.Exec(c.writeOnce)
		}
		return utils.CountOf(p), nil
	}

	// sync write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	if n, err = c.transport.Writev(transport.Buffers{Buffers: p, Indexes: []int{len(p)}}); nil == err {
		err = c.transport.Flush()
	}
	return
}

// IsActive return true if the Channel is active and so connected
func (c *channel) IsActive() bool {
	return 0 == atomic.LoadInt32(&c.closed)
}

// Transport get transport of channel
func (c *channel) Transport() transport.Transport {
	return c.transport
}

// Pipeline get pipeline of channel
func (c *channel) Pipeline() Pipeline {
	return c.pipeline
}

// LocalAddr get local address of channel
func (c *channel) LocalAddr() string {
	return c.transport.LocalAddr().String()
}

// RemoteAddr get remote address of channel
func (c *channel) RemoteAddr() string {
	return c.transport.RemoteAddr().String()
}

// Attachment get attachment of channel
func (c *channel) Attachment() Attachment {
	return c.attachment
}

// SetAttachment set attachment of channel
func (c *channel) SetAttachment(v Attachment) {
	c.attachment = v
}

// Context get context of channel
func (c *channel) Context() context.Context {
	return c.ctx
}

// serveChannel start write & read routines
func (c *channel) serveChannel() {
	c.activeWait.Add(1)
	c.executor.Exec(c.readLoop)
	c.activeWait.Wait()
}

func (c *channel) invokeMethod(fn func()) {

	defer func() {
		if err := recover(); nil != err && 0 == atomic.LoadInt32(&c.closed) {
			c.pipeline.FireChannelException(AsException(err))

			if e, ok := err.(error); ok {
				var ne net.Error
				if errors.As(e, &ne) && !ne.Timeout() {
					c.Close(e)
				}
			}
		}
	}()

	fn()
}

// readLoop reading message of channel
func (c *channel) readLoop() {

	defer func() {
		c.Close(AsException(recover()))
	}()

	func() {
		defer c.activeWait.Done()
		c.invokeMethod(c.pipeline.FireChannelActive)
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.invokeMethod(func() {
				c.pipeline.FireChannelRead(c.transport)
			})
		}
	}
}

// writeOnce sending messages of channel
func (c *channel) writeOnce() {

	defer func() {
		if err := recover(); nil != err {
			c.Close(AsException(err))
		}
	}()

	for {
		// reuse buffer.
		sendBuffers := c.writeBuffers[:0]
		sendIndexes := c.writeIndexes[:0]

		// more packet will be merged
		for c.sendQueue.Len() > 0 && len(sendBuffers) < cap(sendBuffers) {
			// poll packet
			item, err := c.sendQueue.Poll(-1)
			if nil != err {
				break
			}

			// combine send bytes to reduce syscall.
			pkts := item.([][]byte)
			sendBuffers = append(sendBuffers, pkts...)
			sendIndexes = append(sendIndexes, len(sendBuffers))
		}

		if len(sendBuffers) > 0 {
			utils.AssertLong(c.transport.Writev(transport.Buffers{Buffers: sendBuffers, Indexes: sendIndexes}))
			utils.Assert(c.transport.Flush())

			// clear buffer ref
			for index := range sendBuffers {
				sendBuffers[index] = nil // avoid memory leak
				if index < len(sendIndexes) {
					sendIndexes[index] = -1 // for safety
				}
			}
		}

		// double check
		atomic.StoreInt32(&c.running, idle)
		if size := c.sendQueue.Len(); size > 0 {
			if atomic.CompareAndSwapInt32(&c.running, idle, running) {
				continue
			}
		}

		// no packets to send
		break
	}
}
