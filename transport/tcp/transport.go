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
	transport.Transport
	client bool
}

func newTcpTransport(conn *net.TCPConn, tcpOptions *Options, client bool) (*tcpTransport, error) {

	if err := conn.SetKeepAlive(tcpOptions.KeepAlive); nil != err {
		return nil, err
	}

	if err := conn.SetKeepAlivePeriod(tcpOptions.KeepAlivePeriod); nil != err {
		return nil, err
	}

	if err := conn.SetLinger(tcpOptions.Linger); nil != err {
		return nil, err
	}

	if err := conn.SetNoDelay(tcpOptions.NoDelay); nil != err {
		return nil, err
	}

	if tcpOptions.SockBuf > 0 {
		if err := conn.SetReadBuffer(tcpOptions.SockBuf); nil != err {
			return nil, err
		}

		if err := conn.SetWriteBuffer(tcpOptions.SockBuf); nil != err {
			return nil, err
		}
	}

	return &tcpTransport{
		Transport: transport.NewTransport(conn, tcpOptions.ReadBufferSize, tcpOptions.WriteBufferSize),
		client:    client,
	}, nil
}
