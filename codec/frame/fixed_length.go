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
	"fmt"
	"io"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

// FixedLengthCodec create fixed length codec
func FixedLengthCodec(length int) codec.Codec {
	utils.AssertIf(length <= 0, "invalid fixed length")
	return &fixedLengthCodec{length}
}

type fixedLengthCodec struct {
	length int
}

func (*fixedLengthCodec) CodecName() string {
	return "fixed-length-codec"
}

func (f *fixedLengthCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	// 读取指定长度的消息，并传递给下一个处理器处理
	switch r := message.(type) {
	case io.Reader:
		ctx.HandleRead(io.LimitReader(r, int64(f.length)))
	default:
		utils.Assert(fmt.Errorf("unrecognized type: %T", message))
	}
}

func (f *fixedLengthCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	// 直接交由下一个处理器处理
	ctx.HandleWrite(message)
}
