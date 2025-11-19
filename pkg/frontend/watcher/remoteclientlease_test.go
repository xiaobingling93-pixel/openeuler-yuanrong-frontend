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

package watcher

import (
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	. "github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/etcd3"
)

func TestStartWatchRemoteClientLease(t *testing.T) {
	Convey("StartWatchRemoteClientLease", t, func() {
		patches := []*gomonkey.Patches{
			gomonkey.ApplyMethod(reflect.TypeOf(&etcd3.EtcdWatcher{}), "StartWatch", func(ew *etcd3.EtcdWatcher) {
			}),
			gomonkey.ApplyFunc(etcd3.GetRouterEtcdClient, func() *etcd3.EtcdClient {
				return &etcd3.EtcdClient{}
			}),
		}
		defer func() {
			for _, patch := range patches {
				time.Sleep(100 * time.Millisecond)
				patch.Reset()
			}
		}()
		startWatchRemoteClientLease(make(chan struct{}))
	})
}

func TestHandler(t *testing.T) {
	events := []etcd3.Event{
		{
			Type: etcd3.PUT,
			Key:  "/sn/instance/business/yrk/tenant/1/function/faasmanager/version/latest/defaultaz/requestID/3f079541-15fc-4009-8c41-50b2b2936772",
			Value: []byte(`{
    "instanceID": "3f079541-15fc-4009-8c41-50b2b2936772",
    "instanceStatus": {
        "code": 3,
        "msg": "running"
    }}`),
		},
		{
			Type: etcd3.DELETE,
			Key:  "/sn/instance/business/yrk/tenant/1/function/faasmanager/version/latest/defaultaz/requestID/3f079541-15fc-4009-8c41-50b2b2936772",
		},
		{
			Type: etcd3.PUT,
			Key:  "/business/yrk/tenant/1/function/xxxxmanager/version/latest/defaultaz/3f079541-15fc-4009-8c41-50b2b2936772",
			Value: []byte(`{
    "instanceID": "3f079541-15fc-4009-8c41-50b2b2936772",
    "instanceStatus": {
        "code": 5,
        "msg": "exiting"
    }}`),
		},
		{
			Type: etcd3.DELETE,
			Key:  "/business/yrk/tenant/1/function/xxxxmanager/version/latest/defaultaz/3f079541-15fc-4009-8c41-50b2b2936772",
		},
	}
	Convey("Test faas manager handler", t, func() {
		remoteClientLeaseHandler(&events[0])
		So(events[0].Type, ShouldEqual, etcd3.PUT)
		remoteClientLeaseHandler(&events[1])
		So(events[1].Type, ShouldEqual, etcd3.DELETE)
		remoteClientLeaseHandler(&events[2])
		So(events[2].Type, ShouldEqual, etcd3.PUT)
		remoteClientLeaseHandler(&events[3])
		So(events[3].Type, ShouldEqual, etcd3.DELETE)
	})
}

func Test_isFaaSManager(t *testing.T) {
	Convey("test isFaaSManager", t, func() {
		manager := isFaaSManager("/sn/instance/business/yrk/tenant/0/function/0-system-faasmanager/version/$latest/defaultaz/d9300b9aec177ed300/0826cb0b-40bc-4e90-ab19-eb2e16b223eb")
		So(manager, ShouldBeTrue)
		manager = isFaaSManager("aaa")
		So(manager, ShouldBeFalse)
	})
}
