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
	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/utils"
	"os"
	"os/signal"
)

// Bootstrap
type Bootstrap interface {
	Context() context.Context
	WithContext(ctx context.Context) Bootstrap
	ChannelExecutor(executorFactory ChannelExecutorFactory) Bootstrap
	ChannelId(channelIdFactory ChannelIdFactory) Bootstrap
	Pipeline(pipelineFactory PipelineFactory) Bootstrap
	Channel(channelFactory ChannelFactory) Bootstrap
	Transport(factory TransportFactory) Bootstrap
	ChildInitializer(initializer ChannelInitializer) Bootstrap
	ClientInitializer(initializer ChannelInitializer) Bootstrap
	Listen(url string, option ...transport.Option) Bootstrap
	Connect(url string, attachment Attachment, option ...transport.Option) (Channel, error)
	Action(action func(Bootstrap)) Bootstrap
	Stop() Bootstrap
}

// Wait for signal.
func WaitSignal(signals ...os.Signal) func(Bootstrap) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, signals...)

	return func(bs Bootstrap) {
		select {
		case <-bs.Context().Done():
		case <-sigChan:
		}
	}
}

// Create a new Bootstrap.
func NewBootstrap() Bootstrap {
	return new(bootstrap).WithContext(context.Background()).ChannelId(SequenceId()).Pipeline(NewPipeline()).Channel(NewChannel(128))
}

type bootstrap struct {
	bootstrapOptions
	acceptor transport.Acceptor
}

func (b *bootstrap) WithContext(ctx context.Context) Bootstrap {
	b.bootstrapCtx, b.bootstrapCancel = context.WithCancel(ctx)
	return b
}

func (b *bootstrap) Context() context.Context {
	return b.bootstrapCtx
}

func (b *bootstrap) ChannelExecutor(executor ChannelExecutorFactory) Bootstrap {
	b.executorFactory = executor
	return b
}

func (b *bootstrap) ChannelId(channelIdFactory ChannelIdFactory) Bootstrap {
	b.channelIdFactory = channelIdFactory
	return b
}

func (b *bootstrap) Pipeline(pipelineFactory PipelineFactory) Bootstrap {
	b.pipelineFactory = pipelineFactory
	return b
}

func (b *bootstrap) Channel(channelFactory ChannelFactory) Bootstrap {
	b.channelFactory = channelFactory
	return b
}

func (b *bootstrap) Transport(factory TransportFactory) Bootstrap {
	b.transportFactory = factory
	return b
}

func (b *bootstrap) ChildInitializer(initializer ChannelInitializer) Bootstrap {
	b.childInitializer = initializer
	return b
}

func (b *bootstrap) ClientInitializer(initializer ChannelInitializer) Bootstrap {
	b.clientInitializer = initializer
	return b
}

func (b *bootstrap) serveChannel(channelExecutor ChannelExecutor, channel Channel, childChannel bool) {

	// initialization pipeline
	if childChannel {
		b.childInitializer(channel)
	} else {
		b.clientInitializer(channel)
	}

	// 需要插入Executor
	if channelExecutor != nil {

		// 找到最后一个解码器的位置
		position := channel.Pipeline().LastIndexOf(func(handler Handler) bool {
			_, ok := handler.(CodecHandler)
			return ok
		})

		// 必须要有Codec
		utils.AssertIf(-1 == position, "missing codec.")

		// 插入到解码器后面
		channel.Pipeline().AddHandler(position, channelExecutor)
	}

	// serve channel.
	channel.Pipeline().serveChannel(channel)
}

func (b *bootstrap) serveTransport(transport transport.Transport, attachment Attachment, childChannel bool) Channel {

	// 创建一个流水线, 用于定义事件处理流程
	pipeline := b.pipelineFactory()

	// 生成ChanelId
	cid := b.channelIdFactory()

	// 创建一个Channel用于读写数据
	channel := b.channelFactory(cid, b.bootstrapCtx, pipeline, transport)

	// 挂载附件
	if nil != attachment {
		channel.SetAttachment(attachment)
	}

	// Channel Executor
	var chExecutor ChannelExecutor
	if nil != b.executorFactory {
		chExecutor = b.executorFactory(channel.Context())
	}

	b.serveChannel(chExecutor, channel, childChannel)
	return channel
}

func (b *bootstrap) createListener(listenOptions ...transport.Option) error {

	// 不需要创建
	if len(listenOptions) <= 0 {
		return nil
	}

	options, err := transport.ParseOptions(listenOptions...)
	if nil != err {
		return err
	}

	// 监听服务
	l, err := b.transportFactory.Listen(options)
	if nil != err {
		return err
	}

	// 关闭已有的监听器
	b.stopListener()

	b.acceptor = l
	return nil
}

func (b *bootstrap) startListener() {

	if nil == b.acceptor {
		return
	}

	go func() {

		for {
			// 接受一个连接
			t, err := b.acceptor.Accept()
			if nil != err {
				break
			}

			select {
			case <-b.Context().Done():
				// 程序需要退出
				_ = t.Close()
				return
			default:
				// 开始服务
				b.serveTransport(t, nil, true)
			}
		}
	}()
}

func (b *bootstrap) stopListener() {
	if b.acceptor != nil {
		_ = b.acceptor.Close()
		b.acceptor = nil
	}
}

func (b *bootstrap) Connect(url string, attachment Attachment, option ...transport.Option) (Channel, error) {

	transOptions := []transport.Option{
		// remote address.
		transport.WithAddress(url),
		// context
		transport.WithContext(b.Context()),
	}
	transOptions = append(transOptions, option...)

	options, err := transport.ParseOptions(transOptions...)
	if nil != err {
		return nil, err
	}

	// 连接对端
	t, err := b.transportFactory.Connect(options)
	if nil != err {
		return nil, err
	}

	// 开始服务
	return b.serveTransport(t, attachment, false), nil
}

func (b *bootstrap) Listen(url string, option ...transport.Option) Bootstrap {
	listenOptions := []transport.Option{
		// remote address
		transport.WithAddress(url),
		// context.
		transport.WithContext(b.Context()),
	}
	listenOptions = append(listenOptions, option...)
	// create listener
	utils.Assert(b.createListener(listenOptions...))
	// start acceptor
	b.startListener()
	return b
}

func (b *bootstrap) Action(action func(Bootstrap)) Bootstrap {
	defer b.Stop()
	action(b)
	return b
}

func (b *bootstrap) Stop() Bootstrap {
	b.stopListener()
	b.bootstrapCancel()
	return b
}
