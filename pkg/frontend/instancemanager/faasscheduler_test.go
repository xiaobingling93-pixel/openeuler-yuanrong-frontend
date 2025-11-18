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

package instancemanager

import (
	"fmt"
	"sync"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/logger/log"
	commontype "frontend/pkg/common/faas_common/types"
)

func clearTest() {
	GetFaaSSchedulerInstanceManager().faaSSchedulerInstanceMap = make(map[string]*commontype.InstanceSpecification)
	GetFaaSSchedulerInstanceManager().synced.Store(false)
}

func Test_faaSSchedulerInstanceManager_complex(t *testing.T) {
	convey.Convey("faaSSchedulerInstanceManager_complex", t, func() {
		clearTest()
		GetFaaSSchedulerInstanceManager().addInstance("0", nil, log.GetLogger())
		convey.So(GetFaaSSchedulerInstanceManager().size(), convey.ShouldEqual, 1)

		wg := sync.WaitGroup{}

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				for j := 0; j < 10; j++ {
					GetFaaSSchedulerInstanceManager().addInstance(fmt.Sprintf("%d%d", i, j), nil, log.GetLogger())
					GetFaaSSchedulerInstanceManager().addInstance(fmt.Sprintf("%d%d", i, j), nil, log.GetLogger())
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		convey.So(GetFaaSSchedulerInstanceManager().size(), convey.ShouldEqual, 10*10+1)

		GetFaaSSchedulerInstanceManager().delInstance("0", log.GetLogger())
		convey.So(GetFaaSSchedulerInstanceManager().size(), convey.ShouldEqual, 10*10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				for j := 0; j < 10; j++ {
					GetFaaSSchedulerInstanceManager().delInstance(fmt.Sprintf("%d%d", i, j), log.GetLogger())
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		convey.So(GetFaaSSchedulerInstanceManager().IsEmpty(), convey.ShouldBeTrue)
	})
}

func Test_faaSSchedulerInstanceManager_alarm(t *testing.T) {
	convey.Convey("Test_faaSSchedulerInstanceManager_alarm", t, func() {
		clearTest()
		reportTrigger := false
		clearTrigger := false
		defer gomonkey.ApplyFunc(reportNoAvailableSchedulerInstAlarm, func() {
			reportTrigger = true
		}).Reset()
		defer gomonkey.ApplyFunc(clearNoAvailableSchedulerInstAlarm, func() {
			clearTrigger = true
		}).Reset()
		GetFaaSSchedulerInstanceManager().addInstance("0", nil, log.GetLogger())
		convey.So(clearTrigger, convey.ShouldBeFalse)
		convey.So(reportTrigger, convey.ShouldBeFalse)

		GetFaaSSchedulerInstanceManager().delInstance("0", log.GetLogger())
		convey.So(reportTrigger, convey.ShouldBeTrue)
		reportTrigger = false

		GetFaaSSchedulerInstanceManager().sync(log.GetLogger())
		convey.So(reportTrigger, convey.ShouldBeTrue)
		convey.So(clearTrigger, convey.ShouldBeFalse)

		GetFaaSSchedulerInstanceManager().addInstance("0", nil, log.GetLogger())
		convey.So(clearTrigger, convey.ShouldBeTrue)
		clearTest()
	})
}
