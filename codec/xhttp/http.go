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
	"fmt"
	"github.com/go-netty/go-netty/utils"
	"net/http"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
)

// ClientCodec create a http client codec
func ClientCodec() codec.Codec {
	return codec.Combine("http-client-codec", new(responseCodec), new(requestCodec))
}

// ServerCodec create a http server codec
func ServerCodec() codec.Codec {
	return codec.Combine("http-server-codec", new(requestCodec), new(responseCodec))
}

// Handler to convert http.Handler to codec.Codec
func Handler(handler http.Handler) codec.Codec {
	if nil == handler {
		handler = http.DefaultServeMux
	}
	return &handlerAdapter{handler: handler}
}

type handlerAdapter struct {
	handler http.Handler
}

func (*handlerAdapter) CodecName() string {
	return "http-handler-adapter"
}

func (h *handlerAdapter) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case *http.Request:
		// create a response writer
		writer := NewResponseWriter(r.ProtoMajor, r.ProtoMinor)
		// serve the http request
		h.handler.ServeHTTP(writer, r)
		// write the response
		ctx.Write(writer)
	default:
		utils.Assert(fmt.Errorf("unrecognized type: %T", message))
	}
}

func (*handlerAdapter) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
