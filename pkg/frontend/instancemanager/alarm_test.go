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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/alarm"
)

func TestReportOrClearNoAvailableSchedulerInstAlarm(t *testing.T) {
	convey.Convey("report or clear no available scheduler instance alarm", t, func() {
		var alarmInfoArgs alarm.LogAlarmInfo
		var detailArgs alarm.Detail
		patch := gomonkey.ApplyFunc(alarm.ReportOrClearAlarm, func(alarmInfo *alarm.LogAlarmInfo, detail *alarm.Detail) {
			alarmInfoArgs = *alarmInfo
			detailArgs = *detail
		})
		defer patch.Reset()

		convey.Convey("report no available scheduler instance alarm", func() {
			reportNoAvailableSchedulerInstAlarm()

			alarmInfoExpected := alarm.LogAlarmInfo{
				AlarmID:    alarm.NoAvailableSchedulerInstance00001,
				AlarmName:  "NoAvailableSchedulerInstance",
				AlarmLevel: alarm.Level2,
			}
			alarmDetailExpected := alarm.Detail{
				OpType:       alarm.GenerateAlarmLog,
				SourceTag:    "|" + "|" + "|NoAvailableSchedulerInstance",
				Details:      "There is no available scheduler instance",
				EndTimestamp: 0,
			}
			convey.So(alarmInfoArgs, convey.ShouldResemble, alarmInfoExpected)
			convey.So(detailArgs.OpType, convey.ShouldEqual, alarmDetailExpected.OpType)
			convey.So(detailArgs.SourceTag, convey.ShouldEqual, alarmDetailExpected.SourceTag)
			convey.So(detailArgs.Details, convey.ShouldEqual, alarmDetailExpected.Details)
			convey.So(detailArgs.EndTimestamp, convey.ShouldEqual, alarmDetailExpected.EndTimestamp)
		})

		convey.Convey("clear no available scheduler instance alarm", func() {
			clearNoAvailableSchedulerInstAlarm()

			alarmInfoExpected := alarm.LogAlarmInfo{
				AlarmID:    alarm.NoAvailableSchedulerInstance00001,
				AlarmName:  "NoAvailableSchedulerInstance",
				AlarmLevel: alarm.Level2,
			}
			alarmDetailExpected := alarm.Detail{
				OpType:         alarm.ClearAlarmLog,
				SourceTag:      "|" + "|" + "|NoAvailableSchedulerInstance",
				Details:        "There is available scheduler instance",
				StartTimestamp: 0,
			}
			convey.So(alarmInfoArgs, convey.ShouldResemble, alarmInfoExpected)
			convey.So(detailArgs.OpType, convey.ShouldEqual, alarmDetailExpected.OpType)
			convey.So(detailArgs.SourceTag, convey.ShouldEqual, alarmDetailExpected.SourceTag)
			convey.So(detailArgs.Details, convey.ShouldEqual, alarmDetailExpected.Details)
			convey.So(detailArgs.StartTimestamp, convey.ShouldEqual, alarmDetailExpected.StartTimestamp)
		})
	})
}
