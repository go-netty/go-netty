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
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-netty/go-netty"
)

func TestServerCodec(t *testing.T) {

	var wg = sync.WaitGroup{}

	var httpMessage = []byte("hello go-netty")

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/test", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(httpMessage)
	})

	var childInitializer = func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(ServerCodec()).
			AddLast(Handler(httpMux))
	}

	var clientInitializer = func(channel netty.Channel) {
		channel.Pipeline().
			AddLast(ClientCodec()).
			AddLast(netty.ActiveHandlerFunc(func(ctx netty.ActiveContext) {
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
			})).
			AddLast(netty.ExceptionHandlerFunc(func(ctx netty.ExceptionContext, ex netty.Exception) {
				ctx.Close(ex)
			}))
	}

	var bootstrap = netty.NewBootstrap(netty.WithChildInitializer(childInitializer), netty.WithClientInitializer(clientInitializer))

	bootstrap.Listen("127.0.0.1:9526").Async(func(err error) {
		if nil != err && netty.ErrServerClosed != err {
			t.Fatal(err)
		}
	})

	wg.Add(1)

	time.Sleep(time.Second * 2)
	channel, err := bootstrap.Connect("127.0.0.1:9526")
	if nil != err {
		t.Fatal(err)
	}

	request, err := http.NewRequest("GET", "http://127.0.0.1:9526/test", nil)
	if nil != err {
		t.Fatal(err)
	}

	channel.Write(request)

	wg.Wait()
	bootstrap.Shutdown()
}
