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
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-netty/go-netty/transport"
	"github.com/go-netty/go-netty/transport/tcp"
	"github.com/go-netty/go-netty/utils"
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

	bs := NewBootstrap(
		WithChildInitializer(pipelineInitializer),
		WithClientInitializer(pipelineInitializer),
		WithTransport(tcp.New()),
	)

	bs.Listen("127.0.0.1:9527", tcp.WithOptions(tcpOptions)).Async(func(err error) {
		if nil != err && ErrServerClosed != err {
			t.Fatal(err)
		}
	})

	ch, err := bs.Connect("tcp://127.0.0.1:9527", transport.WithAttachment("go-netty"))
	if nil != err {
		t.Fatal(err)
	}

	ch.Write("https://go-netty.com")

	time.Sleep(time.Second * 3)
	bs.Shutdown()
	time.Sleep(time.Second)
}

type eventHandler struct {
	idleEvent int32
}

func (e *eventHandler) HandleEvent(ctx EventContext, event Event) {

	switch event.(type) {
	case ReadIdleEvent:
		fmt.Println("read idle event", " isActive: ", ctx.Channel().IsActive())
		if 2 == atomic.AddInt32(&e.idleEvent, 1) {
			panic("read/write idle")
		}
	case WriteIdleEvent:
		fmt.Println("write idle event", " isActive: ", ctx.Channel().IsActive())
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
	fmt.Println("active: ", ctx.Channel().ID(), ctx.Channel().LocalAddr(), "->", ctx.Channel().RemoteAddr(), " isActive: ", ctx.Channel().IsActive())

	ctx.SetAttachment("https://go-netty.com")
	if ctx.Attachment().(string) != "https://go-netty.com" {
		panic("set attachment fail")
	}

	ctx.HandleActive()
}

func (e echoHandler) HandleRead(ctx InboundContext, message Message) {
	fmt.Println("read: ", ctx.Channel().ID(), message, " isActive: ", ctx.Channel().IsActive())
	ctx.HandleRead(message)
}

func (e echoHandler) HandleWrite(ctx OutboundContext, message Message) {
	fmt.Println("write: ", ctx.Channel().ID(), message, " isActive: ", ctx.Channel().IsActive())
	ctx.HandleWrite(message)
}

func (e echoHandler) HandleException(ctx ExceptionContext, ex Exception) {
	fmt.Println("exception: ", ctx.Channel().ID(), ex, " isActive: ", ctx.Channel().IsActive())
	ctx.Channel().Close(ex)
}

func (e echoHandler) HandleInactive(ctx InactiveContext, ex Exception) {
	fmt.Println("inactive: ", ctx.Channel().ID(), ex, " isActive: ", ctx.Channel().IsActive())
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
