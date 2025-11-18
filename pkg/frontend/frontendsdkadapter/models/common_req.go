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

// Package models multidata request response transfer object
package models

import (
	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
)

// RequestParser -
type RequestParser interface {
	ParseRequest(ctx *gin.Context, maxBodySize int64) error
}

// CommonReqHeader -
type CommonReqHeader struct {
	TraceID  string
	TenantID string
	Headers  map[string]string
}

// ParserHeader -
func (reqHeader *CommonReqHeader) ParserHeader(ctx *gin.Context) {
	reqHeader.Headers = make(map[string]string)
	for key, values := range ctx.Request.Header {
		if len(values) > 0 {
			reqHeader.Headers[key] = values[0]
		}
	}
	reqHeader.TenantID = reqHeader.Headers[constant.HeaderTenantID]
	reqHeader.TraceID = reqHeader.Headers[constant.HeaderTraceID]
}
