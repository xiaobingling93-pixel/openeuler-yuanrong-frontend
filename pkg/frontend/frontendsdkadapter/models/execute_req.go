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
	"io"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/frontend/common/httputil"
)

// ExecuteRequest -
type ExecuteRequest struct {
	CommonReqHeader
	DataSystemPayloadInfo *DataSystemPayloadInfo
	FunctionName          string
	PayloadData           *PayloadData
	RawData               []byte
}

// ParseRequest -
func (req *ExecuteRequest) ParseRequest(ctx *gin.Context, maxBodySize int64) error {
	err := req.parseHeader(ctx)
	if err != nil {
		return err
	}
	err = req.readBody(ctx.Request.Body, maxBodySize)
	if err != nil {
		return err
	}
	return nil
}

func (req *ExecuteRequest) parseHeader(ctx *gin.Context) error {
	req.CommonReqHeader.ParserHeader(ctx)
	req.FunctionName = req.Headers[constant.HeaderFunctionName]
	return nil
}

func (req *ExecuteRequest) readBody(inputStream io.ReadCloser, maxBodySize int64) error {
	rawData, err := httputil.ReadLimitedBody(inputStream, maxBodySize)
	if err != nil {
		return err
	}
	req.RawData = rawData
	return nil
}
