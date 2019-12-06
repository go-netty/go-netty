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

package utils

import (
	"fmt"
	"io"
)

// 限制最大可读字节数
func NewMaxBytesReader(r io.Reader, maxBytes int) io.Reader {
	return &maxBytesReader{reader: r, n: maxBytes}
}

type maxBytesReader struct {
	reader io.Reader
	n      int
	err    error
}

func (m *maxBytesReader) Read(p []byte) (n int, err error) {

	// 上一轮留下的错误
	if m.err != nil {
		return 0, m.err
	}

	// 不做任何IO操作
	if len(p) == 0 {
		return 0, nil
	}

	// 最多读取_n + 1个字节, 多出的一个字节用于判断是否超出最大限制
	if len(p) > m.n+1 {
		p = p[:m.n+1]
	}

	n, err = m.reader.Read(p)
	if n > 0 {
		// 减去已经读取的字节数
		m.n -= n

		// 超出最大允许字节数
		if m.n < 0 {
			err = fmt.Errorf("read bytes too large")
			return
		}

		// 本轮有数据，那么EOF放到下一轮返回
		if io.EOF == err {
			m.err, err = err, nil
			return
		}

		// 已经全部正常读完，下一轮返回EOF
		if 0 == m.n {
			m.err, err = io.EOF, nil
			return
		}

		// 产生错误，或者正常未读取完毕，下一轮继续
	}

	return
}
