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

import "io"

// ByteReader defines byte reader
type ByteReader interface {
	io.Reader
	io.ByteReader
}

// NewByteReader create a ByteReader from io.Reader
func NewByteReader(r io.Reader) ByteReader {
	if br, ok := r.(ByteReader); ok {
		return br
	}
	return &byteReader{Reader: r}
}

type byteReader struct {
	io.Reader
}

func (r *byteReader) ReadByte() (byte, error) {
	var buff = [1]byte{}
	_, err := r.Read(buff[:])
	return buff[0], err
}
