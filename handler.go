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
	"os"
	"runtime/debug"
	"sync"
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

func (*headHandler) HandleWrite(ctx OutboundContext, message Message) {

	var dataBytes [][]byte
	switch m := message.(type) {
	case []byte:
		dataBytes = [][]byte{m}
	case [][]byte:
		dataBytes = m
	case io.Reader:
		data := utils.AssertBytes(ioutil.ReadAll(m))
		dataBytes = [][]byte{data}
	default:
		panic(fmt.Errorf("unsupported type: %T", m))
	}

	writeN, err := ctx.Channel().Writev(dataBytes)
	if totalN := utils.CountOf(dataBytes); totalN != writeN && nil == err {
		err = fmt.Errorf("short write: %d != %d", totalN, writeN)
	}

	if nil != err {
		panic(err)
	}
}

// default: tailHandler
// The final closing operation will be provided when the user registered handler is not processing.
type tailHandler struct{}

func (*tailHandler) HandleException(ctx ExceptionContext, ex Exception) {
	ex.PrintStackTrace(os.Stderr, "An HandleException() event was fired, and it reached at the tail of the pipeline. ",
		"It usually means the last handler in the pipeline did not handle the exception. ",
		fmt.Sprintf("We will close the channel(%d: %s), If you don't want to close the channel please add HandleException() to the pipeline.\n", ctx.Channel().ID(), ctx.Channel().RemoteAddr()),
	)
	ctx.Channel().Close(ex)
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
	mutex        sync.RWMutex
	idleTime     time.Duration
	lastReadTime time.Time
	readTimer    *time.Timer
	handlerCtx   HandlerContext
}

func (r *readIdleHandler) withLock(fn func()) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	fn()
}

func (r *readIdleHandler) withReadLock(fn func()) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	fn()
}

func (r *readIdleHandler) HandleActive(ctx ActiveContext) {
	// cache context.
	r.withLock(func() {
		r.handlerCtx = ctx
		r.lastReadTime = time.Now()
		r.readTimer = time.AfterFunc(r.idleTime, r.onReadTimeout)
	})
	// post the active event.
	ctx.HandleActive()
}

func (r *readIdleHandler) HandleRead(ctx InboundContext, message Message) {
	ctx.HandleRead(message)

	r.withLock(func() {
		// update last read time.
		r.lastReadTime = time.Now()
		// reset timer.
		if r.readTimer != nil {
			r.readTimer.Reset(r.idleTime)
		}
	})
}

func (r *readIdleHandler) HandleInactive(ctx InactiveContext, ex Exception) {

	r.withLock(func() {
		r.handlerCtx = nil
		if r.readTimer != nil {
			r.readTimer.Stop()
			r.readTimer = nil
		}
	})

	// post the inactive event.
	ctx.HandleInactive(ex)
}

func (r *readIdleHandler) onReadTimeout() {

	var expired bool
	var ctx HandlerContext

	r.withReadLock(func() {
		// check if the idle time expires.
		expired = time.Since(r.lastReadTime) >= r.idleTime
		ctx = r.handlerCtx
	})

	if expired && ctx != nil {
		// trigger event.
		func() {
			// capture exception.
			defer func() {
				if err := recover(); nil != err {
					ctx.Channel().Pipeline().FireChannelException(AsException(err, debug.Stack()))
				}
			}()

			// trigger ReadIdleEvent.
			ctx.Trigger(ReadIdleEvent{})
		}()
	}

	// reset timer
	r.withReadLock(func() {
		if r.readTimer != nil {
			r.readTimer.Reset(r.idleTime)
		}
	})
}

// writeIdleHandler
type writeIdleHandler struct {
	mutex         sync.RWMutex
	idleTime      time.Duration
	lastWriteTime time.Time
	writeTimer    *time.Timer
	handlerCtx    HandlerContext
}

func (w *writeIdleHandler) withLock(fn func()) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	fn()
}

func (w *writeIdleHandler) withReadLock(fn func()) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()
	fn()
}

func (w *writeIdleHandler) HandleActive(ctx ActiveContext) {

	// cache context
	w.withLock(func() {
		w.handlerCtx = ctx
		w.lastWriteTime = time.Now()
		w.writeTimer = time.AfterFunc(w.idleTime, w.onWriteTimeout)
	})

	// post the active event.
	ctx.HandleActive()
}

func (w *writeIdleHandler) HandleWrite(ctx OutboundContext, message Message) {

	// update last write time.
	w.withLock(func() {
		w.lastWriteTime = time.Now()
		// reset timer.
		if w.writeTimer != nil {
			w.writeTimer.Reset(w.idleTime)
		}
	})

	// post write event.
	ctx.HandleWrite(message)
}

func (w *writeIdleHandler) HandleInactive(ctx InactiveContext, ex Exception) {

	w.withLock(func() {
		// reset context
		w.handlerCtx = nil
		// stop the timer.
		if w.writeTimer != nil {
			w.writeTimer.Stop()
			w.writeTimer = nil
		}
	})

	// post the inactive event.
	ctx.HandleInactive(ex)
}

func (w *writeIdleHandler) onWriteTimeout() {

	var expired bool
	var ctx HandlerContext

	w.withReadLock(func() {
		// check if the idle time expires.
		expired = time.Since(w.lastWriteTime) >= w.idleTime
		ctx = w.handlerCtx
	})

	// check if the idle time expires
	if expired && ctx != nil {
		// trigger event.
		func() {
			// capture exception
			defer func() {
				if err := recover(); nil != err {
					ctx.Channel().Pipeline().FireChannelException(AsException(err, debug.Stack()))
				}
			}()

			// trigger WriteIdleEvent.
			ctx.Trigger(WriteIdleEvent{})
		}()
	}

	// reset timer.
	w.withReadLock(func() {
		if w.writeTimer != nil {
			w.writeTimer.Reset(w.idleTime)
		}
	})

}
