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
	"net/http"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/utils"
)

type responseCodec struct {
}

func (*responseCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {
	reader := utils.MustToReader(message)
	response, err := http.ReadResponse(bufio.NewReader(reader), nil)
	utils.Assert(err)
	ctx.HandleRead(response)
}

func (*responseCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch r := message.(type) {
	case *http.Response:
		utils.Assert(r.Write(ctx.Channel().Writer()))
	case ResponseWriter:
	default:
		utils.Assert(fmt.Errorf("unrecognized type: %T", message))
	}
}
