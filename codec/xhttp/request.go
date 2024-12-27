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
	"fmt"
	"io"
	"net/http"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/utils"
)

type requestCodec struct {
}

func (*requestCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case io.Reader:
		bufReader := bufio.NewReader(r)
		for {
			request, err := http.ReadRequest(bufReader)
			utils.Assert(err)
			// TODO: replace request context by the channel context
			//
			ctx.HandleRead(request)
			// Close indicates whether to close the connection after
			// replying to this request
			if request.Close {
				ctx.Close(fmt.Errorf("request mark closed"))
				return
			}
		}
	default:
		ctx.HandleRead(message)
	}
}

func (*requestCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch r := message.(type) {
	case *http.Request:
		utils.Assert(r.Write(ctx.Channel().Writer()))
	default:
		ctx.HandleWrite(message)
	}
}
