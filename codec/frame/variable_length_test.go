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
	"testing"

	"github.com/go-netty/go-netty"
)

func TestVariableLengthCodec_HandleWrite(t *testing.T) {

	var text = []byte("Hello go-netty")

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {

			switch m := message.(type) {
			case []byte:
				if !bytes.Equal(m, text) {
					t.Fatal(m, "!=", text)
				}

			default:
				t.Fatal("wrong type", message)
			}
		},
	}

	varLengthCodec := VariableLengthCodec(5)
	varLengthCodec.HandleWrite(ctx, text)

}

func TestVariableLengthCodec_HandleRead(t *testing.T) {

	var text = []byte("Hello go-netty")

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {

			switch m := message.(type) {
			case []byte:

				if len(m) > 5 {
					t.Fatal("wrong length", len(m))
				}

				if !bytes.Equal(m, text[:len(m)]) {
					t.Fatal(m, "!=", text[:len(m)])
				}

			default:
				t.Fatal("wrong type", message)
			}
		},
	}

	varLengthCodec := VariableLengthCodec(5)
	varLengthCodec.HandleRead(ctx, bytes.NewReader(text))

}
