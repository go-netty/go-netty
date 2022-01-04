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
	Connect(url string, attachment Attachment, option ...transport.Option) (Channel, error)
	// Shutdown boostrap
	Shutdown()
}

// NewBootstrap create a new Bootstrap with default config.
func NewBootstrap(option ...Option) Bootstrap {

	opts := &bootstrapOptions{
		channelIDFactory: SequenceID(),
		pipelineFactory:  NewPipeline(),
		channelFactory:   NewChannel(128),
		transportFactory: tcp.New(),
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
func (bs *bootstrap) serveTransport(transport transport.Transport, attachment Attachment, childChannel bool) Channel {

	// create a new pipeline
	pipeline := bs.pipelineFactory()

	// generate a channel id
	cid := bs.channelIDFactory()

	// create a channel
	channel := bs.channelFactory(cid, bs.bootstrapCtx, pipeline, transport)

	// set the attachment if necessary
	if nil != attachment {
		channel.SetAttachment(attachment)
	}

	// initialization pipeline
	if childChannel {
		bs.childInitializer(channel)
	} else {
		bs.clientInitializer(channel)
	}

	// serve channel.
	channel.Pipeline().ServeChannel(channel)
	return channel
}

// Connect to the remote server with options
func (bs *bootstrap) Connect(url string, attachment Attachment, option ...transport.Option) (Channel, error) {

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
	return bs.serveTransport(t, attachment, false), nil
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
			l.bs.serveTransport(t, nil, true)
		}
	}
}

func (l *listener) Async(fn func(err error)) {
	go func() {
		fn(l.Sync())
	}()
}
