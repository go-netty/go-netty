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
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"testing"

	"github.com/go-netty/go-netty"
)

func TestJsonCodec_HandleWrite(t *testing.T) {

	data, err := json.Marshal(map[string]interface{}{"key": "1234567890", "value": 123456789})
	if nil != err {
		t.Fatal(err)
	}

	ctx := netty.MockOutboundContext{
		MockHandleWrite: func(message netty.Message) {
			var dst []byte
			switch m := message.(type) {
			case io.Reader:
				var err error
				if dst, err = ioutil.ReadAll(m); nil != err {
					t.Fatal(err)
				}
			case string:
				dst = []byte(m)
			case []byte:
				dst = m
			default:
				t.Fatal("wrong type", message)
			}

			if !bytes.Equal(data, dst) {
				t.Fatal(data, "!=", dst)
			}
		},
	}

	jsonCodec1 := JSONCodec(true, false)
	jsonCodec1.HandleWrite(ctx, data)

	jsonCodec2 := JSONCodec(false, false)
	jsonCodec2.HandleWrite(ctx, data)

	jsonCodec3 := JSONCodec(true, true)
	jsonCodec3.HandleWrite(ctx, data)

	jsonCodec4 := JSONCodec(false, true)
	jsonCodec4.HandleWrite(ctx, data)
}

func TestJsonCodec_HandleRead(t *testing.T) {

	data, err := json.Marshal(map[string]interface{}{"key": "1234567890", "value": 123456789})
	if nil != err {
		t.Fatal(err)
	}

	ctx := netty.MockInboundContext{
		MockHandleRead: func(message netty.Message) {
			dst, err := json.Marshal(message.(map[string]interface{}))
			if nil != err {
				t.Fatal(err)
			}

			if !bytes.Equal(data, dst) {
				t.Fatal(data, "!=", dst)
			}
		},
	}

	jsonCodec1 := JSONCodec(true, false)
	jsonCodec1.HandleRead(ctx, data)

	jsonCodec2 := JSONCodec(false, false)
	jsonCodec2.HandleRead(ctx, data)

	jsonCodec3 := JSONCodec(true, true)
	jsonCodec3.HandleRead(ctx, data)

	jsonCodec4 := JSONCodec(false, true)
	jsonCodec4.HandleRead(ctx, data)
}
