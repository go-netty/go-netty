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
	"io/ioutil"
)

// LengthFieldCodec create a length field based codec
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
		OutboundHandler:     LengthFieldPrepender(byteOrder, lengthFieldLength, 0, false),
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
	netty.OutboundHandler
}

func (*lengthFieldCodec) CodecName() string {
	return "length-field-codec"
}

func (l *lengthFieldCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	// unwrap to reader
	reader := utils.MustToReader(message)

	// lengthFieldEndOffset
	lengthFieldEndOffset := l.lengthFieldOffset + l.lengthFieldLength

	headerBuffer := make([]byte, lengthFieldEndOffset)
	n, err := io.ReadFull(reader, headerBuffer)

	utils.AssertIf(n != len(headerBuffer) || nil != err, "read header fail, headerLength: %d, read: %d, error: %v", len(headerBuffer), n, err)

	lengthFieldBuff := headerBuffer[l.lengthFieldOffset:lengthFieldEndOffset]

	frameLength := unpackFieldLength(l.byteOrder, l.lengthFieldLength, lengthFieldBuff)

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
		io.LimitReader(reader, frameLength-int64(lengthFieldEndOffset)),
	)

	// strip bytes
	if l.initialBytesToStrip > 0 {
		n, err := io.CopyN(ioutil.Discard, frameReader, int64(l.initialBytesToStrip))
		utils.AssertIf(nil != err || int64(l.initialBytesToStrip) != n, "initialBytesToStrip: %d -> %d, %v", l.initialBytesToStrip, n, err)
	}

	ctx.HandleRead(frameReader)
}

func unpackFieldLength(byteOrder binary.ByteOrder, fieldLen int, buff []byte) (frameLength int64) {
	switch fieldLen {
	case 1:
		frameLength = int64(buff[0])
	case 2:
		frameLength = int64(byteOrder.Uint16(buff))
	case 4:
		frameLength = int64(byteOrder.Uint32(buff))
	case 8:
		frameLength = int64(byteOrder.Uint64(buff))
	default:
		utils.Assert(fmt.Errorf("should not reach here"))
	}
	return
}

func packFieldLength(byteOrder binary.ByteOrder, fieldLen int, dataLen int64) []byte {
	lengthBuff := make([]byte, fieldLen)
	switch fieldLen {
	case 1:
		lengthBuff[0] = byte(dataLen)
	case 2:
		byteOrder.PutUint16(lengthBuff, uint16(dataLen))
	case 4:
		byteOrder.PutUint32(lengthBuff, uint32(dataLen))
	case 8:
		byteOrder.PutUint64(lengthBuff, uint64(dataLen))
	default:
		utils.Assert(fmt.Errorf("should not reach here"))
	}
	return lengthBuff
}
