/*
 * Copyright (c) Huawei Technologies Co., Ltd. 2025. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package subscriber

import (
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestNewSubject(t *testing.T) {
	sub := NewSubject()

	assert.False(t, sub.enable.Load())
	assert.Equal(t, 0, len(sub.observers))
	assert.NotNil(t, sub.eventChan)
}

func TestPublishEvent(t *testing.T) {
	t.Run("禁用状态下不发布事件", func(t *testing.T) {
		sub := NewSubject()
		data := "test data"

		sub.PublishEvent(Update, data)
		sub.PublishEvent(Delete, data)

		// 验证事件没有被放入channel
		assert.Equal(t, 0, len(sub.eventChan))
	})

	t.Run("启用状态下发布Update事件", func(t *testing.T) {
		sub := NewSubject()
		sub.enable.Store(true)
		data := "test data"

		sub.PublishEvent(Update, data)

		assert.Equal(t, 1, len(sub.eventChan))
		e := <-sub.eventChan
		assert.Equal(t, data, e.eventValue)
		assert.Equal(t, Update, e.eventType)
	})

	t.Run("启用状态下发布Delete事件", func(t *testing.T) {
		sub := NewSubject()
		sub.enable.Store(true)
		data := "test data"

		sub.PublishEvent(Delete, data)

		assert.Equal(t, 1, len(sub.eventChan))
		e := <-sub.eventChan
		assert.Equal(t, data, e.eventValue)
		assert.Equal(t, Delete, e.eventType)
	})

	t.Run("不支持的eventType", func(t *testing.T) {
		sub := NewSubject()
		sub.enable.Store(true)
		data := "test data"

		sub.PublishEvent("invalid", data)

		assert.Equal(t, 0, len(sub.eventChan))
	})
}

func TestSubscribe(t *testing.T) {
	sub := NewSubject()
	observer1 := &Observer{
		Update: func(data interface{}) {},
		Delete: func(data interface{}) {},
	}
	observer2 := &Observer{
		Update: func(data interface{}) {},
		Delete: func(data interface{}) {},
	}

	sub.Subscribe(observer1)
	sub.Subscribe(observer2)

	assert.Equal(t, 2, len(sub.observers))
	assert.Equal(t, observer1, sub.observers[0])
	assert.Equal(t, observer2, sub.observers[1])
}

func TestNotifyEvents(t *testing.T) {
	t.Run("通知Update事件", func(t *testing.T) {
		sub := NewSubject()
		data := "test data"

		var updateCalled0 bool
		var updateCalled1 bool
		observer0 := &Observer{
			Update: func(d interface{}) {
				updateCalled0 = true
				assert.Equal(t, data, d)
			},
			Delete: func(d interface{}) {},
		}
		observer1 := &Observer{
			Update: func(d interface{}) {
				updateCalled1 = true
				assert.Equal(t, data, d)
			},
			Delete: func(d interface{}) {},
		}
		sub.Subscribe(observer0)
		sub.Subscribe(observer1)

		sub.notifyUpdateEvent(data)

		assert.True(t, updateCalled0)
		assert.True(t, updateCalled1)
	})

	t.Run("通知Delete事件", func(t *testing.T) {
		sub := NewSubject()
		data := "test data"

		var deleteCalled0 bool
		var deleteCalled1 bool

		observer0 := &Observer{
			Update: func(d interface{}) {},
			Delete: func(d interface{}) {
				deleteCalled0 = true
				assert.Equal(t, data, d)
			},
		}
		observer1 := &Observer{
			Update: func(d interface{}) {},
			Delete: func(d interface{}) {
				deleteCalled1 = true
				assert.Equal(t, data, d)
			},
		}
		sub.Subscribe(observer0)
		sub.Subscribe(observer1)

		sub.notifyDeleteEvent(data)

		assert.True(t, deleteCalled0)
		assert.True(t, deleteCalled1)
	})
}

func TestStartLoop(t *testing.T) {
	t.Run("防重放启动", func(t *testing.T) {
		sub := NewSubject()

		patches := gomonkey.NewPatches()
		defer patches.Reset()

		count := 0
		observer := &Observer{
			Update: func(d interface{}) {
				count++
			},
			Delete: func(d interface{}) {},
		}
		sub.Subscribe(observer)

		stopCh := make(chan struct{})
		sub.StartLoop(stopCh)
		sub.StartLoop(stopCh)
		sub.eventChan <- &event{
			eventType:  Update,
			eventValue: "???",
		}
		time.Sleep(1 * time.Second)
		close(stopCh)
		assert.Equal(t, count, 1)
		sub.eventChan <- &event{
			eventType:  Update,
			eventValue: "???",
		}
		time.Sleep(1 * time.Second)
		assert.Equal(t, count, 1)
	})

	t.Run("正常处理Update事件", func(t *testing.T) {
		sub := NewSubject()
		stopCh := make(chan struct{})
		defer close(stopCh)
		data := "test data"

		var updateCalled bool
		observer := &Observer{
			Update: func(d interface{}) {
				updateCalled = true
				assert.Equal(t, data, d)
			},
			Delete: func(d interface{}) {},
		}
		sub.Subscribe(observer)

		sub.eventChan <- &event{
			eventType:  Update,
			eventValue: data,
		}

		sub.StartLoop(stopCh)
		time.Sleep(100 * time.Millisecond) // 等待goroutine处理

		assert.True(t, updateCalled)
	})

	t.Run("正常处理Delete事件", func(t *testing.T) {
		sub := NewSubject()
		stopCh := make(chan struct{})
		defer close(stopCh)
		data := "test data"

		var deleteCalled bool
		observer := &Observer{
			Update: func(d interface{}) {},
			Delete: func(d interface{}) {
				deleteCalled = true
				assert.Equal(t, data, d)
			},
		}
		sub.Subscribe(observer)

		sub.eventChan <- &event{
			eventType:  Delete,
			eventValue: data,
		}

		sub.StartLoop(stopCh)
		time.Sleep(100 * time.Millisecond) // 等待goroutine处理

		assert.True(t, deleteCalled)
	})
}
