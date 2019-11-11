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

	"github.com/go-netty/go-netty/utils"
)

type (
	// Message abstract type.
	// Represent different objects in different processing steps,
	// in most cases the message type handled by the codec is mainly io.Reader / []byte,
	// in the user handler should have been converted to a protocol object.
	Message interface {
	}

	// Event abstract type
	// User-defined event types, read-write timeout events, etc.
	Event interface {
	}

	// Attachment abstract type.
	// The object or data associated with the Channel.
	Attachment interface {
	}

	// Handler abstract type
	// At least one or more of the following types should be implemented
	// ActiveHandler
	// InboundHandler
	// OutboundHandler
	// ExceptionHandler
	// InactiveHandler
	// EventHandler
	Handler interface {
	}

	// Active event handler.
	ActiveHandler interface {
		HandleActive(ctx ActiveContext)
	}

	// Inbound event handler.
	InboundHandler interface {
		HandleRead(ctx InboundContext, message Message)
	}

	// Outbound event handler.
	OutboundHandler interface {
		HandleWrite(ctx OutboundContext, message Message)
	}

	// Exception event handler.
	ExceptionHandler interface {
		HandleException(ctx ExceptionContext, ex Exception)
	}

	// Inactivation event handler.
	InactiveHandler interface {
		HandleInactive(ctx InactiveContext, ex Exception)
	}

	// User-defined event handler.
	EventHandler interface {
		HandleEvent(ctx EventContext, event Event)
	}
)

// CodecHandler
type CodecHandler interface {
	CodecName() string
	InboundHandler
	OutboundHandler
}

// ChannelHandler
type ChannelHandler interface {
	ActiveHandler
	InboundHandler
	OutboundHandler
	ExceptionHandler
	InactiveHandler
}

// ChannelInboundHandler
type ChannelInboundHandler interface {
	ActiveHandler
	InboundHandler
	InactiveHandler
}

// ChannelOutboundHandler
type ChannelOutboundHandler interface {
	ActiveHandler
	OutboundHandler
	InactiveHandler
}

// SimpleChannelHandler
type SimpleChannelHandler = ChannelInboundHandler

// func for ActiveHandler
type ActiveHandlerFunc func(ctx ActiveContext)

func (fn ActiveHandlerFunc) HandleActive(ctx ActiveContext)                        { fn(ctx) }

// func for InboundHandler
type InboundHandlerFunc func(ctx InboundContext, message Message)

func (fn InboundHandlerFunc) HandleRead(ctx InboundContext, message Message)       { fn(ctx, message) }

// func for OutboundHandler
type OutboundHandlerFunc func(ctx OutboundContext, message Message)

func (fn OutboundHandlerFunc) HandleWrite(ctx OutboundContext, message Message)    { fn(ctx, message) }

// func for ExceptionHandler
type ExceptionHandlerFunc func(ctx ExceptionContext, ex Exception)

func (fn ExceptionHandlerFunc) HandleException(ctx ExceptionContext, ex Exception) { fn(ctx, ex) }

// func for InactiveHandler
type InactiveHandlerFunc func(ctx InactiveContext, ex Exception)

func (fn InactiveHandlerFunc) HandleInactive(ctx InactiveContext, ex Exception)    { fn(ctx, ex) }

// func for EventHandler
type EventHandlerFunc func(ctx EventContext, event Event)

func (fn EventHandlerFunc) HandleEvent(ctx EventContext, event Event)              { fn(ctx, event) }

// default: headHandler.
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
