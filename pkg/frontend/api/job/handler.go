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

// Package job for handle request
package job

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/job"
	"frontend/pkg/frontend/api/app"
)

// SubmitJobHandler -
func SubmitJobHandler(ctx *gin.Context) {
	traceID := ctx.Request.Header.Get(constant.HeaderTraceID)
	logger := log.GetLogger().With(zap.Any("traceID", traceID))
	req := job.SubmitJobHandleReq(ctx)
	if req == nil {
		return
	}
	if req.SubmissionId == "" {
		req.NewSubmissionID()
	} else {
		_, statusCode, err := app.GetAppInfo(req.SubmissionId)
		if err != nil {
			if statusCode != http.StatusNotFound {
				logger.Errorf("failed GetAppInfo, submissionId: %s, err: %v", req.SubmissionId, err)
				ctx.JSON(http.StatusInternalServerError,
					fmt.Sprintf("failed GetAppInfo, submissionId: %s, err: %v", req.SubmissionId, err))
				return
			}
		}
		if statusCode == http.StatusOK {
			logger.Errorf("submit job has already exist, submissionId: %s", req.SubmissionId)
			ctx.JSON(http.StatusBadRequest, fmt.Sprintf("submit job has already exist, submissionId: %s",
				req.SubmissionId))
			return
		}
	}
	logger.Debugf("start to SubmitApp, req:%#v", req)
	job.SubmitJobHandleRes(ctx, job.BuildJobResponse(app.SubmitApp(req)))
}

// ListJobsHandler -
func ListJobsHandler(ctx *gin.Context) {
	job.ListJobsHandleRes(ctx, job.BuildJobResponse(app.ListApps(ctx)))
}

// GetJobInfoHandler -
func GetJobInfoHandler(ctx *gin.Context) {
	submissionId := ctx.Param(job.PathParamSubmissionId)
	job.GetJobInfoHandleRes(ctx, job.BuildJobResponse(app.GetAppInfo(submissionId)))
}

// DeleteJobHandler -
func DeleteJobHandler(ctx *gin.Context) {
	job.DeleteJobHandleRes(ctx, job.BuildJobResponse(app.DeleteApp(ctx)))
}

// StopJobHandler -
func StopJobHandler(ctx *gin.Context) {
	job.StopJobHandleRes(ctx, job.BuildJobResponse(app.StopApp(ctx)))
}
