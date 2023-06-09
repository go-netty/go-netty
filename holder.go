/*
 * Copyright 2023 the go-netty project
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
	"sync"
)

// NewChannelHolder create a new ChannelHolder with initial capacity
func NewChannelHolder(capacity int) ChannelHolder {
	return &channelHolder{channels: make(map[int64]Channel, capacity)}
}

type channelHolder struct {
	channels map[int64]Channel
	mutex    sync.Mutex
}

func (c *channelHolder) HandleActive(ctx ActiveContext) {
	c.addChannel(ctx.Channel())
	ctx.HandleActive()
}

func (c *channelHolder) HandleInactive(ctx InactiveContext, ex Exception) {
	c.delChannel(ctx.Channel())
	ctx.HandleInactive(ex)
}

func (c *channelHolder) CloseAll(err error) {
	c.mutex.Lock()
	channels := c.channels
	c.channels = make(map[int64]Channel, 1024)
	c.mutex.Unlock()

	for _, ch := range channels {
		ch.Close(err)
	}
}

func (c *channelHolder) addChannel(ch Channel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	id := ch.ID()
	if _, ok := c.channels[id]; ok {
		panic(fmt.Errorf("duplicate channel: %d", id))
	}
	c.channels[id] = ch
}

func (c *channelHolder) delChannel(ch Channel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.channels, ch.ID())
}
