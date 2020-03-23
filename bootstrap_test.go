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

package netty

import (
	"bytes"
	"fmt"
	"github.com/go-netty/go-netty/transport/tcp"
	"github.com/go-netty/go-netty/utils"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBootstrap(t *testing.T) {

	pipelineInitializer := func(channel Channel) {
		channel.Pipeline().
			AddLast(delimiterCodec{maxFrameLength: 1024, delimiter: []byte("$"), stripDelimiter: true}).
			AddLast(ReadIdleHandler(time.Second), WriteIdleHandler(time.Second)).
			AddLast(&textCodec{}).
			AddLast(&echoHandler{}).
			AddLast(&eventHandler{})
	}

	tcpOptions := &tcp.Options{
		Timeout:         time.Second * 3,
		KeepAlive:       true,
		KeepAlivePeriod: time.Second * 5,
		Linger:          0,
		NoDelay:         true,
		SockBuf:         1024,
	}

	bootstrap := NewBootstrap()
	bootstrap.Channel(NewBufferedChannel(128, 1024))
	bootstrap.ChildInitializer(pipelineInitializer).ClientInitializer(pipelineInitializer)
	bootstrap.ChannelExecutor(NewFixedChannelExecutor(128, 1))
	bootstrap.Transport(tcp.New())
	bootstrap.Listen("127.0.0.1:9527", tcp.WithOptions(tcpOptions))

	_, err := bootstrap.Connect("tcp://127.0.0.1:9527", "go-netty")
	if nil != err {
		t.Fatal(err)
	}

	bootstrap.Action(func(b Bootstrap) {
		time.Sleep(time.Second * 3)
		b.Stop()
		time.Sleep(time.Second * 1)
	})
}

type eventHandler struct {
	idleEvent int32
}

func (e *eventHandler) HandleEvent(ctx EventContext, event Event) {

	switch event.(type) {
	case ReadIdleEvent:
		fmt.Println("read idle event")
		if 2 == atomic.AddInt32(&e.idleEvent, 1) {
			panic("read/write idle")
		}
	case WriteIdleEvent:
		fmt.Println("write idle event")
		if 2 == atomic.AddInt32(&e.idleEvent, 1) {
			panic("read/write idle")
		}
	default:
		panic(event)
	}
}

type echoHandler struct {
}

func (e echoHandler) HandleActive(ctx ActiveContext) {
	fmt.Println("active: ", ctx.Channel().ID(), ctx.Channel().LocalAddr(), "->", ctx.Channel().RemoteAddr())
	ctx.Write("hello go-netty")
	ctx.Write([]byte("website: https://go-netty.com"))
	ctx.HandleActive()

	ctx.SetAttachment("https://go-netty.com")
	if ctx.Attachment().(string) != "https://go-netty.com" {
		panic("set attachment fail")
	}
}

func (e echoHandler) HandleRead(ctx InboundContext, message Message) {
	fmt.Println("read: ", ctx.Channel().ID(), message)
	ctx.HandleRead(message)
}

func (e echoHandler) HandleWrite(ctx OutboundContext, message Message) {
	fmt.Println("write: ", ctx.Channel().ID(), message)
	ctx.HandleWrite(message)
}

func (e echoHandler) HandleException(ctx ExceptionContext, ex Exception) {
	fmt.Println("exception: ", ctx.Channel().ID(), ex)
	stackBuffer := bytes.NewBuffer(nil)
	ex.PrintStackTrace(stackBuffer)
	ctx.HandleException(ex)
}

func (e echoHandler) HandleInactive(ctx InactiveContext, ex Exception) {
	fmt.Println("inactive: ", ctx.Channel().ID(), ex.Unwrap())
	ctx.HandleInactive(ex)
}

type delimiterCodec struct {
	maxFrameLength int
	delimiter      []byte
	stripDelimiter bool
}

func (delimiterCodec) CodecName() string {
	return "delimiter-codec"
}

func (d delimiterCodec) HandleRead(ctx InboundContext, message Message) {

	// wrap to io.Reader
	reader := utils.MustToReader(message)

	readBuff := make([]byte, 0, 16)
	tempBuff := make([]byte, 1)
	for len(readBuff) < d.maxFrameLength {
		// read 1 byte
		n := utils.AssertLength(reader.Read(tempBuff[:]))

		// append to buffer
		readBuff = append(readBuff, tempBuff[:n]...)

		// check delimiter in received buffer
		if len(readBuff) >= len(d.delimiter) && bytes.Equal(d.delimiter, readBuff[len(readBuff)-len(d.delimiter):]) {

			// strip delimiter
			if d.stripDelimiter {
				readBuff = readBuff[:len(readBuff)-len(d.delimiter)]
			}

			// post message
			ctx.HandleRead(bytes.NewReader(readBuff))
			return
		}
	}

	utils.Assert(fmt.Errorf("frame length too large, readBuffLength(%d) >= maxFrameLength(%d)",
		len(readBuff), d.maxFrameLength))

}

func (d delimiterCodec) HandleWrite(ctx OutboundContext, message Message) {

	switch r := message.(type) {
	case []byte:
		ctx.HandleWrite([][]byte{
			// body
			r,
			// delimiter
			d.delimiter,
		})
	default:
		ctx.HandleWrite(io.MultiReader(
			// body
			utils.MustToReader(message),
			// delimiter
			bytes.NewReader(d.delimiter)),
		)
	}
}

type textCodec struct{}

func (textCodec) CodecName() string {
	return "text-codec"
}

func (textCodec) HandleRead(ctx InboundContext, message Message) {

	// read text bytes
	textBytes := utils.MustToBytes(message)

	// convert from []byte to string
	sb := strings.Builder{}
	sb.Write(textBytes)

	// post text
	ctx.HandleRead(sb.String())
}

func (textCodec) HandleWrite(ctx OutboundContext, message Message) {

	switch s := message.(type) {
	case string:
		ctx.HandleWrite(strings.NewReader(s))
	default:
		ctx.HandleWrite(message)
	}
}
