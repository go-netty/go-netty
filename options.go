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
	"sync/atomic"

	"github.com/go-netty/go-netty/transport"
)

type (
	// ChannelInitializer to init the pipeline of channel
	ChannelInitializer func(Channel)
	// ChannelFactory to create a channel
	ChannelFactory func(id int64, ctx context.Context, pipeline Pipeline, transport transport.Transport) Channel
	// PipelineFactory to create pipeline
	PipelineFactory func() Pipeline
	// TransportFactory tp create transport
	TransportFactory transport.Factory
	// ChannelIDFactory to create channel id
	ChannelIDFactory func() int64

	// bootstrapOptions
	bootstrapOptions struct {
		bootstrapCtx      context.Context
		bootstrapCancel   context.CancelFunc
		clientInitializer ChannelInitializer
		childInitializer  ChannelInitializer
		transportFactory  TransportFactory
		channelFactory    ChannelFactory
		pipelineFactory   PipelineFactory
		channelIDFactory  ChannelIDFactory
	}
)

// SequenceID to generate a sequence id
func SequenceID() ChannelIDFactory {
	var id int64
	return func() int64 {
		return atomic.AddInt64(&id, 1)
	}
}

type Option func(options *bootstrapOptions)

// WithContext fork child context with context.WithCancel
func WithContext(ctx context.Context) Option {
	return func(options *bootstrapOptions) {
		options.bootstrapCtx, options.bootstrapCancel = context.WithCancel(ctx)
	}
}

// WithChannelID to set ChannelIDFactory
func WithChannelID(channelIDFactory ChannelIDFactory) Option {
	return func(options *bootstrapOptions) {
		options.channelIDFactory = channelIDFactory
	}
}

// WithPipeline to set PipelineFactory
func WithPipeline(pipelineFactory PipelineFactory) Option {
	return func(options *bootstrapOptions) {
		options.pipelineFactory = pipelineFactory
	}
}

// WithChannel to set ChannelFactory
func WithChannel(channelFactory ChannelFactory) Option {
	return func(options *bootstrapOptions) {
		options.channelFactory = channelFactory
	}
}

// WithTransport to set TransportFactory
func WithTransport(transportFactory TransportFactory) Option {
	return func(options *bootstrapOptions) {
		options.transportFactory = transportFactory
	}
}

// WithChildInitializer to set server side ChannelInitializer
func WithChildInitializer(initializer ChannelInitializer) Option {
	return func(options *bootstrapOptions) {
		options.childInitializer = initializer
	}
}

// WithClientInitializer to set client side ChannelInitializer
func WithClientInitializer(initializer ChannelInitializer) Option {
	return func(options *bootstrapOptions) {
		options.clientInitializer = initializer
	}
}
