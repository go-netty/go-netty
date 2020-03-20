/*
 *  Copyright 2019 the go-netty project
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

package frame

import (
	"encoding/binary"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/utils"
)

// LengthFieldPrepender for LengthFieldCodec
//
// An encoder to prepends the length of the message
func LengthFieldPrepender(
	byteOrder binary.ByteOrder,
	lengthFieldLength int,
	lengthAdjustment int,
	lengthIncludesLengthFieldLength bool,
) netty.OutboundHandler {
	utils.AssertIf(lengthFieldLength != 1 && lengthFieldLength != 2 &&
		lengthFieldLength != 4 && lengthFieldLength != 8, "lengthFieldLength must be either 1, 2, 3, 4, or 8")
	return &lengthFieldPrepender{
		byteOrder:                       byteOrder,
		lengthFieldLength:               lengthFieldLength,
		lengthAdjustment:                lengthAdjustment,
		lengthIncludesLengthFieldLength: lengthIncludesLengthFieldLength,
	}
}

type lengthFieldPrepender struct {
	byteOrder                       binary.ByteOrder
	lengthFieldLength               int
	lengthAdjustment                int
	lengthIncludesLengthFieldLength bool
}

func (l *lengthFieldPrepender) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	bodyBytes := utils.MustToBytes(message)

	length := len(bodyBytes) + l.lengthAdjustment
	if l.lengthIncludesLengthFieldLength {
		length += l.lengthFieldLength
	}

	// head buffer
	lengthBuff := packFieldLength(l.byteOrder, l.lengthFieldLength, int64(length))

	// HEAD | BODY
	ctx.HandleWrite([][]byte{lengthBuff, bodyBytes})
}
