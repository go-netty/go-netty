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

	// setup websocket params.
	options := &websocket.Options{
		Timeout:  time.Second * 5,
		ServeMux: http.NewServeMux(),
	}

	// index page.
	options.ServeMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(indexHtml)
	})

	// child pipeline initializer.
	setupCodec := func(channel netty.Channel) {
		channel.Pipeline().
			// Exceeding maxFrameLength will throw exception handling
			AddLast(frame.EofCodec(1024)).
			// decode to map[string]interface{}
			AddLast(format.JsonCodec(true, false)).
			// session recorder.
			AddLast(ManagerInst).
			// chat handler.
			AddLast(chatHandler{})
	}

	// setup bootstrap & startup server.
	netty.NewBootstrap().
		ChildInitializer(setupCodec).
		Transport(websocket.New()).
		Listen("ws://0.0.0.0:8080/chat", websocket.WithOptions(options)).
		RunForever(os.Kill, os.Interrupt)
}

type chatHandler struct{}

func (chatHandler) HandleActive(ctx netty.ActiveContext) {
	fmt.Printf("child connection from: %s\n", ctx.Channel().RemoteAddr())
}

func (chatHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {

	fmt.Printf("received child message from: %s, %v\n", ctx.Channel().RemoteAddr(), message)

	if cmd, ok := message.(map[string]interface{}); ok {
		cmd["id"] = ctx.Channel().Id()
	}

	ManagerInst.Broadcast(message)
}

func (chatHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	fmt.Printf("child connection loss: %s %s\n", ctx.Channel().RemoteAddr(), ex.Error())
	ctx.HandleInactive(ex)
}
