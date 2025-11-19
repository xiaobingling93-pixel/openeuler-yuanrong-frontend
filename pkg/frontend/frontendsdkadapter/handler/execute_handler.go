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

// Package handler
package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/uuid"
	"frontend/pkg/frontend/frontendsdkadapter/assembler"
	"frontend/pkg/frontend/frontendsdkadapter/models"
	"frontend/pkg/frontend/frontendsdkadapter/parser"
	"frontend/pkg/frontend/frontendsdkadapter/service"
)

// ExecuteHandler -
func ExecuteHandler(ctx *gin.Context) {
	traceID := ctx.Request.Header.Get(constant.HeaderTraceID)
	if traceID == "" {
		traceID = uuid.New().String()
	}
	traceLogger := models.NewTraceLogger("execute", traceID)

	execRequest, err := parser.NewExecuteRequestContext(ctx, traceID)
	if err != nil {
		msg := fmt.Sprintf("deserialize request failed, err: %s", err.Error())
		errResponse := assembler.NewCommonErrorResponse(statuscode.FrontendStatusBadRequest, msg,
			traceLogger.TraceID)
		traceLogger.Logger.Errorf(errResponse.ErrorRsp.Message)
		assembler.WriteResponse(ctx, nil, errResponse, traceID)
		return
	}

	traceLogger.With("functionName", execRequest.FunctionName)
	traceLogger.TenantID = execRequest.TenantID
	response, errResponse := service.ExecuteService(execRequest, traceLogger)
	if errResponse != nil {
		traceLogger.Logger.Error(errResponse.ErrorRsp.Message)
		assembler.WriteResponse(ctx, nil, errResponse, traceID)
		return
	}
	traceLogger.Logger.Info("execute success")
	assembler.WriteResponse(ctx, response, errResponse, traceID)
}
