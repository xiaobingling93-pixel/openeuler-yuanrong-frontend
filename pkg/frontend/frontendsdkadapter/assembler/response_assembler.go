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

// Package assembler
package assembler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/common/httpconstant"
	"frontend/pkg/frontend/frontendsdkadapter/models"
)

// NewMultiSetSuccessResponse -
func NewMultiSetSuccessResponse(dataKeyList []string, traceID string) *models.MultiSetSuccessResponse {
	return &models.MultiSetSuccessResponse{
		CommonRspHeader: models.CommonRspHeader{
			InnerCode: 0,
			TraceID:   traceID,
		},
		DataKeyList: models.DataKeyList{DataKeys: dataKeyList},
	}
}

// NewMultiGetSuccessResponse -
func NewMultiGetSuccessResponse(payloadInfo *models.DataSystemPayloadInfo, data [][]byte,
	traceID string) *models.MultiGetSuccessResponse {
	return &models.MultiGetSuccessResponse{
		CommonRspHeader: models.CommonRspHeader{
			InnerCode: 0,
			TraceID:   traceID,
		},
		DataSystemPayloadInfo: payloadInfo,
		RawData:               data,
	}
}

// NewMultiDelSuccessResponse -
func NewMultiDelSuccessResponse(
	traceID string) *models.MultiDelSuccessResponse {
	return &models.MultiDelSuccessResponse{
		CommonRspHeader: models.CommonRspHeader{
			InnerCode: 0,
			TraceID:   traceID,
		},
		SuccessResponse: models.SuccessResponse{
			Code:    0,
			Message: "success",
		},
	}
}

// NewCommonErrorResponseByBody -
func NewCommonErrorResponseByBody(body []byte, errMsg string, traceID string) *models.CommonErrorResponse {
	response := &models.CommonErrorResponse{}
	response.TraceID = traceID
	response.ContentType = httpconstant.ApplicationJSON
	if err := json.Unmarshal(body, &response.ErrorRsp); err != nil {
		response.ErrorRsp.Code = statuscode.FrontendStatusInternalError
		response.ErrorRsp.Message = errMsg
	}
	response.InnerCode = int32(response.ErrorRsp.Code)
	return response
}

// NewCommonErrorResponse -
func NewCommonErrorResponse(code int, err string, traceID string) *models.CommonErrorResponse {
	if code == 0 {
		code = statuscode.FrontendStatusInternalError
	}
	return NewCommonErrorResponseByError(snerror.New(code, err), traceID)
}

// NewCommonErrorResponseByError -
func NewCommonErrorResponseByError(err error, traceID string) *models.CommonErrorResponse {
	snError := convertError(err)
	response := &models.CommonErrorResponse{}
	response.ContentType = httpconstant.ApplicationJSON

	response.TraceID = traceID

	response.InnerCode = int32(snError.Code())
	response.ErrorRsp.Code = snError.Code()
	response.ErrorRsp.Message = snError.Error()
	return response
}

// CommonErrorWriter -
type CommonErrorWriter struct{}

// WriteErrorToResponse - 用于不带trace在filter层的错误返回
func (d *CommonErrorWriter) WriteErrorToResponse(ctx *gin.Context, err error) {
	traceID := ctx.Request.Header.Get(constant.HeaderTraceID)
	errResponse := NewCommonErrorResponseByError(err, traceID)
	_ = errResponse.WriteResponse(ctx)
}

// WriteResponse -
func WriteResponse(ctx *gin.Context, response models.ResponseWriter, errResponse *models.CommonErrorResponse,
	traceID string) {
	if errResponse != nil {
		_ = errResponse.WriteResponse(ctx)
		return
	}

	err := response.WriteResponse(ctx)
	if err != nil {
		errResponse := NewCommonErrorResponseByError(err, traceID)
		_ = errResponse.WriteResponse(ctx)
		return
	}
}

// NewExecuteSuccessResponse -
func NewExecuteSuccessResponse(payloadInfo *models.DataSystemPayloadInfo, data [][]byte,
	traceID string) *models.ExecuteSuccessResponse {
	return &models.ExecuteSuccessResponse{
		CommonRspHeader: models.CommonRspHeader{
			InnerCode: 0,
			TraceID:   traceID,
		},
		DataSystemPayloadInfo: payloadInfo,
		RawData:               data,
	}
}

func convertError(err error) snerror.SNError {
	snErr, ok := err.(snerror.SNError)
	if ok {
		return snErr
	}
	return snerror.New(statuscode.FrontendStatusInternalError, err.Error())
}
