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

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

func TextCodec() codec.Codec {
	return &textCodec{}
}

type textCodec struct{}

func (*textCodec) CodecName() string {
	return "text-codec"
}

func (*textCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	switch r := message.(type) {
	case []byte:
		// 二进制转字符串
		ctx.HandleRead(string(r))
	case io.Reader:
		// 读取所有的包二进制
		data := utils.AssertBytes(ioutil.ReadAll(r))
		// 转换为字符串传递给下文
		ctx.HandleRead(string(data))
	default:
		// 不认识的类型传递给下一个处理器处理
		ctx.HandleRead(message)
	}
}

func (*textCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	switch r := message.(type) {
	case string:
		// 字符串转换为二进制流传递给下一个处理器
		ctx.HandleWrite(strings.NewReader(r))
	default:
		// 不认识的类型传递给下一个处理器处理
		ctx.HandleWrite(message)
	}
}
