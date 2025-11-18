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
	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/frontendsdkadapter/assembler"
	"frontend/pkg/frontend/frontendsdkadapter/models"
)

const (
	executeTTLSecond = 1800
	uploadTTLSecond  = 86400
)

// UploadToDataSystemRequest -
type UploadToDataSystemRequest struct {
	PayloadData *models.PayloadData
	TenantID    string
	TraceID     string
	ExecMode    bool
}

// MultiSetService -
func MultiSetService(request *models.MultiSetRequest,
	traceLogger *models.TraceLogger) (*models.MultiSetSuccessResponse,
	*models.CommonErrorResponse) {
	traceLogger.With("tenantId", request.TenantID)
	req := &UploadToDataSystemRequest{
		PayloadData: request.PayloadData,
		TenantID:    request.TenantID,
		TraceID:     traceLogger.TraceID,
		ExecMode:    false,
	}
	dataKeyList, err := UploadToDataSystemWithTrace(req, traceLogger)
	if err != nil {
		traceLogger.Logger.Errorf("failed to upload to data system %s", err.Error())
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}
	traceLogger.Logger.Info("upload success")
	return assembler.NewMultiSetSuccessResponse(dataKeyList, request.TraceID), nil
}

// UploadToDataSystemWithTrace - 返回存入数据系统的拼接后的key
func UploadToDataSystemWithTrace(req *UploadToDataSystemRequest,
	trace *models.TraceLogger) ([]string, error) {
	dataKeyList, err := uploadToDataSystem(req.PayloadData, req.ExecMode, req.TenantID, trace)
	return dataKeyList, err
}

// uploadToDataSystem -
func uploadToDataSystem(payloadData *models.PayloadData, execMode bool, tenantID string,
	trace *models.TraceLogger) ([]string, error) {
	if payloadData.Size == 0 {
		return nil, nil
	}
	trace.Logger.Info("start upload")
	var dataKeyList []string
	var err error
	param := api.SetParam{WriteMode: datasystemclient.UploadWriteMode, TTLSecond: uploadTTLSecond}

	if execMode {
		param.WriteMode = datasystemclient.ExecuteWriteMode
		param.TTLSecond = executeTTLSecond
	}

	for _, data := range payloadData.DataList {
		prefix := ""
		// Do not pass prefix for Execute
		if !execMode {
			prefix = data.DataPrefix
		}

		config := &datasystemclient.Config{TenantID: tenantID, KeyPrefix: prefix, NeedEncrypt: false, DataKey: nil}
		// 返回的是数据系统生成的key，不带前缀拼接
		dataKey, err := datasystemclient.UploadWithKeyRetry(data.Data, config, param, trace.TraceID)
		utils.ClearByteMemory(data.Data)
		if err != nil {
			return nil, snerror.New(statuscode.DsUploadFailed, "internal upload failed")
		}
		trace.Logger.Infof("upload to data system success key %s, len %d", data.DataPrefix+dataKey, "len",
			len(data.Data))
		trace.AppendDataKey(data.DataPrefix, dataKey)
		dataKeyList = append(dataKeyList, dataKey)
	}
	trace.With("uploadkeys", trace.DataKey)
	return dataKeyList, err
}
