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
)

func NewTransport(conn net.Conn, readSize, writeSize int) Transport {
	switch {
	case readSize > 0 && writeSize > 0:
		return &bufConn{Conn: conn, rw: bufio.NewReadWriter(bufio.NewReaderSize(conn, readSize), bufio.NewWriterSize(conn, writeSize))}
	case readSize > 0:
		return &bufReadConn{Conn: conn, reader: bufio.NewReaderSize(conn, readSize)}
	case writeSize > 0:
		return &bufWriteConn{Conn: conn, writer: bufio.NewWriterSize(conn, writeSize)}
	default:
		return &rawConn{Conn: conn}
	}
}

type bufConn struct {
	net.Conn
	rw *bufio.ReadWriter
}

func (b *bufConn) Close() error {
	_ = b.Flush()
	return b.Conn.Close()
}

func (b *bufConn) Read(p []byte) (n int, err error) {
	return b.rw.Reader.Read(p)
}

func (b *bufConn) Write(p []byte) (n int, err error) {
	return b.rw.Writer.Write(p)
}

func (b *bufConn) Writev(buffs Buffers) (int64, error) {
	return buffs.WriteTo(b.rw.Writer)
}

func (b *bufConn) Flush() error {
	return b.rw.Writer.Flush()
}

func (b *bufConn) RawTransport() interface{} {
	return b.Conn
}

type bufReadConn struct {
	net.Conn
	reader *bufio.Reader
}

func (br *bufReadConn) Read(b []byte) (n int, err error) {
	return br.reader.Read(b)
}

func (br *bufReadConn) Writev(buffs Buffers) (int64, error) {
	return buffs.WriteTo(br.Conn)
}

func (br *bufReadConn) Flush() error {
	return nil
}

func (br *bufReadConn) RawTransport() interface{} {
	return br.Conn
}

type bufWriteConn struct {
	net.Conn
	writer *bufio.Writer
}

func (bw *bufWriteConn) Write(b []byte) (n int, err error) {
	return bw.writer.Write(b)
}

func (bw *bufWriteConn) Writev(buffs Buffers) (int64, error) {
	return buffs.WriteTo(bw.writer)
}

func (bw *bufWriteConn) Flush() error {
	return bw.writer.Flush()
}

func (bw *bufWriteConn) RawTransport() interface{} {
	return bw.Conn
}

type rawConn struct {
	net.Conn
}

func (r *rawConn) Writev(buffs Buffers) (int64, error) {
	return buffs.WriteTo(r.Conn)
}

func (r *rawConn) Flush() error {
	return nil
}

func (r *rawConn) RawTransport() interface{} {
	return r.Conn
}
