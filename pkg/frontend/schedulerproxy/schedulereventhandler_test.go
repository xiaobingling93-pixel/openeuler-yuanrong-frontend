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
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/config"
)

func getBytes(info *types.InstanceSpecification) []byte {
	bytes, _ := json.Marshal(info)
	return bytes
}

func TestProcessUpdate(t *testing.T) {
	foundedCall := 0
	missedCall := 0
	defer gomonkey.ApplyMethod(reflect.TypeOf(Proxy), "Add", func(_ *ProxyManager, scheduler *types.InstanceInfo, _ api.FormatLogger) {
		foundedCall++
	}).Reset()
	resetCall := 0
	defer gomonkey.ApplyMethod(reflect.TypeOf(Proxy), "Reset", func(_ *ProxyManager) {
		resetCall++
	}).Reset()
	defer gomonkey.ApplyMethod(reflect.TypeOf(Proxy), "Remove", func(_ *ProxyManager, scheduler *types.InstanceInfo, _ api.FormatLogger) {
		missedCall++
	}).Reset()
	convey.Convey("Test module scheduler ProcessUpdate", t, func() {
		oldType := config.GetConfig().SchedulerKeyPrefixType
		config.GetConfig().SchedulerKeyPrefixType = "module"
		event := &etcd3.Event{
			Key: "/scheduler1", // no need
		}
		info := &types.InstanceInfo{
			InstanceName: "instanceName1",
		}

		insSpec := &types.InstanceSpecification{
			InstanceID: "",
		}
		event.Value = getBytes(insSpec)
		ProcessUpdate(event, info, log.GetLogger())
		convey.So(foundedCall, convey.ShouldEqual, 1)
		convey.So(resetCall, convey.ShouldEqual, 1)

		info = &types.InstanceInfo{InstanceName: "instanceName1"}
		insSpec = &types.InstanceSpecification{InstanceID: "instanceId1"}
		event.Value = getBytes(insSpec)
		ProcessUpdate(event, info, log.GetLogger())
		convey.So(foundedCall, convey.ShouldEqual, 2)
		convey.So(resetCall, convey.ShouldEqual, 2)

		event.Value = []byte(`{"createOptions":{}, "instanceStatus":{"code":3, "msg":"ok"}}`)
		info = &types.InstanceInfo{InstanceName: "instanceName1"}
		insSpec = &types.InstanceSpecification{InstanceID: "instanceId1", CreateOptions: make(map[string]string), InstanceStatus: types.InstanceStatus{
			Code: 3,
			Msg:  "ok",
		}}
		event.Value = getBytes(insSpec)
		ProcessUpdate(event, info, log.GetLogger())
		convey.So(foundedCall, convey.ShouldEqual, 3)
		convey.So(resetCall, convey.ShouldEqual, 3)
		config.GetConfig().SchedulerKeyPrefixType = oldType
	})

	convey.Convey("Test function scheduler ProcessUpdate", t, func() {
		oldType := config.GetConfig().SchedulerKeyPrefixType
		config.GetConfig().SchedulerKeyPrefixType = "function"

		foundedCall = 0
		resetCall = 0
		event := &etcd3.Event{
			Key:   "/scheduler1",
			Value: []byte(""),
		}
		info := &types.InstanceInfo{
			InstanceName: "instanceName1",
		}
		ProcessUpdate(event, info, log.GetLogger())
		convey.So(foundedCall, convey.ShouldEqual, 0)
		convey.So(resetCall, convey.ShouldEqual, 0)

		event.Value = []byte("{}")
		ProcessUpdate(event, info, log.GetLogger())
		convey.So(foundedCall, convey.ShouldEqual, 0)
		convey.So(resetCall, convey.ShouldEqual, 0)

		event.Value = []byte(`{"createOptions":{}, "instanceStatus":{"code":3, "msg":"ok"}}`)
		ProcessUpdate(event, info, log.GetLogger())
		convey.So(foundedCall, convey.ShouldEqual, 1)
		convey.So(resetCall, convey.ShouldEqual, 1)
		config.GetConfig().SchedulerKeyPrefixType = oldType
	})
}
