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

package schedulerproxy

import (
	"reflect"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/loadbalance"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/types"
)

func Test_schedulerProxy_Add(t *testing.T) {
	convey.Convey("Add", t, func() {
		Proxy.Add(&types.InstanceInfo{InstanceName: "instance1"}, log.GetLogger())
		_, ok := Proxy.faasSchedulers.Load("instance1")
		convey.So(ok, convey.ShouldEqual, true)

		convey.So(Proxy.Exist("instance1", ""), convey.ShouldBeTrue)
		convey.So(Proxy.ExistInstanceName("instance1"), convey.ShouldBeTrue)

		Proxy.Add(&types.InstanceInfo{InstanceName: "instance1", InstanceID: "1"}, log.GetLogger())
		convey.So(Proxy.Exist("instance1", "1"), convey.ShouldBeTrue)
		convey.So(Proxy.ExistInstanceName("instance1"), convey.ShouldBeTrue)
		convey.So(Proxy.Exist("instance1", ""), convey.ShouldBeFalse)
	})
}

func Test_schedulerProxy_Remove(t *testing.T) {
	convey.Convey("Remove", t, func() {
		Proxy.Add(&types.InstanceInfo{InstanceName: "instance1"}, log.GetLogger())
		_, ok := Proxy.faasSchedulers.Load("instance1")
		convey.So(ok, convey.ShouldEqual, true)

		Proxy.Remove(&types.InstanceInfo{InstanceName: "instance1"}, log.GetLogger())
		_, ok = Proxy.faasSchedulers.Load("instance1")
		convey.So(ok, convey.ShouldEqual, false)
		convey.So(Proxy.ExistInstanceName("instance1"), convey.ShouldBeFalse)
	})
}

func Test_schedulerProxy_Get(t *testing.T) {
	convey.Convey("Get", t, func() {
		convey.Convey("assert failed", func() {
			patches := []*gomonkey.Patches{
				gomonkey.ApplyMethod(reflect.TypeOf(&loadbalance.ConcurrentCHGeneric{}), "Next",
					func(_ *loadbalance.ConcurrentCHGeneric, name string, move bool) interface{} {
						return 123
					}),
			}
			defer func() {
				for _, patch := range patches {
					time.Sleep(100 * time.Millisecond)
					patch.Reset()
				}
			}()

			_, err := Proxy.Get("functionKey", log.GetLogger())
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("no avaiable faas scheduler was found", func() {
			patches := []*gomonkey.Patches{
				gomonkey.ApplyMethod(reflect.TypeOf(&loadbalance.ConcurrentCHGeneric{}), "Next",
					func(_ *loadbalance.ConcurrentCHGeneric, name string, move bool) interface{} {
						return ""
					}),
			}
			defer func() {
				for _, patch := range patches {
					time.Sleep(100 * time.Millisecond)
					patch.Reset()
				}
			}()

			_, err := Proxy.Get("functionKey", log.GetLogger())
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("failed to get the faas scheduler named", func() {
			patches := []*gomonkey.Patches{
				gomonkey.ApplyMethod(reflect.TypeOf(&loadbalance.ConcurrentCHGeneric{}), "Next",
					func(_ *loadbalance.ConcurrentCHGeneric, name string, move bool) interface{} {
						return "faaSScheduler"
					}),
			}
			defer func() {
				for _, patch := range patches {
					time.Sleep(100 * time.Millisecond)
					patch.Reset()
				}
			}()

			_, err := Proxy.Get("functionKey", log.GetLogger())
			convey.So(err, convey.ShouldBeError)
		})

		convey.Convey("success", func() {
			patches := []*gomonkey.Patches{
				gomonkey.ApplyMethod(reflect.TypeOf(&loadbalance.ConcurrentCHGeneric{}), "Next",
					func(_ *loadbalance.ConcurrentCHGeneric, name string, move bool) interface{} {
						return "instance1"
					}),
			}
			defer func() {
				for _, patch := range patches {
					time.Sleep(100 * time.Millisecond)
					patch.Reset()
				}
			}()
			Proxy.Add(&types.InstanceInfo{InstanceName: "instance1"}, log.GetLogger())
			scheduler, err := Proxy.Get("functionKey", log.GetLogger())
			convey.So(err, convey.ShouldBeNil)
			convey.So(scheduler.InstanceName, convey.ShouldEqual, "instance1")
		})
	})
}

func Test_schedulerProxy_GetSchedulerByInstanceName(t *testing.T) {
	convey.Convey("GetSchedulerByInstanceName", t, func() {
		scheduler := &types.InstanceInfo{InstanceName: "name1", InstanceID: "id1"}
		Proxy.Add(scheduler, log.GetLogger())
		getScheduler, err := Proxy.GetSchedulerByInstanceName("name1", "")
		convey.So(err, convey.ShouldBeNil)
		convey.So(getScheduler.InstanceID, convey.ShouldEqual, "id1")

		getScheduler, err = Proxy.GetSchedulerByInstanceName("name2", "")
		convey.So(err, convey.ShouldNotBeNil)
	})
}
