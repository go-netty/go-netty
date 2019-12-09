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
	"fmt"
	"github.com/go-netty/go-netty"
	"github.com/go-netty/go-netty/codec"
	"github.com/go-netty/go-netty/utils"
	"io"
	"io/ioutil"
)

func LengthFieldPrepender(
	byteOrder binary.ByteOrder,
	lengthFieldLength int,
	lengthAdjustment int,
	lengthIncludesLengthFieldLength bool,
) codec.Codec {
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

func (l *lengthFieldPrepender) CodecName() string {
	return "length-field-prepender"
}

func (l *lengthFieldPrepender) HandleRead(ctx netty.InboundContext, message netty.Message) {
	ctx.HandleRead(message)
}

func (l *lengthFieldPrepender) HandleWrite(ctx netty.OutboundContext, message netty.Message) {

	var bodyBytes []byte

	switch r := message.(type) {
	case []byte:
		bodyBytes = r
	case io.Reader:
		bodyBytes = utils.AssertBytes(ioutil.ReadAll(r))
	default:
		utils.Assert(fmt.Errorf("unrecognized type: %T", message))
	}

	length := len(bodyBytes) + l.lengthAdjustment
	if l.lengthIncludesLengthFieldLength {
		length += l.lengthFieldLength
	}

	lengthBuff := make([]byte, l.lengthFieldLength)

	switch l.lengthFieldLength {
	case 1:
		lengthBuff[0] = byte(length)
	case 2:
		l.byteOrder.PutUint16(lengthBuff, uint16(length))
	case 4:
		l.byteOrder.PutUint32(lengthBuff, uint32(length))
	case 8:
		l.byteOrder.PutUint64(lengthBuff, uint64(length))
	default:
		utils.Assert(fmt.Errorf("should not reach here"))
	}

	// 优化掉一次组包操作，降低内存分配操作
	// 批量写数据
	ctx.HandleWrite([][]byte{lengthBuff, bodyBytes})
}
