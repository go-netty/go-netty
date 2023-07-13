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
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/utils"
	"github.com/go-netty/go-netty/utils/pool/pbytes"
)

// ErrAsyncNoSpace is returned when an write queue full if not writeForever flags.
var ErrAsyncNoSpace = errors.New("async write queue is full")

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

	// Write1 to write []byte to channel
	Write1(p []byte) (n int, err error)

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
		return newChannelWith(ctx, pipeline, transport, executor, id, 0, false)
	}
}

// NewAsyncWriteChannel create an async write ChannelFactory.
func NewAsyncWriteChannel(writeQueueSize int, writeForever bool) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor) Channel {
		return newChannelWith(ctx, pipeline, transport, executor, id, writeQueueSize, writeForever)
	}
}

// newChannelWith internal method for NewChannel & NewBufferedChannel
func newChannelWith(ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor, id int64, writeQueueSize int, writeForever bool) Channel {
	childCtx, cancel := context.WithCancel(ctx)

	var (
		writeQueue   chan [][]byte
		writeBuffers net.Buffers
		writeIndexes []int
	)

	// enable async write
	if writeQueueSize > 0 {
		writeQueue = make(chan [][]byte, writeQueueSize)
		writeBuffers = make(net.Buffers, 0, (writeQueueSize/5)*2+1)
		writeIndexes = make([]int, 0, writeQueueSize/5+1)
	}

	return &channel{
		id:           id,
		ctx:          childCtx,
		cancel:       cancel,
		pipeline:     pipeline,
		transport:    transport,
		executor:     executor,
		writeQueue:   writeQueue,
		writeBuffers: writeBuffers,
		writeIndexes: writeIndexes,
		writeForever: writeForever,
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
	writeQueue   chan [][]byte
	writeBuffers net.Buffers
	writeIndexes []int
	writeForever bool
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
	if nil != c.writeQueue {
		return c.asyncWrite(p)
	}

	// sync write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	if n, err = c.transport.Writev(transport.Buffers{Buffers: p, Indexes: []int{len(p)}}); nil == err {
		err = c.transport.Flush()
	}
	return
}

// Write1 to write []byte to channel
func (c *channel) Write1(p []byte) (n int, err error) {
	if !c.IsActive() {
		select {
		case <-c.ctx.Done():
			return 0, c.closeErr
		}
	}

	// enable async write
	if nil != c.writeQueue {
		wn, err := c.asyncWrite([][]byte{p})
		return int(wn), err
	}

	// sync write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	if n, err = c.transport.Write(p); nil == err {
		err = c.transport.Flush()
	}
	return
}

func (c *channel) asyncWrite(p [][]byte) (int64, error) {
	// count of data length
	dataLen := utils.CountOf(p)

	// get buffer from asyncWrite
	// put buffer from writeOnce
	dataBuff := pbytes.GetLen(int(dataLen))
	offset := 0
	for _, b := range p {
		if cn := copy(dataBuff[offset:], b); cn != len(b) {
			panic(fmt.Errorf("%w: want: %d, got: %d", io.ErrShortWrite, len(b), cn))
		} else {
			offset += cn
		}
	}

	// put packet to send queue
	var packet = [][]byte{dataBuff}

	if c.writeForever {
		select {
		case <-c.ctx.Done():
			return 0, c.ctx.Err()
		case c.writeQueue <- packet:
			// write queue
		}
	} else {
		select {
		case <-c.ctx.Done():
			return 0, c.ctx.Err()
		case c.writeQueue <- packet:
			// write queue
		default:
			return 0, ErrAsyncNoSpace
		}
	}

	// try send
	if atomic.CompareAndSwapInt32(&c.running, idle, running) {
		c.executor.Exec(c.writeOnce)
	}
	return dataLen, nil
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
	signal := make(chan struct{})
	defer func() { <-signal }()

	c.executor.Exec(func() {
		c.readLoop(func() {
			close(signal)
		})
	})
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
func (c *channel) readLoop(done func()) {

	defer func() {
		c.Close(AsException(recover()))
	}()

	func() {
		defer done()
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
		for len(sendBuffers) < cap(c.writeQueue) {
			// poll packet
			select {
			case pkts := <-c.writeQueue:
				// combine send bytes to reduce syscall.
				sendBuffers = append(sendBuffers, pkts...)
				sendIndexes = append(sendIndexes, len(sendBuffers))
				continue
			default:
			}

			// no more packet to write
			break
		}

		if len(sendBuffers) > 0 {
			utils.AssertLong(c.transport.Writev(transport.Buffers{Buffers: sendBuffers, Indexes: sendIndexes}))

			// clear buffer ref
			for index, buf := range sendBuffers {
				// reuse buffer
				pbytes.Put(buf)
				// avoid memory leak
				sendBuffers[index] = nil
				// for safety
				if index < len(sendIndexes) {
					sendIndexes[index] = -1
				}
			}

			// continue to send remain packets
			if len(c.writeQueue) > 0 {
				continue
			}
		}

		// flush transport buffer
		utils.Assert(c.transport.Flush())

		// double check
		atomic.StoreInt32(&c.running, idle)
		if size := len(c.writeQueue); size > 0 {
			if atomic.CompareAndSwapInt32(&c.running, idle, running) {
				continue
			}
		}

		// no packets to send
		break
	}
}
