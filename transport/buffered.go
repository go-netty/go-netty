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

import "bufio"

// BufferedTransport for optimize system calls with bufio.Reader
func BufferedTransport(transport Transport, sizeRead int) Transport {
	switch transport.(type) {
	case *bufferedTransport:
		return transport
	default:
		return &bufferedTransport{
			Transport: transport,
			reader:    bufio.NewReaderSize(transport, sizeRead),
		}
	}
}

type bufferedTransport struct {
	Transport
	reader *bufio.Reader
}

func (bt *bufferedTransport) Read(b []byte) (int, error) {
	return bt.reader.Read(b)
}
