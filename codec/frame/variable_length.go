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

package frame

import (
	"io"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

func VariableLengthCodec(maxReadLength int) codec.Codec {
	return &variableLengthCodec{maxReadLength: maxReadLength}
}

type variableLengthCodec struct {
	maxReadLength int // maximum received length
}

func (*variableLengthCodec) CodecName() string {
	return "variable-length-codec"
}

func (v *variableLengthCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case io.Reader:
		var buff = make([]byte, v.maxReadLength)
		var n = utils.AssertLength(r.Read(buff))
		ctx.HandleRead(buff[:n])
	default:
		ctx.HandleRead(message)
	}
}

func (*variableLengthCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
