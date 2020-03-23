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

package transport

import (
	"net/url"
	"testing"
)

func TestSchemes(t *testing.T) {

	var schemes = Schemes{}.Add("tcp").Add("tcp4").Add("tcp6")

	if !schemes.Valid("tcp") {
		t.Fatal("fail")
	}

	if schemes.Valid("udp") {
		t.Fatal("fail")
	}

	if !schemes.ValidURL("tcp://127.0.0.1:9527") {
		t.Fatal("fail")
	}

	if schemes.ValidURL("udp://127.0.0.1:9527") {
		t.Fatal("fail")
	}

	u, err := url.Parse("//127.0.0.1:9527")
	if nil != err {
		t.Fatal(err)
	}

	if err := schemes.FixedURL(u); nil != err || u.Scheme != "tcp" {
		t.Fatal(err, u.Scheme)
	}

}
