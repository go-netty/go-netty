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
	"fmt"
	"sync"

	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/transport/tcp"
)

// Bootstrap makes it easy to bootstrap a channel
type Bootstrap interface {
	// Context return context
	Context() context.Context
	// Listen create a listener
	Listen(url string, option ...transport.Option) Listener
	// Connect to remote endpoint
	Connect(url string, option ...transport.Option) (Channel, error)
	// Shutdown boostrap
	Shutdown()
}

// NewBootstrap create a new Bootstrap with default config.
func NewBootstrap(option ...Option) Bootstrap {

	opts := &bootstrapOptions{
		channelIDFactory: SequenceID(),
		pipelineFactory:  NewPipeline,
		channelFactory:   NewChannel(64),
		transportFactory: tcp.New(),
		executor:         AsyncExecutor(),
	}
	opts.bootstrapCtx, opts.bootstrapCancel = context.WithCancel(context.Background())

	for i := range option {
		option[i](opts)
	}

	return &bootstrap{bootstrapOptions: opts}
}

// bootstrap implement
type bootstrap struct {
	*bootstrapOptions
	listeners sync.Map // url - Listener
}

// Context to get context
func (bs *bootstrap) Context() context.Context {
	return bs.bootstrapCtx
}

// serveTransport to serve channel
func (bs *bootstrap) serveTransport(ctx context.Context, transport transport.Transport, attachment Attachment, childChannel bool) Channel {

	// create a new pipeline
	pl := bs.pipelineFactory()

	// generate a channel id
	cid := bs.channelIDFactory()

	// create a channel
	ch := bs.channelFactory(cid, ctx, pl, transport, bs.executor)

	// set the attachment if necessary
	if nil != attachment {
		ch.SetAttachment(attachment)
	}

	// initialization pipeline
	if childChannel {
		bs.childInitializer(ch)
	} else {
		bs.clientInitializer(ch)
	}

	// serve channel.
	ch.Pipeline().ServeChannel(ch)
	return ch
}

// Connect to the remote server with options
func (bs *bootstrap) Connect(url string, option ...transport.Option) (Channel, error) {

	options, err := transport.ParseOptions(bs.Context(), url, option...)
	if nil != err {
		return nil, err
	}

	// connect to remote endpoint
	t, err := bs.transportFactory.Connect(options)
	if nil != err {
		return nil, err
	}

	// serve client transport
	return bs.serveTransport(options.Context, t, options.Attachment, false), nil
}

// Listen to the address with options
func (bs *bootstrap) Listen(url string, option ...transport.Option) Listener {
	l := &listener{bs: bs, url: url, option: option}
	bs.listeners.Store(url, l)
	return l
}

// Shutdown the bootstrap
func (bs *bootstrap) Shutdown() {
	bs.bootstrapCancel()

	bs.listeners.Range(func(key, value interface{}) bool {
		value.(Listener).Close()
		return true
	})
}

// removeListener close the listener with url
func (bs *bootstrap) removeListener(url string) {
	bs.listeners.Delete(url)
}

type Listener interface {
	// Close the listener
	Close() error
	// Sync waits for this listener until it is done
	Sync() error
	// Async nonblock waits for this listener
	Async(func(error))
}

// impl Listener
type listener struct {
	bs       *bootstrap
	url      string
	option   []transport.Option
	options  *transport.Options
	acceptor transport.Acceptor
}

// Close listener
func (l *listener) Close() error {
	if l.acceptor != nil {
		l.bs.removeListener(l.url)
		return l.acceptor.Close()
	}
	return nil
}

// Sync accept new transport from listener
func (l *listener) Sync() error {

	if nil != l.acceptor {
		return fmt.Errorf("duplicate call Listener:Sync")
	}

	var err error
	if l.options, err = transport.ParseOptions(l.bs.Context(), l.url, l.option...); nil != err {
		return err
	}

	if l.acceptor, err = l.bs.transportFactory.Listen(l.options); nil != err {
		return err
	}

	for {
		// accept the transport
		t, err := l.acceptor.Accept()
		if nil != err {
			return err
		}

		select {
		case <-l.bs.Context().Done():
			// bootstrap has been closed
			return t.Close()
		default:
			// serve child transport
			l.bs.serveTransport(l.options.Context, t, l.options.Attachment, true)
		}
	}
}

// Async accept new transport from listener
func (l *listener) Async(fn func(err error)) {
	l.bs.executor.Exec(func() {
		fn(l.Sync())
	})
}
