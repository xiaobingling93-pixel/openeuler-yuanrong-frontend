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

package instanceleasemanager

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/tls"
	commType "frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/common/util"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/schedulerproxy"
	"frontend/pkg/frontend/types"
)

func TestInstanceLeasePool_releaseInstanceLease(t *testing.T) {
	convey.Convey("release instance lease test", t, func() {
		convey.Convey("baseline", func() {
			pool := newInstanceLeasePool("func1")
			freeCalled := 0
			p := gomonkey.ApplyFunc((*InstanceLease).free, func(_ *InstanceLease, abnormal bool, record bool) {
				freeCalled++
			})
			defer p.Reset()
			pool.leaseRecord["aaa"] = &instanceLeaseRecord{
				lease:   &InstanceLease{},
				element: nil,
			}
			pool.releaseInstanceLease("aaa", true)
			convey.So(freeCalled, convey.ShouldEqual, 1)
		})
	})
}

func TestInstanceManager_ReleaseInstanceAllocation(t *testing.T) {
	convey.Convey("release instance allocation test", t, func() {
		convey.Convey("baseline", func() {
			im := GetInstanceManager()
			im.leasePools["func1"] = &InstanceLeasePool{}
			im.ReleaseInstanceAllocation(&commType.InstanceAllocationInfo{
				FuncKey:       "func1",
				FuncSig:       "",
				InstanceID:    "aaa",
				ThreadID:      "func1",
				LeaseInterval: 0,
			}, true, "")
		})
		convey.Convey("not exsit", func() {
			im := GetInstanceManager()
			im.ReleaseInstanceAllocation(&commType.InstanceAllocationInfo{
				FuncKey:       "func1",
				FuncSig:       "",
				InstanceID:    "aaa",
				ThreadID:      "func1",
				LeaseInterval: 0,
			}, true, "")
		})
	})
}

func Test_AcquireRepeatedLease(t *testing.T) {
	convey.Convey("AcquireRepeatedLease", t, func() {
		leasePool := newInstanceLeasePool("test-function")
		schedulerproxy.Proxy.Add(&commType.InstanceInfo{
			FunctionName: "test-scheduler",
			InstanceName: "test-schedulerID",
			Address:      "127.0.0.1",
		}, log.GetLogger())
		resp := &commType.InstanceResponse{
			InstanceAllocationInfo: commType.InstanceAllocationInfo{ThreadID: "lease1-1",
				InstanceID: "lease1", LeaseInterval: 100000},
			ErrorCode:     constant.InsReqSuccessCode,
			ErrorMessage:  "",
			SchedulerTime: 0,
		}
		body, _ := json.Marshal(resp)
		c := &fasthttp.Client{}
		defer gomonkey.ApplyMethod(reflect.TypeOf(c),
			"DoTimeout", func(c *fasthttp.Client, req *fasthttp.Request,
				resp *fasthttp.Response, timeout time.Duration) error {
				resp.SetBody(body)
				resp.SetStatusCode(200)
				return nil
			}).Reset()
		defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
			return &types.Config{
				HTTPSConfig: &tls.InternalHTTPSConfig{},
				LocalAuth:   &localauth.AuthConfig{},
			}
		}).Reset()
		lease1, snError := leasePool.acquireInstanceLease("", util.AcquireOption{
			DesignateInstanceID: "",
			SchedulerFuncKey:    "test-scheduler",
			SchedulerID:         "test-schedulerID",
			RequestID:           "123456789",
			TraceID:             "123456",
			ResourceSpecs:       nil,
			Timeout:             30,
			FuncSig:             "",
		})
		convey.So(leasePool.idleLeaseList.Len(), convey.ShouldEqual, 0)
		convey.So(lease1.ThreadID, convey.ShouldEqual, "lease1-1")
		convey.So(snError, convey.ShouldBeNil)

		// releaseLease and add to idleLeaseList
		leasePool.releaseInstanceLease(lease1.ThreadID, false)
		convey.So(leasePool.idleLeaseList.Len(), convey.ShouldEqual, 1)
		// acquire again within 100 ms,get lease from idleLeaseList
		lease1, snError = leasePool.acquireInstanceLease("", util.AcquireOption{
			DesignateInstanceID: "",
			SchedulerFuncKey:    "test-scheduler",
			SchedulerID:         "test-schedulerID",
			RequestID:           "123456789",
			TraceID:             "123456",
			ResourceSpecs:       nil,
			Timeout:             30,
			FuncSig:             "",
		})
		convey.So(leasePool.idleLeaseList.Len(), convey.ShouldEqual, 0)
		convey.So(lease1.ThreadID, convey.ShouldEqual, "lease1-1")
		convey.So(snError, convey.ShouldBeNil)

		// releaseLease and add to idleLeaseList
		leasePool.releaseInstanceLease(lease1.ThreadID, false)
		convey.So(leasePool.idleLeaseList.Len(), convey.ShouldEqual, 1)

		// 100ms no acquire, delete
		time.Sleep(120 * time.Millisecond)
		convey.So(len(leasePool.leaseRecord), convey.ShouldEqual, 0)
		convey.So(leasePool.idleLeaseList.Len(), convey.ShouldEqual, 0)

	})
}

func TestFree(t *testing.T) {
	convey.Convey("Test Free", t, func() {
		il := &InstanceLease{
			releaseCh:    make(chan struct{}, 1),
			reportRecord: &ReportRecord{},
			claimTime:    time.Now(),
		}
		convey.Convey("Free Abnormal", func() {

			il.free(true, true)
			if !il.reportRecord.isAbnormal {
				t.Error("recordAbnormal was not called")
			}
			<-il.releaseCh
			convey.So(il.reportRecord.isAbnormal, convey.ShouldBeTrue)
		})
		convey.Convey("Free Normal", func() {
			il.free(false, true)
			<-il.releaseCh
			convey.So(il.available, convey.ShouldBeFalse)
		})
	})
}

func TestInstanceLeasePool_handleLeaseLifeCycle(t *testing.T) {
	convey.Convey("test handleLeaseLifeCycle", t, func() {
		convey.Convey("renew failed", func() {
			pool := &InstanceLeasePool{leaseRecord: map[string]*instanceLeaseRecord{}}
			il := &InstanceLease{
				available:              false,
				releaseCh:              make(chan struct{}),
				InstanceAllocationInfo: commType.InstanceAllocationInfo{ThreadID: "aaa"},
				reportRecord:           &ReportRecord{},
			}
			pool.handleLeaseLifeCycle(il, nil)
			pool.handleLeaseLifeCycle(il, time.NewTimer(100*time.Millisecond))
			p := gomonkey.ApplyFunc((*InstanceLeasePool).renewHandler, func(_ *InstanceLeasePool, leaseID string, option util.AcquireOption,
				report InstanceReport) (*commType.InstanceResponse, error) {
				return nil, fmt.Errorf("error")
			})
			defer p.Reset()
			convey.So(len(pool.leaseRecord), convey.ShouldEqual, 0)
		})
	})
}

func TestInstanceLeasePool_renewHandler(t *testing.T) {
	convey.Convey("test renewHandler", t, func() {
		convey.Convey("baseline", func() {
			pool := &InstanceLeasePool{leaseRecord: map[string]*instanceLeaseRecord{}}
			gomonkey.ApplyFunc((*InstanceLeasePool).processRenewAndRelease, func(_ *InstanceLeasePool, action, leaseID string, option util.AcquireOption,
				report InstanceReport) (*commType.InstanceResponse, error) {
				return &commType.InstanceResponse{}, nil
			})
			response, err := pool.renewHandler("aaa", util.AcquireOption{}, InstanceReport{})
			convey.So(err, convey.ShouldBeNil)
			convey.So(response, convey.ShouldNotBeNil)
		})
		convey.Convey("renew failed", func() {
			pool := &InstanceLeasePool{leaseRecord: map[string]*instanceLeaseRecord{}}
			gomonkey.ApplyFunc((*InstanceLeasePool).processRenewAndRelease, func(_ *InstanceLeasePool, action, leaseID string, option util.AcquireOption,
				report InstanceReport) (*commType.InstanceResponse, error) {
				return nil, fmt.Errorf("error")
			})
			response, err := pool.renewHandler("aaa", util.AcquireOption{}, InstanceReport{})
			convey.So(response, convey.ShouldBeNil)
			convey.So(err, convey.ShouldNotBeNil)
		})
	})
}
