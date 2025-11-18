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

// Package instancleaseemanager for message process
package instanceleasemanager

import (
	"sync"
	"time"
)

// InstanceReport contains the necessary metric info
type InstanceReport struct {
	ProcReqNum  int64 `json:"procReqNum"`
	AvgProcTime int64 `json:"avgProcTime"`
	MaxProcTime int64 `json:"maxProcTime"`
	IsAbnormal  bool  `json:"isAbnormal"`
}

// ReportRecord is a counter to calculate the metric
type ReportRecord struct {
	// these two field will be accessed by only one go routine
	// the requests completed at the current report period
	requestsCount int64
	// the total time spent by the requests completed at the current report period
	totalDuration int64
	// the max of the time spent by all the requests yet
	maxDuration int64
	isAbnormal  bool
	sync.RWMutex
}

func (mc *ReportRecord) recordAbnormal() {
	mc.Lock()
	mc.isAbnormal = true
	mc.Unlock()
}

func (mc *ReportRecord) recordRequest(duration time.Duration) {
	mc.Lock()
	mc.requestsCount++
	durationInMill := duration.Milliseconds()
	mc.totalDuration += durationInMill
	if durationInMill > mc.maxDuration {
		mc.maxDuration = durationInMill
	}
	mc.Unlock()
}

func (mc *ReportRecord) report(reset bool) InstanceReport {
	mc.Lock()
	report := InstanceReport{
		ProcReqNum:  mc.requestsCount,
		MaxProcTime: mc.maxDuration,
		IsAbnormal:  mc.isAbnormal,
	}
	if mc.requestsCount == 0 {
		report.AvgProcTime = -1
	} else {
		report.AvgProcTime = mc.totalDuration / mc.requestsCount
	}
	if reset {
		mc.requestsCount = 0
		mc.totalDuration = 0
	}
	mc.Unlock()
	return report
}
