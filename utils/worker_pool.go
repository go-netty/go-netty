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

package utils

import (
	"context"
	"time"
)

// WorkerPool defines worker pool.
type WorkerPool interface {
	// add the task
	AddTask(task func())
	// run the task
	RunTask(task func())
	// stop the worker pool
	Stop()
	// Stop all workers and wait for task execution timeout
	StopWait(timeout time.Duration)
}

// NewWorkerPool create a worker pool
func NewWorkerPool(ctx context.Context, taskCap, idleWorker, maxWorker int) WorkerPool {

	if idleWorker < 1 {
		idleWorker = 1
	}

	if maxWorker < idleWorker {
		maxWorker = idleWorker
	}

	poolCtx, cancel := context.WithCancel(ctx)

	pool := &workerPool{
		maxWorkers:   maxWorker,
		idleWorkers:  idleWorker,
		idleTimeout:  time.Second * 5,
		taskQueue:    make(chan func(), taskCap),
		readyWorkers: make(chan chan func(), idleWorker),
		poolCtx:      poolCtx,
		poolCancel:   cancel,
		closedSignal: make(chan struct{}),
	}

	go pool.dispatch()
	return pool
}

type workerPool struct {
	maxWorkers   int
	idleWorkers  int
	idleTimeout  time.Duration
	taskQueue    chan func()
	readyWorkers chan chan func()
	poolCtx      context.Context
	poolCancel   context.CancelFunc
	closedSignal chan struct{}
}

func (wp *workerPool) Stop() {
	select {
	case <-wp.poolCtx.Done():
		// closed pool
	default:
		// notify all worker to closed
		wp.poolCancel()
	}
}

func (wp *workerPool) StopWait(timeout time.Duration) {
	// 停止工作队列
	wp.Stop()

	// 设置超时时间用于指定最多等待时长
	// 超时将直接返回
	var timeChan <-chan time.Time
	if timeout > 0 {
		timeChan = time.After(timeout)
	}

	// 等待所有任务和worker退出
	select {
	case <-wp.closedSignal:
		// 所有的任务都已经执行完毕且worker已经退出
	case <-timeChan:
		// 等待超时
	}
}

func (wp *workerPool) AddTask(task func()) {
	if nil != task {
		select {
		case <-wp.poolCtx.Done():
			// 已经关闭的任务池，不接受任务
		default:
			wp.taskQueue <- task
		}
	}
}

func (wp *workerPool) RunTask(task func()) {
	done := make(chan struct{})
	wp.AddTask(func() {
		// 执行任务
		task()
		// 通知任务执行完毕
		close(done)
	})
	<-done
}

func (wp *workerPool) dispatch() {

	// 用于检查是否有必要关闭空闲的worker
	checkTimer := time.NewTicker(wp.idleTimeout)
	defer checkTimer.Stop()

	// 当前创建的worker数量
	workerCount := 0

	for {
		select {
		case task := <-wp.taskQueue:
			// 拿到一个任务，分配给有空闲的worker
			select {
			case worker := <-wp.readyWorkers:
				// 分配执行任务给这个worker
				worker <- task
			default:
				// 没有空闲的worker, 检查是否需要创建更多的worker来执行任务
				if workerCount < wp.maxWorkers {
					// 创建一个worker来执行这个任务
					wp.startWorker()
					// 记录worker数量
					workerCount++
				}

				// 等到一个worker
				// 达到上限之前，这里将会等到一个新创建的worker(也有可能是正好别的worker空闲了)
				// 达到上限之后，这里将会等待之前的worker有一个任务结束
				worker := <-wp.readyWorkers
				worker <- task
			}
		case <-checkTimer.C:
			// 检查是否有空闲的worker需要关闭掉
			// 最终会保留一定数量的worker处于待命状态
			// 多出的worker将会被关闭
			if workerCount > wp.idleWorkers {
				// 一次性关闭多余的闲置worker
				for i := workerCount - wp.idleWorkers; i > 0; i-- {
					select {
					case worker := <-wp.readyWorkers:
						// 通知worker退出
						worker <- nil
						// 减少worker数量
						workerCount--
					default:
						// 剩下的worker都在工作中，那么本轮结束清理
						// 等待下一轮再次尝试清理
						i = 0
						break
					}
				}
			}
		case <-wp.poolCtx.Done():
			// 任务池关闭了
			// 等待剩余的任务执行完毕
			for len(wp.taskQueue) > 0 {
				select {
				case task := <-wp.taskQueue:
					worker := <-wp.readyWorkers
					worker <- task
				}
			}

			// 通知所有的worker也退出
			for workerCount > 0 {
				// 通知所有的worker可以退出了
				worker := <-wp.readyWorkers
				worker <- nil
				workerCount--
			}

			// 所有的worker都退出了，可以安全的关闭了
			close(wp.readyWorkers)
			close(wp.taskQueue)
			close(wp.closedSignal)
			return
		}
	}
}

func (wp *workerPool) startWorker() {

	workerChan := make(chan func(), 1)

	go func() {
		// worker退出时释放工作队列
		defer close(workerChan)

		// 准备工作,直到外部通知退出
		for {
			// 通知外部开始工作，可以投递任务过来了
			wp.readyWorkers <- workerChan

			// 拿到一个任务, 没有任务将会阻塞
			task := <-workerChan
			if nil == task {
				// 外部通知本worker可以退出了
				break
			}

			// 执行任务
			// 任务里面必须要recover否则将会在发生panic时导致程序退出
			task()
		}
	}()
}
