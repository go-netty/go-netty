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

package codec

import "github.com/go-netty/go-netty"

// Codec
type Codec interface {
	netty.CodecHandler
}

// Wrap InboundHandler and OutboundHandler into Codec.
func Combine(name string, inbound netty.InboundHandler, outbound netty.OutboundHandler) Codec {
	return &combineCodec{name: name, InboundHandler: inbound, OutboundHandler: outbound}
}

type combineCodec struct {
	name string
	netty.InboundHandler
	netty.OutboundHandler
}

func (c *combineCodec) CodecName() string {
	return c.name
}
