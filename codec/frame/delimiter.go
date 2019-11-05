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

func DelimiterCodec(maxFrameLength int, delimiter string, stripDelimiter bool) codec.Codec {
	utils.AssertIf(maxFrameLength <= 0, "maxFrameLength must be a positive integer")
	utils.AssertIf(len(delimiter) <= 0, "delimiter must be a non empty string")
	return &delimiterCodec{
		maxFrameLength: maxFrameLength,
		delimiter:      []byte(delimiter),
		stripDelimiter: stripDelimiter,
	}
}

type delimiterCodec struct {
	maxFrameLength int
	delimiter      []byte
	stripDelimiter bool
}

func (*delimiterCodec) CodecName() string {
	return "delimiter-codec"
}

func (d *delimiterCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	var msgReader io.Reader

	switch r := message.(type) {
	case []byte:
		msgReader = bytes.NewReader(r)
	case io.Reader:
		msgReader = r
	default:
		ctx.HandleRead(message)
		return
	}

	readBuff := make([]byte, 0, 16)
	tempBuff := make([]byte, len(d.delimiter))
	for len(readBuff) < d.maxFrameLength {
		// 每次读取len(delimiter)个字节
		n := utils.AssertLength(msgReader.Read(tempBuff[:]))

		// 追加数据
		readBuff = append(readBuff, tempBuff[:n]...)

		// 比较分割符
		if len(readBuff) >= len(d.delimiter) && bytes.Equal(d.delimiter, readBuff[len(readBuff)-len(d.delimiter):]) {

			// 去除分隔符
			if d.stripDelimiter {
				readBuff = readBuff[:len(readBuff)-len(d.delimiter)]
			}

			// 将读取到的数据交由下一个处理器处理
			ctx.HandleRead(bytes.NewReader(readBuff))
			return
		}
	}

	utils.Assert(fmt.Errorf("frame length too large, readBuffLength(%d) >= maxFrameLength(%d)",
		len(readBuff), d.maxFrameLength))

}

func (d *delimiterCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch r := message.(type) {
	case []byte:
		// 使用批量写优化一次内存分配
		ctx.HandleWrite([][]byte{
			// 数据包
			r,
			// 尾部分割符
			d.delimiter,
		})
	case io.Reader:
		ctx.HandleWrite(io.MultiReader(
			// 数据包
			r,
			// 尾部分割符
			bytes.NewReader(d.delimiter)),
		)
	default:
		ctx.HandleWrite(message)
	}
}

