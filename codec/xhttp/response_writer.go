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

// NewResponseWriter create a http response writer
func NewResponseWriter(protoMajor, protoMinor int) http.ResponseWriter {
	return &responseWriter{protoMajor: protoMajor, protoMinor: protoMinor}
}

type responseWriter struct {
	protoMajor, protoMinor int
	statusCode             int
	header                 http.Header
	body                   bytes.Buffer
}

func (r *responseWriter) Header() http.Header {
	if nil == r.header {
		r.header = make(http.Header)
		r.header.Set("server", "go-netty")
	}
	return r.header
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseWriter) Write(b []byte) (int, error) {
	return r.body.Write(b)
}

func (r *responseWriter) response() *http.Response {
	if 0 == r.statusCode {
		r.WriteHeader(http.StatusOK)
	}
	return &http.Response{
		ProtoMajor:    r.protoMajor,
		ProtoMinor:    r.protoMinor,
		StatusCode:    r.statusCode,
		Header:        r.Header(),
		Body:          ioutil.NopCloser(&r.body),
		ContentLength: int64(r.body.Len()),
	}
}
