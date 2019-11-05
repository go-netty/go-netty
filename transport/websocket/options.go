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

package websocket

import (
	"context"
	"net/http"
	"time"

	"github.com/go-netty/go-netty/transport"
)

var DefaultOptions = &Options{
	Timeout:  time.Second * 5,
	ServeMux: http.DefaultServeMux,
}

type Options struct {
	Timeout  time.Duration  `json:"timeout"`
	Cert     string         `json:"cert"`
	Key      string         `json:"key"`
	Binary   bool           `json:"binary,string"`
	Stream   bool           `json:"stream,string"`
	Routers  []string       `json:"routers"`
	ServeMux *http.ServeMux `json:"-"`
}

const contextKey = "go-netty-transport-websocket-options"

func WithOptions(option *Options) transport.Option {
	return func(options *transport.Options) error {
		options.Context = context.WithValue(options.Context, contextKey, option)
		return nil
	}
}

func FromContext(ctx context.Context, def *Options) *Options {
	if v, ok := ctx.Value(contextKey).(*Options); ok {
		return v
	}
	return def
}
