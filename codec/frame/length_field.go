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
	"encoding/binary"
	"fmt"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
	"io"
)

func LengthFieldCodec(
	byteOrder binary.ByteOrder, // 字节序，大端 & 小端
	maxFrameLength int, // 最大允许数据包长度
	lengthFieldOffset int, // 长度域的偏移量，表示跳过指定长度个字节之后的才是长度域
	lengthFieldLength int, // 记录该帧数据长度的字段本身的长度 1, 2, 4, 8
	lengthAdjustment int, // 包体长度调整的大小，长度域的数值表示的长度加上这个修正值表示的就是带header的包长度
	initialBytesToStrip int, // 拿到一个完整的数据包之后向业务解码器传递之前，应该跳过多少字节
) codec.Codec {
	utils.AssertIf(maxFrameLength <= 0, "maxFrameLength must be a positive integer")
	utils.AssertIf(lengthFieldOffset < 0, "lengthFieldOffset must be a non-negative integer")
	utils.AssertIf(initialBytesToStrip < 0, "initialBytesToStrip must be a non-negative integer")
	utils.AssertIf(lengthFieldLength != 1 && lengthFieldLength != 2 &&
		lengthFieldLength != 4 && lengthFieldLength != 8, "lengthFieldLength must be either 1, 2, 3, 4, or 8")
	utils.AssertIf(lengthFieldOffset > maxFrameLength-lengthFieldLength,
		"maxFrameLength must be equal to or greater than lengthFieldOffset + lengthFieldLength")

	return &lengthFieldCodec{
		byteOrder:           byteOrder,
		maxFrameLength:      maxFrameLength,
		lengthFieldOffset:   lengthFieldOffset,
		lengthFieldLength:   lengthFieldLength,
		lengthAdjustment:    lengthAdjustment,
		initialBytesToStrip: initialBytesToStrip,
		prepender:           LengthFieldPrepender(byteOrder, lengthFieldLength, 0, false),
	}
}

type lengthFieldCodec struct {
	byteOrder           binary.ByteOrder
	maxFrameLength      int
	lengthFieldOffset   int
	lengthFieldLength   int
	lengthAdjustment    int
	initialBytesToStrip int

	// default encoder
	prepender netty.OutboundHandler
}

func (*lengthFieldCodec) CodecName() string {
	return "length-field-codec"
}

func (l *lengthFieldCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case io.Reader:

		lengthFieldEndOffset := l.lengthFieldOffset + l.lengthFieldLength

		headerBuffer := make([]byte, lengthFieldEndOffset)
		n, err := io.ReadFull(r, headerBuffer)

		utils.AssertIf(n != len(headerBuffer) || nil != err, "read header fail, headerLength: %d, read: %d, error: %v", len(headerBuffer), n, err)

		lengthFieldBuff := headerBuffer[l.lengthFieldOffset:lengthFieldEndOffset]

		var frameLength int64
		switch l.lengthFieldLength {
		case 1:
			frameLength = int64(lengthFieldBuff[0])
		case 2:
			frameLength = int64(l.byteOrder.Uint16(lengthFieldBuff))
		case 4:
			frameLength = int64(l.byteOrder.Uint32(lengthFieldBuff))
		case 8:
			frameLength = int64(l.byteOrder.Uint64(lengthFieldBuff))
		default:
			utils.Assert(fmt.Errorf("should not reach here"))
		}

		utils.AssertIf(frameLength < 0, "negative pre-adjustment length field: %d", frameLength)

		frameLength += int64(l.lengthAdjustment + lengthFieldEndOffset)

		utils.AssertIf(frameLength < int64(lengthFieldEndOffset),
			"Adjusted frame length (%d) is less than lengthFieldEndOffset: %d", frameLength, lengthFieldEndOffset)

		utils.AssertIf(frameLength > int64(l.maxFrameLength),
			"Frame length too large, frameLength(%d) > maxFrameLength(%d)", frameLength, l.maxFrameLength)

		utils.AssertIf(int64(l.initialBytesToStrip) > frameLength,
			"Adjusted frame length (%d) is less than initialBytesToStrip: %d", frameLength, l.initialBytesToStrip)

		frameReader := io.MultiReader(
			// lengthFieldOffset + lengthFieldLength
			bytes.NewReader(headerBuffer),
			// frameLength - len(headerBuffer)
			io.LimitReader(r, frameLength-int64(lengthFieldEndOffset)),
		)

		// extract frame
		actualFrameLength := frameLength - int64(l.initialBytesToStrip)
		ctx.HandleRead(io.NewSectionReader(utils.ReaderAt(frameReader), int64(l.initialBytesToStrip), actualFrameLength))

	default:
		utils.Assert(fmt.Errorf("unrecognized type: %T", message))
	}
}

func (l *lengthFieldCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	l.prepender.HandleWrite(ctx, message)
}
