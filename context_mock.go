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

// MockHandlerContext for mock handler context
type MockHandlerContext struct {
	MockChannel       func() Channel
	MockHandler       func() Handler
	MockWrite         func(message Message)
	MockClose         func(err error)
	MockTrigger       func(event Event)
	MockAttachment    func() Attachment
	MockSetAttachment func(attachment Attachment)
}

// Channel to mock Channel of HandlerContext
func (m MockHandlerContext) Channel() Channel {
	if m.MockChannel != nil {
		return m.MockChannel()
	}
	return nil
}

// Handler to mock Handler of HandlerContext
func (m MockHandlerContext) Handler() Handler {
	if m.MockHandler != nil {
		return m.MockHandler()
	}
	return nil
}

// Write to mock Write of HandlerContext
func (m MockHandlerContext) Write(message Message) {
	if m.MockWrite != nil {
		m.MockWrite(message)
	}
}

// Close to mock Close of HandlerContext
func (m MockHandlerContext) Close(err error) {
	if m.MockClose != nil {
		m.MockClose(err)
	}
}

// Trigger to mock Trigger of HandlerContext
func (m MockHandlerContext) Trigger(event Event) {
	if m.MockTrigger != nil {
		m.MockTrigger(event)
	}
}

// Attachment to mock Attachment of HandlerContext
func (m MockHandlerContext) Attachment() Attachment {
	if m.MockAttachment != nil {
		return m.MockAttachment()
	}
	return nil
}

// SetAttachment to mock SetAttachment of HandlerContext
func (m MockHandlerContext) SetAttachment(attachment Attachment) {
	if nil != m.MockSetAttachment {
		m.SetAttachment(attachment)
	}
}

// MockOutboundContext mock for OutboundContext
type MockOutboundContext struct {
	MockHandlerContext
	MockHandleWrite func(message Message)
}

// HandleWrite to mock HandleWrite of OutboundContext
func (m MockOutboundContext) HandleWrite(message Message) {
	if m.MockHandleWrite != nil {
		m.MockHandleWrite(message)
	}
}

// MockInboundContext mock for InboundContext
type MockInboundContext struct {
	MockHandlerContext
	MockHandleRead func(message Message)
}

// HandleRead to mock HandleRead of InboundContext
func (m MockInboundContext) HandleRead(message Message) {
	if m.MockHandleRead != nil {
		m.MockHandleRead(message)
	}
}
