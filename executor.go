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
	"runtime/debug"

	"github.com/go-netty/go-netty/utils"
)

// ChannelExecutor
type ChannelExecutor interface {
	InboundHandler
	EventHandler
}

// A fixed number of coroutines for event processing
func NewFixedChannelExecutor(taskCap int, workerNum int) ChannelExecutorFactory {
	return func(_ctx context.Context) ChannelExecutor {
		return &channelExecutor{WorkerPool: utils.NewWorkerPool(taskCap, workerNum, workerNum, _ctx)}
	}
}

// Flexible number of coroutine event processing, allowing setting maximum
func NewFlexibleChannelExecutor(taskCap int, idleWorker, maxWorker int) ChannelExecutorFactory {
	return func(_ctx context.Context) ChannelExecutor {
		return &channelExecutor{WorkerPool: utils.NewWorkerPool(taskCap, idleWorker, maxWorker, _ctx)}
	}
}

type channelExecutor struct {
	utils.WorkerPool
}

func (ce *channelExecutor) HandleRead(ctx InboundContext, message Message) {

	ce.AddTask(func() {

		// capture exception
		defer func() {
			if err := recover(); nil != err {
				ctx.Channel().Pipeline().fireChannelException(AsException(err, debug.Stack()))
			}
		}()

		// do HandleRead
		ctx.HandleRead(message)
	})
}

func (ce *channelExecutor) HandleEvent(ctx EventContext, event Event) {

	ce.AddTask(func() {

		// capture exception
		defer func() {
			if err := recover(); nil != err {
				ctx.Channel().Pipeline().fireChannelException(AsException(err, debug.Stack()))
			}
		}()

		// do HandleEvent
		ctx.HandleEvent(event)
	})
}
