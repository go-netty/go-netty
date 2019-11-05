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

package tcp

import (
	"net"

	"github.com/go-netty/go-netty/transport"
)

type tcpTransport struct {
	*net.TCPConn
}

func (t *tcpTransport) Writev(buffs transport.Buffers) (int64, error) {
	// 利用tcp的writev接口批量写数据达到优化发送效率的目的
	return buffs.Buffers.WriteTo(t.TCPConn)
}

func (t *tcpTransport) Flush() error {
	return nil
}

func (t *tcpTransport) RawTransport() interface{} {
	return t.TCPConn
}

func (t *tcpTransport) applyOptions(tcpOptions *Options, client bool) (*tcpTransport, error) {

	if err := t.SetKeepAlive(tcpOptions.KeepAlive); nil != err {
		return t, err
	}

	if err := t.SetKeepAlivePeriod(tcpOptions.KeepAlivePeriod); nil != err {
		return t, err
	}

	if err := t.SetLinger(tcpOptions.Linger); nil != err {
		return t, err
	}

	if err := t.SetNoDelay(tcpOptions.NoDelay); nil != err {
		return t, err
	}

	if tcpOptions.SockBuf > 0 {
		if err := t.SetReadBuffer(tcpOptions.SockBuf); nil != err {
			return t, err
		}

		if err := t.SetWriteBuffer(tcpOptions.SockBuf); nil != err {
			return t, err
		}
	}

	return t, nil
}
