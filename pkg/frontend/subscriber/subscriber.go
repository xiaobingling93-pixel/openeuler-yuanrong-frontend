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

// Package subscriber -
package subscriber

import (
	"sync"
	"sync/atomic"
)

const (
	// Delete -
	Delete = "delete"
	// Update -
	Update = "update"
)

var (
	enableEventTypes = map[string]struct{}{
		Update: {},
		Delete: {},
	}
)

// Observer -
type Observer struct {
	Update func(data interface{})
	Delete func(data interface{})
}

type event struct {
	eventType  string
	eventValue interface{}
}

// Subject -
type Subject struct {
	enable    atomic.Bool
	observers []*Observer
	eventChan chan *event
	sync.RWMutex
}

// NewSubject -
func NewSubject() *Subject {
	return &Subject{
		observers: make([]*Observer, 0),
		eventChan: make(chan *event, 1000),
		RWMutex:   sync.RWMutex{},
	}
}

// PublishEvent -
func (s *Subject) PublishEvent(eventType string, data interface{}) {
	if !s.enable.Load() {
		return
	}

	if _, ok := enableEventTypes[eventType]; !ok {
		return
	}

	s.eventChan <- &event{
		eventType:  eventType,
		eventValue: data,
	}
}

// Subscribe -
func (s *Subject) Subscribe(o *Observer) {
	s.Lock()
	defer s.Unlock()
	s.observers = append(s.observers, o)
}

func (s *Subject) notifyUpdateEvent(data interface{}) {
	s.RLock()
	defer s.RUnlock()
	for _, observer := range s.observers {
		observer.Update(data)
	}
}

func (s *Subject) notifyDeleteEvent(data interface{}) {
	s.RLock()
	defer s.RUnlock()
	for _, observer := range s.observers {
		observer.Delete(data)
	}
}

// StartLoop -
func (s *Subject) StartLoop(stopCh <-chan struct{}) {
	if s.enable.Load() {
		return // 防重放
	}
	s.enable.Store(true)
	go func() {
		for {
			select {
			case e, ok := <-s.eventChan:
				if !ok {
					return
				}
				switch e.eventType {
				case Update:
					s.notifyUpdateEvent(e.eventValue)
				case Delete:
					s.notifyDeleteEvent(e.eventValue)
				default: // do nothing
				}
			case <-stopCh:
				return
			}
		}
	}()
}
