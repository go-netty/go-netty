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
	"io/ioutil"
	"testing"

	"github.com/go-netty/go-netty"
)

func TestDelimiterCodec_HandleWrite(t *testing.T) {

	var src = []byte("Hello go-netty")
	var dst = append(append([]byte{}, src...), '\n')

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {

			var msg []byte

			switch m := message.(type) {
			case [][]byte:
				msg = append(msg, m[0]...)
				msg = append(msg, m[1]...)
			case io.Reader:
				var err error
				if msg, err = ioutil.ReadAll(m); nil != err {
					t.Fatal(err)
				}
			default:
				t.Fatal("wrong type", message)
			}

			if !bytes.Equal(msg, dst) {
				t.Fatal(msg, "!=", dst)
			}
		},
	}

	delimiterCodec := DelimiterCodec(1024, "\n", true)
	delimiterCodec.HandleWrite(ctx, src)
	delimiterCodec.HandleWrite(ctx, bytes.NewReader(src))
}

func TestDelimiterCodec_HandleRead(t *testing.T) {

	var src = []byte("Hello go-netty")
	var dst = append(append([]byte{}, src...), '\n')

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {

			var msg []byte

			switch m := message.(type) {
			case io.Reader:
				var err error
				if msg, err = ioutil.ReadAll(m); nil != err {
					t.Fatal(err)
				}
			default:
				t.Fatal("wrong type", message)
			}

			if !bytes.Equal(msg, src) {
				t.Fatal(msg, "!=", src)
			}
		},
	}

	delimiterCodec := DelimiterCodec(1024, "\n", true)
	delimiterCodec.HandleRead(ctx, dst)
	delimiterCodec.HandleRead(ctx, bytes.NewReader(dst))
}
