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

// Package schedulerproxy -
package schedulerproxy

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/loadbalance"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/instancemanager"
)

const (
	// hashRingSize the concurrent hash ring length
	hashRingSize = 100
	limiterTime  = 1 * time.Millisecond
)

const (
	etcdPathElementsLen = 14
	tenantIndex         = 6
	functionNameIndex   = 8
	versionIndex        = 10
	instanceNameIndex   = 13
	funcKeyElementsLen  = 3
)

const (
	addSchedulerInfoOption    = "ADD"
	removeSchedulerInfoOption = "REMOVE"
)

// Proxy is the singleton proxy
var Proxy *ProxyManager

func init() {
	Proxy = newSchedulerProxy(
		loadbalance.NewLimiterCHGeneric(limiterTime),
	)
}

// ProxyManager is used to get instances from FaaSScheduler via a grpc stream
type ProxyManager struct {
	faasSchedulers sync.Map
	// key is tenantID, value is instanceID
	exclusivitySchedulers sync.Map
	// used to select a FaaSScheduler by the func info Concurrent Consistent Hash
	loadBalance loadbalance.LoadBalance
	RTAPI       api.LibruntimeAPI
}

// Add an FaaSScheduler
func (im *ProxyManager) Add(scheduleInfo *types.InstanceInfo, logger api.FormatLogger) {
	if im.RTAPI != nil {
		switch scheduleInfo.InstanceID != "" {
		case true:
			im.RTAPI.UpdateSchdulerInfo(scheduleInfo.InstanceName, scheduleInfo.InstanceID, addSchedulerInfoOption)
		case false:
			im.RTAPI.UpdateSchdulerInfo(scheduleInfo.InstanceName, scheduleInfo.InstanceID, removeSchedulerInfoOption)
		default:

		}
	}
	im.faasSchedulers.Store(scheduleInfo.InstanceName, scheduleInfo)
	im.exclusivitySchedulers.Store(scheduleInfo.Exclusivity, scheduleInfo.InstanceName)
	if scheduleInfo.Exclusivity != "" {
		logger.Infof("no need to add scheduler to load balance for exclusivity %s", scheduleInfo.Exclusivity)
		return
	}
	im.loadBalance.Add(scheduleInfo.InstanceName, 0)
	logger.Infof("add scheduler to load balance")
}

// Exist -
func (im *ProxyManager) Exist(instanceName string, instanceId string) bool {
	value, ok := im.faasSchedulers.Load(instanceName)
	if !ok {
		return false
	}
	info, _ := value.(*types.InstanceInfo) // no need judge
	if info == nil {
		return false
	}
	return info.InstanceID == instanceId
}

// ExistInstanceName -
func (im *ProxyManager) ExistInstanceName(instanceName string) bool {
	_, ok := im.faasSchedulers.Load(instanceName)
	return ok
}

// Remove a FaaSScheduler
func (im *ProxyManager) Remove(schedulerInfo *types.InstanceInfo, logger api.FormatLogger) {
	if _, ok := im.faasSchedulers.Load(schedulerInfo.InstanceName); !ok {
		logger.Infof("no need delete unexist scheduler")
		return
	}
	if im.RTAPI != nil {
		im.RTAPI.UpdateSchdulerInfo(schedulerInfo.InstanceName, schedulerInfo.InstanceID, removeSchedulerInfoOption)
	}
	im.faasSchedulers.Delete(schedulerInfo.InstanceName)
	im.exclusivitySchedulers.Range(func(key, value interface{}) bool {
		instanceID, ok := value.(string)
		if !ok {
			return true
		}
		if instanceID == schedulerInfo.InstanceName {
			im.exclusivitySchedulers.Delete(key)
		}
		return true
	})
	im.loadBalance.Remove(schedulerInfo.InstanceName)
	logger.Infof("deleted from load balance")
}

// Get an instance for this request
func (im *ProxyManager) Get(funcKey string, logger api.FormatLogger) (*types.InstanceInfo, error) {
	logger.Debugf("begin to get scheduler for funcKey: %s", funcKey)
	next, err := im.getNextScheduler(funcKey, logger)
	if err != nil {
		return nil, err
	}
	faasSchedulerName, ok := next.(string)
	if !ok {
		return nil, fmt.Errorf("failed to parse the result of loadbanlance: %+v", next)
	}
	if strings.TrimSpace(faasSchedulerName) == "" {
		return nil, fmt.Errorf("no avaiable faas scheduler was found")
	}
	faaSSchedulerData, ok := im.faasSchedulers.Load(faasSchedulerName)
	if !ok {
		return nil, fmt.Errorf("failed to get the faas scheduler named %s", faasSchedulerName)
	}
	faaSScheduler, ok := faaSSchedulerData.(*types.InstanceInfo)
	if !ok {
		return nil, fmt.Errorf("invalid faas scheduler named %s: %#v", faasSchedulerName, faaSSchedulerData)
	}
	logger.Infof("succeed to get scheduler instanceID: %s for funcKey: %s", faasSchedulerName, funcKey)
	return faaSScheduler, nil
}

// IsEmpty -
func (im *ProxyManager) IsEmpty() bool {
	flag := false
	im.faasSchedulers.Range(func(k, v any) bool {
		instance, ok := v.(*types.InstanceInfo)
		if !ok {
			return true
		}

		ok = instancemanager.GetFaaSSchedulerInstanceManager().IsExist(instance.InstanceID)
		if ok {
			flag = true
			return false
		}
		return true
	})
	return !flag
}

// GetSchedulerByInstanceName -
func (im *ProxyManager) GetSchedulerByInstanceName(instanceName string, traceID string) (*types.InstanceInfo, error) {
	faaSSchedulerData, ok := im.faasSchedulers.Load(instanceName)
	if !ok {
		return nil, fmt.Errorf("failed to get the faas scheduler named %s,traceID %s", instanceName, traceID)
	}
	faaSScheduler, ok := faaSSchedulerData.(*types.InstanceInfo)
	if !ok {
		return nil, fmt.Errorf("invalid faas scheduler named %s: %#v, traceID: %s",
			instanceName, faaSSchedulerData, traceID)
	}
	return faaSScheduler, nil
}

func (im *ProxyManager) getNextScheduler(funcKey string, logger api.FormatLogger) (any, error) {
	var next interface{}
	elements := strings.Split(funcKey, constant.KeySeparator)
	if len(elements) == funcKeyElementsLen {
		var ok bool
		tenantID := elements[0]
		next, ok = im.exclusivitySchedulers.Load(tenantID)
		if ok && next != nil {
			return next, nil
		}
	} else {
		logger.Warnf("invalid funcKey: %s", funcKey)
	}

	// select one FaaSScheduler by the func key
	next = im.loadBalance.Next(funcKey, false)
	if next == nil {
		log.GetLogger().Errorf("failed to get faaSScheduler instance, function: %s", funcKey)
		return nil, fmt.Errorf("failed to get faaSScheduler instance")
	}
	return next, nil
}

// DeleteBalancer -
func (im *ProxyManager) DeleteBalancer(funcKey string) {
	im.loadBalance.DeleteBalancer(funcKey)
}

// SetStain -
func (im *ProxyManager) SetStain(funcKey, instanceName string) {
	if v, ok := im.loadBalance.(*loadbalance.LimiterCHGeneric); ok {
		v.SetStain(funcKey, instanceName)
	}
}

// Reset - reset hash anchor point
func (im *ProxyManager) Reset() {
	im.loadBalance.Reset()
}

// newSchedulerProxy return an instance pool which get the instance from the remote FaaSScheduler
func newSchedulerProxy(lb loadbalance.LoadBalance) *ProxyManager {
	return &ProxyManager{
		loadBalance: lb,
	}
}
