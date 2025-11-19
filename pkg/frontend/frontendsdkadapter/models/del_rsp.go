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

// SuccessResponse HTTP request message
type SuccessResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MultiDelSuccessResponse -
type MultiDelSuccessResponse struct {
	CommonRspHeader
	SuccessResponse
}

// WriteResponse -
func (rsp *MultiDelSuccessResponse) WriteResponse(ctx *gin.Context) error {
	ctx.Header(constant.HeaderContentType, httpconstant.ApplicationJSON)
	ctx.Header(constant.HeaderInnerCode, fmt.Sprint(rsp.InnerCode))
	ctx.Header(constant.HeaderTraceID, fmt.Sprint(rsp.TraceID))

	ctx.Writer.WriteHeader(http.StatusOK)

	marshal, err := json.Marshal(rsp.SuccessResponse)
	if err != nil {
		log.GetLogger().Errorf("failed to write rsp err %s", err.Error())
		return err
	}

	_, err = ctx.Writer.Write(marshal)
	if err != nil {
		log.GetLogger().Errorf("failed to write rsp err %s", err.Error())
		return err
	}

	return nil
}
