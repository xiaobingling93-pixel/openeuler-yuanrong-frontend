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
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
	"time"

	"frontend/pkg/common/faas_common/etcd3"
)

func TestStartWatchFunctionMeta(t *testing.T) {
	convey.Convey("StartWatch", t, func() {
		patches := []*gomonkey.Patches{
			gomonkey.ApplyMethod(reflect.TypeOf(&etcd3.EtcdWatcher{}), "StartWatch", func(ew *etcd3.EtcdWatcher) {
			}),
			gomonkey.ApplyFunc(etcd3.GetMetaEtcdClient, func() *etcd3.EtcdClient {
				return &etcd3.EtcdClient{}
			}),
		}
		defer func() {
			for _, patch := range patches {
				time.Sleep(100 * time.Millisecond)
				patch.Reset()
			}
		}()
		startWatchFunctionMeta(make(chan struct{}))
	})
}

func TestStartWatchCAEFunctionMeta(t *testing.T) {
	convey.Convey("StartWatch 01", t, func() {
		patches := []*gomonkey.Patches{
			gomonkey.ApplyMethod(reflect.TypeOf(&etcd3.EtcdWatcher{}), "StartWatch", func(ew *etcd3.EtcdWatcher) {
			}),
			gomonkey.ApplyFunc(etcd3.GetCAEMetaEtcdClient, func() *etcd3.EtcdClient {
				return &etcd3.EtcdClient{}
			}),
		}
		defer func() {
			for _, patch := range patches {
				time.Sleep(100 * time.Millisecond)
				patch.Reset()
			}
		}()
		startWatchCAEFunctionMeta(make(chan struct{}))
	})

	convey.Convey("StartWatch 02", t, func() {
		patches := []*gomonkey.Patches{
			gomonkey.ApplyMethod(reflect.TypeOf(&etcd3.EtcdWatcher{}), "StartWatch", func(ew *etcd3.EtcdWatcher) {
			}),
		}
		defer func() {
			for _, patch := range patches {
				time.Sleep(100 * time.Millisecond)
				patch.Reset()
			}
		}()
		startWatchCAEFunctionMeta(make(chan struct{}))
	})
}

func Test_FunctionMetaFilter(t *testing.T) {
	convey.Convey("filter", t, func() {
		convey.Convey("len false", func() {
			filter := functionMetaFilter(&etcd3.Event{
				Key: "sn/functions/business/<businessID>/tenant/<tenantID>/function/<functionName>/version/<version>"})
			convey.So(filter, convey.ShouldEqual, true)
		})

		convey.Convey("false", func() {
			filter := functionMetaFilter(&etcd3.Event{
				Key: "/sn/functions/business/<businessID>/tenant/<tenantID>/function/<functionName>/version/<version>"})
			convey.So(filter, convey.ShouldEqual, false)
		})

		convey.Convey("true", func() {
			filter := functionMetaFilter(&etcd3.Event{
				Key: "/sn/instance/business/<businessID>/tenant/<tenantID>/function/<functionName>/version/<version>"})
			convey.So(filter, convey.ShouldEqual, true)
		})
	})
}

func Test_FunctionMetaHandler(t *testing.T) {
	convey.Convey("handler", t, func() {
		convey.Convey("PUT", func() {
			functionMetaHandler(&etcd3.Event{Type: etcd3.PUT})
		})

		convey.Convey("DELETE", func() {
			functionMetaHandler(&etcd3.Event{Type: etcd3.DELETE})
		})

		convey.Convey("SYNCED", func() {
			functionMetaHandler(&etcd3.Event{Type: etcd3.SYNCED})
		})

		convey.Convey("DEFAULT", func() {
			functionMetaHandler(&etcd3.Event{})
		})
	})
}
