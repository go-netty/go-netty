/*
 *  Copyright 2020 the go-netty project
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package xhttp

import (
	"bytes"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/transport/tcp"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
)

func TestServerCodec(t *testing.T) {

	var httpMessage = []byte("hello go-netty")

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(httpMessage)
	})

	var bootstrap = netty.NewBootstrap()

	bootstrap.ChildInitializer(func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(ServerCodec()).
			AddLast(Handler(httpMux))
	})

	wg := sync.WaitGroup{}
	wg.Add(1)

	bootstrap.ClientInitializer(func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(ClientCodec()).
			AddLast(netty.ActiveHandlerFunc(func(ctx netty.ActiveContext) {
				request, err := http.NewRequestWithContext(ctx.Channel().Context(), "GET", "http://127.0.0.1:9527/test", nil)
				if nil != err {
					t.Fatal(err)
				}
				ctx.Write(request)
				ctx.HandleActive()
			})).
			AddLast(netty.InboundHandlerFunc(func(ctx netty.InboundContext, message netty.Message) {
				defer wg.Done()
				response := message.(*http.Response)
				respBytes, err := ioutil.ReadAll(response.Body)
				if nil != err {
					t.Fatal(err)
				}

				if !bytes.Equal(httpMessage, respBytes) {
					t.Fatal(httpMessage, "!=", respBytes)
				}
				ctx.HandleRead(message)
			}))
	})

	bootstrap.Transport(tcp.New()).Listen("127.0.0.1:9527")

	if _, err := bootstrap.Connect("127.0.0.1:9527", nil); nil != err {
		t.Fatal(err)
	}

	wg.Wait()
}
