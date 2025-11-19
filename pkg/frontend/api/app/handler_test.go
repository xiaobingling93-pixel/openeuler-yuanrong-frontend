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

package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/types"
	mockUtils "frontend/pkg/common/faas_common/utils"
	"frontend/pkg/common/job"
	"frontend/pkg/frontend/common/util"
	"frontend/pkg/frontend/instancemanager"
)

func TestCreateHandler(t *testing.T) {
	mock := &mockUtils.FakeLibruntimeSdkClient{}
	util.SetAPIClientLibruntime(mock)
	req1 := &job.SubmitRequest{
		Entrypoint:   "sleep 200",
		SubmissionId: "app-scrpit-1",
		RuntimeEnv: &job.RuntimeEnv{
			WorkingDir: "file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip",
			Pip:        []string{"numpy==1.24", "scipy==1.25"},
			EnvVars: map[string]string{
				"SOURCE_REGION": "suzhou_std",
				"DEPLOY_REGION": "suzhou_std",
			},
		},
		EntrypointNumCpus: 300,
		EntrypointNumGpus: 0,
		EntrypointMemory:  0,
		EntrypointResources: map[string]float64{
			"NPU": 0,
		},
	}

	convey.Convey("create successfully", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := json.Marshal(req1)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		util.SetAPIClientLibruntime(mock)
		CreateHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	convey.Convey("io read failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := json.Marshal(req1)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		p := gomonkey.ApplyFunc(io.ReadAll, func(r io.Reader) ([]byte, error) {
			return nil, fmt.Errorf("error")
		})
		defer p.Reset()
		CreateHandler(ctx)
		convey.So(rw.Code, convey.ShouldEqual, http.StatusInternalServerError)
	})

	convey.Convey("unmarshal failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer([]byte("aaa")))
		util.SetAPIClientLibruntime(mock)
		CreateHandler(ctx)
		convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
	})

	convey.Convey("create failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := json.Marshal(req1)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		util.SetAPIClientLibruntime(mock)
		p := gomonkey.ApplyFunc((*mockUtils.FakeLibruntimeSdkClient).CreateInstance, func(_ *mockUtils.FakeLibruntimeSdkClient, funcMeta api.FunctionMeta, args []api.Arg, invokeOpt api.InvokeOptions) (instanceID string, err error) {
			return "", fmt.Errorf("error")
		})
		defer p.Reset()
		CreateHandler(ctx)
		convey.So(rw.Code, convey.ShouldEqual, http.StatusInternalServerError)
	})

}

func TestGetInfoHandler(t *testing.T) {

	instancemanager.StoreAppInfo("app-123456", &types.InstanceSpecification{
		InstanceID: "app-123456",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3, // RUNNING
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "sleep 200",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	convey.Convey("get info successfully", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "/app/v1/getappinfo/app-123456", nil)
		ctx.AddParam("submissionId", "app-123456")
		GetInfoHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	convey.Convey("get info failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "/app/v1/getappinfo/app-123", nil)
		GetInfoHandler(ctx)
		assert.Equal(t, http.StatusNotFound, rw.Code)
	})
}

func TestListHandler(t *testing.T) {
	instancemanager.StoreAppInfo("app-script-1", &types.InstanceSpecification{
		InstanceID: "app-script-1",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3, // RUNNING
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "sleep 200",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	instancemanager.StoreAppInfo("app-script-2", &types.InstanceSpecification{
		InstanceID: "app-script-2",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3, // RUNNING
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "sleep 200",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	convey.Convey("get info successfully", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "/app/v1/list", nil)
		ListHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
	})
}

func TestKillHandler(t *testing.T) {
	mock := &mockUtils.FakeLibruntimeSdkClient{}
	util.SetAPIClientLibruntime(mock)
	instancemanager.StoreAppInfo("app-script-1", &types.InstanceSpecification{
		InstanceID: "app-script-1",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3, // RUNNING
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "python script.py",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	instancemanager.StoreAppInfo("app-script-2", &types.InstanceSpecification{
		InstanceID: "app-script-2",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     1,
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "python script.py",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	convey.Convey("kill successfully", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", nil)
		ctx.AddParam("submissionId", "app-script-2")
		StopHandler(ctx)
		assert.Equal(t, http.StatusForbidden, rw.Code)
	})

	convey.Convey("kill failed, status not allowed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.AddParam("submissionId", "app-script-1")
		ctx.Request, _ = http.NewRequest("", "", nil)
		StopHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	convey.Convey("kill failed, job not existed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.AddParam("submissionId", "app-script-3")
		ctx.Request, _ = http.NewRequest("", "", nil)
		StopHandler(ctx)
		assert.Equal(t, http.StatusNotFound, rw.Code)
	})
}

func TestDeleteHandler(t *testing.T) {
	mock := &mockUtils.FakeLibruntimeSdkClient{}
	util.SetAPIClientLibruntime(mock)
	instancemanager.StoreAppInfo("app-script-1", &types.InstanceSpecification{
		InstanceID: "app-script-1",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     3, // RUNNING
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "python script.py",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	instancemanager.StoreAppInfo("app-script-2", &types.InstanceSpecification{
		InstanceID: "app-script-2",
		StartTime:  "",
		InstanceStatus: types.InstanceStatus{
			Code:     7,
			ExitCode: 0,
		},
		CreateOptions: map[string]string{
			"POST_START_EXEC":        "pip3.9 install numpy==1.24 scipy==1.25",
			"DELEGATE_DOWNLOAD":      "{\"code_path\":\"file:///home/ray/codeNew/god_gt_factory/file/lidar_seg_042801.zip\",\"storage_type\":\"working_dir\"}",
			"ENTRYPOINT":             "python script.py",
			"DELEGATE_ENV_VAR":       "{\"DEPLOY_REGION\":\"suzhou_std\",\"SOURCE_REGION\":\"suzhou_std\"}",
			"USER_PROVIDED_METADATA": "{\"autoscenes_ids\":\"auto_1-test\",\"task_type\":\"task_1\",\"ttl\":\"1250\"}",
		},
	})
	convey.Convey("delete successfully", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "", nil)
		ctx.AddParam("submissionId", "app-script-2")
		DeleteHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	convey.Convey("delete failed, status not allowed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "", nil)
		ctx.AddParam("submissionId", "app-script-1")
		DeleteHandler(ctx)
		assert.Equal(t, http.StatusForbidden, rw.Code)
	})

	convey.Convey("kill failed, job not existed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest(http.MethodGet, "/app/v1/getappinfo/app-script-3", nil)
		ctx.AddParam("submissionId", "app-script-3")
		DeleteHandler(ctx)
		assert.Equal(t, http.StatusNotFound, rw.Code)
	})
}

func TestSetCtxResponse(t *testing.T) {
	convey.Convey("test SetCtxResponse", t, func() {
		gin.SetMode(gin.TestMode)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		convey.Convey("when process success", func() {
			jsonBytes, err := json.Marshal(job.Response{
				Code: http.StatusOK,
				Data: nil,
			})
			if err != nil {
				t.Errorf("json.Marshal failed: %v", err)
			}
			SetCtxResponse(c, nil, http.StatusOK, nil)
			convey.So(w.Code, convey.ShouldEqual, http.StatusOK)
			convey.So(w.Body.String(), convey.ShouldEqual, string(jsonBytes))
		})
	})
}

func TestCreateInvokeOpts(t *testing.T) {
	convey.Convey("test createInvokeOpts", t, func() {
		convey.Convey("when process success", func() {
			defer gomonkey.ApplyFunc(buildCreateOpt, func(reqBody *job.SubmitRequest) map[string]string {
				return map[string]string{"abc": "123"}
			}).Reset()
			defer gomonkey.ApplyFunc(generateScheduleAffinity, func(scheduleAffinity []api.Affinity, label string) []api.Affinity {
				return []api.Affinity{
					{
						Kind:                     api.AffinityKindResource,
						Affinity:                 api.PreferredAffinity,
						PreferredPriority:        true,
						PreferredAntiOtherLabels: !strings.Contains(label, constant.UnUseAntiOtherLabelsKey),
						LabelOps: []api.LabelOperator{
							{
								Type:        api.LabelOpExists,
								LabelKey:    strings.TrimSpace(label),
								LabelValues: nil,
							},
						},
					},
				}
			}).Reset()
			reqBody := &job.SubmitRequest{
				EntrypointNumCpus:   0,
				EntrypointMemory:    0,
				EntrypointResources: map[string]float64{"abc": 0},
				Labels:              "poolLabel",
			}
			expectedResult := api.InvokeOptions{
				Cpu:             0,
				Memory:          0,
				CustomResources: map[string]float64{"abc": 0},
				CreateOpt:       map[string]string{"abc": "123"},
				Timeout:         constant.AppInvokeTimeout,
				ScheduleAffinities: []api.Affinity{
					{
						Kind:                     api.AffinityKindResource,
						Affinity:                 api.PreferredAffinity,
						PreferredPriority:        true,
						PreferredAntiOtherLabels: !strings.Contains("poolLabel", constant.UnUseAntiOtherLabelsKey),
						LabelOps: []api.LabelOperator{
							{
								Type:        api.LabelOpExists,
								LabelKey:    strings.TrimSpace("poolLabel"),
								LabelValues: nil,
							},
						},
					},
				},
			}
			result := createInvokeOpts(reqBody)
			convey.So(result, convey.ShouldResemble, expectedResult)
		})
	})
}

func TestBuildCreateOpt(t *testing.T) {
	convey.Convey("test buildCreateOpt", t, func() {
		convey.Convey("when reqBody is empty", func() {
			reqBody := &job.SubmitRequest{}
			result := buildCreateOpt(reqBody)
			convey.So(result[constant.EntryPointKey], convey.ShouldEqual, reqBody.Entrypoint)
		})
		convey.Convey("when reqBody.CreateOptions is not nil", func() {
			reqBody := &job.SubmitRequest{
				CreateOptions: map[string]string{"abc": "123"},
			}
			result := buildCreateOpt(reqBody)
			convey.So(result, convey.ShouldResemble, map[string]string{
				"abc":                  "123",
				constant.EntryPointKey: "",
			})
		})
		convey.Convey("when reqBody.RuntimeEnv is not nil", func() {
			reqBody := &job.SubmitRequest{
				RuntimeEnv: &job.RuntimeEnv{
					WorkingDir: "file:///home/disk/tk/file.zip",
					Pip:        []string{"numpy==1.24", "scipy==1.11.0"},
					EnvVars: map[string]string{
						"SOURCE_REGION": "suzhou_std",
					},
				},
				Metadata: map[string]string{
					"autoscenes_ids": "auto_1-test",
				},
			}
			result := buildCreateOpt(reqBody)
			convey.So(result, convey.ShouldResemble, map[string]string{
				constant.DelegateDownloadKey: "{\"storage_type\":\"working_dir\",\"code_path\":\"file:///home/disk/tk/file.zip\"}",
				constant.DelegateEnvVar:      "{\"SOURCE_REGION\":\"suzhou_std\"}",
				constant.EntryPointKey:       "",
				constant.PostStartExec:       "pip3.9 install numpy==1.24 scipy==1.11.0 && pip3.9 check",
				constant.UserMetadataKey:     "{\"autoscenes_ids\":\"auto_1-test\"}",
			})
		})
	})
}

func TestGenerateScheduleAffinity(t *testing.T) {
	convey.Convey("test generateScheduleAffinity", t, func() {
		convey.Convey("when scheduleAffinity is nil, and labels is empty", func() {
			result := generateScheduleAffinity(nil, "")
			convey.So(result, convey.ShouldBeNil)
		})
		convey.Convey("when scheduleAffinity is nil, label equal "+constant.UnUseAntiOtherLabelsKey, func() {
			result := generateScheduleAffinity(nil, constant.UnUseAntiOtherLabelsKey)
			convey.So(result, convey.ShouldBeNil)
		})
		convey.Convey("when scheduleAffinity is nil, label equal 'aaa,bbb'", func() {
			result := generateScheduleAffinity(nil, "aaa,bbb")
			convey.So(len(result), convey.ShouldEqual, 2)
		})
	})
}
