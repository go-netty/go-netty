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

package format

import (
	"encoding/json"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
)

// JSONCodec create a json codec
func JSONCodec(useNumber bool, disAllowUnknownFields bool) codec.Codec {
	return &jsonCodec{
		useNumber:             useNumber,
		disAllowUnknownFields: disAllowUnknownFields,
	}
}

type jsonCodec struct {
	useNumber             bool
	disAllowUnknownFields bool
}

func (*jsonCodec) CodecName() string {
	return "json-codec"
}

func (j *jsonCodec) HandleRead(ctx netty.InboundContext, message netty.Message) {

	// new json decoder from reader
	jsonDecoder := json.NewDecoder(utils.MustToReader(message))

	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a
	// Number instead of as a float64.
	if j.useNumber {
		jsonDecoder.UseNumber()
	}

	// DisallowUnknownFields causes the Decoder to return an error when the destination
	// is a struct and the input contains object keys which do not match any
	// non-ignored, exported fields in the destination.
	if j.disAllowUnknownFields {
		jsonDecoder.DisallowUnknownFields()
	}

	// decode to map
	var object = make(map[string]interface{})
	utils.Assert(jsonDecoder.Decode(&object))

	// post object
	ctx.HandleRead(object)
}

func (j *jsonCodec) HandleWrite(ctx netty.OutboundContext, message netty.Message) {
	// marshal object to json bytes
	data := utils.AssertBytes(json.Marshal(message))
	// post json
	ctx.HandleWrite(data)
}
