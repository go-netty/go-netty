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
	"io"
	"io/ioutil"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
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
	}
}

type lengthFieldCodec struct {
	byteOrder           binary.ByteOrder
	maxFrameLength      int
	lengthFieldOffset   int
	lengthFieldLength   int
	lengthAdjustment    int
	initialBytesToStrip int
}

func (*lengthFieldCodec) CodecName() string {
	return "length-field-codec"
}

func (l *lengthFieldCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case io.Reader:

		// 读取包头
		headerBuffer := make([]byte, l.lengthFieldOffset+l.lengthFieldLength+l.initialBytesToStrip)
		n, err := io.ReadFull(r, headerBuffer)
		utils.AssertIf(n != len(headerBuffer) || nil != err, "read header fail, headerLength: %d, read: %d, error: %v", len(headerBuffer), n, err)

		// 长度域数据
		lengthFieldBuff := headerBuffer[l.lengthFieldOffset : l.lengthFieldOffset+l.lengthFieldLength]

		// 开始读取长度域
		var frameLength uint64
		switch l.lengthFieldLength {
		case 1:
			frameLength = uint64(lengthFieldBuff[0])
		case 2:
			frameLength = uint64(l.byteOrder.Uint16(lengthFieldBuff))
		case 4:
			frameLength = uint64(l.byteOrder.Uint32(lengthFieldBuff))
		case 8:
			frameLength = uint64(l.byteOrder.Uint64(lengthFieldBuff))
		default:
			utils.Assert(fmt.Errorf("should not reach here"))
		}

		utils.AssertIf(frameLength > uint64(l.maxFrameLength),
			"frame length too large, frameLength(%d) > maxFrameLength(%d)", frameLength, l.maxFrameLength)

		// 调整数据包位置
		if l.lengthAdjustment != 0 {
			// 回退时最多可以回退到包头的位置
			utils.AssertIf(l.lengthAdjustment < 0 && (l.lengthAdjustment+len(headerBuffer)) < 0,
				"lengthAdjustment(%d) > (initialBytesToStrip(%d) + lengthFieldLength(%d) + lengthFieldOffset(%d))",
				-l.lengthAdjustment, l.initialBytesToStrip, l.lengthFieldLength, l.lengthFieldOffset)
			// 前进时最多可以前进到包尾的位置
			utils.AssertIf(l.lengthAdjustment > 0 && l.lengthAdjustment > int(int64(frameLength-uint64(l.initialBytesToStrip))),
				"lengthAdjustment(%d) < (frameLength(%d) - initialBytesToStrip(%d))",
				l.lengthAdjustment, frameLength, l.initialBytesToStrip)

			// 处理回退和前进
			if l.lengthAdjustment < 0 {
				// 处理后退
				ctx.HandleRead(io.MultiReader(
					// 回退的数据
					bytes.NewReader(headerBuffer[len(headerBuffer)+l.lengthAdjustment:]),
					// 未读取的数据
					io.LimitReader(r, int64(frameLength-uint64(l.initialBytesToStrip))),
				))
			} else {
				// 处理前进
				ctx.HandleRead(io.NewSectionReader(
					// 未读取的数据
					utils.ReaderAt(r),
					// 跳过的字节
					int64(l.lengthAdjustment),
					// 计算剩余的包长度
					int64(frameLength-uint64(l.initialBytesToStrip))-int64(l.lengthAdjustment)),
				)
			}

			// 已经处理过前进或者回退
			return
		}

		// 不需要前进和后退的数据包
		// 将解析出来的数据包交由下一个处理器处理
		ctx.HandleRead(io.LimitReader(r, int64(frameLength-uint64(l.initialBytesToStrip))))
	default:
		// 不认识的类型，交由下一个处理器处理
		ctx.HandleRead(message)
	}
}

func (l *lengthFieldCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	var bodyBytes []byte

	switch r := message.(type) {
	case []byte:
		bodyBytes = r
	case io.Reader:
		bodyBytes = utils.AssertBytes(ioutil.ReadAll(r))
	default:
		ctx.HandleWrite(message)
		return
	}

	utils.AssertIf(len(bodyBytes) > l.maxFrameLength,
		"frame length too large, frameLength(%d) > maxFrameLength(%d)", len(bodyBytes), l.maxFrameLength)

	lengthBuff := make([]byte, l.lengthFieldLength)

	switch l.lengthFieldLength {
	case 1:
		lengthBuff[0] = byte(len(bodyBytes))
	case 2:
		l.byteOrder.PutUint16(lengthBuff, uint16(len(bodyBytes)))
	case 4:
		l.byteOrder.PutUint32(lengthBuff, uint32(len(bodyBytes)))
	case 8:
		l.byteOrder.PutUint64(lengthBuff, uint64(len(bodyBytes)))
	default:
		utils.Assert(fmt.Errorf("should not reach here"))
	}

	// 优化掉一次组包操作，降低内存分配操作
	// 批量写数据
	ctx.HandleWrite([][]byte{lengthBuff, bodyBytes})
}
