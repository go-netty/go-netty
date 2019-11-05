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

package kcp

import (
	"github.com/go-netty/go-netty/transport"
	"github.com/xtaci/kcp-go"
)

type kcpTransport struct {
	*kcp.UDPSession
}

func (t *kcpTransport) Writev(buffs transport.Buffers) (int64, error) {
	// kcp 底层会自动分片，可以当做普通的TCP来使用
	n, err := t.UDPSession.WriteBuffers(buffs.Buffers)
	return int64(n), err
}

func (t *kcpTransport) Flush() error {
	return nil
}

func (t *kcpTransport) RawTransport() interface{} {
	return t.UDPSession
}

func (t *kcpTransport) applyOptions(kcpOptions *Options, client bool) (*kcpTransport, error) {

	t.SetStreamMode(true)
	t.SetWriteDelay(false)
	t.SetNoDelay(kcpOptions.NoDelay, kcpOptions.Interval, kcpOptions.Resend, kcpOptions.NoCongestion)
	t.SetMtu(kcpOptions.MTU)
	t.SetWindowSize(kcpOptions.SndWnd, kcpOptions.RcvWnd)
	t.SetACKNoDelay(kcpOptions.AckNodelay)

	// 客户端需要设置更多参数
	if client {

		if err := t.SetDSCP(kcpOptions.DSCP); nil != err {
			return t, err
		}

		if err := t.SetReadBuffer(kcpOptions.SockBuf); nil != err {
			return t, err
		}

		if err := t.SetWriteBuffer(kcpOptions.SockBuf); nil != err {
			return t, err
		}
	}

	return t, nil
}
