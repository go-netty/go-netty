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
	"errors"
	"io"
	"io/ioutil"
)

func ReaderAt(r io.Reader) io.ReaderAt {
	return &readerAt{r}
}

type readerAt struct {
	_reader io.Reader
}

func (r *readerAt) ReadAt(p []byte, off int64) (n int, err error) {

	if off < 0 {
		return 0, errors.New("negative offset")
	}

	// 丢弃前面指定的字节数
	written, err := io.CopyN(ioutil.Discard, r._reader, off)
	if written < off {
		return 0, io.EOF
	}

	// 开始读取指定大小的数据
	return io.ReadFull(r._reader, p)
}
