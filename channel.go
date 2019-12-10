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

// Channel
type Channel interface {
	// channel id.
	Id() int64

	// reader & writer & closer
	io.WriteCloser

	// write vector
	Writev([][]byte) (int64, error)

	// local address.
	LocalAddr() string

	// remote address.
	RemoteAddr() string

	// transport
	Transport() transport.Transport

	// pipeline
	Pipeline() Pipeline

	// get attachment.
	Attachment() Attachment

	// set attachment.
	SetAttachment(Attachment)

	// channel context.
	Context() context.Context

	// start read & write message routines.
	serveChannel()
}

// NewChannel
func NewChannel(capacity int) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport) Channel {
		return newChannelWith(id, ctx, pipeline, transport, capacity)
	}
}

// NewBufferedChannel
func NewBufferedChannel(capacity int, sizeRead int) ChannelFactory {
	return func(id int64, ctx context.Context, pipeline Pipeline, tran transport.Transport) Channel {
		tran = transport.BufferedTransport(tran, sizeRead)
		return newChannelWith(id, ctx, pipeline, tran, capacity)
	}
}

func newChannelWith(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport, capacity int) Channel {
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

type channel struct {
	id         int64
	ctx        context.Context
	cancel     context.CancelFunc
	transport  transport.Transport
	pipeline   Pipeline
	sendQueue  chan [][]byte
	closed     int32
	posted     int32
	attachment Attachment
}

func (c *channel) Id() int64 {
	return c.id
}

func (c *channel) Write(p []byte) (n int, err error) {

	select {
	case <-c.ctx.Done():
		return 0, errors.New("broken pipe")
	case c.sendQueue <- [][]byte{p}:
		return len(p), nil
	}
}

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

func (c *channel) Close() error {
	if atomic.CompareAndSwapInt32(&c.closed, 0, 1) {
		c.cancel()
		return c.transport.Close()
	}
	return nil
}

func (c *channel) Transport() transport.Transport {
	return c.transport
}

func (c *channel) Pipeline() Pipeline {
	return c.pipeline
}

func (c *channel) LocalAddr() string {
	return c.transport.LocalAddr().String()
}

func (c *channel) RemoteAddr() string {
	return c.transport.RemoteAddr().String()
}

func (c *channel) Attachment() Attachment {
	return c.attachment
}

func (c *channel) SetAttachment(v Attachment) {
	c.attachment = v
}

func (c *channel) Context() context.Context {
	return c.ctx
}

func (c *channel) serveChannel() {
	go c.writeLoop()
	go c.readLoop()
}

func (c *channel) readLoop() {

	defer func() {
		c.postCloseEvent(AsException(recover(), debug.Stack()))
	}()

	for {
		// 循环读取数据，直到连接关闭
		c.pipeline.fireChannelRead(c.transport)
	}
}

func (c *channel) writeLoop() {

	defer func() {
		if err := recover(); nil != err {
			c.postCloseEvent(AsException(err, debug.Stack()))
		}
	}()

	// 复用buff
	const BufferCap = 64
	var buffers = make(net.Buffers, 0, BufferCap)
	var indexes = make([]int, 0, BufferCap)

	// 尽量一次性发送多个数据
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
				// 最大一次合并发送的size由buffer的cap来决定
				// 这里会影响吞吐，先屏蔽掉
				// if len(buffers) >= BufferCap {
				//	return c.transport.Writev(&buffers)
				// }
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

func (c *channel) postCloseEvent(ex Exception) {
	// 防止错误事件重复投递
	if atomic.CompareAndSwapInt32(&c.posted, 0, 1) {
		// 非主动关闭需要投递异常事件
		if 0 == atomic.LoadInt32(&c.closed) {
			c.pipeline.fireChannelException(ex)
		}
	}
}
