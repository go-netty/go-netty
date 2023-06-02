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

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

// PacketCodec create packet codec
func PacketCodec(readBuffSize int) codec.Codec {
	return &packetCodec{readBuffer: bytes.NewBuffer(make([]byte, 0, readBuffSize))}
}

type packetCodec struct {
	readBuffer *bytes.Buffer
}

func (packetCodec) CodecName() string {
	return "packet-codec"
}

func (p packetCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {
	// reset read buffer
	p.readBuffer.Reset()
	// read packet
	utils.AssertLong(p.readBuffer.ReadFrom(utils.MustToReader(message)))
	// post packet
	ctx.HandleRead(p.readBuffer)
}

func (packetCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	ctx.HandleWrite(message)
}
