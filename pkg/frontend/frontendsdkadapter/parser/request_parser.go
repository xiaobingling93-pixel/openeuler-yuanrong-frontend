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

// Package parser
package parser

import (
	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/frontendsdkadapter/models"
)

const (
	megabytes = 1024 * 1024
)

// NewMultiSetRequestContext -
func NewMultiSetRequestContext(ctx *gin.Context, traceID string) (*models.MultiSetRequest, error) {
	setRequest := &models.MultiSetRequest{}

	err := setRequest.ParseRequest(ctx, config.GetConfig().HTTPConfig.MaxDataSystemMultiDataBodySize*megabytes)
	if err != nil {
		return nil, err
	}
	setRequest.TraceID = traceID
	// header
	payloadInfoStr := setRequest.Headers[constant.HeaderDataSystemPayloadInfo]
	payloadInfo, err := ParsePayloadHeaderJSON(payloadInfoStr)
	if err != nil {
		return nil, err
	}

	setRequest.DataSystemPayloadInfo = payloadInfo
	// body
	data, err := ReadPayloadData(setRequest.RawData, setRequest.DataSystemPayloadInfo)
	if err != nil {
		return nil, err
	}
	setRequest.PayloadData = data

	return setRequest, nil
}

// NewMultiGetRequestContext -
func NewMultiGetRequestContext(ctx *gin.Context, traceID string) (*models.MultiGetRequest, error) {
	getRequest := &models.MultiGetRequest{}
	err := getRequest.ParseRequest(ctx, int64(config.GetConfig().HTTPConfig.MaxDataSystemMultiDataBodySize*megabytes))
	if err != nil {
		return nil, err
	}
	getRequest.TraceID = traceID
	// header
	payloadInfoStr := getRequest.Headers[constant.HeaderDataSystemPayloadInfo]
	payloadInfo, err := ParsePayloadHeaderJSON(payloadInfoStr)
	if err != nil {
		return nil, err
	}

	getRequest.DataSystemPayloadInfo = payloadInfo
	return getRequest, nil
}

// NewMultiDelRequestContext -
func NewMultiDelRequestContext(ctx *gin.Context, traceID string) (*models.MultiDelRequest, error) {
	delRequest := &models.MultiDelRequest{}
	err := delRequest.ParseRequest(ctx, int64(config.GetConfig().HTTPConfig.MaxDataSystemMultiDataBodySize*megabytes))
	if err != nil {
		return nil, err
	}
	delRequest.TraceID = traceID
	// header
	payloadInfoStr := delRequest.Headers[constant.HeaderDataSystemPayloadInfo]
	payloadInfo, err := ParsePayloadHeaderJSON(payloadInfoStr)
	if err != nil {
		return nil, err
	}

	delRequest.DataSystemPayloadInfo = payloadInfo
	return delRequest, nil
}

// NewExecuteRequestContext -
func NewExecuteRequestContext(ctx *gin.Context, traceID string) (*models.ExecuteRequest, error) {
	execRequest := &models.ExecuteRequest{}
	err := execRequest.ParseRequest(ctx, int64(config.GetConfig().HTTPConfig.MaxDataSystemMultiDataBodySize*megabytes))
	if err != nil {
		return nil, err
	}
	execRequest.TraceID = traceID

	// header
	payloadInfoStr := execRequest.Headers[constant.HeaderDataSystemPayloadInfo]
	payloadInfo, err := ParsePayloadHeaderJSON(payloadInfoStr)
	if err != nil {
		return nil, err
	}

	execRequest.DataSystemPayloadInfo = payloadInfo
	// body
	data, err := ReadPayloadData(execRequest.RawData, execRequest.DataSystemPayloadInfo)
	if err != nil {
		return nil, err
	}
	execRequest.PayloadData = data
	return execRequest, nil
}
