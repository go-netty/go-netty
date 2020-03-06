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
	"fmt"
	"io"
	"net"
	"net/url"
)

// 传输层定义，一般按照传输协议可以简单分类为两种:
//      1. 数据流模式    # 无数据边界(需要处理分包)
//      2. 数据包模式    # 有数据边界(无须处理分包)
// 常见的数据流模式传输层协议: TCP, KCP
// 常见的数据包模式传输层协议: UDP, WebSocket
//
// Transport 提供了io.ReadWriter接口, 为了适应两种不同的传输层协议，在使用时会有细微差别
//      数据流模式下: 按照传统的 FrameCodec -> MessageCodec 处理即可
//      数据包模式下: FrameCodec 需要指定为 PacketCodec -> [FrameCodec -> MessageCodec]
//

// Addr alias to net.Addr
type Addr = net.Addr

// Buffers to optimize message merging
// 用于优化消息的发送
// 1. 利用net.Buffers的接口实现Writev的系统调用优化
// 2. 免除合并buffer时导致的多余内存分配和内存考虑消耗
// tcp中最常见的协议头一般需要在最后发送之前在头部追加协议头，通常需要分配一个
// (sizeof(协议头) + sizeof(payload))的内存用于拼接最终的协议包用于发送
// 这里使用索引的方式标记出包与包的界限，这样就可以免除合并操作，可以极大的降低发送开销
// 下图中Buffers中的一个.（点）代表一个[]byte，[..] 通常代表[header, payload]
// [ [(header), (payload)], [(header), (payload1), (payload2)] ]
//            2                                 5
type Buffers struct {
	// [[..], [...], [...], [....]]
	Buffers net.Buffers
	// [2, 5, 8, 12]
	Indexes []int
}

// BuffersWriter defines writev for optimized syscall
type BuffersWriter interface {
	// writev
	Writev(buffs Buffers) (int64, error)
}

// Transport defines a transport
type Transport interface {
	// read & write & close
	io.ReadWriteCloser

	// syscall: sendmsg
	BuffersWriter

	// local address.
	LocalAddr() Addr

	// remote address.
	RemoteAddr() Addr

	// flush buffer.
	Flush() error

	// raw transport object.
	RawTransport() interface{}
}

// Acceptor defines transport acceptor
type Acceptor interface {
	Accept() (Transport, error)
	Close() error
}

// Factory defines transport factory
type Factory interface {

	// Supported schemes.
	Schemes() Schemes

	// Connect to the peer with the specified address.
	Connect(options *Options) (Transport, error)

	// Listen for an address and accept the connection request.
	Listen(options *Options) (Acceptor, error)
}

// Schemes to define scheme list
type Schemes []string

// FixedURL to fix scheme
func (ss Schemes) FixedURL(u *url.URL) error {
	switch {
	case "" == u.Scheme:
		u.Scheme = ss[0]
	case !ss.Valid(u.Scheme):
		return fmt.Errorf("invalid scheme, %s, available: %v", u.Scheme, ss)
	}
	return nil
}

// ValidURL to check url scheme
func (ss Schemes) ValidURL(address string) bool {
	u, err := url.Parse(address)
	if nil != err {
		return false
	}
	return ss.Valid(u.Scheme)
}

// Valid scheme
func (ss Schemes) Valid(scheme string) bool {
	return -1 != ss.indexOf(scheme)
}

// Add scheme
func (ss Schemes) Add(scheme string) Schemes {
	if -1 != ss.indexOf(scheme) {
		return ss
	}
	return append(ss, scheme)
}

// find scheme
func (ss Schemes) indexOf(scheme string) int {
	for index, s := range ss {
		if s == scheme {
			return index
		}
	}
	return -1
}
