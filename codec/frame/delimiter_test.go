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
	"github.com/go-netty/go-netty/utils"
	"strings"
	"testing"

	"github.com/go-netty/go-netty"
)

func TestDelimiterCodec(t *testing.T) {

	var cases = []struct {
		delimiter      string
		stripDelimiter bool
		input          []byte
		output         interface{}
	}{
		{delimiter: "\n", stripDelimiter: true, output: []byte("123456789")},
		{delimiter: "\n", stripDelimiter: false, output: []byte("123456789")},
		{delimiter: "\n", stripDelimiter: true, output: []byte("123456789")},
		{delimiter: "\n", stripDelimiter: false, output: []byte("123456789")},
		{delimiter: "\r\n", stripDelimiter: true, output: []byte("123456789")},
		{delimiter: "\r\n", stripDelimiter: false, output: []byte("123456789")},
		{delimiter: "\r\n", stripDelimiter: true, output: []byte("123456789")},
		{delimiter: "\r\n", stripDelimiter: false, output: []byte("123456789")},
		{delimiter: "\n", stripDelimiter: false, output: strings.NewReader("123456789")},
		{delimiter: "\r\n", stripDelimiter: true, output: strings.NewReader("123456789")},
	}

	for index, c := range cases {
		codec := DelimiterCodec(1024, c.delimiter, c.stripDelimiter)
		t.Run(fmt.Sprint(codec.CodecName(), "#", index), func(t *testing.T) {
			ctx := netty.MockHandlerContext{
				MockHandleRead: func(message netty.Message) {
					if c.stripDelimiter {
						c.input = bytes.TrimRight(c.input, c.delimiter)
					}
					if dst := utils.MustToBytes(message); !bytes.Equal(dst, c.input) {
						t.Fatal(dst, "!=", c.input)
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
