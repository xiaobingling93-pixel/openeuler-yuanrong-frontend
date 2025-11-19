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

// MultiGetHandler -
func MultiGetHandler(ctx *gin.Context) {
	traceID := ctx.Request.Header.Get(constant.HeaderTraceID)
	if traceID == "" {
		traceID = uuid.New().String()
	}
	traceLogger := models.NewTraceLogger("multiget", traceID)
	getRequest, err := parser.NewMultiGetRequestContext(ctx, traceID)
	if err != nil {
		msg := fmt.Sprintf("deserialize request failed, err: %s", err.Error())
		errResponse := assembler.NewCommonErrorResponse(statuscode.FrontendStatusBadRequest, msg,
			traceID)
		traceLogger.Logger.Error(msg)
		assembler.WriteResponse(ctx, nil, errResponse, traceID)
		return
	}
	traceLogger.TenantID = getRequest.TenantID
	response, errResponse := service.MultiGetService(getRequest, traceLogger)
	assembler.WriteResponse(ctx, response, errResponse, traceID)
}

// MultiDelHandler -
func MultiDelHandler(ctx *gin.Context) {
	traceID := ctx.Request.Header.Get(constant.HeaderTraceID)
	if traceID == "" {
		traceID = uuid.New().String()
	}
	traceLogger := models.NewTraceLogger("multidel", traceID)
	delRequest, err := parser.NewMultiDelRequestContext(ctx, traceID)
	if err != nil {
		msg := fmt.Sprintf("deserialize request failed, err: %s", err.Error())
		errResponse := assembler.NewCommonErrorResponse(statuscode.FrontendStatusBadRequest, msg, traceID)
		traceLogger.Logger.Errorf(errResponse.ErrorRsp.Message)
		assembler.WriteResponse(ctx, nil, errResponse, traceID)
		return
	}
	response, errResponse := service.MultiDelService(delRequest, traceLogger)
	assembler.WriteResponse(ctx, response, errResponse, traceID)
}

// MultiSetHandler -
func MultiSetHandler(ctx *gin.Context) {
	traceID := ctx.Request.Header.Get(constant.HeaderTraceID)
	if traceID == "" {
		traceID = uuid.New().String()
	}
	traceLogger := models.NewTraceLogger("multiset", traceID)
	setRequest, err := parser.NewMultiSetRequestContext(ctx, traceLogger.TraceID)
	if err != nil {
		msg := fmt.Sprintf("deserialize request failed, err: %s", err.Error())
		errResponse := assembler.NewCommonErrorResponse(statuscode.FrontendStatusBadRequest, msg,
			traceID)
		traceLogger.Logger.Error(errResponse.ErrorRsp.Message)
		assembler.WriteResponse(ctx, nil, errResponse, traceID)
		return
	}
	traceLogger.TenantID = setRequest.TenantID
	response, errResponse := service.MultiSetService(setRequest, traceLogger)
	assembler.WriteResponse(ctx, response, errResponse, traceID)
}
