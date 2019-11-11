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
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/go-netty/go-netty/transport"
	"github.com/gobwas/ws"
)

func New() transport.Factory {
	return new(websocketFactory)
}

type acceptEvent struct {
	conn   net.Conn
	closer func() error
	path   string
}

type websocketFactory struct {
	httpServer   *http.Server
	incoming     chan acceptEvent
	closedSignal chan struct{}
	wsOptions    *Options
}

func (*websocketFactory) Schemes() transport.Schemes {
	return transport.Schemes{"ws", "wss"}
}

func (w *websocketFactory) Connect(options *transport.Options) (transport.Transport, error) {

	if !w.Schemes().Valid(options.Address.Scheme) {
		return nil, fmt.Errorf("Invalid scheme, %v://[host]:port ", w.Schemes())
	}

	wsOptions := FromContext(options.Context, DefaultOptions)
	wsDialer := ws.DefaultDialer
	wsDialer.Timeout = wsOptions.Timeout

	u := url.URL{Scheme: options.Address.Scheme, Host: options.Address.Host, Path: options.Address.Path}
	conn, _, _, err := wsDialer.Dial(options.Context, u.String())
	if nil != err {
		return nil, err
	}

	return (&websocketTransport{conn: conn.(*net.TCPConn), closer: conn.Close, path: u.Path}).applyOptions(wsOptions, true)
}

func (w *websocketFactory) upgradeHTTP(writer http.ResponseWriter, request *http.Request) {

	conn, _, _, err := ws.UpgradeHTTP(request, writer)
	if conn != nil {
		defer conn.Close()
	}

	if nil != err {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	connCloseSignal := make(chan struct{})

	select {
	case <-w.closedSignal:
		return
	case w.incoming <- acceptEvent{conn: conn, closer: func() error {
		select {
		case <-connCloseSignal:
		default:
			close(connCloseSignal)
		}
		return nil
	}, path: request.URL.Path}:
	}

	// waiting for connection to close
	<-connCloseSignal
}

func (w *websocketFactory) Listen(options *transport.Options) (transport.Acceptor, error) {

	if !w.Schemes().Valid(options.Address.Scheme) {
		return nil, fmt.Errorf("Invalid scheme, %v://[host]:port ", w.Schemes())
	}

	_ = w.Close()

	listen, err := net.Listen("tcp", options.AddressWithoutHost())
	if nil != err {
		return nil, err
	}

	w.wsOptions = FromContext(options.Context, DefaultOptions)
	w.incoming = make(chan acceptEvent, 128)
	w.httpServer = &http.Server{Addr: listen.Addr().String(), Handler: w.wsOptions.ServeMux}
	w.closedSignal = make(chan struct{})

	var routers = []string{options.Address.Path}
	if len(w.wsOptions.Routers) > 0 {
		routers = w.wsOptions.Routers
	}

	for _, router := range routers {
		w.wsOptions.ServeMux.HandleFunc(router, w.upgradeHTTP)
	}

	errorChan := make(chan error, 1)

	go func() {
		switch options.Address.Scheme {
		case "ws":
			errorChan <- w.httpServer.Serve(listen)
		case "wss":
			errorChan <- w.httpServer.ServeTLS(listen, w.wsOptions.Cert, w.wsOptions.Key)
		}
	}()

	// temporary plan
	// waiting for server initialization
	time.Sleep(time.Second)

	select {
	case err := <-errorChan:
		return nil, err
	default:
		return w, nil
	}
}

func (w *websocketFactory) Accept() (transport.Transport, error) {

	accept, ok := <-w.incoming
	if !ok {
		return nil, errors.New("server has been closed")
	}

	return (&websocketTransport{conn: accept.conn.(*net.TCPConn), closer: accept.closer, path: accept.path}).applyOptions(w.wsOptions, false)
}

func (w *websocketFactory) Close() error {

	if nil == w.closedSignal {
		return nil
	}

	select {
	case <-w.closedSignal:
		return nil
	default:
		close(w.closedSignal)

		if w.httpServer != nil {
			return w.httpServer.Close()
		}
	}

	return nil
}
