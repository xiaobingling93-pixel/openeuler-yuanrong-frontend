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
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/aliasroute"
	"frontend/pkg/common/faas_common/etcd3"
)

func TestStartWatchAlias(t *testing.T) {
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
		startWatchAlias(make(chan struct{}))
	})
}

func Test_AliasHandler(t *testing.T) {
	aliasByte, _ := json.Marshal(&aliasroute.AliasElement{})
	convey.Convey("handler", t, func() {
		convey.Convey("PUT", func() {
			aliasHandler(&etcd3.Event{
				Type:  etcd3.PUT,
				Value: aliasByte,
			})
		})

		convey.Convey("DELETE", func() {
			aliasHandler(&etcd3.Event{Type: etcd3.DELETE})
		})

		convey.Convey("SYNCED", func() {
			aliasHandler(&etcd3.Event{Type: etcd3.SYNCED})
		})

		convey.Convey("DEFAULT", func() {
			aliasHandler(&etcd3.Event{})
		})
	})
}

func Test_AliasFilter(t *testing.T) {
	convey.Convey("filter", t, func() {
		convey.Convey("len true", func() {
			filter := aliasFilter(&etcd3.Event{
				Key: "sn/aliases/business/yrk/tenant/{tenantId}/function/{function-name}/{alias-name}"})
			convey.So(filter, convey.ShouldEqual, true)
		})

		convey.Convey("true", func() {
			filter := aliasFilter(&etcd3.Event{
				Key: "sn/functions/business/yrk/tenant/{tenantId}/function/{function-name}/{alias-name}"})
			convey.So(filter, convey.ShouldEqual, true)
		})

		convey.Convey("success", func() {
			filter := aliasFilter(&etcd3.Event{
				Key: "/sn/aliases/business/yrk/tenant/{tenantId}/function/{function-name}/{alias-name}"})
			convey.So(filter, convey.ShouldEqual, false)
		})
	})
}
