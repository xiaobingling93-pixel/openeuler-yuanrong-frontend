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
	"github.com/gin-gonic/gin"

	"frontend/pkg/frontend/common/httputil"
)

// MultiDelRequest -
type MultiDelRequest struct {
	CommonReqHeader
	DataSystemPayloadInfo *DataSystemPayloadInfo
	DataKeys              string
}

// ParseRequest -
func (req *MultiDelRequest) ParseRequest(context *gin.Context, maxBodySize int64) error {
	req.CommonReqHeader.ParserHeader(context)
	body, err := httputil.ReadLimitedBody(context.Request.Body, maxBodySize)
	if err != nil {
		return err
	}
	req.DataKeys = string(body)
	return nil
}
