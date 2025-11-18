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

// Package models
package models

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/common/httpconstant"
)

// ResponseWriter -
type ResponseWriter interface {
	WriteResponse(ctx *gin.Context) error
}

// CommonRspHeader -
type CommonRspHeader struct {
	InnerCode   int32
	TraceID     string
	Headers     map[string]string
	ContentType string
}

// BadResponse HTTP request message that does not return 200
type BadResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// CommonErrorResponse -
type CommonErrorResponse struct {
	CommonRspHeader
	ErrorRsp BadResponse
}

// WriteResponse -
func (rsp *CommonErrorResponse) WriteResponse(ctx *gin.Context) error {
	return writeJSONResponse(ctx, http.StatusOK, rsp.InnerCode, rsp.TraceID, rsp.ErrorRsp)
}

func writeJSONResponse(ctx *gin.Context, statusCode int, innerCode int32, traceID string, payload interface{}) error {
	ctx.Writer.WriteHeader(statusCode)
	ctx.Writer.Header().Set(constant.HeaderContentType, httpconstant.ApplicationJSON)
	ctx.Writer.Header().Set(constant.HeaderInnerCode, fmt.Sprint(innerCode))
	ctx.Writer.Header().Set(constant.HeaderTraceID, fmt.Sprint(traceID))

	body, err := json.Marshal(payload)
	if err != nil {
		log.GetLogger().Errorf("failed to marshal response body: %s", err.Error())
		return err
	}
	_, err = ctx.Writer.Write(body)
	if err != nil {
		log.GetLogger().Errorf("failed to write response: %s", err.Error())
		return err
	}
	return nil
}
