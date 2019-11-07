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

package format

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/go-netty/go-netty"
)

func TestTextCodec_HandleWrite(t *testing.T) {

	const text = "Hello go-netty!!!"

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {
			switch m := message.(type) {
			case string:
				if m != text {
					t.Fatal(m, "!=", text)
				}
			case []byte:
				if ds := string(m); ds != text {
					t.Fatal(ds, "!=", text)
				}
			case io.Reader:
				dst, err := ioutil.ReadAll(m)
				if nil != err {
					t.Fatal(err)
				}
				if ds := string(dst); ds != text {
					t.Fatal(ds, "!=", text)
				}
			default:
				t.Fatal("wrong type", message)
			}
		},
	}

	textCodec := TextCodec()
	textCodec.HandleWrite(ctx, text)
	textCodec.HandleWrite(ctx, []byte(text))
	textCodec.HandleWrite(ctx, strings.NewReader(text))
}

func TestTextCodec_HandleRead(t *testing.T) {

	const text = "Hello go-netty!!!"

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {
			switch m := message.(type) {
			case string:
				if m != text {
					t.Fatal(m, "!=", text)
				}
			case io.Reader:
				dst, err := ioutil.ReadAll(m)
				if nil != err {
					t.Fatal(err)
				}
				if ds := string(dst); ds != text {
					t.Fatal(ds, "!=", text)
				}
			default:
				t.Fatal("wrong type", message)
			}
		},
	}

	textCodec := TextCodec()
	textCodec.HandleRead(ctx, text)
	textCodec.HandleRead(ctx, []byte(text))
	textCodec.HandleRead(ctx, strings.NewReader(text))
}
