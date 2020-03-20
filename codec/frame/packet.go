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
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
	"io"
)

// PacketCodec create packet codec
func PacketCodec(maxFrameLength int) codec.Codec {
	utils.AssertIf(maxFrameLength <= 0, "maxFrameLength must be a positive integer")
	return &packetCodec{maxFrameLength: maxFrameLength, buffer: make([]byte, maxFrameLength)}
}

type packetCodec struct {
	maxFrameLength int
	buffer         []byte
}

func (*packetCodec) CodecName() string {
	return "packet-codec"
}

func (p *packetCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	reader := utils.MustToReader(message)
	n, err := reader.Read(p.buffer)
	utils.AssertIf(nil != err && io.EOF != err, "%v", err)

	ctx.HandleRead(p.buffer[:n])
}

func (*packetCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
