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

package transport

import (
	"bufio"
	"net"
	"time"
)

type Buffered = Transport

func NewBuffered(conn net.Conn, readSize, writeSize int) Buffered {
	return &bufConn{conn: conn, ReadWriter: bufio.NewReadWriter(bufio.NewReaderSize(conn, readSize), bufio.NewWriterSize(conn, writeSize))}
}

type bufConn struct {
	*bufio.ReadWriter
	conn net.Conn
}

func (b *bufConn) Close() error {
	_ = b.Flush()
	return b.conn.Close()
}

func (b *bufConn) LocalAddr() net.Addr {
	return b.conn.LocalAddr()
}

func (b *bufConn) RemoteAddr() net.Addr {
	return b.conn.RemoteAddr()
}

func (b *bufConn) SetDeadline(t time.Time) error {
	return b.conn.SetDeadline(t)
}

func (b *bufConn) SetReadDeadline(t time.Time) error {
	return b.conn.SetReadDeadline(t)
}

func (b *bufConn) SetWriteDeadline(t time.Time) error {
	return b.conn.SetWriteDeadline(t)
}

func (b *bufConn) Writev(buffs Buffers) (int64, error) {
	return buffs.Buffers.WriteTo(b)
}

func (b *bufConn) RawTransport() interface{} {
	return b.conn
}
