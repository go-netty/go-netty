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

import "runtime/debug"

type (
	// HandlerContext defines a base handler context
	HandlerContext interface {
		Channel() Channel
		Handler() Handler
		Write(message Message)
		Close(err error)
		Trigger(event Event)
		Attachment() Attachment
		SetAttachment(Attachment)
	}

	// ActiveContext defines an active handler
	ActiveContext interface {
		HandlerContext
		HandleActive()
	}

	// InboundContext defines an inbound handler
	InboundContext interface {
		HandlerContext
		HandleRead(message Message)
	}

	// OutboundContext defines an outbound handler
	OutboundContext interface {
		HandlerContext
		HandleWrite(message Message)
	}

	// ExceptionContext defines an exception handler
	ExceptionContext interface {
		HandlerContext
		HandleException(ex Exception)
	}

	// InactiveContext defines an inactive handler
	InactiveContext interface {
		HandlerContext
		HandleInactive(ex Exception)
	}

	// EventContext defines an event handler
	EventContext interface {
		HandlerContext
		HandleEvent(event Event)
	}
)

// handlerContext impl HandlerContext
type handlerContext struct {
	pipeline Pipeline
	handler  Handler
	prev     *handlerContext
	next     *handlerContext
}

func (hc *handlerContext) prevContext() *handlerContext {
	return hc.prev
}

func (hc *handlerContext) nextContext() *handlerContext {
	return hc.next
}

func (hc *handlerContext) Write(message Message) {
	var next = hc

	for {
		if next = next.prevContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(OutboundHandler); ok {
			handler.HandleWrite(next, message)
			break
		}
	}
}

func (hc *handlerContext) Close(err error) {

	ex, ok := err.(Exception)
	if !ok && err != nil {
		ex = AsException(err, debug.Stack())
	}

	var prev = hc
	for {
		if prev = prev.prevContext(); nil == prev {
			break
		}

		if handler, ok := prev.Handler().(InactiveHandler); ok {
			handler.HandleInactive(prev, ex)
			break
		}
	}
}

func (hc *handlerContext) Trigger(event Event) {
	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(EventHandler); ok {
			handler.HandleEvent(next, event)
			break
		}
	}
}

func (hc *handlerContext) Channel() Channel {
	return hc.pipeline.Channel()
}

func (hc *handlerContext) Handler() Handler {
	return hc.handler
}

func (hc *handlerContext) Attachment() Attachment {
	return hc.Channel().Attachment()
}

func (hc *handlerContext) SetAttachment(v Attachment) {
	hc.Channel().SetAttachment(v)
}

func (hc *handlerContext) HandleActive() {

	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(ActiveHandler); ok {
			handler.HandleActive(next)
			break
		}
	}
}

func (hc *handlerContext) HandleRead(message Message) {
	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(InboundHandler); ok {
			handler.HandleRead(next, message)
			break
		}
	}
}

func (hc *handlerContext) HandleWrite(message Message) {
	var prev = hc

	for {
		if prev = prev.prevContext(); nil == prev {
			break
		}

		if handler, ok := prev.Handler().(OutboundHandler); ok {
			handler.HandleWrite(prev, message)
			break
		}
	}
}

func (hc *handlerContext) HandleException(ex Exception) {
	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(ExceptionHandler); ok {
			handler.HandleException(next, ex)
			break
		}
	}
}

func (hc *handlerContext) HandleInactive(ex Exception) {
	var next = hc

	for {
		if next = next.prevContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(InactiveHandler); ok {
			handler.HandleInactive(next, ex)
			break
		}
	}
}

func (hc *handlerContext) HandleEvent(event Event) {
	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler, ok := next.Handler().(EventHandler); ok {
			handler.HandleEvent(next, event)
			break
		}
	}
}
