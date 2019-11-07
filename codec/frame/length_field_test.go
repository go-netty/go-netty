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

func TestLengthFieldCodec_HandleWrite(t *testing.T) {

	var text = []byte("Hello go-netty")

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {

			switch m := message.(type) {
			case [][]byte:

				n := binary.LittleEndian.Uint16(m[0])
				if int(n) != len(text) {
					t.Fatal(n, "!=", len(text))
				}

				if !bytes.Equal(m[1], text) {
					t.Fatal(m[1], "!=", text)
				}

			default:
				t.Fatal("wrong type", message)
			}
		},
	}

	lengthFieldCodec := LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)
	lengthFieldCodec.HandleWrite(ctx, text)
	lengthFieldCodec.HandleWrite(ctx, bytes.NewReader(text))
}

func TestLengthFieldCodec_HandleRead(t *testing.T) {

	var text = []byte("Hello go-netty")
	var lengthBuff [2]byte
	binary.LittleEndian.PutUint16(lengthBuff[:], uint16(len(text)))

	var packet = append(append([]byte{}, lengthBuff[:]...), text...)

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {

			data, err := ioutil.ReadAll(message.(io.Reader))
			if nil != err {
				t.Fatal(err)
			}

			if !bytes.Equal(data, text) {
				t.Fatal(data, "!=", text)
			}
		},
	}

	lengthFieldCodec := LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)
	lengthFieldCodec.HandleRead(ctx, bytes.NewReader(packet))
}
