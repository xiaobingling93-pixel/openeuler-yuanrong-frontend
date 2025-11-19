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
	"encoding/json"

	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/config"
)

// ProcessDelete -
func ProcessDelete(info *types.InstanceInfo, logger api.FormatLogger) {
	defer logger.Infof("process delete event over")
	if Proxy.ExistInstanceName(info.InstanceName) {
		Proxy.Remove(info, logger)
		Proxy.Reset()
		logger.Infof("deleted from ProxyManager")
	}
}

// ProcessUpdate -
func ProcessUpdate(event *etcd3.Event, info *types.InstanceInfo, logger api.FormatLogger) {
	defer logger.Infof("process update event over")

	instanceInfo := &types.InstanceSpecification{}
	if len(event.Value) != 0 {
		err := json.Unmarshal(event.Value, instanceInfo)
		if err != nil {
			logger.Errorf("failed to unmarshal ProxyManager to instanceInfo, error %s", err.Error())
		}
	}

	logger = logger.With(zap.Any("instanceName", info.InstanceName), zap.Any("instanceId", instanceInfo.InstanceID))
	if instanceInfo.CreateOptions != nil {
		info.Exclusivity = instanceInfo.CreateOptions[constant.SchedulerExclusivityKey]
	}
	info.InstanceID = instanceInfo.InstanceID
	info.Address = instanceInfo.RuntimeAddress

	isExist := Proxy.Exist(info.InstanceName, info.InstanceID)
	isRunning := instanceInfo.InstanceStatus.Code == int32(constant.KernelInstanceStatusRunning)
	logger = logger.With(zap.Any("isExistInProxy", isExist), zap.Any("instanceStatus", instanceInfo.InstanceStatus.Code))

	// scheduler实例添加到环的逻辑：如果是终端云融合架构场景，则无论scheduler状态如何，亦添加到hash环中,但是如果scheduler id为空，则删除该实例。否则 需要判断其实例状态
	switch config.GetConfig().SchedulerKeyPrefixType {
	case constant.SchedulerKeyTypeModule:
		Proxy.Add(info, logger)
		Proxy.Reset()
		logger.Infof("add to ProxyManager")
	case constant.SchedulerKeyTypeFunction:
		fallthrough
	default:
		if !isExist && (isRunning || instanceInfo.InstanceStatus.Code == int32(constant.KernelInstanceStatusCreating)) {
			Proxy.Add(info, logger)
			Proxy.Reset()
			logger.Infof("added to ProxyManager")
		} else if utils.CheckFaaSSchedulerInstanceFault(instanceInfo.InstanceStatus) && isExist {
			Proxy.Remove(info, logger)
			Proxy.Reset()
			logger.Infof("deleted from ProxyManager")
		} else {
			logger.Infof("do nothing")
		}
	}
}
