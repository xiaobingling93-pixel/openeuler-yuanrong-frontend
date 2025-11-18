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

package remoteclientlease

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/types"
)

func TestFaasManager(t *testing.T) {
	events := []etcd3.Event{
		{
			Type: etcd3.PUT,
			Value: []byte(`{
    "instanceID": "3f079541-15fc-4009-8c41-50b2b2936772",
    "instanceStatus": {
        "code": 3,
        "msg": "running"
    }}`),
		},
		{
			Type: etcd3.PUT,
			Value: []byte(`{
    "instanceID": "3f079541-15fc-4009-8c41-50b2b2936772",
    "instanceStatus": {
        "code": 5,
        "msg": "exiting"
    }}`),
		},
		{
			Type:  etcd3.PUT,
			Value: []byte("value1"),
		},
	}
	Convey("Test faas manager handler", t, func() {
		instanceInfo := &types.InstanceInfo{
			FunctionName: "faasmanager",
			InstanceName: "3f079541-15fc-4009-8c41-50b2b2936772",
		}
		UpdateFaasManager(&events[0], instanceInfo)
		So(info.funcKey, ShouldEqual, "faasmanager")
		So(info.instanceID, ShouldEqual, "3f079541-15fc-4009-8c41-50b2b2936772")
		DeleteFaasManager(instanceInfo)
		So(info, ShouldBeNil)
		instanceInfo = &types.InstanceInfo{
			FunctionName: "faasmanager",
			InstanceName: "3f079541-15fc-4009-8c41-50b2b2936772",
		}
		UpdateFaasManager(&events[1], instanceInfo)
		So(info, ShouldBeNil)
		DeleteFaasManager(instanceInfo)
		So(info, ShouldBeNil)
		UpdateFaasManager(&events[2], instanceInfo)
		So(info, ShouldBeNil)
		DeleteFaasManager(instanceInfo)
		So(info, ShouldBeNil)
	})
}
