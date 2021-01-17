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

// Bootstrap makes it easy to bootstrap a channel
type Bootstrap interface {
	Context() context.Context
	WithContext(ctx context.Context) Bootstrap
	ChannelExecutor(executorFactory ChannelExecutorFactory) Bootstrap
	ChannelID(channelIDFactory ChannelIDFactory) Bootstrap
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

// WaitSignal for your signals
func WaitSignal(signals ...os.Signal) func(Bootstrap) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, signals...)

	return func(bs Bootstrap) {
		select {
		case <-bs.Context().Done():
		case <-sigChan:
			bs.Stop()
		}
	}
}

// NewBootstrap create a new Bootstrap with default config.
func NewBootstrap() Bootstrap {
	return new(bootstrap).WithContext(context.Background()).ChannelID(SequenceID()).Pipeline(NewPipeline()).Channel(NewChannel(128))
}

// bootstrap implement
type bootstrap struct {
	bootstrapOptions
	acceptor transport.Acceptor
}

// WithContext fork child context with context.WithCancel
func (b *bootstrap) WithContext(ctx context.Context) Bootstrap {
	b.bootstrapCtx, b.bootstrapCancel = context.WithCancel(ctx)
	return b
}

// Context to get context
func (b *bootstrap) Context() context.Context {
	return b.bootstrapCtx
}

// ChannelExecutor to set ChannelExecutorFactory
func (b *bootstrap) ChannelExecutor(executor ChannelExecutorFactory) Bootstrap {
	b.executorFactory = executor
	return b
}

// ChannelID to set ChannelIDFactory
func (b *bootstrap) ChannelID(channelIDFactory ChannelIDFactory) Bootstrap {
	b.channelIDFactory = channelIDFactory
	return b
}

// Pipeline to set PipelineFactory
func (b *bootstrap) Pipeline(pipelineFactory PipelineFactory) Bootstrap {
	b.pipelineFactory = pipelineFactory
	return b
}

// Channel to set ChannelFactory
func (b *bootstrap) Channel(channelFactory ChannelFactory) Bootstrap {
	b.channelFactory = channelFactory
	return b
}

// Transport to set TransportFactory
func (b *bootstrap) Transport(factory TransportFactory) Bootstrap {
	b.transportFactory = factory
	return b
}

// ChildInitializer to set server side ChannelInitializer
func (b *bootstrap) ChildInitializer(initializer ChannelInitializer) Bootstrap {
	b.childInitializer = initializer
	return b
}

// ClientInitializer to set client side ChannelInitializer
func (b *bootstrap) ClientInitializer(initializer ChannelInitializer) Bootstrap {
	b.clientInitializer = initializer
	return b
}

// serveChannel to startup channel
func (b *bootstrap) serveChannel(channelExecutor ChannelExecutor, channel Channel, childChannel bool) {

	// initialization pipeline
	if childChannel {
		b.childInitializer(channel)
	} else {
		b.clientInitializer(channel)
	}

	// need insert executor handler
	if channelExecutor != nil {

		// find the last codec handler position
		position := channel.Pipeline().LastIndexOf(func(handler Handler) bool {
			_, ok := handler.(CodecHandler)
			return ok
		})

		// assert checker
		utils.AssertIf(-1 == position, "missing codec.")

		// insert executor to pipeline at position
		channel.Pipeline().AddHandler(position, channelExecutor)
	}

	// serve channel.
	channel.Pipeline().ServeChannel(channel)
}

// serveTransport to serve channel
func (b *bootstrap) serveTransport(transport transport.Transport, attachment Attachment, childChannel bool) Channel {

	// create a new pipeline
	pipeline := b.pipelineFactory()

	// generate a channel id
	cid := b.channelIDFactory()

	// create a channel
	channel := b.channelFactory(cid, b.bootstrapCtx, pipeline, transport)

	// set the attachment if necessary
	if nil != attachment {
		channel.SetAttachment(attachment)
	}

	// Channel Executor
	var chExecutor ChannelExecutor
	if nil != b.executorFactory {
		chExecutor = b.executorFactory(channel.Context())
	}

	// serve the channel
	b.serveChannel(chExecutor, channel, childChannel)
	return channel
}

// createListener to create a listener with options
func (b *bootstrap) createListener(listenOptions ...transport.Option) error {

	// no need to create
	if len(listenOptions) <= 0 {
		return nil
	}

	// parse options
	options, err := transport.ParseOptions(listenOptions...)
	if nil != err {
		return err
	}

	// create a listener with options
	l, err := b.transportFactory.Listen(options)
	if nil != err {
		return err
	}

	// stop the old listener
	b.stopListener()

	b.acceptor = l
	return nil
}

// startListener to accept child transport
func (b *bootstrap) startListener() {

	if nil == b.acceptor {
		return
	}

	go func() {

		for {
			// accept the transport
			t, err := b.acceptor.Accept()
			if nil != err {
				break
			}

			select {
			case <-b.Context().Done():
				// bootstrap has been closed
				_ = t.Close()
				return
			default:
				// serve child transport
				b.serveTransport(t, nil, true)
			}
		}
	}()
}

// stopListener to close the listener
func (b *bootstrap) stopListener() {
	if b.acceptor != nil {
		_ = b.acceptor.Close()
		b.acceptor = nil
	}
}

// Connect to the remote server with options
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

	// connect to remote endpoint
	t, err := b.transportFactory.Connect(options)
	if nil != err {
		return nil, err
	}

	// serve client transport
	return b.serveTransport(t, attachment, false), nil
}

// Listen to the address with options
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

// Action to call action function
func (b *bootstrap) Action(action func(Bootstrap)) Bootstrap {
	action(b)
	return b
}

// Stop the bootstrap
func (b *bootstrap) Stop() Bootstrap {
	b.stopListener()
	b.bootstrapCancel()
	return b
}
