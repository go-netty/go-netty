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

package xhttp

import (
	"net/http"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
)

// http客户端编解码器
// 发送请求 -> 解码回应
func ClientCodec() codec.Codec {
	return codec.Combine("http-client-codec", new(responseCodec), new(requestCodec))
}

// http服务端编解码器
// 读取请求 -> 写入回应
func ServerCodec() codec.Codec {
	return codec.Combine("http-server-codec", new(requestCodec), new(responseCodec))
}

// 适配http标准Handler
func Handler(handler http.Handler) codec.Codec {
	if nil == handler {
		handler = http.DefaultServeMux
	}
	return &handlerAdapter{_handler: handler}
}

type handlerAdapter struct {
	_handler http.Handler
}

func (*handlerAdapter) CodecName() string {
	return "http-handler-adapter"
}

func (h *handlerAdapter) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case *http.Request:
		// 准备ResponseWriter
		writer := NewResponseWriter(r.ProtoMajor, r.ProtoMinor)
		// 回调处理器
		h._handler.ServeHTTP(writer, r)
		// 写回响应
		ctx.Write(writer)
	default:
		ctx.HandleRead(message)
	}
}

func (*handlerAdapter) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
