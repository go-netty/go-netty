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

package format

import (
	"github.com/go-netty/go-netty/utils"
	"strings"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
)

// TextCodec create a text codec
func TextCodec() codec.Codec {
	return &textCodec{}
}

type textCodec struct{}

func (*textCodec) CodecName() string {
	return "text-codec"
}

func (*textCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	// read text bytes
	textBytes := utils.MustToBytes(message)

	// convert from []byte to string
	sb := strings.Builder{}
	sb.Write(textBytes)

	// post text
	ctx.HandleRead(sb.String())
}

func (*textCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch s := message.(type) {
	case string:
		ctx.HandleWrite(strings.NewReader(s))
	default:
		ctx.HandleWrite(message)
	}
}
