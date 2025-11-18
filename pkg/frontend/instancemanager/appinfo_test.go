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
	"encoding/json"
	"testing"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/types"
)

func TestGetAppStatusByID(t *testing.T) {
	StoreAppInfo("app-123456", &types.InstanceSpecification{
		InstanceID: "app-123456",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3,
			ExitCode: 1,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy=1.24 scipy=1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "sleep 200",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})

	type args struct {
		submissionID string
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"case1", args{submissionID: "app-123456"}, "RUNNING"},
		{"case2", args{submissionID: "app-654321"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAppStatusByID(tt.args.submissionID); got != tt.want {
				t.Errorf("GetNodeIPFromInstanceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAppDetailsByID(t *testing.T) {
	StoreAppInfo("app-123456", &types.InstanceSpecification{
		InstanceID: "app-123456",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3,
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25 && pip3.9 check",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "sleep 200",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})

	type args struct {
		submissionID string
	}

	tests := []struct {
		name string
		args args
		want *constant.AppInfo
	}{
		{"case1", args{submissionID: "app-123456"}, &constant.AppInfo{
			Key:          "app-123456",
			Type:         "SUBMISSION",
			Entrypoint:   "sleep 200",
			SubmissionID: "app-123456",
			Status:       "RUNNING",
			StartTime:    "",
			EndTime:      "",
			Metadata: map[string]string{
				"autoscenes_ids": "suzhou_std",
				"task_type":      "task_1",
				"ttl":            "1250",
			},
			RuntimeEnv: map[string]interface{}{
				"envVars":    "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
				"pip":        "pip3.9 install numpy==1.24 scipy==1.11.0 \u0026\u0026 pip3.9 check",
				"workingDir": "file:///usr1/deploy/file.zip",
			},
			DriverAgentHttpAddress: "",
			DriverNodeID:           "",
			DriverExitCode:         0,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := GetAppDetailsByID(tt.args.submissionID)
			if got.Status != tt.want.Status {
				t.Errorf("GetAppDetailsByID().Status = %v, want.Status %v", got.Status, tt.want.Status)
			}
			if got.Type != tt.want.Type {
				t.Errorf("GetAppDetailsByID().Type = %v, want.Type %v", got.Type, tt.want.Type)
			}
			if got.Entrypoint != tt.want.Entrypoint {
				t.Errorf("GetAppDetailsByID().entrypoint = %v, want.entrypoint %v", got.Entrypoint, tt.want.Entrypoint)
			}
			if got.DriverExitCode != tt.want.DriverExitCode {
				t.Errorf("GetAppDetailsByID().driverExitCode = %v, want.driverExitCode %v", got.DriverExitCode, tt.want.DriverExitCode)
			}
		})
	}

}

func TestProcessUpdateAndDelete(t *testing.T) {
	appInfo := &types.InstanceSpecification{
		InstanceID: "app-666666",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3,
			ExitCode: 1,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy=1.24 scipy=1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "sleep 200",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	}
	info, _ := json.Marshal(appInfo)
	event := &etcd3.Event{
		Key:   "x/x/x/x/x/x/x/x/x/x/x/x/x/app-666666",
		Value: info,
	}
	ProcessAppInfoUpdate(event)

	type args struct {
		submissionID string
		event        *etcd3.Event
	}

	tests := []struct {
		name string
		args args
		want string
	}{
		{"case1", args{submissionID: "app-666666"}, "RUNNING"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAppStatusByID(tt.args.submissionID); got != tt.want {
				t.Errorf("GetNodeIPFromInstanceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
