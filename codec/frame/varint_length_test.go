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
	"io"
	"io/ioutil"
	"testing"

	"github.com/go-netty/go-netty"
)

func TestVarintLengthFieldCodec_HandleWrite(t *testing.T) {

	var text = []byte("Hello go-netty")
	var head = [binary.MaxVarintLen64]byte{}
	n := binary.PutUvarint(head[:], uint64(len(text)))
	headBuff := head[:n]

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {

			switch m := message.(type) {
			case [][]byte:

				if !bytes.Equal(m[0], headBuff) {
					t.Fatal(m[0], "!=", headBuff)
				}

				if !bytes.Equal(m[1], text) {
					t.Fatal(m[1], "!=", text)
				}
			default:
				t.Fatal("wrong type", message)
			}
		},
	}

	varintCodec := VarintLengthFieldCodec(1024)
	varintCodec.HandleWrite(ctx, text)
	varintCodec.HandleWrite(ctx, bytes.NewReader(text))
}

func TestVarintLengthFieldCodec_HandleRead(t *testing.T) {

	var text = []byte("Hello go-netty")
	var head = [binary.MaxVarintLen64]byte{}
	n := binary.PutUvarint(head[:], uint64(len(text)))

	packet := append(append([]byte{}, head[:n]...), text...)

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {

			var data []byte

			switch m := message.(type) {
			case io.Reader:
				var err error
				if data, err = ioutil.ReadAll(m); nil != err {
					t.Fatal(err)
				}
			default:
				t.Fatal("wrong type", message)
			}

			if !bytes.Equal(data, text) {
				t.Fatal(data, "!=", text)
			}
		},
	}

	varintCodec := VarintLengthFieldCodec(1024)
	varintCodec.HandleRead(ctx, packet)
	varintCodec.HandleRead(ctx, bytes.NewReader(packet))
}
