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

// New tcp factory
func New() transport.Factory {
	return new(tcpFactory)
}

type tcpFactory struct{}

func (*tcpFactory) Schemes() transport.Schemes {
	return transport.Schemes{"tcp", "tcp4", "tcp6"}
}

func (f *tcpFactory) Connect(options *transport.Options) (transport.Transport, error) {

	if err := f.Schemes().FixedURL(options.Address); nil != err {
		return nil, err
	}

	tcpOptions := FromContext(options.Context, DefaultOption)

	var d = net.Dialer{Timeout: tcpOptions.Timeout}
	conn, err := d.DialContext(options.Context, options.Address.Scheme, options.Address.Host)
	if nil != err {
		return nil, err
	}

	return (&tcpTransport{TCPConn: conn.(*net.TCPConn)}).applyOptions(tcpOptions, true)
}

func (f *tcpFactory) Listen(options *transport.Options) (transport.Acceptor, error) {

	if err := f.Schemes().FixedURL(options.Address); nil != err {
		return nil, err
	}

	l, err := net.Listen(options.Address.Scheme, options.AddressWithoutHost())
	if nil != err {
		return nil, err
	}

	return &tcpAcceptor{listener: l.(*net.TCPListener), options: FromContext(options.Context, DefaultOption)}, nil
}

type tcpAcceptor struct {
	listener *net.TCPListener
	options  *Options
}

func (t *tcpAcceptor) Accept() (transport.Transport, error) {

	conn, err := t.listener.AcceptTCP()
	if nil != err {
		return nil, err
	}

	return (&tcpTransport{TCPConn: conn}).applyOptions(t.options, false)
}

func (t *tcpAcceptor) Close() error {
	if t.listener != nil {
		defer func() { t.listener = nil }()
		return t.listener.Close()
	}
	return nil
}
