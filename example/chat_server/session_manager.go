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

package main

import (
	"sync"

	"github.com/go-netty/go-netty"
)

type Manager interface {
	// middleware
	netty.ActiveHandler
	netty.InactiveHandler
	// size of active channels.
	Size() int
	// find channel context with id.
	Context(id int64) netty.HandlerContext
	// foreach active channels.
	ForEach(func(netty.HandlerContext) bool)
	// broadcast message.
	Broadcast(message netty.Message)
	// broadcast message filter.
	BroadcastIf(message netty.Message, fn func(netty.HandlerContext) bool)
}

func NewManager() Manager {
	return &sessionManager{
		_sessions: make(map[int64]netty.HandlerContext, 64),
	}
}

type sessionManager struct {
	_sessions map[int64]netty.HandlerContext
	_mutex    sync.RWMutex
}

func (s *sessionManager) Size() int {
	s._mutex.RLock()
	size := len(s._sessions)
	s._mutex.RUnlock()
	return size
}

func (s *sessionManager) Context(id int64) netty.HandlerContext {
	s._mutex.RLock()
	ctx, _ := s._sessions[id]
	s._mutex.RUnlock()
	return ctx
}

func (s *sessionManager) ForEach(fn func(netty.HandlerContext) bool) {
	s._mutex.RLock()
	defer s._mutex.RUnlock()

	for _, ctx := range s._sessions {
		fn(ctx)
	}
}

func (s *sessionManager) Broadcast(message netty.Message) {
	s.ForEach(func(ctx netty.HandlerContext) bool {
		ctx.Write(message)
		return true
	})
}

func (s *sessionManager) BroadcastIf(message netty.Message, fn func(netty.HandlerContext) bool) {
	s.ForEach(func(ctx netty.HandlerContext) bool {
		if fn(ctx) {
			ctx.Write(message)
		}
		return true
	})
}

func (s *sessionManager) HandleActive(ctx netty.ActiveContext) {

	s._mutex.Lock()
	s._sessions[ctx.Channel().Id()] = ctx
	s._mutex.Unlock()

	ctx.HandleActive()
}

func (s *sessionManager) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	s._mutex.Lock()
	delete(s._sessions, ctx.Channel().Id())
	s._mutex.Unlock()

	ctx.HandleInactive(ex)
}

func (*sessionManager) HandleRead(ctx netty.InboundContext, message netty.Message) {
	ctx.HandleRead(message)
}

func (*sessionManager) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}

