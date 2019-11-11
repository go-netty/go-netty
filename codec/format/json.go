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
	"fmt"
	"io"
	"strings"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

func JsonCodec(useNumber bool, disAllowUnknownFields bool) codec.Codec {
	return &jsonCodec{
		useNumber:             useNumber,
		disAllowUnknownFields: disAllowUnknownFields,
	}
}

type jsonCodec struct {
	useNumber             bool
	disAllowUnknownFields bool
}

func (*jsonCodec) CodecName() string {
	return "json-codec"
}

func (j *jsonCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	var msgReader io.Reader

	switch r := message.(type) {
	case string:
		msgReader = strings.NewReader(r)
	case []byte:
		msgReader = bytes.NewReader(r)
	case io.Reader:
		msgReader = r
	default:
		utils.Assert(fmt.Errorf("unsupported type: %T", message))
		return
	}

	jsonDecoder := json.NewDecoder(msgReader)

	if j.useNumber {
		jsonDecoder.UseNumber()
	}

	if j.disAllowUnknownFields {
		jsonDecoder.DisallowUnknownFields()
	}

	// 解析到万能结构
	var object = make(map[string]interface{})
	utils.Assert(jsonDecoder.Decode(&object))

	// 交给下一个处理器处理
	ctx.HandleRead(object)
}

func (j *jsonCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch message.(type) {
	case io.Reader, []byte, string:
		ctx.HandleWrite(message)
	default:
		// 序列化数据
		data := utils.AssertBytes(json.Marshal(message))
		// 传递给下一个处理器处理
		ctx.HandleWrite(bytes.NewReader(data))
	}
}
