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

// Package util -
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	commonType "frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

// Retry -
func Retry(execute func() error, shouldRetry func() bool, maxRetries int, sleep time.Duration) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = execute()
		if err == nil {
			return nil
		}
		if shouldRetry() {
			time.Sleep(sleep)
		} else {
			return err
		}
	}
	return err
}

// LibruntimeCustomResources -
func LibruntimeCustomResources(res map[string]int64) (int, int, map[string]float64) {
	var cpu, mem int
	rtRes := make(map[string]float64, len(res))
	for k, v := range res {
		rtRes[k] = float64(v)
		if k == constant.ResourceCPUName {
			cpu = int(v)
		} else if k == constant.ResourceMemoryName {
			mem = int(v)
		}
	}
	return cpu, mem, rtRes
}

// ConvertResourceSpecs -
func ConvertResourceSpecs(ctx *types.InvokeProcessContext, funcSpec *commonType.FuncSpec) (map[string]int64, error) {
	if config.GetConfig().BusinessType == constant.BusinessTypeWiseCloud {
		return nil, nil
	}
	resourceSpecs := make(map[string]int64)
	setCPUMemory(ctx, funcSpec, resourceSpecs)
	if funcSpec.ResourceMetaData.CustomResources != "" {
		var customResources map[string]int64
		if err := json.Unmarshal([]byte(funcSpec.ResourceMetaData.CustomResources), &customResources); err != nil {
			log.GetLogger().Errorf("failed to unmarshal custom resources %s", err.Error())
			return nil, err
		}
		for resourceType, resource := range customResources {
			if resource > constant.MinCustomResourcesSize {
				resourceSpecs[resourceType] = resource
			} else {
				log.GetLogger().Warnf("ignore invalid value %f of custom resource %s", resource, resourceType)
			}
		}
	}
	return resourceSpecs, nil
}

func setCPUMemory(ctx *types.InvokeProcessContext, funcSpec *commonType.FuncSpec, resourceSpecs map[string]int64) {
	if resourceSpecs == nil {
		return
	}
	resourceSpecs[constant.ResourceCPUName] = funcSpec.ResourceMetaData.CPU
	resourceSpecs[constant.ResourceMemoryName] = funcSpec.ResourceMetaData.Memory
	if ctx == nil || ctx.ReqHeader == nil {
		return
	}
	if cpuString := PeekIgnoreCase(ctx.ReqHeader, constant.HeaderCPUSize); cpuString != "" {
		cpu, err := strconv.Atoi(cpuString)
		if err != nil {
			log.GetLogger().Warnf("invalid value %s from request header", constant.HeaderCPUSize)
			resourceSpecs[constant.ResourceCPUName] = funcSpec.ResourceMetaData.CPU
		} else {
			resourceSpecs[constant.ResourceCPUName] = int64(cpu)
		}
	}

	if memoryString := PeekIgnoreCase(ctx.ReqHeader, constant.HeaderMemorySize); memoryString != "" {
		memory, err := strconv.Atoi(memoryString)
		if err != nil {
			log.GetLogger().Warnf("invalid value %s from request header", constant.ResourceMemoryName)
			resourceSpecs[constant.ResourceMemoryName] = funcSpec.ResourceMetaData.Memory
		} else {
			resourceSpecs[constant.ResourceMemoryName] = int64(memory)
		}
	}
}

// UnmarshalCallResp -
func UnmarshalCallResp(message []byte) (*types.CallResp, error) {
	respMsg := &types.CallResp{}
	err := json.Unmarshal(message, respMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal call response data: %s", err)
	}
	return respMsg, nil
}

// GetAcquireTimeout -
func GetAcquireTimeout(funcSpec *commonType.FuncSpec) int64 {
	acquireTimeout := funcSpec.ExtendedMetaData.Initializer.Timeout
	if funcSpec.FuncMetaData.Runtime == constant.CustomContainerRuntimeType {
		acquireTimeout += constant.CustomImageExtraTimeout
	}
	// if acquireTimeout is 0,use default 120s set by libruntime, add CommonExtraTimeout two times, one for scheduler,
	// one for kernel
	if acquireTimeout > 0 {
		acquireTimeout += 2*constant.CommonExtraTimeout + constant.KernelScheduleTimeout
	}
	return acquireTimeout
}

// PeekIgnoreCase Compatible with uppercase and lowercase letters
func PeekIgnoreCase(reqHeader map[string]string, name string) string {
	if value, ok := reqHeader[name]; ok {
		return value
	}
	for key, value := range reqHeader {
		if bytes.EqualFold([]byte(key), []byte(name)) {
			return value
		}
	}
	return ""
}
