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

package wisecloud

import (
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/instanceconfig"
	"frontend/pkg/common/faas_common/resspeckey"
	commontypes "frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/functionmeta"
	"frontend/pkg/frontend/instancemanager"
	"frontend/pkg/frontend/types"
)

// 测试数据准备
var (
	mockFuncSpec = &commontypes.FuncSpec{
		FunctionKey:  "test-function",
		FuncMetaData: commontypes.FuncMetaData{
			// 填充元数据字段
		},
	}
	mockInsConfig = &instanceconfig.Configuration{
		FuncKey:       "test-function",
		InstanceLabel: "test-label",
	}
	mockInstance = &commontypes.InstanceSpecification{
		CreateOptions: map[string]string{
			constant.FunctionKeyNote:  "test-function",
			constant.ResourceSpecNote: "test-res-spec",
		},
		Extensions: commontypes.Extensions{
			PodNamespace:      "test-ns",
			PodDeploymentName: "test-deploy",
			PodName:           "test-pod",
		},
		InstanceID: "test-instance-id",
	}
)

func TestWiseCloudMetricsManager_ProcessFunctionDelete(t *testing.T) {
	convey.Convey("测试 ProcessFunctionDelete", t, func() {
		// 使用 gomonkey mock MetricProvider
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		var called bool
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "ClearMetricsForFunction",
			func(*commontypes.FuncMetaData) {
				called = true
			})

		// 执行测试
		metricsManager.ProcessFunctionDelete(mockFuncSpec)

		// 验证结果
		convey.So(called, convey.ShouldBeTrue)
	})
}

func TestWiseCloudMetricsManager_ProcessInsConfigDelete(t *testing.T) {
	convey.Convey("测试 ProcessInsConfigDelete", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		// mock 函数元数据加载
		patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*commontypes.FuncSpec, bool) {
			return mockFuncSpec, true
		})

		var called bool
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "ClearMetricsForInsConfig",
			func(*commontypes.FuncMetaData, string) {
				called = true
			})

		// 执行测试
		metricsManager.ProcessInsConfigDelete(mockInsConfig)

		// 验证结果
		convey.So(called, convey.ShouldBeTrue)
	})

	convey.Convey("测试 ProcessInsConfigDelete 函数元数据不存在", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		// mock 函数元数据加载返回不存在
		patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*commontypes.FuncSpec, bool) {
			return nil, false
		})
		var called bool
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "ClearMetricsForInsConfig",
			func(*commontypes.FuncMetaData, string) {
				called = true
			})
		// 执行测试并验证日志输出
		metricsManager.ProcessInsConfigDelete(mockInsConfig)
		convey.So(called, convey.ShouldBeFalse)
	})
}

func TestWiseCloudMetricsManager_ProcessInstanceDelete(t *testing.T) {
	// 使用 gomonkey mock 依赖
	patches := gomonkey.NewPatches()
	defer patches.Reset()
	var leaseCalled, concurrencyCalled bool
	// mock 函数元数据加载
	patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*commontypes.FuncSpec, bool) {
		return mockFuncSpec, true
	})
	patches.ApplyMethodFunc(metricsManager.metricsProvider, "ClearLeaseRequestTotalWithLabel",
		func([]string) error {
			leaseCalled = true
			return nil
		})
	patches.ApplyMethodFunc(metricsManager.metricsProvider, "ClearConcurrencyGaugeWithLabel",
		func([]string) error {
			concurrencyCalled = true
			return nil
		})
	// mock 资源规格解析
	patches.ApplyFunc(resspeckey.GetResKeyFromStr, func(string) (resspeckey.ResSpecKey, error) {
		return resspeckey.ResSpecKey{InvokeLabel: "test-label"}, nil
	})

	convey.Convey("测试 ProcessInstanceDelete", t, func() {
		leaseCalled = false
		concurrencyCalled = false

		// 执行测试
		metricsManager.ProcessInstanceDelete(mockInstance)

		// 验证结果
		convey.So(leaseCalled, convey.ShouldBeTrue)
		convey.So(concurrencyCalled, convey.ShouldBeTrue)
	})

	convey.Convey("测试 ProcessInstanceDelete 异常情况", t, func() {
		convey.Convey("函数元数据不存在", func() {
			localPatches := gomonkey.NewPatches()
			defer localPatches.Reset()
			leaseCalled = false
			concurrencyCalled = false
			localPatches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*commontypes.FuncSpec, bool) {
				return nil, false
			})
			metricsManager.ProcessInstanceDelete(mockInstance)

			convey.So(leaseCalled, convey.ShouldBeFalse)
			convey.So(concurrencyCalled, convey.ShouldBeFalse)
		})

		convey.Convey("资源规格解析失败", func() {
			localPatches := gomonkey.NewPatches()
			defer localPatches.Reset()
			leaseCalled = false
			concurrencyCalled = false
			localPatches.ApplyFunc(resspeckey.GetResKeyFromStr, func(string) (*resspeckey.ResSpecKey, error) {
				return nil, fmt.Errorf("parse error")
			})

			metricsManager.ProcessInstanceDelete(mockInstance)
			convey.So(leaseCalled, convey.ShouldBeFalse)
			convey.So(concurrencyCalled, convey.ShouldBeFalse)
		})
	})
}

func TestWiseCloudMetricsManager_InvokeStart(t *testing.T) {
	convey.Convey("测试 InvokeStart", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		// mock 配置检查
		patches.ApplyFunc(config.GetConfig, func() *types.Config {
			return &types.Config{BusinessType: constant.BusinessTypeWiseCloud}
		})

		// mock 函数元数据加载
		patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*commontypes.FuncSpec, bool) {
			return mockFuncSpec, true
		})

		// mock 资源规格解析
		patches.ApplyFunc(resspeckey.GetResKeyFromStr, func(string) (resspeckey.ResSpecKey, error) {
			return resspeckey.ResSpecKey{InvokeLabel: "test-label"}, nil
		})

		var leaseCalled, concurrencyCalled bool
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "IncLeaseRequestTotalWithLabel",
			func([]string) error {
				leaseCalled = true
				return nil
			})
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "IncConcurrencyGaugeWithLabel",
			func([]string) error {
				concurrencyCalled = true
				return nil
			})
		patches.ApplyMethodFunc(instancemanager.GetGlobalInstanceScheduler(), "GetInstance",
			func(string, string, string) *commontypes.InstanceSpecification {
				return &commontypes.InstanceSpecification{
					Extensions: commontypes.Extensions{
						PodName:           "1",
						PodNamespace:      "2",
						PodDeploymentName: "3",
					},
				}
			})

		// 执行测试
		metricsManager.InvokeStart("test-function", "test-res-spec", "test-function")

		// 验证结果
		convey.So(leaseCalled, convey.ShouldBeTrue)
		convey.So(concurrencyCalled, convey.ShouldBeTrue)
	})

	convey.Convey("测试 InvokeStart 非CaaS业务类型", t, func() {
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		patches.ApplyFunc(config.GetConfig, func() *types.Config {
			return &types.Config{BusinessType: 2}
		})

		var called bool
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "IncConcurrencyGaugeWithLabel",
			func([]string) error {
				called = true
				return nil
			})

		metricsManager.InvokeStart("test-function", "test-res-spec", "test-function")
		convey.So(called, convey.ShouldBeFalse)
	})
}

func TestWiseCloudMetricsManager_InvokeEnd(t *testing.T) {
	convey.Convey("测试 InvokeEnd", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		// mock 配置检查
		patches.ApplyFunc(config.GetConfig, func() *types.Config {
			return &types.Config{BusinessType: constant.BusinessTypeWiseCloud}
		})

		// mock 函数元数据加载
		patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*commontypes.FuncSpec, bool) {
			return mockFuncSpec, true
		})

		// mock 实例获取
		patches.ApplyMethodFunc(instancemanager.GetGlobalInstanceScheduler(), "GetInstance",
			func(string, string, string) *commontypes.InstanceSpecification {
				return mockInstance
			})

		// mock 资源规格解析
		patches.ApplyFunc(resspeckey.GetResKeyFromStr, func(string) (resspeckey.ResSpecKey, error) {
			return resspeckey.ResSpecKey{InvokeLabel: "test-label"}, nil
		})

		patches.ApplyMethodFunc(instancemanager.GetGlobalInstanceScheduler(), "GetInstance",
			func(string, string, string) *commontypes.InstanceSpecification {
				return &commontypes.InstanceSpecification{
					Extensions: commontypes.Extensions{
						PodName:           "1",
						PodNamespace:      "2",
						PodDeploymentName: "3",
					},
				}
			})

		var called bool
		patches.ApplyMethodFunc(metricsManager.metricsProvider, "DecConcurrencyGaugeWithLabel",
			func([]string) error {
				called = true
				return nil
			})

		// 执行测试
		metricsManager.InvokeEnd("test-function", "test-res-spec", "test-instance-id")

		// 验证结果
		convey.So(called, convey.ShouldBeTrue)
	})

	convey.Convey("测试 InvokeEnd 异常情况", t, func() {
		convey.Convey("非CaaS业务类型", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(config.GetConfig, func() *types.Config {
				return &types.Config{BusinessType: 2}
			})

			var called bool
			patches.ApplyMethodFunc(metricsManager.metricsProvider, "DecConcurrencyGaugeWithLabel",
				func([]string) error {
					called = true
					return nil
				})

			metricsManager.InvokeEnd("test-function", "test-res-spec", "test-instance-id")
			convey.So(called, convey.ShouldBeFalse)
		})

		convey.Convey("实例不存在", func() {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyMethodFunc(instancemanager.GetGlobalInstanceScheduler(), "GetInstance",
				func(string, string, string) *commontypes.InstanceSpecification {
					return nil
				})

			var called bool
			patches.ApplyMethodFunc(metricsManager.metricsProvider, "DecConcurrencyGaugeWithLabel",
				func([]string) error {
					called = true
					return nil
				})

			metricsManager.InvokeEnd("test-function", "test-res-spec", "test-instance-id")
			convey.So(called, convey.ShouldBeFalse)
		})
	})
}
