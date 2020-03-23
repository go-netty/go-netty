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
	"github.com/go-netty/go-netty/utils"
	"testing"

	"github.com/go-netty/go-netty"
)

func TestLengthFieldCodec(t *testing.T) {

	var cases = []struct {
		byteOrder        binary.ByteOrder
		maxFrameLen      int
		fieldOffset      int
		fieldLen         int
		lengthAdjustment int
		bytesToStrip     int
		input            []byte
		output           interface{}
	}{
		{byteOrder: binary.LittleEndian, maxFrameLen: 1024, fieldLen: 1, lengthAdjustment: 0, bytesToStrip: 1, output: []byte("123456789")},
		{byteOrder: binary.LittleEndian, maxFrameLen: 1024, fieldLen: 2, lengthAdjustment: 0, bytesToStrip: 2, output: []byte("123456789")},
		{byteOrder: binary.LittleEndian, maxFrameLen: 1024, fieldLen: 4, lengthAdjustment: 0, bytesToStrip: 4, output: []byte("123456789")},
		{byteOrder: binary.LittleEndian, maxFrameLen: 1024, fieldLen: 8, lengthAdjustment: 0, bytesToStrip: 8, output: []byte("123456789")},

		{byteOrder: binary.BigEndian, maxFrameLen: 1024, fieldLen: 1, lengthAdjustment: 0, bytesToStrip: 1, output: []byte("123456789")},
		{byteOrder: binary.BigEndian, maxFrameLen: 1024, fieldLen: 2, lengthAdjustment: 0, bytesToStrip: 2, output: []byte("123456789")},
		{byteOrder: binary.BigEndian, maxFrameLen: 1024, fieldLen: 4, lengthAdjustment: 0, bytesToStrip: 4, output: []byte("123456789")},
		{byteOrder: binary.BigEndian, maxFrameLen: 1024, fieldLen: 8, lengthAdjustment: 0, bytesToStrip: 8, output: []byte("123456789")},
	}

	for _, c := range cases {
		codec := LengthFieldCodec(c.byteOrder, c.maxFrameLen, c.fieldOffset, c.fieldLen, c.lengthAdjustment, c.bytesToStrip)
		t.Run(codec.CodecName(), func(t *testing.T) {
			ctx := MockHandlerContext{
				MockHandleRead: func(message netty.Message) {
					if dst := utils.MustToBytes(message); !bytes.Equal(dst, c.input[c.bytesToStrip:]) {
						t.Fatalf("%v != %v", dst, c.input[c.bytesToStrip:])
					}
				},

				MockHandleWrite: func(message netty.Message) {
					c.input = utils.MustToBytes(message)
				},
			}
			codec.HandleWrite(ctx, c.output)
			codec.HandleRead(ctx, c.input)
		})
	}
}
