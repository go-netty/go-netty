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
	"fmt"
	"io"
	"io/ioutil"
	"runtime/debug"
	"sync/atomic"
	"time"

	"github.com/go-netty/go-netty/utils"
)

type (
	// Message defines an any message
	//
	// Represent different objects in different processing steps,
	// in most cases the message type handled by the codec is mainly io.Reader / []byte,
	// in the user handler should have been converted to a protocol object.
	Message interface {
	}

	// Event defines user-defined event types, read-write timeout events, etc
	Event interface {
	}

	// Attachment defines the object or data associated with the Channel
	Attachment interface {
	}

	// Handler defines an any handler
	// At least one or more of the following types should be implemented
	// ActiveHandler
	// InboundHandler
	// OutboundHandler
	// ExceptionHandler
	// InactiveHandler
	// EventHandler
	Handler interface {
	}

	// ActiveHandler defines an active handler
	ActiveHandler interface {
		HandleActive(ctx ActiveContext)
	}

	// InboundHandler defines an Inbound handler
	InboundHandler interface {
		HandleRead(ctx InboundContext, message Message)
	}

	// OutboundHandler defines an outbound handler
	OutboundHandler interface {
		HandleWrite(ctx OutboundContext, message Message)
	}

	// ExceptionHandler defines an exception handler
	ExceptionHandler interface {
		HandleException(ctx ExceptionContext, ex Exception)
	}

	// InactiveHandler defines an inactive handler
	InactiveHandler interface {
		HandleInactive(ctx InactiveContext, ex Exception)
	}

	// EventHandler defines an event handler
	EventHandler interface {
		HandleEvent(ctx EventContext, event Event)
	}
)

// CodecHandler defines an codec handler
type CodecHandler interface {
	CodecName() string
	InboundHandler
	OutboundHandler
}

// ChannelHandler defines a channel handler
type ChannelHandler interface {
	ActiveHandler
	InboundHandler
	OutboundHandler
	ExceptionHandler
	InactiveHandler
}

// ChannelInboundHandler defines a channel inbound handler
type ChannelInboundHandler interface {
	ActiveHandler
	InboundHandler
	InactiveHandler
}

// ChannelOutboundHandler defines a channel outbound handler
type ChannelOutboundHandler interface {
	ActiveHandler
	OutboundHandler
	InactiveHandler
}

// SimpleChannelHandler defines ChannelInboundHandler alias
type SimpleChannelHandler = ChannelInboundHandler

// ActiveHandlerFunc impl ActiveHandler
type ActiveHandlerFunc func(ctx ActiveContext)

// HandleActive to impl ActiveHandler
func (fn ActiveHandlerFunc) HandleActive(ctx ActiveContext) { fn(ctx) }

// InboundHandlerFunc impl InboundHandler
type InboundHandlerFunc func(ctx InboundContext, message Message)

// HandleRead to impl InboundHandler
func (fn InboundHandlerFunc) HandleRead(ctx InboundContext, message Message) { fn(ctx, message) }

// OutboundHandlerFunc impl OutboundHandler
type OutboundHandlerFunc func(ctx OutboundContext, message Message)

// HandleWrite to impl OutboundHandler
func (fn OutboundHandlerFunc) HandleWrite(ctx OutboundContext, message Message) { fn(ctx, message) }

// ExceptionHandlerFunc impl ExceptionHandler
type ExceptionHandlerFunc func(ctx ExceptionContext, ex Exception)

// HandleException to impl ExceptionHandler
func (fn ExceptionHandlerFunc) HandleException(ctx ExceptionContext, ex Exception) { fn(ctx, ex) }

// InactiveHandlerFunc impl InactiveHandler
type InactiveHandlerFunc func(ctx InactiveContext, ex Exception)

// HandleInactive to impl InactiveHandler
func (fn InactiveHandlerFunc) HandleInactive(ctx InactiveContext, ex Exception) { fn(ctx, ex) }

// EventHandlerFunc impl EventHandler
type EventHandlerFunc func(ctx EventContext, event Event)

// HandleEvent to impl EventHandler
func (fn EventHandlerFunc) HandleEvent(ctx EventContext, event Event) { fn(ctx, event) }

// headHandler
type headHandler struct{}

func (*headHandler) HandleActive(ctx ActiveContext) {
	ctx.HandleActive()
}

func (*headHandler) HandleRead(ctx InboundContext, message Message) {
	ctx.HandleRead(message)
}

func (*headHandler) HandleWrite(ctx OutboundContext, message Message) {

	switch m := message.(type) {
	case []byte:
		_, _ = ctx.Channel().Write(m)
	case [][]byte:
		_, _ = ctx.Channel().Writev(m)
	case io.WriterTo:
		_, _ = m.WriteTo(ctx.Channel())
	case io.Reader:
		data := utils.AssertBytes(ioutil.ReadAll(m))
		_, _ = ctx.Channel().Write(data)
	default:
		utils.Assert(fmt.Errorf("unsupported type: %T", m))
	}
}

func (*headHandler) HandleException(ctx ExceptionContext, ex Exception) {
	ctx.HandleException(ex)
}

func (*headHandler) HandleInactive(ctx InactiveContext, ex Exception) {
	_ = ctx.Channel().Close()
}

// default: tailHandler
// The final closing operation will be provided when the user registered handler is not processing.
type tailHandler struct{}

func (*tailHandler) HandleException(ctx ExceptionContext, ex Exception) {
	ctx.Close(ex)
}

func (*tailHandler) HandleInactive(ctx InactiveContext, ex Exception) {
	ctx.HandleInactive(ex)
}

func (*tailHandler) HandleWrite(ctx OutboundContext, message Message) {
	ctx.HandleWrite(message)
}

type (
	// ReadIdleEvent define a ReadIdleEvent
	ReadIdleEvent struct{}

	// WriteIdleEvent define a WriteIdleEvent
	WriteIdleEvent struct{}
)

// ReadIdleHandler fire ReadIdleEvent after waiting for a reading timeout
func ReadIdleHandler(idleTime time.Duration) ChannelInboundHandler {
	utils.AssertIf(idleTime < time.Second, "idleTime must be greater than one second")
	return &readIdleHandler{
		idleTime: idleTime,
	}
}

// WriteIdleHandler fire WriteIdleEvent after waiting for a sending timeout
func WriteIdleHandler(idleTime time.Duration) ChannelOutboundHandler {
	utils.AssertIf(idleTime < time.Second, "idleTime must be greater than one second")
	return &writeIdleHandler{
		idleTime: idleTime,
	}
}

// readIdleHandler
type readIdleHandler struct {
	idleTime     time.Duration
	lastReadTime atomic.Value // time.Time
	readTimer    atomic.Value // *time.Timer
	handlerCtx   atomic.Value // HandlerContext
}

func (r *readIdleHandler) HandleActive(ctx ActiveContext) {
	// cache context.
	r.handlerCtx.Store(ctx)
	r.lastReadTime.Store(time.Now())
	r.readTimer.Store(time.AfterFunc(r.idleTime, r.onReadTimeout))
	// post the active event.
	ctx.HandleActive()
}

func (r *readIdleHandler) HandleRead(ctx InboundContext, message Message) {
	ctx.HandleRead(message)
	// update last read time.
	r.lastReadTime.Store(time.Now())
}

func (r *readIdleHandler) HandleInactive(ctx InactiveContext, ex Exception) {

	// reset timer.
	r.handlerCtx.Store(nil)
	if readTimer, ok := r.readTimer.Load().(*time.Timer); ok {
		r.readTimer.Store(nil)
		readTimer.Stop()
	}

	// post the inactive event.
	ctx.HandleInactive(ex)
}

func (r *readIdleHandler) onReadTimeout() {

	ctx, ok := r.handlerCtx.Load().(HandlerContext)
	if !ok {
		return
	}

	lastReadTime := r.lastReadTime.Load().(time.Time)

	// check if the idle time expires.
	if d := time.Since(lastReadTime); d >= r.idleTime {

		// trigger event.
		func() {
			// capture exception.
			defer func() {
				if err := recover(); nil != err {
					ctx.Channel().Pipeline().fireChannelException(AsException(err, debug.Stack()))
				}
			}()

			// trigger ReadIdleEvent.
			ctx.Trigger(ReadIdleEvent{})
		}()
	}

	// reset timer.
	if readTimer, ok := r.readTimer.Load().(*time.Timer); ok {
		readTimer.Reset(r.idleTime)
	}
}

// writeIdleHandler
type writeIdleHandler struct {
	idleTime      time.Duration
	lastWriteTime atomic.Value // time.Time
	writeTimer    atomic.Value // *time.Timer
	handlerCtx    atomic.Value // HandlerContext
}

func (w *writeIdleHandler) HandleActive(ctx ActiveContext) {
	// cache context
	w.handlerCtx.Store(ctx)
	w.lastWriteTime.Store(time.Now())
	w.writeTimer.Store(time.AfterFunc(w.idleTime, w.onWriteTimeout))
	// post the active event.
	ctx.HandleActive()
}

func (w *writeIdleHandler) HandleWrite(ctx OutboundContext, message Message) {
	// update last write time.
	w.lastWriteTime.Store(time.Now())
	// post write event.
	ctx.HandleWrite(message)
}

func (w *writeIdleHandler) HandleInactive(ctx InactiveContext, ex Exception) {
	// reset context
	w.handlerCtx.Store(nil)

	// stop the timer.
	if writeTimer, ok := w.writeTimer.Load().(*time.Timer); ok {
		w.writeTimer.Store(nil)
		writeTimer.Stop()
	}

	// post the inactive event.
	ctx.HandleInactive(ex)
}

func (w *writeIdleHandler) onWriteTimeout() {

	ctx, ok := w.handlerCtx.Load().(HandlerContext)
	if !ok {
		return
	}

	lastWriteTime := w.lastWriteTime.Load().(time.Time)

	// check if the idle time expires
	if d := time.Since(lastWriteTime); d >= w.idleTime {

		// trigger event.
		func() {
			// capture exception
			defer func() {
				if err := recover(); nil != err {
					ctx.Channel().Pipeline().fireChannelException(AsException(err, debug.Stack()))
				}
			}()

			// trigger WriteIdleEvent.
			ctx.Trigger(WriteIdleEvent{})
		}()
	}

	// reset timer.
	if writeTimer, ok := w.writeTimer.Load().(*time.Timer); ok {
		writeTimer.Reset(w.idleTime)
	}
}
