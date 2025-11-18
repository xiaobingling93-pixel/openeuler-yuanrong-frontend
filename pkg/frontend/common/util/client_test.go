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

package util

import (
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"
	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	mockUtils "frontend/pkg/common/faas_common/utils"
	"frontend/pkg/common/uuid"
)

func TestNewClientLibruntime(t *testing.T) {
	mock := &mockUtils.FakeLibruntimeSdkClient{}
	Convey("TestNewClientLibruntime", t, func() {
		testInstID := uuid.New().String()
		returnObjID := uuid.New().String()
		result := []byte(uuid.New().String())
		req := InvokeRequest{
			Function:   "test",
			Args:       nil,
			InstanceID: testInstID,
		}

		patches := []*gomonkey.Patches{
			gomonkey.ApplyMethod(reflect.TypeOf(mock), "GetAsync",
				func(_ *mockUtils.FakeLibruntimeSdkClient, objectID string, cb api.GetAsyncCallback) {
					cb(result, nil)
					return
				}),
			gomonkey.ApplyMethod(reflect.TypeOf(mock), "InvokeByFunctionName",
				func(_ *mockUtils.FakeLibruntimeSdkClient, funcMeta api.FunctionMeta, args []api.Arg,
					invokeOpt api.InvokeOptions) (string, error) {
					return testInstID, nil
				}),
			gomonkey.ApplyMethod(reflect.TypeOf(mock), "InvokeByInstanceId",
				func(_ *mockUtils.FakeLibruntimeSdkClient, funcMeta api.FunctionMeta, instanceID string, args []api.Arg,
					invokeOpt api.InvokeOptions) (string, error) {
					So(instanceID, ShouldEqual, testInstID)
					return returnObjID, nil
				}),
		}
		defer func() {
			for _, patch := range patches {
				patch.Reset()
			}
		}()

		client := newDefaultClientLibruntime(mock)
		So(client, ShouldNotBeNil)
		res, err := client.InvokeByName(req)
		So(err, ShouldBeNil)
		So(res, ShouldResemble, result)

		res, err = client.Invoke(req)
		So(err, ShouldBeNil)
		So(res, ShouldResemble, result)
	})
}

func Test_defaultClient_AcquireInstance(t *testing.T) {
	Convey("test AcquireInstance", t, func() {
		Convey("baseline", func() {
			mock := &mockUtils.FakeLibruntimeSdkClient{}
			client := newDefaultClientLibruntime(mock)
			instance, err := client.AcquireInstance("func", AcquireOption{
				DesignateInstanceID: "id",
				FuncSig:             "aaa",
				ResourceSpecs: map[string]int64{
					constant.ResourceCPUName:    1000,
					constant.ResourceMemoryName: 1000,
				},
				Timeout:        100,
				TrafficLimited: false,
			})
			So(err, ShouldBeNil)
			So(instance, ShouldNotBeNil)
		})
	})
}
