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
	"runtime/debug"

	"github.com/go-netty/go-netty/utils"
)

type Pipeline interface {

	// add handler to first.
	AddFirst(handlers ...Handler) Pipeline

	// add handler to last.
	AddLast(handlers ...Handler) Pipeline

	// add handlers in position.
	AddHandler(position int, handlers ...Handler) Pipeline

	// find fist index of handler.
	IndexOf(func(Handler) bool) int

	// find last index of handler.
	LastIndexOf(func(Handler) bool) int

	// get context by position.
	ContextAt(position int) HandlerContext

	// size of handler
	Size() int

	// channel.
	Channel() Channel

	// serve the channel.
	serveChannel(channel Channel)

	// internal use.
	fireChannelActive()
	fireChannelRead(message Message)
	fireChannelWrite(message Message)
	fireChannelException(ex Exception)
	fireChannelInactive(ex Exception)
	fireChannelEvent(event Event)
}

func NewPipeline() PipelineFactory {
	return NewPipelineWith
}

func NewPipelineWith() Pipeline {

	p := &pipeline{}

	p.head = &handlerContext{
		pipeline: p,
		handler:  new(headHandler),
	}

	p.tail = &handlerContext{
		pipeline: p,
		handler:  new(tailHandler),
	}

	p.head.next = p.tail
	p.tail.prev = p.head

	// head + tail
	p.size = 2
	return p
}

type pipeline struct {
	head    *handlerContext
	tail    *handlerContext
	channel Channel
	size    int
}

func (p *pipeline) AddFirst(handlers ...Handler) Pipeline {
	for _, h := range handlers {
		p.addFirst(h)
	}
	return p
}

func (p *pipeline) AddLast(handlers ...Handler) Pipeline {
	for _, h := range handlers {
		p.addLast(h)
	}
	return p
}

// add handlers in position.
func (p *pipeline) AddHandler(position int, handlers ...Handler) Pipeline {

	// checking handler.
	checkHandler(handlers)

	if -1 == position {
		return p.AddLast(handlers...)
	}

	curNode := p.head
	for i := 0; i < position; i++ {
		curNode = curNode.next
	}

	for _, h := range handlers {
		oldNext := curNode.next
		curNode.next = &handlerContext{
			pipeline: p,
			handler:  h,
			prev:     curNode,
			next:     oldNext,
		}

		oldNext.prev = curNode.next
		curNode = curNode.next
		p.size++
	}

	return p
}

// find fist index of handler.
func (p *pipeline) IndexOf(comp func(Handler) bool) int {

	head := p.head

	for i := 0; ; i++ {
		if comp(head.handler) {
			return i
		}
		if head = head.next; head != nil {
			continue
		}

		break
	}
	return -1
}

// find last index of handler.
func (p *pipeline) LastIndexOf(comp func(Handler) bool) int {

	tail := p.tail

	for i := p.size - 1; ; i-- {
		if comp(tail.handler) {
			return i
		}
		if tail = tail.prev; tail != nil {
			continue
		}

		break
	}

	return -1
}

func (p *pipeline) ContextAt(position int) HandlerContext {

	if -1 == position || position >= p.size {
		return nil
	}

	curNode := p.head
	for i := 0; i < position; i++ {
		curNode = curNode.next
	}
	return curNode
}

// size of handler
func (p *pipeline) Size() int {
	return p.size
}

func (p *pipeline) addFirst(handler Handler) {

	// checking handler.
	checkHandler(handler)

	oldNext := p.head.next
	p.head.next = &handlerContext{
		pipeline: p,
		handler:  handler,
		prev:     p.head,
		next:     oldNext,
	}

	oldNext.prev = p.head.next
	p.size++
}

func (p *pipeline) addLast(handler Handler) {

	// checking handler.
	checkHandler(handler)

	oldPrev := p.tail.prev
	p.tail.prev = &handlerContext{
		pipeline: p,
		handler:  handler,
		prev:     oldPrev,
		next:     p.tail,
	}

	oldPrev.next = p.tail.prev
	p.size++
}

func (p *pipeline) Channel() Channel {
	return p.channel
}

func (p *pipeline) serveChannel(channel Channel) {

	utils.AssertIf(nil != p.channel, "already attached channel.")
	p.channel = channel

	defer func() {
		if err := recover(); nil != err {
			p.fireChannelException(AsException(err, debug.Stack()))
		}
	}()

	p.fireChannelActive()
	go p.channel.serveChannel()
}

func (p *pipeline) fireChannelActive() {
	p.head.HandleActive()
}

func (p *pipeline) fireChannelRead(message Message) {
	p.head.HandleRead(message)
}

func (p *pipeline) fireChannelWrite(message Message) {
	p.tail.HandleWrite(message)
}

func (p *pipeline) fireChannelException(ex Exception) {
	p.head.HandleException(ex)
}

func (p *pipeline) fireChannelInactive(ex Exception) {
	p.tail.HandleInactive(ex)
}

func(p *pipeline) fireChannelEvent(event Event) {
	p.head.HandleEvent(event)
}

func checkHandler(handlers ...Handler) {

	for _, h := range handlers {
		switch h.(type) {
		case ActiveHandler:
		case InboundHandler:
		case OutboundHandler:
		case ExceptionHandler:
		case InactiveHandler:
		case EventHandler:
		default:
			utils.Assert(fmt.Errorf("unrecognized Handler: %T", h))
		}
	}
}
