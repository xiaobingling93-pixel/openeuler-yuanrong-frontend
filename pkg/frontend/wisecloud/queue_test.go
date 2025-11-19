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
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/instanceconfig"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/queue"
	"frontend/pkg/common/faas_common/resspeckey"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/wisecloudtool"
	"frontend/pkg/frontend/functionmeta"
	"frontend/pkg/frontend/instanceconfigmanager"
	"frontend/pkg/frontend/instancemanager"
)

func TestQueueManager_ProcessFunctionDelete(t *testing.T) {
	convey.Convey("测试 ProcessFunctionDelete", t, func() {
		// 初始化数据
		funcKey1 := "test-func1"
		resSpec1 := &resspeckey.ResSpecKey{InvokeLabel: "test-label1"}

		funcKey2 := "test-func2"
		resSpec2 := &resspeckey.ResSpecKey{InvokeLabel: "test-label2"}
		// 创建 QueueManager 并添加测试队列
		manager := &QueueManager{
			queuesMap: map[string]map[string]*reqQueue{
				funcKey1: {
					resSpec1.String(): newQueue(funcKey1, resSpec1, &instanceconfig.Configuration{}),
				},
				funcKey2: {
					resSpec2.String(): newQueue(funcKey2, resSpec2, &instanceconfig.Configuration{}),
				},
			},
			logger:  log.GetLogger(),
			RWMutex: sync.RWMutex{},
		}

		// 执行测试
		manager.ProcessFunctionDelete(&types.FuncSpec{FunctionKey: funcKey1})
		manager.ProcessFunctionDelete(&types.FuncSpec{FunctionKey: funcKey2})

		// 验证结果
		convey.So(manager.queuesMap, convey.ShouldNotContainKey, funcKey1)
		convey.So(manager.queuesMap, convey.ShouldNotContainKey, funcKey2)
	})
}

func TestQueueManager_ProcessInsConfigDelete(t *testing.T) {
	convey.Convey("测试 ProcessInsConfigDelete", t, func() {
		// 初始化数据
		funcKey := "test-func"
		invokeLabel1 := "test-label1"
		resSpec1 := &resspeckey.ResSpecKey{InvokeLabel: invokeLabel1}

		invokeLabel2 := "test-label2"
		resSpec2 := &resspeckey.ResSpecKey{InvokeLabel: invokeLabel2}

		// 创建 QueueManager 并添加测试队列
		queue1 := newQueue(funcKey, resSpec1, &instanceconfig.Configuration{})
		queue2 := newQueue(funcKey, resSpec2, &instanceconfig.Configuration{})

		manager := &QueueManager{
			queuesMap: map[string]map[string]*reqQueue{
				funcKey: {
					resSpec1.String(): queue1,
					resSpec2.String(): queue2,
				},
			},
			logger:  log.GetLogger(),
			RWMutex: sync.RWMutex{},
		}

		// 执行测试1
		manager.ProcessInsConfigDelete(&instanceconfig.Configuration{
			FuncKey:       funcKey,
			InstanceLabel: invokeLabel1,
		})

		// 验证结果1
		convey.So(len(manager.queuesMap[funcKey]), convey.ShouldEqual, 1)

		// 执行测试2
		manager.ProcessInsConfigDelete(&instanceconfig.Configuration{
			FuncKey:       funcKey,
			InstanceLabel: invokeLabel2,
		})

		// 验证结果2
		convey.So(len(manager.queuesMap[funcKey]), convey.ShouldEqual, 0)
	})
}

func TestQueueManager_AddPendingRequest(t *testing.T) {
	convey.Convey("测试 addPendingRequest", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		funcKey := "test-func"
		invokeLabel := "test-label"
		resSpec := &resspeckey.ResSpecKey{InvokeLabel: invokeLabel}
		pendingReq0 := &PendingRequest{
			CreatedTime:     time.Now(),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}
		pendingReq1 := &PendingRequest{
			CreatedTime:     time.Now(),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}

		// mock 函数元数据加载
		patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*types.FuncSpec, bool) {
			return &types.FuncSpec{FunctionKey: funcKey}, true
		})

		// mock 实例配置加载
		patches.ApplyFunc(instanceconfigmanager.Load, func(string, string) (*instanceconfig.Configuration, bool) {
			return &instanceconfig.Configuration{
				FuncKey:       funcKey,
				InstanceLabel: invokeLabel,
			}, true
		})

		coldStartCount := 0
		wg := sync.WaitGroup{}
		wg.Add(1)
		coldStartProvider = &wisecloudtool.PodOperator{}
		patches.ApplyMethod(reflect.TypeOf(coldStartProvider), "ColdStart", func(_ *wisecloudtool.PodOperator, _ string, _ resspeckey.ResSpecKey) error {
			coldStartCount++
			if coldStartCount == 1 {
				wg.Done()
			}
			return nil
		})

		// 创建 QueueManager
		manager := &QueueManager{
			queuesMap: make(map[string]map[string]*reqQueue),
			logger:    log.GetLogger(),
		}

		// 执行测试
		manager.AddPendingRequest(funcKey, resSpec, pendingReq0)
		manager.AddPendingRequest(funcKey, resSpec, pendingReq1)

		// 验证结果
		convey.So(manager.queuesMap[funcKey], convey.ShouldNotBeNil)
		convey.So(manager.queuesMap[funcKey][resSpec.String()], convey.ShouldNotBeNil)

		wg.Wait()
		convey.So(coldStartCount, convey.ShouldEqual, 1)

		// 收尾
		manager.ProcessFunctionDelete(&types.FuncSpec{FunctionKey: funcKey})

		// 测试冷启动失败场景
		wgFailed := sync.WaitGroup{}
		wgFailed.Add(1)
		coldStartCount = 0
		pendingReq0 = &PendingRequest{
			CreatedTime:     time.Now(),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}
		pendingReq1 = &PendingRequest{
			CreatedTime:     time.Now(),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}
		patches.ApplyMethod(reflect.TypeOf(coldStartProvider), "ColdStart", func(_ *wisecloudtool.PodOperator, _ string, _ resspeckey.ResSpecKey) error {
			coldStartCount++
			if coldStartCount == 1 {
				wgFailed.Done()
			}
			return fmt.Errorf("error")
		})

		processEmptyChan := sync.WaitGroup{}
		processEmptyChan.Add(1)
		processEmptyCount := 0
		patches.ApplyMethod(reflect.TypeOf(queueManager), "ProcessQueueEmpty", func(_ *QueueManager, _ string, _ *resspeckey.ResSpecKey) {
			processEmptyCount++
			if processEmptyCount == 1 {
				processEmptyChan.Done()
			}
		})

		manager.AddPendingRequest(funcKey, resSpec, pendingReq0)
		manager.AddPendingRequest(funcKey, resSpec, pendingReq1)
		wgFailed.Wait()

		convey.So(coldStartCount, convey.ShouldEqual, 1)
		result0 := <-pendingReq0.ResultChan
		convey.So(result0.Error, convey.ShouldNotBeNil)
		result1 := <-pendingReq1.ResultChan
		convey.So(result1.Error, convey.ShouldNotBeNil)
		processEmptyChan.Wait()
	})
}

func TestQueueManager_AddPendingRequest_FuncMetaNotFound(t *testing.T) {
	convey.Convey("测试 addPendingRequest 函数元数据不存在", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		// mock 函数元数据加载返回不存在
		patches.ApplyFunc(functionmeta.LoadFuncSpec, func(string) (*types.FuncSpec, bool) {
			return nil, false
		})

		manager := &QueueManager{
			queuesMap: make(map[string]map[string]*reqQueue),
		}

		// 创建测试用的 pending 请求
		resultChan := make(chan *PendingResponse, 1)
		pendingReq := &PendingRequest{
			ResultChan: resultChan,
		}

		// 执行测试
		manager.AddPendingRequest("not-exist", &resspeckey.ResSpecKey{}, pendingReq)

		// 验证结果
		convey.So(len(resultChan), convey.ShouldEqual, 1)
		resp := <-resultChan
		convey.So(resp.Error, convey.ShouldNotBeNil)
		convey.So(resp.Error.Error(), convey.ShouldContainSubstring, "function metadata not found")
	})
}

func TestQueueManager_ProcessInstanceUpdate(t *testing.T) {
	convey.Convey("测试 ProcessInstanceUpdate", t, func() {
		// 使用 gomonkey mock 依赖
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		funcKey := "test-func"
		invokeLabel := "test-label"
		invokeLabelFake := "test-label-fake"
		resSpec := &resspeckey.ResSpecKey{InvokeLabel: "test-label"}
		resJson := &resspeckey.ResourceSpecification{
			InvokeLabel: invokeLabel,
		}

		resSpecFake := &resspeckey.ResSpecKey{InvokeLabel: invokeLabelFake}
		resJsonBytes, _ := json.Marshal(resJson)
		resJsonFakeBytes, _ := json.Marshal(resSpecFake)

		instance := &types.InstanceSpecification{
			CreateOptions: map[string]string{
				constant.FunctionKeyNote:  funcKey,
				constant.ResourceSpecNote: string(resJsonBytes),
			},
		}

		// 创建 QueueManager 并添加测试队列
		queue := &reqQueue{
			funcKey:   funcKey,
			resSpec:   resSpec,
			FifoQueue: queue.NewFifoQueue(nil),
		}

		// 添加一个待处理请求
		queue.Lock()
		queue.PushBack(&PendingRequest{
			ResultChan: make(chan *PendingResponse, 1),
		})
		queue.Unlock()

		manager := &QueueManager{
			queuesMap: map[string]map[string]*reqQueue{
				funcKey: {
					resSpec.String(): queue,
				},
			},
		}

		// mock 实例管理器
		patches.ApplyMethodFunc(instancemanager.GetGlobalInstanceScheduler(),
			"GetRandomInstanceWithoutUnexpectedInstance",
			func(string, string, []string, api.FormatLogger) *types.InstanceSpecification {
				return instance
			})

		// 执行测试,传入非对应函数的key
		manager.ProcessInstanceUpdate(&types.InstanceSpecification{CreateOptions: map[string]string{
			constant.FunctionKeyNote:  "mock-test",
			constant.ResourceSpecNote: string(resJsonBytes),
		}})
		// 验证结果
		convey.So(queue.Len(), convey.ShouldEqual, 1)

		// 执行测试,传入对应函数非对应label的key
		manager.ProcessInstanceUpdate(&types.InstanceSpecification{CreateOptions: map[string]string{
			constant.FunctionKeyNote:  funcKey,
			constant.ResourceSpecNote: string(resJsonFakeBytes),
		}})
		// 验证结果
		convey.So(queue.Len(), convey.ShouldEqual, 1)

		// 执行测试，传入对应的函数的key
		manager.ProcessInstanceUpdate(instance)

		// 验证结果
		convey.So(queue.Len(), convey.ShouldEqual, 0)
	})
}

func TestQueue_AddPendingRequest_ExceedLimit(t *testing.T) {
	convey.Convey("测试队列请求超过限制", t, func() {
		queue := &reqQueue{
			FifoQueue: queue.NewFifoQueue(nil),
			logger:    log.GetLogger().With(zap.String("funcKey", "test"), zap.String("resSpecKey", "test")),
		}

		// 填充队列到上限
		for i := 0; i < 100; i++ {
			queue.PushBack(&PendingRequest{
				ResultChan: make(chan *PendingResponse, 1),
			})
		}

		// 创建一个会触发超限的请求
		resultChan := make(chan *PendingResponse, 1)
		pendingReq := &PendingRequest{
			ResultChan: resultChan,
		}

		// 执行测试
		queue.addPendingRequest(pendingReq)

		// 验证结果
		convey.So(len(resultChan), convey.ShouldEqual, 1)
		resp := <-resultChan
		convey.So(resp.Error, convey.ShouldNotBeNil)
		convey.So(resp.Error.Error(), convey.ShouldContainSubstring, "too many request")
	})
}

func TestQueueManager_ProcessQueueEmpty(t *testing.T) {
	convey.Convey("测试 ProcessQueueEmpty 方法", t, func() {
		// 准备测试数据
		funcKey := "test-func"
		resSpec := &resspeckey.ResSpecKey{InvokeLabel: "test-label"}
		resSpecStr := resSpec.String()

		// 测试用例组
		convey.Convey("当队列存在且为空时", func() {
			// 创建 mock 队列
			mockQueue := &reqQueue{
				RWMutex:   sync.RWMutex{},
				FifoQueue: queue.NewFifoQueue(nil),
			}

			// 准备 QueueManager
			manager := &QueueManager{
				queuesMap: map[string]map[string]*reqQueue{
					funcKey: {
						resSpecStr: mockQueue,
					},
				},
			}

			// mock reqQueue 方法
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			var destroyCalled bool
			wg := sync.WaitGroup{}
			wg.Add(1)
			patches.ApplyPrivateMethod(mockQueue, "destroy", func() {
				destroyCalled = true
				wg.Done()
			})

			// 执行测试
			manager.ProcessQueueEmpty(funcKey, resSpec)

			// 验证结果
			wg.Wait()
			convey.So(destroyCalled, convey.ShouldBeTrue)
			convey.So(manager.queuesMap[funcKey], convey.ShouldNotContainKey, resSpecStr)
		})

		convey.Convey("当队列不存在时", func() {
			manager := &QueueManager{
				queuesMap: make(map[string]map[string]*reqQueue),
			}

			// 执行测试
			manager.ProcessQueueEmpty(funcKey, resSpec)

			// 验证 queuesMap 未被修改
			convey.So(manager.queuesMap, convey.ShouldNotContainKey, funcKey)
		})

		convey.Convey("当队列不为空时", func() {
			mockQueue := &reqQueue{
				RWMutex:   sync.RWMutex{},
				FifoQueue: queue.NewFifoQueue(nil),
			}
			mockQueue.PushBack(&PendingRequest{
				CreatedTime:     time.Time{},
				ScheduleTimeout: 0,
				ResultChan:      make(chan *PendingResponse),
			})

			manager := &QueueManager{
				queuesMap: map[string]map[string]*reqQueue{
					funcKey: {
						resSpecStr: mockQueue,
					},
				},
			}

			// mock 队列不为空
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			var destroyCalled bool
			patches.ApplyPrivateMethod(mockQueue, "destroy", func() {
				destroyCalled = true
			})

			// 执行测试
			manager.ProcessQueueEmpty(funcKey, resSpec)

			// 验证队列未被销毁
			time.Sleep(500 * time.Millisecond)
			convey.So(destroyCalled, convey.ShouldBeFalse)
			convey.So(manager.queuesMap[funcKey], convey.ShouldContainKey, resSpecStr)
		})

		convey.Convey("当删除最后一个队列时清理父map", func() {
			mockQueue := &reqQueue{
				RWMutex:   sync.RWMutex{},
				FifoQueue: queue.NewFifoQueue(nil),
			}

			manager := &QueueManager{
				queuesMap: map[string]map[string]*reqQueue{
					funcKey: {
						resSpecStr: mockQueue,
					},
				},
			}

			// 执行测试
			manager.ProcessQueueEmpty(funcKey, resSpec)

			// 验证整个 funcKey 映射被清除
			convey.So(manager.queuesMap, convey.ShouldNotContainKey, funcKey)
		})
	})
}

func TestQueue_TimeoutLoop(t *testing.T) {
	convey.Convey("测试队列超时处理循环", t, func() {
		// 使用 gomonkey mock 时间
		patches := gomonkey.NewPatches()
		defer patches.Reset()

		now := time.Now()
		coldStartProvider = &wisecloudtool.PodOperator{}
		patches.ApplyMethodFunc(reflect.TypeOf(coldStartProvider), "ColdStart", func(funcKeyWithRes string, resSpec resspeckey.ResSpecKey, nuwaRuntimeInfo *types.NuwaRuntimeInfo) error {
			return nil
		})
		queue := newQueue("test-func", &resspeckey.ResSpecKey{}, &instanceconfig.Configuration{})

		// 添加一个已经超时的请求
		timeoutReq := &PendingRequest{
			CreatedTime:     now.Add(-11 * time.Second),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}
		timeoutReq0 := &PendingRequest{
			CreatedTime:     now.Add(-11 * time.Second),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}
		timeoutReq1 := &PendingRequest{
			CreatedTime:     now.Add(-8 * time.Second),
			ScheduleTimeout: 10 * time.Second,
			ResultChan:      make(chan *PendingResponse, 1),
		}

		queue.Lock()
		queue.PushBack(timeoutReq)
		queue.PushBack(timeoutReq0)
		queue.PushBack(timeoutReq1)
		queue.Unlock()

		// 等待超时处理
		time.Sleep(1*time.Second + 100*time.Millisecond)

		// 验证结果
		convey.So(len(timeoutReq.ResultChan), convey.ShouldEqual, 1)
		convey.So(queue.Len(), convey.ShouldEqual, 1)

		time.Sleep(2 * time.Second)
		convey.So(queue.Len(), convey.ShouldEqual, 0)

		// 停止队列
		queue.destroy()
	})
}
