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

// mock for HandlerContext
type MockHandlerContext struct {
	MockChannel       func() Channel
	MockHandler       func() Handler
	MockWrite         func(message Message)
	MockClose         func(err error)
	MockTrigger       func(event Event)
	MockAttachment    func() Attachment
	MockSetAttachment func(attachment Attachment)
}

func (m MockHandlerContext) Channel() Channel {
	if m.MockChannel != nil {
		return m.MockChannel()
	}
	return nil
}

func (m MockHandlerContext) Handler() Handler {
	if m.MockHandler != nil {
		return m.MockHandler()
	}
	return nil
}

func (m MockHandlerContext) Write(message Message) {
	if m.MockWrite != nil {
		m.MockWrite(message)
	}
}

func (m MockHandlerContext) Close(err error) {
	if m.MockClose != nil {
		m.MockClose(err)
	}
}

func (m MockHandlerContext) Trigger(event Event) {
	if m.MockTrigger != nil {
		m.MockTrigger(event)
	}
}

func (m MockHandlerContext) Attachment() Attachment {
	if m.MockAttachment != nil {
		return m.MockAttachment()
	}
	return nil
}

func (m MockHandlerContext) SetAttachment(attachment Attachment) {
	if nil != m.MockSetAttachment {
		m.SetAttachment(attachment)
	}
}

// mock for OutboundContext
type MockOutboundContext struct {
	MockHandlerContext
	MockHandleWrite func(message Message)
}

func (m MockOutboundContext) HandleWrite(message Message) {
	if m.MockHandleWrite != nil {
		m.MockHandleWrite(message)
	}
}

// mock for InboundContext
type MockInboundContext struct {
	MockHandlerContext
	MockHandleRead func(message Message)
}

func (m MockInboundContext) HandleRead(message Message) {
	if m.MockHandleRead != nil {
		m.MockHandleRead(message)
	}
}
