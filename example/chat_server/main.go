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
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec/format"
	"github.com/go-netty/go-netty/codec/frame"
	"github.com/go-netty/go-netty/transport/websocket"
)

var ManagerInst = NewManager()

func main() {

	setupCodec := func(channel netty.Channel) {
		channel.Pipeline().
			// 超出maxFrameLength将引发异常处理
			AddLast(frame.EofCodec(1024)).
			// 解析json数据到map
			AddLast(format.JsonCodec(true, false)).
			// 记录会话
			AddLast(ManagerInst).
			// 自定义处理器
			AddLast(new(chatHandler))
	}

	// 设置一些参数
	options := &websocket.Options{
		Timeout:  time.Second * 5,
		ServeMux: http.NewServeMux(),
	}

	// 渲染首页数据
	options.ServeMux.HandleFunc("/", renderIndex)

	// 配置并启动服务
	netty.NewBootstrap().
		ChildInitializer(setupCodec).
		Transport(websocket.New()).
		Listen("ws://0.0.0.0:8080/chat", websocket.WithOptions(options)).
		RunForever(os.Kill, os.Interrupt)
}

func renderIndex(writer http.ResponseWriter, request *http.Request) {
	writer.Write(indexHtml)
}

type chatHandler struct{}

func (*chatHandler) HandleActive(ctx netty.ActiveContext) {
	fmt.Printf("child connection from: %s\n", ctx.Channel().RemoteAddr())
	ctx.HandleActive()
}

func (*chatHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {

	path := ctx.Channel().Transport().(interface{ Path() string }).Path()

	fmt.Printf("received child message from: %s%s, %v\n", ctx.Channel().RemoteAddr(), path, message)

	if cmd, ok := message.(map[string]interface{}); ok {
		cmd["id"] = ctx.Channel().Id()
	}

	ManagerInst.Broadcast(message)
}

func (*chatHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	fmt.Printf("child connection loss: %s %s\n", ctx.Channel().RemoteAddr(), ex.Error())
	ctx.HandleInactive(ex)
}
