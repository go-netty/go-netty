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
	"bufio"
	"bytes"
	"io"
	"net/http"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/utils"
)

type responseCodec struct {
}

func (*responseCodec) CodecName() string {
	return "http-response-codec"
}

func (*responseCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case io.Reader:
		response, err := http.ReadResponse(bufio.NewReader(r), nil)
		utils.Assert(err)
		ctx.HandleRead(response)

		// auto close the response body.
		if response.Body != nil {
			_ = response.Body.Close()
		}
	default:
		ctx.HandleRead(message)
	}
}

func (*responseCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch r := message.(type) {
	case *http.Response:
		buffer := bytes.NewBuffer(nil)
		utils.Assert(r.Write(buffer))
		ctx.HandleWrite(buffer)
	case *responseWriter:
		response := r.response()
		buffer := bytes.NewBuffer(nil)
		utils.Assert(response.Write(buffer))
		ctx.HandleWrite(buffer)
	default:
		ctx.HandleWrite(message)
	}
}

