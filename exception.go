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

package netty

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// Exception defines an exception
type Exception interface {
	// unwrap inner error.
	Unwrap() error
	// error message.
	Error() string
	// error stack.
	Stack() []byte
	// dump stack trace to writers.
	PrintStackTrace(writer ...io.Writer)
}

// AsException to wrap error to Exception
func AsException(e interface{}, stack []byte) Exception {

	switch err := e.(type) {
	case error:
		return exception{error: err, stack: stack}
	default:
		return exception{error: fmt.Errorf("%v", e), stack: stack}
	}
}

// exception impl Exception
type exception struct {
	error error
	stack []byte
}

// Unwrap to unwrap inner error
func (e exception) Unwrap() error {
	return e.error
}

// Error to get error message
func (e exception) Error() string {
	return e.error.Error()
}

// Stack to get exception stack trace
func (e exception) Stack() []byte {
	return e.stack
}

// PrintStackTrace to write stack trance info to writer
func (e exception) PrintStackTrace(writer ...io.Writer) {

	// default: write to stderr.
	if 0 == len(writer) {
		writer = append(writer, os.Stderr)
	}

	// build output information.
	var sb strings.Builder
	sb.WriteString("exception: ")
	sb.WriteString(e.Error())
	sb.WriteString("\n")
	sb.Write(e.Stack())

	// write stack trace to writer
	_, _ = io.Copy(io.MultiWriter(writer...), strings.NewReader(sb.String()))
}
