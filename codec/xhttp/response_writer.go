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
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"sync"
)

type ResponseWriter interface {
	http.ResponseWriter
	http.Flusher

	Request() *http.Request
}

var bufioWriterPool sync.Pool

// NewResponseWriter create a http response writer
func NewResponseWriter(request *http.Request, writer io.Writer) ResponseWriter {

	var bufWriter *bufio.Writer
	if value := bufioWriterPool.Get(); value != nil {
		bufWriter = value.(*bufio.Writer)
	} else {
		bufWriter = bufio.NewWriterSize(writer, 2048)
	}

	// reset writer
	bufWriter.Reset(writer)

	return &responseWriter{
		request: request,
		writer:  bufWriter,
	}
}

type responseWriter struct {
	request     *http.Request
	statusCode  int
	writer      *bufio.Writer
	header      http.Header
	wroteHeader bool
	chunkWriter io.WriteCloser
}

func (r *responseWriter) Request() *http.Request {
	return r.request
}

func (r *responseWriter) Header() http.Header {
	if nil == r.header {
		r.header = make(http.Header)
		r.header.Set("server", "go-netty")
	}
	return r.header
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if !r.wroteHeader {
		r.wroteHeader = true
		r.statusCode = statusCode
		fmt.Fprintf(r.writer, "HTTP/%d.%d %d OK\r\n", r.request.ProtoMajor, r.request.ProtoMinor, statusCode)
		for key, values := range r.Header() {
			for _, value := range values {
				fmt.Fprintf(r.writer, "%s: %s\r\n", key, value)
			}
		}
		fmt.Fprint(r.writer, "\r\n")

		if r.Header().Get("Transfer-Encoding") == "chunked" {
			r.chunkWriter = httputil.NewChunkedWriter(r.writer)
		}
	}
}

func (r *responseWriter) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	if r.chunkWriter != nil {
		return r.chunkWriter.Write(b)
	}

	return r.writer.Write(b)
}

func (r *responseWriter) Flush() {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	_ = r.Close()
}

func (r *responseWriter) Close() (err error) {

	if nil != r.chunkWriter {
		err = r.chunkWriter.Close()
		if nil == err && r.Header().Get("Trailer") == "" {
			// no trailer
			_, err = fmt.Fprint(r.writer, "\r\n")
		}
	}

	// flush response.
	if nil == err {
		err = r.writer.Flush()
	}

	// connection mark close
	if r.shouldClose() {
		r.request.Close = true
	}

	r.writer.Reset(nil)
	bufioWriterPool.Put(r.writer)
	r.writer = nil
	return
}

func (r *responseWriter) shouldClose() bool {
	if r.request.Close {
		return true
	}

	header := r.Header()
	if header.Get("Content-Length") == "" && header.Get("Transfer-Encoding") == "" {
		return true
	}
	return false
}
