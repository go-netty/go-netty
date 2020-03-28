/*
 *  Copyright 2020 the go-netty project
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       https://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package utils

import (
	"bytes"
	"io"
	"io/ioutil"
	"testing"
)

type testReader struct {
	reader io.Reader
}

func (t testReader) Read(p []byte) (n int, err error) {
	return t.reader.Read(p)
}

func TestByteReader(t *testing.T) {
	byteString := []byte("GO-NETTY")
	br := NewByteReader(bytes.NewReader(byteString))
	for _, b1 := range byteString {
		if b2, err := br.ReadByte(); nil != err || b1 != b2 {
			t.Fatalf("unexpected byte: %d, want: %d, err: %v", b2, b1, err)
		}
	}

	br = NewByteReader(testReader{bytes.NewReader(byteString)})
	for _, b1 := range byteString {
		if b2, err := br.ReadByte(); nil != err || b1 != b2 {
			t.Fatalf("unexpected byte: %d, want: %d, err: %v", b2, b1, err)
		}
	}
}

func TestToReader(t *testing.T) {
	byteString := []byte("GO-NETTY")
	runTest := func(reader io.Reader, err error) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			if nil != err {
				t.Fatal(err)
			}
			readBytes, err := ioutil.ReadAll(reader)
			if nil != err {
				t.Fatal(err)
			}
			if !bytes.Equal(byteString, readBytes) {
				t.Fatalf("unexpecteded bytes: %v != %v", byteString, readBytes)
			}
		}
	}

	t.Run("[]byte", runTest(ToReader(byteString)))
	t.Run("[][]byte", runTest(ToReader([][]byte{byteString[:2], byteString[2:]})))
	t.Run("string", runTest(ToReader(string(byteString))))
	t.Run("io.Reader", runTest(ToReader(bytes.NewReader(byteString))))
}

func TestToBytes(t *testing.T) {
	byteString := []byte("GO-NETTY")
	runTest := func(readBytes []byte, err error) func(t *testing.T) {
		return func(t *testing.T) {
			t.Helper()
			if nil != err {
				t.Fatal(err)
			}
			if !bytes.Equal(byteString, readBytes) {
				t.Fatalf("unexpecteded bytes: %v != %v", byteString, readBytes)
			}
		}
	}

	t.Run("[]byte", runTest(ToBytes(byteString)))
	t.Run("[][]byte", runTest(ToBytes([][]byte{byteString[:2], byteString[2:]})))
	t.Run("string", runTest(ToBytes(string(byteString))))
	t.Run("io.Reader", runTest(ToBytes(bytes.NewReader(byteString))))
}
