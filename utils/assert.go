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

import "fmt"

func Assert(err error, msg ...interface{}) {
	if nil != err {
		panic(fmt.Sprint(err, fmt.Sprint(msg...)))
	}
}

func AssertIf(exp bool, msg string, args ...interface{}) {
	if exp {
		panic(fmt.Sprintf(msg, args...))
	}
}

func AssertLength(n int, err error) int {
	if nil != err {
		panic(err)
	}
	return n
}

func AssertLong(n int64, err error) int64 {
	if nil != err {
		panic(err)
	}
	return n
}

func AssertBytes(b []byte, err error) []byte {
	if nil != err {
		panic(err)
	}
	return b
}
