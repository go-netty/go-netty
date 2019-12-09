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

func TestEofCodec_HandleWrite(t *testing.T) {

	var text = []byte("Hello go-netty")

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {

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

			if !bytes.Equal(msg, text) {
				t.Fatal(msg, "!=", text)
			}
		},
	}

	eofCodec := EofCodec(1024)
	eofCodec.HandleWrite(ctx, bytes.NewReader(text))
}

func TestEofCodec_HandleRead(t *testing.T) {

	var text = []byte("Hello go-netty")

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {

			msg, err := ioutil.ReadAll(message.(io.Reader))
			if nil != err {
				t.Fatal(err)
			}

			if !bytes.Equal(msg, text) {
				t.Fatal(msg, "!=", text)
			}
		},
	}

	eofCodec := EofCodec(1024)
	eofCodec.HandleRead(ctx, bytes.NewReader(text))
}
