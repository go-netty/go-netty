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
	"time"

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

	// CtxWrite1 channels with asynchronous write enabled, writes will block until the write is successfully sent to the queue or times out.
	// for synchronous write channels, SetWriteDeadline will be called to ensure that the blocking write operation is interrupted after a timeout.
	CtxWrite1(ctx context.Context, p []byte) (n int, err error)

	// CtxWritev channels with asynchronous write enabled, writes will block until the write is successfully sent to the queue or times out.
	// for synchronous write channels, SetWriteDeadline will be called to ensure that the blocking write operation is interrupted after a timeout.
	CtxWritev(ctx context.Context, pv [][]byte) (n int64, err error)

	// ReadFrom reads data from r until EOF or error.
	// The return value n is the number of bytes read.
	ReadFrom(r io.Reader) (n int64, err error)

	// Writer return the channel writer.
	Writer() io.Writer

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
func NewAsyncWriteChannel(writeQueueSize int, untilWrite bool) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor) Channel {
		return newChannelWith(ctx, pipeline, transport, executor, id, writeQueueSize, untilWrite)
	}
}

// newChannelWith internal method for NewChannel & NewBufferedChannel
func newChannelWith(ctx context.Context, pipeline Pipeline, transport transport.Transport, executor Executor, id int64, writeQueueSize int, untilWrite bool) Channel {
	childCtx, cancel := context.WithCancel(ctx)

	var (
		writeQueue     chan []byte
		writeBuffers   net.Buffers
		recycleBuffers net.Buffers
	)

	// enable async write
	if writeQueueSize > 0 {
		writeQueue = make(chan []byte, writeQueueSize)
		writeBuffers = make(net.Buffers, 0, writeQueueSize/2+1)
		recycleBuffers = make(net.Buffers, 0, writeQueueSize/2+1)
	}

	return &channel{
		id:             id,
		ctx:            childCtx,
		cancel:         cancel,
		pipeline:       pipeline,
		transport:      transport,
		executor:       executor,
		writeQueue:     writeQueue,
		recycleBuffers: recycleBuffers,
		writeBuffers:   writeBuffers,
		untilWrite:     untilWrite,
	}
}

const idle = 0
const running = 1

// implement of Channel
type channel struct {
	id             int64
	ctx            context.Context
	cancel         context.CancelFunc
	transport      transport.Transport
	executor       Executor
	pipeline       Pipeline
	attachment     Attachment
	writeQueue     chan []byte
	recycleBuffers net.Buffers
	writeBuffers   net.Buffers
	untilWrite     bool
	closed         int32
	running        int32
	closeErr       error
	writeLock      sync.Mutex // for sync write
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

		// wait async send finished.
		if nil != c.writeQueue {
			var maxWaitNum int
			for (c.untilWrite || maxWaitNum < 10) && atomic.LoadInt32(&c.running) != idle {
				maxWaitNum++
				time.Sleep(time.Millisecond * 100)
			}
		}

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
	if nil != c.closeErr {
		return 0, c.closeErr
	}

	// enable async write
	if nil != c.writeQueue {
		return c.asyncWritev(context.Background(), p)
	}

	// sync write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	if n, err = c.transport.Writev(p); nil == err {
		err = c.transport.Flush()
	}
	return
}

// Write1 to write []byte to channel
func (c *channel) Write1(p []byte) (n int, err error) {
	return c.write1(p, true)
}

// CtxWrite1 channels with asynchronous write enabled, writes will block until the write is successfully sent to the queue or times out.
// for synchronous write channels, SetDeadline will be called to ensure that the blocking write operation is interrupted after a timeout.
func (c *channel) CtxWrite1(ctx context.Context, p []byte) (n int, err error) {
	// enable async write
	if nil != c.writeQueue {
		wn, err := c.asyncWrite(ctx, p, true)
		return int(wn), err
	}

	// sync write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		if err = c.transport.SetWriteDeadline(deadline); nil != err {
			return
		}
		// reset write deadline
		defer c.transport.SetWriteDeadline(time.Time{})
	}

	if n, err = c.transport.Write(p); nil == err {
		err = c.transport.Flush()
	}
	return
}

// CtxWritev channels with asynchronous write enabled, writes will block until the write is successfully sent to the queue or times out.
// for synchronous write channels, SetDeadline will be called to ensure that the blocking write operation is interrupted after a timeout.
func (c *channel) CtxWritev(ctx context.Context, pv [][]byte) (n int64, err error) {
	// enable async write
	if nil != c.writeQueue {
		wn, err := c.asyncWritev(ctx, pv)
		return wn, err
	}

	// sync write
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	if deadline, ok := ctx.Deadline(); ok {
		if err = c.transport.SetWriteDeadline(deadline); nil != err {
			return
		}
		// reset write deadline
		defer c.transport.SetWriteDeadline(time.Time{})
	}

	if n, err = c.transport.Writev(pv); nil == err {
		err = c.transport.Flush()
	}
	return
}

// ReadFrom reads data from r until EOF or error.
// The return value n is the number of bytes read.
func (c *channel) ReadFrom(r io.Reader) (n int64, err error) {
	if nil != c.closeErr {
		return 0, c.closeErr
	}

	const MinRead = 1024

	for {
		dataBuff := *pbytes.Get(MinRead)
		dataBuff = dataBuff[:MinRead]

		rn, rerr := r.Read(dataBuff)
		if rn < 0 {
			panic("reader returned negative count from Read")
		}
		n += int64(rn)
		if rn > 0 {
			wn, werr := c.write1(dataBuff[:rn], false)
			if nil != werr {
				return n, werr
			}
			if wn != rn {
				return n, io.ErrShortWrite
			}
		} else {
			dataBuff = dataBuff[:0]
			pbytes.Put(&dataBuff)
		}

		if rerr != nil {
			if rerr == io.EOF {
				return n, nil
			}
			return n, rerr
		}
	}
}

func (c *channel) Writer() io.Writer {
	return channelWriter{c}
}

func (c *channel) write1(p []byte, clone bool) (n int, err error) {
	if nil != c.closeErr {
		return 0, c.closeErr
	}

	// enable async write
	if nil != c.writeQueue {
		wn, err := c.asyncWrite(context.Background(), p, clone)
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

func (c *channel) asyncWrite(ctx context.Context, p []byte, clone bool) (int64, error) {
	// count of data length
	dataLen := len(p)

	if clone {
		dataBuff := *pbytes.Get(dataLen)
		dataBuff = dataBuff[:dataLen]

		if n := copy(dataBuff, p); n != dataLen {
			return 0, fmt.Errorf("%w: want: %d, got: %d", io.ErrShortWrite, len(p), n)
		}

		p = dataBuff
	}

	// put packet to send queue
	var packet = p

	if c.untilWrite {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-c.ctx.Done():
			return 0, c.closeErr
		case c.writeQueue <- packet:
			// write queue
		}
	} else {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-c.ctx.Done():
			return 0, c.closeErr
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
	return int64(dataLen), nil
}

func (c *channel) asyncWritev(ctx context.Context, p [][]byte) (int64, error) {
	// count of data length
	dataLen := utils.CountOf(p)

	// get buffer from asyncWrite
	// put buffer from writeOnce
	dataBuff := *pbytes.Get(int(dataLen))
	dataBuff = dataBuff[:0]
	offset := 0
	for _, b := range p {
		if cn := copy(dataBuff[offset:cap(dataBuff)], b); cn != len(b) {
			return 0, fmt.Errorf("%w: want: %d, got: %d", io.ErrShortWrite, len(b), cn)
		} else {
			offset += cn
		}
	}

	// put packet to send queue
	var packet = dataBuff[:offset]

	if c.untilWrite {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-c.ctx.Done():
			return 0, c.closeErr
		case c.writeQueue <- packet:
			// write queue
		}
	} else {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-c.ctx.Done():
			return 0, c.closeErr
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
			atomic.StoreInt32(&c.running, idle)
			c.Close(AsException(err))
		}
	}()

	for {
		// reuse buffer.
		sendBuffers := c.writeBuffers[:0]
		recycleBuffers := c.recycleBuffers[:0]

		// more packet will be merged
		for len(sendBuffers) < cap(sendBuffers) {
			// poll packet
			select {
			case pkt := <-c.writeQueue:
				// combine send bytes to reduce syscall.
				sendBuffers = append(sendBuffers, pkt)
				recycleBuffers = append(recycleBuffers, pkt)
				continue
			default:
			}

			// no more packet to write
			break
		}

		if len(sendBuffers) > 0 {

			utils.AssertLong(c.transport.Writev(sendBuffers))

			// clear buffer ref
			for index, buf := range recycleBuffers {
				// reuse buffer
				buf := buf[:0]
				pbytes.Put(&buf)
				// avoid memory leak
				sendBuffers[index] = nil
				recycleBuffers[index] = nil
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

type channelWriter struct {
	channel Channel
}

func (c channelWriter) Write(p []byte) (n int, err error) {
	return c.channel.Write1(p)
}
