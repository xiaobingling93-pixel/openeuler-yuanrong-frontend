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
	"testing"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/types"
)

func Test_handler(t *testing.T) {
	instanceEtcdInfoBytes, _ := json.Marshal(&types.InstanceSpecification{InstanceStatus: types.InstanceStatus{}})
	type args struct {
		event *etcd3.Event
	}
	tests := []struct {
		name string
		args args
	}{
		{"case1 event put", args{event: &etcd3.Event{
			Type:  etcd3.PUT,
			Value: instanceEtcdInfoBytes,
		}}},
		{"case2 event delete", args{event: &etcd3.Event{
			Type: etcd3.DELETE,
			Key:  "/sn/instance/business/yrk/tenant/12/function/0-system-faasscheduler/version/$latest/defaultaz/requestID/123",
		}}},
		{"case3 event error", args{event: &etcd3.Event{
			Type: etcd3.ERROR,
		}}},
		{"case4 event default", args{event: &etcd3.Event{
			Type: etcd3.SYNCED,
		}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instanceInfoHandler(tt.args.event)
		})
	}
}

func Test_InstanceInfoFilter(t *testing.T) {
	type args struct {
		event *etcd3.Event
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"case1", args{event: &etcd3.Event{
			Type: etcd3.PUT,
			Key:  "/sn/instance/business/yrk/tenant/12/function/0-system-faasscheduler/version/$latest/defaultaz/requestID/123",
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := instanceInfoFilter(tt.args.event); got != tt.want {
				t.Errorf("filter() = %v, want %v", got, tt.want)
			}
		})
	}
}
