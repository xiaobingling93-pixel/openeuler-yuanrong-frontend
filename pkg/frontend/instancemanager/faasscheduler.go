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

// Package instancemanager -
package instancemanager

import (
	"sync"
	"sync/atomic"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/types"
)

var globalFaaSSchedulerInstanceManager = &FaaSSchedulerInstanceManager{
	lock:                     sync.RWMutex{},
	faaSSchedulerInstanceMap: make(map[string]*types.InstanceSpecification),
}

// GetFaaSSchedulerInstanceManager -
func GetFaaSSchedulerInstanceManager() *FaaSSchedulerInstanceManager {
	return globalFaaSSchedulerInstanceManager
}

// FaaSSchedulerInstanceManager -
type FaaSSchedulerInstanceManager struct {
	lock                     sync.RWMutex
	synced                   atomic.Bool
	faaSSchedulerInstanceMap map[string]*types.InstanceSpecification
}

func (f *FaaSSchedulerInstanceManager) sync(logger api.FormatLogger) {
	f.synced.Store(true)
	if f.IsEmpty() {
		logger.Warnf("trigger no scheduler instances alarm")
		reportNoAvailableSchedulerInstAlarm()
	}
	logger.Infof("sync scheduler instance event over")
}

func (f *FaaSSchedulerInstanceManager) addInstance(instanceId string, instance *types.InstanceSpecification,
	logger api.FormatLogger) {
	f.lock.Lock()
	if _, ok := f.faaSSchedulerInstanceMap[instanceId]; ok {
		f.lock.Unlock()
		return
	}
	f.faaSSchedulerInstanceMap[instanceId] = instance
	f.lock.Unlock()
	logger.Infof("add instance to faaSSchedulerInstanceMap")
	if f.size() == 1 && f.synced.Load() {
		f.lock.Lock()
		if len(f.faaSSchedulerInstanceMap) == 1 {
			logger.Infof("clear no scheduler instances alarm")
			clearNoAvailableSchedulerInstAlarm()
		}
		f.lock.Unlock()
	}
}

func (f *FaaSSchedulerInstanceManager) delInstance(instanceId string, logger api.FormatLogger) {
	f.lock.Lock()
	if _, ok := f.faaSSchedulerInstanceMap[instanceId]; !ok {
		f.lock.Unlock()
		logger.Infof("no need delete, %s not in faaSSchedulerInstanceManager", instanceId)
		return
	}
	delete(f.faaSSchedulerInstanceMap, instanceId)
	f.lock.Unlock()
	logger.Infof("delete instance from faaSSchedulerInstanceMap")
	if f.IsEmpty() {
		f.lock.Lock()
		if len(f.faaSSchedulerInstanceMap) == 0 {
			logger.Warnf("trigger no scheduler instances alarm")
			reportNoAvailableSchedulerInstAlarm()
		}
		f.lock.Unlock()
	}
}

func (f *FaaSSchedulerInstanceManager) size() int {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return len(f.faaSSchedulerInstanceMap)
}

// IsEmpty -
func (f *FaaSSchedulerInstanceManager) IsEmpty() bool {
	return f.size() == 0
}

// Reset - just for test
func (f *FaaSSchedulerInstanceManager) Reset() {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.faaSSchedulerInstanceMap = make(map[string]*types.InstanceSpecification)
	f.synced.Store(false)
}

// IsExist -
func (f *FaaSSchedulerInstanceManager) IsExist(instanceId string) bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	_, ok := f.faaSSchedulerInstanceMap[instanceId]
	return ok
}
