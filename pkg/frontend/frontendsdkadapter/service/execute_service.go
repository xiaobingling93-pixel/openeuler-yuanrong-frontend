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

// Package service
package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"frontend/pkg/common/faas_common/aliasroute"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/urnutils"
	"frontend/pkg/frontend/frontendsdkadapter/assembler"
	"frontend/pkg/frontend/frontendsdkadapter/models"
	"frontend/pkg/frontend/invocation"
	"frontend/pkg/frontend/types"
)

const (
	acquireTimeout          = 55
	caaSHeaderDataSystemKey = "X-Caas-Data-System-Key"
)

// ExecuteService -
func ExecuteService(request *models.ExecuteRequest,
	traceLogger *models.TraceLogger) (*models.ExecuteSuccessResponse,
	*models.CommonErrorResponse) {
	traceLogger.With("tenantId", request.TenantID)
	// build invokeCtx 包括函数名解析及别名路由
	invokeCtx, err := buildProcessContext(request)
	if err != nil {
		return nil, assembler.NewCommonErrorResponse(statuscode.FrontendStatusBadRequest, err.Error(),
			request.TraceID)
	}

	traceLogger.Logger.Info("start execute")
	// 上传
	dataKeyList, err := uploadData(request, traceLogger)
	if err != nil {
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}

	// 删除文件
	defer func() {
		cleanupData(dataKeyList, request.TenantID, traceLogger)
	}()

	resDataKeyList, errorResponse := executeInvoke(invokeCtx, dataKeyList, traceLogger)
	if errorResponse != nil {
		return nil, errorResponse
	}

	// 下载
	return downloadAndGenerateResponse(resDataKeyList, request, traceLogger)
}

// uploadData 上传数据到数据系统
func uploadData(request *models.ExecuteRequest, traceLogger *models.TraceLogger) ([]string, error) {
	uploadReq := &UploadToDataSystemRequest{
		PayloadData: request.PayloadData,
		TenantID:    request.TenantID,
		TraceID:     traceLogger.TraceID,
		ExecMode:    true,
	}

	dataKeyList, err := UploadToDataSystemWithTrace(uploadReq, traceLogger)
	if err != nil {
		traceLogger.Logger.Errorf("failed to upload to data system %s", err)
		return nil, err
	}

	return dataKeyList, nil
}

// cleanupData 清理数据系统中的临时文件
func cleanupData(dataKeyList []string, tenantID string, traceLogger *models.TraceLogger) {
	delReq := &DeleteFromDataSystemRequest{
		DataKeyList: dataKeyList,
		TenantID:    tenantID,
		TraceID:     traceLogger.TraceID,
		NeedEncrypt: false,
	}

	if err := DeleteFromDataSystemWithTrace(delReq, traceLogger); err != nil {
		traceLogger.Logger.Errorf("failed to delete from data system %s", err)
	}
}

// executeInvoke 执行函数调用
func executeInvoke(invokeCtx *types.InvokeProcessContext, dataKeyList []string,
	traceLogger *models.TraceLogger) ([]string, *models.CommonErrorResponse) {
	traceLogger.Logger.Info("start invoke")

	resDataKeyList, errorResponse := invoke(invokeCtx, dataKeyList)
	if errorResponse != nil {
		traceLogger.Logger.Errorf("failed to invoke %s", errorResponse.ErrorRsp.Message)
		return nil, errorResponse
	}

	return resDataKeyList, nil
}

func invoke(ctx *types.InvokeProcessContext, dataKeyList []string) ([]string, *models.CommonErrorResponse) {
	if len(dataKeyList) == 0 {
		return []string{}, nil
	}
	// 数据系统key塞到请求头里
	ctx.ReqHeader[caaSHeaderDataSystemKey] = strings.Join(dataKeyList, "|")
	// invoke
	err := invocation.InvokeHandler(ctx)
	if err != nil {
		return nil, assembler.NewCommonErrorResponse(statuscode.UserFunctionInvokeError, err.Error(), ctx.TraceID)
	}
	if ctx.StatusCode != http.StatusOK || ctx.RequestTraceInfo.InnerCode != 0 {
		return nil, assembler.NewCommonErrorResponseByBody(ctx.RespBody, "invoke err", ctx.TraceID)
	}
	keys := ctx.RespHeader[caaSHeaderDataSystemKey]
	resDataKeyList := strings.FieldsFunc(keys, func(r rune) bool {
		return r == '|'
	})

	return resDataKeyList, nil
}

// downloadAndGenerateResponse 下载结果数据并生成响应
func downloadAndGenerateResponse(resDataKeyList []string, request *models.ExecuteRequest,
	traceLogger *models.TraceLogger) (*models.ExecuteSuccessResponse, *models.CommonErrorResponse) {

	// 下载结果数据
	data, err := downloadResultData(resDataKeyList, request, traceLogger)
	if err != nil {
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}

	// 生成响应载荷
	payload, err := genExecPayloadInfo(resDataKeyList, data)
	if err != nil {
		traceLogger.Logger.Errorf("failed to genExecPayloadInfo %s", err)
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}

	return assembler.NewExecuteSuccessResponse(payload, data, traceLogger.TraceID), nil
}

// downloadResultData 从数据系统下载结果数据
func downloadResultData(resDataKeyList []string, request *models.ExecuteRequest,
	traceLogger *models.TraceLogger) ([][]byte, error) {

	downloadReq := &DownloadFromDataSystemRequest{
		DataKeyList: resDataKeyList,
		TenantID:    request.TenantID,
		TraceID:     request.TraceID,
		NeedEncrypt: false,
	}

	data, err := DownloadFromDataSystemWithTrace(downloadReq, traceLogger)
	if err != nil {
		traceLogger.Logger.Errorf("failed to download from data system %s", err)
		return nil, err
	}

	return data, nil
}

func buildProcessContext(request *models.ExecuteRequest) (*types.InvokeProcessContext,
	error) {
	processCtx := &types.InvokeProcessContext{
		ReqHeader:  make(map[string]string),
		RespHeader: make(map[string]string),
		StartTime:  time.Now(),
	}
	processCtx.TraceID = request.TraceID
	processCtx.RequestID = request.TraceID
	processCtx.ReqHeader = request.Headers
	processCtx.NeedReadRespHeader = true
	functionURN, err := urnutils.GetFuncInfoWithVersion(extractFunctionURN(request))
	if err != nil {
		return nil, err
	}
	processCtx.FuncKey = urnutils.CombineFunctionKey(functionURN.TenantID,
		functionURN.FuncName, functionURN.FuncVersion)
	processCtx.RequestTraceInfo = &types.RequestTraceInfo{}
	processCtx.AcquireTimeout = acquireTimeout
	return processCtx, nil
}

func extractFunctionURN(request *models.ExecuteRequest) string {
	return aliasroute.ResolveAliasedFunctionNameToURN(request.FunctionName, request.TenantID, request.Headers)
}

func genExecPayloadInfo(keys []string, data [][]byte) (*models.DataSystemPayloadInfo, error) {
	if len(keys) != len(data) {
		return nil, fmt.Errorf("keylen = %d is not equals to data len = %d", len(keys), len(data))
	}

	if len(keys) == 0 {
		return nil, nil
	}

	payloadInfos := make([]*models.PayloadInfo, len(data))
	currentOffset := 0
	for i, item := range data {
		payloadInfos[i] = &models.PayloadInfo{
			DataKey: keys[i],
			Offset:  currentOffset,
			Length:  len(item),
		}

		currentOffset += len(item)
	}

	return &models.DataSystemPayloadInfo{
		Data: payloadInfos,
	}, nil
}
