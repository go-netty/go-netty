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

// ChannelExecutor defines an executor
type ChannelExecutor interface {
}

// NewFixedChannelExecutor create a fixed number of coroutines for event processing
func NewFixedChannelExecutor(taskCap int, workerNum int) ChannelExecutorFactory {
	return func(ctx context.Context) ChannelExecutor {
		return &channelExecutor{worker: utils.NewWorkerPool(ctx, taskCap, workerNum, workerNum)}
	}
}

// NewFlexibleChannelExecutor create a flexible number of coroutine event processing, allowing setting maximum
func NewFlexibleChannelExecutor(taskCap int, idleWorker, maxWorker int) ChannelExecutorFactory {
	return func(ctx context.Context) ChannelExecutor {
		return &channelExecutor{worker: utils.NewWorkerPool(ctx, taskCap, idleWorker, maxWorker)}
	}
}

// channelExecutor impl ChannelExecutor
type channelExecutor struct {
	worker utils.WorkerPool
}

func(ce *channelExecutor) execute(ctx HandlerContext, task func()) {

	ce.worker.AddTask(func() {
		// capture exception
		defer func() {
			if err := recover(); nil != err {
				ctx.Channel().Pipeline().FireChannelException(AsException(err, debug.Stack()))
			}
		}()

		// do task
		task()
	})
}

func (ce *channelExecutor) HandleActive(ctx ActiveContext) {
	ce.execute(ctx, func() {
		ctx.HandleActive()
	})
}

func (ce *channelExecutor) HandleRead(ctx InboundContext, message Message) {
	ce.execute(ctx, func() {
		ctx.HandleRead(message)
	})
}

/*func (ce *channelExecutor) HandleWrite(ctx OutboundContext, message Message) {
	panic("implement me")
}*/

func (ce *channelExecutor) HandleException(ctx ExceptionContext, ex Exception) {
	ce.execute(ctx, func() {
		ctx.HandleException(ex)
	})
}

func (ce *channelExecutor) HandleInactive(ctx InactiveContext, ex Exception) {
	ce.execute(ctx, func() {
		ctx.HandleInactive(ex)
	})
}

func (ce *channelExecutor) HandleEvent(ctx EventContext, event Event) {
	ce.execute(ctx, func() {
		ctx.HandleEvent(event)
	})
}



