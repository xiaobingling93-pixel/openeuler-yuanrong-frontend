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

// Package instanceconfigmanager -
package instanceconfigmanager

import (
	"sync"

	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/instanceconfig"
	"frontend/pkg/frontend/subscriber"
)

// manager -
var manager = &Manager{
	lock:               sync.RWMutex{},
	instanceConfigMaps: make(map[string]map[string]*instanceconfig.Configuration),
}

// subject -
var subject = subscriber.NewSubject()

// GetInstanceConfigSubject -
func GetInstanceConfigSubject() *subscriber.Subject {
	return subject
}

// Manager -
type Manager struct {
	lock               sync.RWMutex
	instanceConfigMaps map[string]map[string]*instanceconfig.Configuration
}

// Load -
func Load(funcKey, invokeLabel string) (*instanceconfig.Configuration, bool) {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	insConfigs, ok := manager.instanceConfigMaps[funcKey]
	if !ok {
		return nil, false
	}

	insConfig, ok := insConfigs[invokeLabel]
	if !ok {
		return nil, false
	}
	return insConfig, true
}

// ProcessUpdate -
func ProcessUpdate(event *etcd3.Event, logger api.FormatLogger) {
	instanceConfig, err := instanceconfig.ParseInstanceConfigFromEtcdEvent(event.Key, event.Value)
	if err != nil {
		logger.Warnf("ParseInstanceConfigFromEtcdEvent failed, err: %s", err.Error())
		return
	}
	logger = logger.With(zap.Any("funcKey", instanceConfig.FuncKey), zap.Any("label", instanceConfig.InstanceLabel))

	manager.lock.Lock()
	defer manager.lock.Unlock()
	instanceConfigMap, ok := manager.instanceConfigMaps[instanceConfig.FuncKey]
	if !ok {
		instanceConfigMap = make(map[string]*instanceconfig.Configuration)
		manager.instanceConfigMaps[instanceConfig.FuncKey] = instanceConfigMap
	}
	instanceConfigMap[instanceConfig.InstanceLabel] = instanceConfig
	subject.PublishEvent(subscriber.Update, instanceConfig)
	logger.Infof("add instanceConfig ok")
}

// ProcessDelete -
func ProcessDelete(event *etcd3.Event, logger api.FormatLogger) {
	instanceConfig, err := instanceconfig.ParseInstanceConfigFromEtcdEvent(event.Key, event.PrevValue)
	if err != nil {
		logger.Warnf("ParseInstanceConfigFromEtcdEvent failed, err: %s", err.Error())
		return
	}
	logger = logger.With(zap.Any("funcKey", instanceConfig.FuncKey), zap.Any("label", instanceConfig.InstanceLabel))

	manager.lock.Lock()
	defer manager.lock.Unlock()
	instanceConfigMap, ok := manager.instanceConfigMaps[instanceConfig.FuncKey]
	if !ok {
		logger.Infof("funcKey not exist")
		return
	}

	insConfig, ok := instanceConfigMap[instanceConfig.InstanceLabel]
	if ok {
		delete(instanceConfigMap, instanceConfig.InstanceLabel)
		logger.Infof("delete instanceConfig ok")
	} else {
		logger.Infof("delete duplicates")
	}
	if len(instanceConfigMap) == 0 {
		delete(manager.instanceConfigMaps, instanceConfig.FuncKey)
	}
	subject.PublishEvent(subscriber.Delete, insConfig)
}
