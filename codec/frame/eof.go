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
	"bytes"
	"io"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

func EofCodec(maxFrameLength int) codec.Codec {
	utils.AssertIf(maxFrameLength <= 0, "maxFrameLength must be a positive integer")
	return &eofCodec{maxFrameLength: maxFrameLength}
}

type eofCodec struct {
	maxFrameLength int // 最大允许数据包长度
}

func (*eofCodec) CodecName() string {
	return "eof-codec"
}

func (p *eofCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case io.Reader:
		// 数据包模式直接需要读取到EOF，代表当前数据包已经读完
		var packetBuffer = new(bytes.Buffer)
		// 需要处理最大包字节， 最大允许读取maxFrameLength, 超出则报错，防止超大数据包导致内存溢出
		r = utils.NewMaxBytesReader(r, p.maxFrameLength)
		utils.AssertLong(packetBuffer.ReadFrom(r))

		// 将读取的包丢给下一个处理器处理
		ctx.HandleRead(packetBuffer)
	default:
		ctx.HandleRead(message)
	}
}

func (*eofCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
