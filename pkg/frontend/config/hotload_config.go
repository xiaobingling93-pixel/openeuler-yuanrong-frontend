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

// Package config is used to keep the config used by the faas frontend function
package config

import (
	"encoding/json"
	"io/ioutil"

	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/monitor"
	types2 "frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/types"
)

var (
	configWatcher         monitor.FileWatcher
	configChangedCallback ChangedCallback
)

// ChangedCallback config change callback func
type ChangedCallback func()

// WatchConfig describe config watch api
func WatchConfig(configPath string, stopCh <-chan struct{}, callback ChangedCallback) error {

	watcher, err := monitor.CreateFileWatcher(stopCh)
	if err != nil {
		return err
	}
	configWatcher = watcher
	configChangedCallback = callback
	configWatcher.RegisterCallback(configPath, hotLoadConfig)
	return nil
}

func hotLoadConfig(filename string, opType monitor.OpType) {
	log.GetLogger().Infof("file %s hot load start", filename)
	config, err := loadConfig(filename)
	if err != nil {
		log.GetLogger().Errorf("hotLoadConfig failed file: %s, opType: %d, err: %s",
			filename, opType, err.Error())
		return
	}
	hotLoadMemoryControlConfig(config)
	hotLoadMemoryEvaluatorConfig(config)
	hotLoadEtcdLeaseConfig(config)
	hotLoadCAEEtcdConfig(config)
	hotLoadMetaEtcdConfig(config)
	hotLoadRouterEtcdConfig(config)

	if configChangedCallback != nil {
		configChangedCallback()
	}
}

func loadConfig(configPath string) (*types.Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.GetLogger().Errorf("read file error, file path is %s", configPath)
		return nil, err
	}
	config := &types.Config{}
	err = json.Unmarshal(data, config)
	if err != nil {
		log.GetLogger().Errorf("failed to parse the config data: %s", err)
		return nil, err
	}
	err = loadFunctionConfig(config)
	if err != nil {
		return nil, err
	}
	return config, err
}

func hotLoadMemoryControlConfig(newAllConfig *types.Config) {
	if newAllConfig.MemoryControlConfig == nil {
		return
	}
	updateMemoryControlConfig(newAllConfig.MemoryControlConfig, GetConfig().MemoryControlConfig)
}

func hotLoadMemoryEvaluatorConfig(newAllConfig *types.Config) {
	if newAllConfig.MemoryEvaluatorConfig == nil {
		return
	}
	oldConfig := GetConfig().MemoryEvaluatorConfig
	newConfig := newAllConfig.MemoryEvaluatorConfig
	if newConfig.RequestMemoryEvaluator > 0 {
		log.GetLogger().Infof("RequestMemoryEvaluator update old: %f, new: %f",
			oldConfig.RequestMemoryEvaluator, newConfig.RequestMemoryEvaluator)
		oldConfig.RequestMemoryEvaluator = newConfig.RequestMemoryEvaluator
	}
}

func hotLoadEtcdLeaseConfig(newAllConfig *types.Config) {
	if newAllConfig.EtcdLeaseConfig == nil {
		return
	}
	if GetConfig().EtcdLeaseConfig == nil {
		GetConfig().EtcdLeaseConfig = &types.EtcdLeaseConfig{}
	}
	newConfig := newAllConfig.EtcdLeaseConfig
	oldConfig := GetConfig().EtcdLeaseConfig
	if newConfig.LeaseTTL > 0 {
		oldConfig.LeaseTTL = newConfig.LeaseTTL
		log.GetLogger().Infof("LeaseTTL update, new: %d", newConfig.LeaseTTL)

	}
	if newConfig.RenewTTL > 0 {
		oldConfig.RenewTTL = newConfig.RenewTTL
		log.GetLogger().Infof("RenewTTL update, new: %d", newConfig.RenewTTL)
	}
}

func hotLoadCAEEtcdConfig(newAllConfig *types.Config) {
	if newAllConfig.CAEMetaEtcd.Servers != nil && len(newAllConfig.CAEMetaEtcd.Servers) > 0 {
		newConfig := newAllConfig.CAEMetaEtcd
		oldConfig := GetConfig().CAEMetaEtcd
		oldConfig.Servers = newConfig.Servers
		log.GetLogger().Infof("etcd serverList update, new: %v", newConfig.Servers)
	}
	return
}

func hotLoadMetaEtcdConfig(newAllConfig *types.Config) {
	if newAllConfig.MetaEtcd.Servers != nil && len(newAllConfig.MetaEtcd.Servers) > 0 {
		newConfig := newAllConfig.MetaEtcd
		oldConfig := GetConfig().MetaEtcd
		oldConfig.Servers = newConfig.Servers
		log.GetLogger().Infof("etcd serverList update, new: %v", newConfig.Servers)
	}
	return
}

func hotLoadRouterEtcdConfig(newAllConfig *types.Config) {
	if newAllConfig.RouterEtcd.Servers != nil && len(newAllConfig.RouterEtcd.Servers) > 0 {
		newConfig := newAllConfig.RouterEtcd
		oldConfig := GetConfig().RouterEtcd
		oldConfig.Servers = newConfig.Servers
		log.GetLogger().Infof("etcd serverList update, new: %v", newConfig.Servers)
	}
	return
}

// UpdateMemoryControlConfig update memory control config
func updateMemoryControlConfig(newConfig *types2.MemoryControlConfig, oldConfig *types2.MemoryControlConfig) {
	if newConfig == nil || oldConfig == nil {
		log.GetLogger().Infof("MemoryControlConfig is nil")
		return
	}
	if newConfig.LowerMemoryPercent > 0 {
		log.GetLogger().Infof("LowerMemoryPercent update old: %f, new: %f",
			oldConfig.LowerMemoryPercent, newConfig.LowerMemoryPercent)
		oldConfig.LowerMemoryPercent = newConfig.LowerMemoryPercent
	}

	if newConfig.HighMemoryPercent > 0 {
		log.GetLogger().Infof("HighMemoryPercent update old: %f, new: %f",
			oldConfig.HighMemoryPercent, newConfig.HighMemoryPercent)
		oldConfig.HighMemoryPercent = newConfig.HighMemoryPercent
	}

	if newConfig.StatefulHighMemPercent > 0 {
		log.GetLogger().Infof("StatefulHighMemPercent update old: %f, new: %f",
			oldConfig.StatefulHighMemPercent, newConfig.StatefulHighMemPercent)
		oldConfig.StatefulHighMemPercent = newConfig.StatefulHighMemPercent
	}

	if newConfig.BodyThreshold > 0 {
		log.GetLogger().Infof("BodyThreshold update old: %d, new: %d",
			oldConfig.BodyThreshold, newConfig.BodyThreshold)
		oldConfig.BodyThreshold = newConfig.BodyThreshold
	}

	if newConfig.MemDetectIntervalMs > 0 {
		log.GetLogger().Infof("MemDetectIntervalMs update old: %d, new: %d",
			oldConfig.MemDetectIntervalMs, newConfig.MemDetectIntervalMs)
		oldConfig.MemDetectIntervalMs = newConfig.MemDetectIntervalMs
	}
}
