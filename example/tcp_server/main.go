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

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/format"
	"github.com/go-netty/go-netty/codec/frame"
	"github.com/go-netty/go-netty/transport/tcp"
	"github.com/go-netty/go-netty/utils"
)

func main() {

	// 创建bootstrap
	var bootstrap = netty.NewBootstrap()

	// 配置服务器连接的编解码器
	bootstrap.ChildInitializer(func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(frame.LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)).
			AddLast(format.TextCodec()).
			AddLast(LogHandler{"Server"})
	})

	// 配置客户端连接的编解码器
	bootstrap.ClientInitializer(func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(frame.LengthFieldCodec(binary.LittleEndian, 1024, 0, 2, 0, 0)).
			AddLast(format.TextCodec()).
			AddLast(LogHandler{"Client"})
	})

	// 稍后建立客户端连接
	time.AfterFunc(time.Second, func() {
		_, err := bootstrap.Connect("tcp://127.0.0.1:6565", nil)
		utils.Assert(err)
	})

	// 指定使用TCP作为传输层并工作在指定的端口上
	// 开始服务并阻塞到接受退出信号为止
	bootstrap.
		Transport(tcp.New()).
		Listen("tcp://0.0.0.0:6565").
		RunForever(os.Kill, os.Interrupt)
}

type LogHandler struct {
	role string
}

func (l LogHandler) HandleActive(ctx netty.ActiveContext) {
	fmt.Println(l.role, "->", "active:", ctx.Channel().RemoteAddr())

	// 给对端发送一条消息，将进入如下流程（视编解码配置）
	// Text -> TextCodec -> LengthFieldCodec   -> Channel.Write
	// 文本     文本编码      组装协议格式（长度字段）     网络发送
	ctx.Write("Hello I'm " + l.role)

	// 向后续的handler传递控制权
	// 如果是最后一个handler或者需要中断请求可以不用调用
	ctx.HandleActive()
}

func (l LogHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
	fmt.Println(l.role, "->", "handle read:", message)

	// 向后续的handler传递控制权
	// 如果是最后一个handler或者需要中断请求可以不用调用
	ctx.HandleRead(message)
}

func (l LogHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	fmt.Println(l.role, "->", "inactive:", ctx.Channel().RemoteAddr(), ex)

	// 向后续的handler传递控制权
	// 如果是最后一个handler或者需要中断请求可以不用调用
	ctx.HandleInactive(ex)
}

