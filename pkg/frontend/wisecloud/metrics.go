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

// Package wisecloud -
package wisecloud

import (
	"sync"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/instanceconfig"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/resspeckey"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/wisecloudtool"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/functionmeta"
	"frontend/pkg/frontend/instancemanager"
)

var metricsManager = &MetricsManager{
	RWMutex:         sync.RWMutex{},
	metricsProvider: wisecloudtool.NewMetricProvider(),
	logger:          log.GetLogger(),
}

// GetMetricsManager -
func GetMetricsManager() *MetricsManager {
	return metricsManager
}

// MetricsManager -
type MetricsManager struct {
	sync.RWMutex
	metricsProvider *wisecloudtool.MetricProvider
	logger          api.FormatLogger // key: {funcKey}#{invokeLabel}, value: {namespace, podName}
}

// ProcessFunctionDelete -
func (m *MetricsManager) ProcessFunctionDelete(funcSpec *types.FuncSpec) {
	if funcSpec == nil {
		return
	}
	m.Lock()
	defer m.Unlock()
	m.metricsProvider.ClearMetricsForFunction(&funcSpec.FuncMetaData)
	m.logger.Infof("delete function: %s wisecloud metrics", funcSpec.FunctionKey)
}

// ProcessInsConfigDelete -
func (m *MetricsManager) ProcessInsConfigDelete(insConfig *instanceconfig.Configuration) {
	if insConfig == nil {
		return
	}
	m.Lock()
	defer m.Unlock()
	funcSpec, ok := functionmeta.LoadFuncSpec(insConfig.FuncKey)
	if !ok {
		m.logger.Warnf("funcKey: %s's functionMetaData not found", insConfig.FuncKey)
		return
	}
	m.metricsProvider.ClearMetricsForInsConfig(&funcSpec.FuncMetaData, insConfig.InstanceLabel)
	m.logger.Infof("delete function: %s, invokeLabel: %s wisecloud metrics",
		funcSpec.FunctionKey, insConfig.InstanceLabel)
}

// ProcessInstanceDelete -
func (m *MetricsManager) ProcessInstanceDelete(instance *types.InstanceSpecification) {
	if instance == nil {
		return
	}
	m.Lock()
	defer m.Unlock()
	funcKey, ok := instance.CreateOptions[constant.FunctionKeyNote]
	if !ok {
		m.logger.Warnf("delete instance: %s wisecloud metrics failed, no functionMeta", instance.InstanceID)
		return
	}
	funcSpec, ok := functionmeta.LoadFuncSpec(funcKey)
	if !ok {
		return
	}
	resSpecKey, err := resspeckey.GetResKeyFromStr(instance.CreateOptions[constant.ResourceSpecNote])
	if err != nil {
		return
	}
	labels := wisecloudtool.GetMetricLabels(&funcSpec.FuncMetaData, resSpecKey.InvokeLabel,
		instance.Extensions.PodNamespace, instance.Extensions.PodDeploymentName, instance.Extensions.PodName)
	m.metricsProvider.ClearLeaseRequestTotalWithLabel(labels)
	m.metricsProvider.ClearConcurrencyGaugeWithLabel(labels)
	m.logger.Infof("delete instance: %s wisecloud metrics, function: %s, invokeLabel: %s",
		instance.InstanceID, funcSpec.FunctionKey, resSpecKey.InvokeLabel)
}

// InvokeStart -
func (m *MetricsManager) InvokeStart(funcKey string, resSpecKeyStr string, instanceId string) {
	if config.GetConfig().BusinessType != constant.BusinessTypeWiseCloud {
		return
	}
	funcSpec, ok := functionmeta.LoadFuncSpec(funcKey)
	if !ok {
		return
	}
	instance := instancemanager.GetGlobalInstanceScheduler().GetInstance(funcKey, resSpecKeyStr, instanceId)
	if instance == nil {
		return
	}
	resSpecKey, err := resspeckey.GetResKeyFromStr(instance.CreateOptions[constant.ResourceSpecNote])
	if err != nil {
		return
	}
	labels := wisecloudtool.GetMetricLabels(&funcSpec.FuncMetaData, resSpecKey.InvokeLabel,
		instance.Extensions.PodNamespace, instance.Extensions.PodDeploymentName, instance.Extensions.PodName)
	m.metricsProvider.IncConcurrencyGaugeWithLabel(labels)
	m.metricsProvider.IncLeaseRequestTotalWithLabel(labels)
}

// InvokeEnd -
func (m *MetricsManager) InvokeEnd(funcKey, resSpecKeyStr string, instanceId string) {
	if config.GetConfig().BusinessType != constant.BusinessTypeWiseCloud {
		return
	}
	funcSpec, ok := functionmeta.LoadFuncSpec(funcKey)
	if !ok {
		return
	}
	instance := instancemanager.GetGlobalInstanceScheduler().GetInstance(funcKey, resSpecKeyStr, instanceId)
	if instance == nil {
		return
	}
	resSpecKey, err := resspeckey.GetResKeyFromStr(instance.CreateOptions[constant.ResourceSpecNote])
	if err != nil {
		return
	}
	labels := wisecloudtool.GetMetricLabels(&funcSpec.FuncMetaData, resSpecKey.InvokeLabel,
		instance.Extensions.PodNamespace, instance.Extensions.PodDeploymentName, instance.Extensions.PodName)
	m.metricsProvider.DecConcurrencyGaugeWithLabel(labels)
}
