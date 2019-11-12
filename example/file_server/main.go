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
	"github.com/go-netty/go-netty/codec/xhttp"
	"github.com/go-netty/go-netty/transport/tcp"
)

func main() {

	// http file server handler.
	httpMux := http.NewServeMux()
	httpMux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("./"))))

	// channel pipeline initializer.
	setupCodec := func(channel netty.Channel) {
		channel.Pipeline().
			// decode http request from channel
			AddLast(xhttp.ServerCodec()).
			// print http access log
			AddLast(new(httpStateHandler)).
			// compatible with http.Handler
			AddLast(xhttp.Handler(httpMux))
	}

	// setup bootstrap & startup server.
	netty.NewBootstrap().
		ChildInitializer(setupCodec).
		Transport(tcp.New()).
		Listen("tcp://0.0.0.0:8080").
		RunForever(os.Kill, os.Interrupt)
}

type httpStateHandler struct{}

func (*httpStateHandler) HandleActive(ctx netty.ActiveContext) {
	fmt.Printf("http client active: %s\n", ctx.Channel().RemoteAddr())
}

func (*httpStateHandler) HandleRead(ctx netty.InboundContext, message netty.Message) {
	if request, ok := message.(*http.Request); ok {
		fmt.Printf("[%d]%s: %s %s\n", ctx.Channel().Id(), ctx.Channel().RemoteAddr(), request.Method, request.URL.Path)
	}
	ctx.HandleRead(message)
}

func (*httpStateHandler) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	if responseWriter, ok := message.(http.ResponseWriter); ok {
		// set response header.
		responseWriter.Header().Add("x-time", time.Now().String())
	}
	ctx.HandleWrite(message)
}

func (*httpStateHandler) HandleInactive(ctx netty.InactiveContext, ex netty.Exception) {
	fmt.Printf("http client inactive: %s %v\n", ctx.Channel().RemoteAddr(), ex)
}
