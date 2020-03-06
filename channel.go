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
	"io"
	"net"
	"runtime/debug"
	"sync/atomic"

	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/utils"
)

// Channel is defines a server-side-channel & client-side-channel
type Channel interface {
	// Channel id
	ID() int64

	// reader & writer & closer
	io.WriteCloser

	// Writev to write [][]byte for optimize syscall
	Writev([][]byte) (int64, error)

	// Local address
	LocalAddr() string

	// Remote address
	RemoteAddr() string

	// Transport
	Transport() transport.Transport

	// Pipeline
	Pipeline() Pipeline

	// Get attachment
	Attachment() Attachment

	// Set attachment
	SetAttachment(Attachment)

	// Channel context
	Context() context.Context

	// Start send & write routines.
	serveChannel()
}

// NewChannel create a ChannelFactory
func NewChannel(capacity int) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport) Channel {
		return newChannelWith(ctx, pipeline, transport, id, capacity)
	}
}

// NewBufferedChannel create a ChannelFactory with buffered transport
func NewBufferedChannel(capacity int, sizeRead int) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, tran transport.Transport) Channel {
		tran = transport.BufferedTransport(tran, sizeRead)
		return newChannelWith(ctx, pipeline, tran, id, capacity)
	}
}

// newChannelWith internal method for NewChannel & NewBufferedChannel
func newChannelWith(ctx context.Context, pipeline Pipeline, transport transport.Transport, id int64, capacity int) Channel {
	childCtx, cancel := context.WithCancel(ctx)
	return &channel{
		id:        id,
		ctx:       childCtx,
		cancel:    cancel,
		pipeline:  pipeline,
		transport: transport,
		sendQueue: make(chan [][]byte, capacity),
	}
}

// implement of Channel
type channel struct {
	id             int64
	ctx            context.Context
	cancel         context.CancelFunc
	transport      transport.Transport
	pipeline       Pipeline
	attachment     Attachment
	sendQueue      chan [][]byte
	closed, posted int32
}

// Get channel id
func (c *channel) ID() int64 {
	return c.id
}

// Write bytes to the channel
func (c *channel) Write(p []byte) (n int, err error) {

	select {
	case <-c.ctx.Done():
		return 0, errors.New("broken pipe")
	case c.sendQueue <- [][]byte{p}:
		return len(p), nil
	}
}

// Writev to write [][]byte for optimize syscall
func (c *channel) Writev(p [][]byte) (n int64, err error) {

	select {
	case <-c.ctx.Done():
		return 0, errors.New("broken pipe")
	case c.sendQueue <- p:
		for _, d := range p {
			n += int64(len(d))
		}
		return
	}
}

// Close the channel
func (c *channel) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.cancel()
		return c.transport.Close()
	}
	return nil
}

// Get transport of channel
func (c *channel) Transport() transport.Transport {
	return c.transport
}

// Get pipeline of channel
func (c *channel) Pipeline() Pipeline {
	return c.pipeline
}

// Get local address of channel
func (c *channel) LocalAddr() string {
	return c.transport.LocalAddr().String()
}

// Get remote address of channel
func (c *channel) RemoteAddr() string {
	return c.transport.RemoteAddr().String()
}

// Get attachment of channel
func (c *channel) Attachment() Attachment {
	return c.attachment
}

// Set attachment of channel
func (c *channel) SetAttachment(v Attachment) {
	c.attachment = v
}

// Get context of channel
func (c *channel) Context() context.Context {
	return c.ctx
}

// start write & read routines
func (c *channel) serveChannel() {
	go c.writeLoop()
	go c.readLoop()
}

// reading message of channel
func (c *channel) readLoop() {

	defer func() {
		c.postCloseEvent(AsException(recover(), debug.Stack()))
	}()

	for {
		// 循环读取数据，直到连接关闭
		c.pipeline.fireChannelRead(c.transport)
	}
}

// sending message of channel
func (c *channel) writeLoop() {

	defer func() {
		if err := recover(); nil != err {
			c.postCloseEvent(AsException(err, debug.Stack()))
		}
	}()

	var BufferCap = cap(c.sendQueue)
	var buffers = make(net.Buffers, 0, BufferCap)
	var indexes = make([]int, 0, BufferCap)

	// Try to combine packet sending to optimize sending performance
	sendWithWritev := func(data [][]byte, queue <-chan [][]byte) (int64, error) {

		// reuse buffer.
		buffers = buffers[:0]
		indexes = indexes[:0]

		// append first packet.
		buffers = append(buffers, data...)
		indexes = append(indexes, len(buffers))

		// more packet will be merged.
		for {
			select {
			case data := <-queue:
				buffers = append(buffers, data...)
				indexes = append(indexes, len(buffers))
				// 合并到一定数量的buffer之后直接发送，防止无限撑大buffer
				// 最大一次合并发送的size由sendQueue的cap来决定
				if len(indexes) >= BufferCap {
					return c.transport.Writev(transport.Buffers{Buffers: buffers, Indexes: indexes})
				}
			default:
				return c.transport.Writev(transport.Buffers{Buffers: buffers, Indexes: indexes})
			}
		}
	}

	for {
		select {
		case buf := <-c.sendQueue:
			// combine send bytes to reduce syscall.
			utils.AssertLong(sendWithWritev(buf, c.sendQueue))
			// flush buffer
			utils.Assert(c.transport.Flush())
		case <-c.ctx.Done():
			return
		}
	}
}

// once post the exception
func (c *channel) postCloseEvent(ex Exception) {
	if atomic.CompareAndSwapInt32(&c.posted, 0, 1) {
		// 主动关闭的不需要触发异常流程
		// 非主动关闭需要投递异常事件
		if 0 == atomic.LoadInt32(&c.closed) {
			c.pipeline.fireChannelException(ex)
		}
	}
}
