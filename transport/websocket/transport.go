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

package websocket

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/go-netty/go-netty/transport"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type websocketTransport struct {
	conn    *net.TCPConn
	closer  func() error
	options *Options
	state   ws.State
	reader  io.Reader
	path    string
}

func (t *websocketTransport) Path() string {
	return t.path
}

func (t *websocketTransport) readFrame(want ws.OpCode) (io.Reader, error) {
	controlHandler := wsutil.ControlFrameHandler(t.conn, t.state)
	rd := &wsutil.Reader{
		Source:          t.conn,
		State:           t.state,
		CheckUTF8:       !t.options.Binary,
		SkipHeaderCheck: false,
		OnIntermediate:  controlHandler,
	}

	for {
		// read frame.
		hdr, err := rd.NextFrame()
		if err != nil {
			return nil, err
		}

		// handle websocket control frame.
		if hdr.OpCode.IsControl() {
			if err := controlHandler(hdr, rd); err != nil {
				return nil, err
			}
			continue
		}

		// check frame.
		if hdr.OpCode&want == 0 {
			if err := rd.Discard(); err != nil {
				return nil, err
			}
			continue
		}

		return rd, nil
	}

}

func (t *websocketTransport) Read(p []byte) (n int, err error) {

	if t.reader != nil {
		n, err = t.reader.Read(p)

		switch {
		case n > 0:

			// 数据包包模式下，需要将剩余的数据包读完(丢弃)
			if !t.options.Stream {
				// 包模式下，可以将数据包读完之后返回一个错误，上层将可以感知到当前数据包已经读取完成
				// 这样可以不用将数据丢弃

				if io.EOF == err {
					// 当前数据包已经读完了, 下一次将要读取下一个包
					t.reader = nil

					// 保留err信息，告知上层本数据包已经读完了
				}

				return
			}

			// 数据流模式下ws Reader行为与标准Reader行为不一致
			// 额外处理一下
			if io.EOF == err {
				// 当前数据包已经读完了, 下一次将要读取下一个包
				t.reader = nil

				// 确保与标准接口行为一致, 重置err信息，继续读取下一帧数据
				err = nil
			}
			return
		case io.EOF != err:
			return
		}
	}

	// 读取下一个帧(包)
	if t.reader, err = t.readFrame(ws.OpText | ws.OpBinary); nil != err {
		// 数据流模式下，直接返回错误（EOF），上层会认为连接断开
		if t.options.Stream {
			return 0, err
		}
		// 数据包模式下，上层通过EOF来判断是否读取完一个包所以此处的错误不能直接返回EOF
		if err != io.EOF {
			return 0, err
		}
		return 0, fmt.Errorf("read frame: %v", err)
	}

	return t.Read(p)
}

func (t *websocketTransport) Write(p []byte) (n int, err error) {
	frame := t.buildFrame([][]byte{p})
	return len(p), ws.WriteFrame(t.conn, frame)
}

func (t *websocketTransport) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *websocketTransport) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

func (t *websocketTransport) Close() error {
	return t.closer()
}

func (t *websocketTransport) Writev(buffs transport.Buffers) (int64, error) {
	// header1 + payload1 | header2 + payload2 | ...
	var combineBuffers = make(net.Buffers, 0, len(buffs.Indexes)+len(buffs.Buffers))

	var i = 0
	for _, j := range buffs.Indexes {

		pkt := buffs.Buffers[i:j]
		i = j

		var header = [ws.MaxHeaderSize]byte{}
		headerWriter := bytes.NewBuffer(header[:])
		headerWriter.Reset()

		frame := t.buildFrame(pkt)
		if err := ws.WriteHeader(headerWriter, frame.Header); nil != err {
			return 0, err
		}

		// header
		combineBuffers = append(combineBuffers, headerWriter.Bytes())
		// payload
		combineBuffers = append(combineBuffers, pkt...)
	}

	return combineBuffers.WriteTo(t.conn)
}

func (t *websocketTransport) mode() ws.OpCode {
	if t.options.Binary {
		return ws.OpBinary
	}
	return ws.OpText
}

func (t *websocketTransport) buildFrame(buffs [][]byte) (frame ws.Frame) {

	// 二进制 & 文本模式
	frame = ws.NewFrame(t.mode(), true, nil)

	// 客户端需要对数据进行编码操作
	if t.state.ClientSide() {
		frame.Header.Masked = true
		frame.Header.Mask = ws.NewMask()
	}

	for _, buf := range buffs {
		frame.Header.Length += int64(len(buf))

		if frame.Header.Masked {
			ws.Cipher(buf, frame.Header.Mask, 0)
		}
	}

	return
}

func (t *websocketTransport) Flush() error {
	return nil
}

func (t *websocketTransport) RawTransport() interface{} {
	return t.conn
}

func (t *websocketTransport) applyOptions(wsOptions *Options, client bool) (*websocketTransport, error) {
	// save options.
	t.options = wsOptions
	// client or server
	if t.state = ws.StateServerSide; client {
		t.state = ws.StateClientSide
	}
	return t, nil
}
