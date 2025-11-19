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
	"strings"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/frontendsdkadapter/assembler"
	"frontend/pkg/frontend/frontendsdkadapter/models"
)

// DownloadFromDataSystemRequest -
type DownloadFromDataSystemRequest struct {
	DataKeyList []string
	TenantID    string
	TraceID     string
	NeedEncrypt bool
}

// DownloadFromDataSystemResponse -
type DownloadFromDataSystemResponse struct {
	DataSystemPayloadInfo *models.DataSystemPayloadInfo
	RawData               [][]byte
}

// MultiGetService -
func MultiGetService(request *models.MultiGetRequest,
	traceLogger *models.TraceLogger) (*models.MultiGetSuccessResponse,
	*models.CommonErrorResponse) {
	if len(request.DataSystemPayloadInfo.Data) == 0 {
		return assembler.NewMultiGetSuccessResponse(nil, nil, request.TraceID), nil
	}
	traceLogger.With("tenantId", request.TenantID)
	dataKeyList := models.DataKeyPrefixJoin(request.DataSystemPayloadInfo)
	traceLogger.DataKey = strings.Join(dataKeyList, "&")
	req := &DownloadFromDataSystemRequest{
		DataKeyList: dataKeyList,
		TenantID:    request.TenantID,
		TraceID:     traceLogger.TraceID,
		NeedEncrypt: request.DataSystemPayloadInfo.Data[0].NeedEncrypt,
	}
	data, err := DownloadFromDataSystemWithTrace(req, traceLogger)
	if err != nil {
		traceLogger.Logger.Errorf("failed to download from data system %s", err.Error())
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}
	payload, err := genGetPayloadInfo(request.DataSystemPayloadInfo, data)
	if err != nil {
		traceLogger.Logger.Errorf("failed to generate get payload info %s", err)
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}
	traceLogger.Logger.Info("download success")
	return assembler.NewMultiGetSuccessResponse(payload, data, request.TraceID), nil
}

// DownloadFromDataSystemWithTrace - key中如果有某个key不存在会整个报错
func DownloadFromDataSystemWithTrace(req *DownloadFromDataSystemRequest,
	trace *models.TraceLogger) ([][]byte, error) {
	return downloadFromDataSystem(req, trace)
}

func downloadFromDataSystem(req *DownloadFromDataSystemRequest, trace *models.TraceLogger) ([][]byte, error) {
	trace.With("downloadkeys", strings.Join(req.DataKeyList, "|"))
	if len(req.DataKeyList) == 0 {
		return nil, nil
	}
	var err error
	trace.Logger.Info("start download")

	dataConfig := &datasystemclient.Config{
		TenantID:    req.TenantID,
		NeedEncrypt: false,
		DataKey:     nil,
	}
	dataArray, err := datasystemclient.DownloadArrayRetry(req.DataKeyList, dataConfig, trace.TraceID)
	if err != nil {
		if err.Error() == datasystemclient.ErrKeyNotFound.Error() {
			return nil, snerror.New(statuscode.DsKeyNotFound, err.Error())
		}
		if err.Error() == datasystemclient.ErrValueSizeExceeded.Error() {
			return nil, snerror.New(statuscode.DsDownloadFailed, "download body too large")
		}
		return nil, snerror.New(statuscode.DsDownloadFailed, "internal download failed")
	}
	return dataArray, nil
}

func genGetPayloadInfo(payloadInfo *models.DataSystemPayloadInfo, data [][]byte) (*models.DataSystemPayloadInfo,
	error) {
	if len(payloadInfo.Data) != len(data) {
		return nil, fmt.Errorf("keylen = %d is not equals to data len = %d", len(payloadInfo.Data), len(data))
	}

	if len(payloadInfo.Data) == 0 {
		return nil, nil
	}

	payloadInfos := make([]*models.PayloadInfo, len(data))
	currentOffset := 0
	for i, item := range data {
		payloadInfos[i] = &models.PayloadInfo{
			DataKey: payloadInfo.Data[i].DataKey,
			Offset:  currentOffset,
			Length:  len(item),
		}

		currentOffset += len(item)
	}

	return &models.DataSystemPayloadInfo{
		Data: payloadInfos,
	}, nil
}
