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
	"fmt"
	"io"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

func PacketCodec(maxFrameLength int) codec.Codec {
	utils.AssertIf(maxFrameLength <= 0, "maxFrameLength must be a positive integer")
	return &packetCodec{maxFrameLength: maxFrameLength}
}

type packetCodec struct {
	maxFrameLength int // 最大允许数据包长度
	buffer         []byte
}

func (*packetCodec) CodecName() string {
	return "packet-codec"
}

func (p *packetCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	// alloc packet buffer.
	if nil == p.buffer {
		p.buffer = make([]byte, p.maxFrameLength)
	}

	switch r := message.(type) {
	case io.Reader:
		n, err := r.Read(p.buffer)
		if nil != err && err != io.EOF {
			panic(err)
		}

		ctx.HandleRead(bytes.NewReader(p.buffer[:n]))
	default:
		utils.Assert(fmt.Errorf("unrecognized type: %T", message))
	}
}

func (*packetCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
