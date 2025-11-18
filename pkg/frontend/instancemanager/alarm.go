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

// Package instancemanager -
package instancemanager

import (
	"fmt"
	"os"
	"time"

	"frontend/pkg/common/faas_common/alarm"
	"frontend/pkg/common/faas_common/constant"
)

func reportNoAvailableSchedulerInstAlarm() {
	alarmInfo := &alarm.LogAlarmInfo{
		AlarmID:    alarm.NoAvailableSchedulerInstance00001,
		AlarmName:  "NoAvailableSchedulerInstance",
		AlarmLevel: alarm.Level2,
	}
	alarmDetail := &alarm.Detail{
		OpType: alarm.GenerateAlarmLog,
		SourceTag: os.Getenv(constant.PodNameEnvKey) + "|" + os.Getenv(constant.PodIPEnvKey) +
			"|" + os.Getenv(constant.ClusterName) + "|NoAvailableSchedulerInstance",
		Details:        fmt.Sprintf("There is no available scheduler instance"),
		StartTimestamp: int(time.Now().Unix()),
		EndTimestamp:   0,
	}
	if os.Getenv(constant.CloudMapId) != "" {
		alarmDetail.Details += fmt.Sprintf(", environment name: %s", os.Getenv(constant.CloudMapId))
		alarmDetail.SourceTag = os.Getenv(constant.CloudMapId) + "|" + alarmDetail.SourceTag
	}
	alarm.ReportOrClearAlarm(alarmInfo, alarmDetail)
}

func clearNoAvailableSchedulerInstAlarm() {
	alarmInfo := &alarm.LogAlarmInfo{
		AlarmID:    alarm.NoAvailableSchedulerInstance00001,
		AlarmName:  "NoAvailableSchedulerInstance",
		AlarmLevel: alarm.Level2,
	}
	alarmDetail := &alarm.Detail{
		OpType: alarm.ClearAlarmLog,
		SourceTag: os.Getenv(constant.PodNameEnvKey) + "|" + os.Getenv(constant.PodIPEnvKey) +
			"|" + os.Getenv(constant.ClusterName) + "|NoAvailableSchedulerInstance",
		Details:        fmt.Sprintf("There is available scheduler instance"),
		StartTimestamp: 0,
		EndTimestamp:   int(time.Now().Unix()),
	}
	if os.Getenv(constant.CloudMapId) != "" {
		alarmDetail.Details += fmt.Sprintf(", environment name: %s", os.Getenv(constant.CloudMapId))
		alarmDetail.SourceTag = os.Getenv(constant.CloudMapId) + "|" + alarmDetail.SourceTag
	}
	alarm.ReportOrClearAlarm(alarmInfo, alarmDetail)
}
