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

package transport

import (
	"context"
	"net"
	"net/url"

	"github.com/go-netty/go-netty/utils"
)

type Option func(options *Options) error

type Options struct {
	Address *url.URL

	Context context.Context
}

func(lo *Options) AddressWithoutHost() string {
	_, port, err := net.SplitHostPort(lo.Address.Host)
	utils.Assert(err)
	return net.JoinHostPort("", port)
}

func (lo *Options) Apply(options ...Option) error {
	for _, option := range options {
		if err := option(lo); nil != err {
			return err
		}
	}
	return nil
}

func ParseOptions(options ...Option) (*Options, error) {
	option := &Options{Context: context.Background()}
	return option, option.Apply(options...)
}

func WithAddress(address string) Option {
	return func(options *Options) (err error) {
		options.Address, err = url.Parse(address)
		return err
	}
}

func WithContext(ctx context.Context) Option {
	return func(options *Options) error {
		options.Context = ctx
		return nil
	}
}
