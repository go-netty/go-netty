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

package xhttp

import (
	"bytes"
	"io/ioutil"
	"net/http"
)


// 构造响应
func NewResponseWriter(protoMajor, protoMinor int) http.ResponseWriter {
	return &responseWriter{_protoMajor: protoMajor, _protoMinor: protoMinor}
}

type responseWriter struct {
	_protoMajor, _protoMinor int
	_statusCode              int
	_header                  http.Header
	_body                    bytes.Buffer
}

func (r *responseWriter) Header() http.Header {
	if nil == r._header {
		r._header = make(http.Header)
		r._header.Set("server", "go-netty")
	}
	return r._header
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r._statusCode = statusCode
}

func (r *responseWriter) Write(b []byte) (int, error) {
	return r._body.Write(b)
}

func (r *responseWriter) response() *http.Response {
	if 0 == r._statusCode {
		r._statusCode = http.StatusOK
	}
	return &http.Response{
		ProtoMajor:    r._protoMajor,
		ProtoMinor:    r._protoMinor,
		StatusCode:    r._statusCode,
		Header:        r._header,
		Body:          ioutil.NopCloser(&r._body),
		ContentLength: int64(r._body.Len()),
	}
}
