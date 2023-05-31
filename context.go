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
		Trigger(event Event)
		Close(err error)
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
	pipeline       Pipeline
	handler        Handler
	prev           *handlerContext
	next           *handlerContext
	cast2Active    ActiveHandler
	cast2Inbound   InboundHandler
	cast2Outbound  OutboundHandler
	cast2Exception ExceptionHandler
	cast2Inactive  InactiveHandler
	cast2Event     EventHandler
}

func newHandlerContext(p Pipeline, handler Handler, prev, next *handlerContext) *handlerContext {
	hc := &handlerContext{
		pipeline: p,
		handler:  handler,
		prev:     prev,
		next:     next,
	}

	hc.cast2Active, _ = handler.(ActiveHandler)
	hc.cast2Inbound, _ = handler.(InboundHandler)
	hc.cast2Outbound, _ = handler.(OutboundHandler)
	hc.cast2Exception, _ = handler.(ExceptionHandler)
	hc.cast2Inactive, _ = handler.(InactiveHandler)
	hc.cast2Event, _ = handler.(EventHandler)
	return hc
}

func (hc *handlerContext) prevContext() *handlerContext {
	return hc.prev
}

func (hc *handlerContext) nextContext() *handlerContext {
	return hc.next
}

func (hc *handlerContext) Write(message Message) {

	defer func() {
		if err := recover(); nil != err {
			hc.Channel().Pipeline().FireChannelException(AsException(err, debug.Stack()))
		}
	}()

	var next = hc

	for {
		if next = next.prevContext(); nil == next {
			break
		}

		if handler := next.cast2Outbound; nil != handler {
			handler.HandleWrite(next, message)
			break
		}
	}
}

func (hc *handlerContext) Trigger(event Event) {

	defer func() {
		if err := recover(); nil != err {
			hc.Channel().Pipeline().FireChannelException(AsException(err, debug.Stack()))
		}
	}()

	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler := next.cast2Event; nil != handler {
			handler.HandleEvent(next, event)
			break
		}
	}
}

func (hc *handlerContext) Close(err error) {
	hc.Channel().Close(err)
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

		if handler := next.cast2Active; nil != handler {
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

		if handler := next.cast2Inbound; nil != handler {
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

		if handler := prev.cast2Outbound; nil != handler {
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

		if handler := next.cast2Exception; nil != handler {
			handler.HandleException(next, ex)
			break
		}
	}
}

func (hc *handlerContext) HandleInactive(ex Exception) {
	var next = hc

	for {
		if next = next.nextContext(); nil == next {
			break
		}

		if handler := next.cast2Inactive; nil != handler {
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

		if handler := next.cast2Event; nil != handler {
			handler.HandleEvent(next, event)
			break
		}
	}
}
