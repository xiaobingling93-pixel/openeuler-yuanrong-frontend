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
	"strings"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/urnutils"
	"frontend/pkg/frontend/frontendsdkadapter/assembler"
	"frontend/pkg/frontend/frontendsdkadapter/models"
)

// DeleteFromDataSystemRequest -
type DeleteFromDataSystemRequest struct {
	DataKeyList []string
	TenantID    string
	TraceID     string
	NeedEncrypt bool
}

// MultiDelService -
func MultiDelService(request *models.MultiDelRequest,
	traceLogger *models.TraceLogger) (*models.MultiDelSuccessResponse,
	*models.CommonErrorResponse) {
	if len(request.DataSystemPayloadInfo.Data) == 0 {
		return assembler.NewMultiDelSuccessResponse(request.TraceID), nil
	}

	traceLogger.With("tenantId", request.TenantID)
	dataKeyList := models.DataKeyPrefixJoin(request.DataSystemPayloadInfo)
	traceLogger.DataKey = strings.Join(dataKeyList, "&")
	req := &DeleteFromDataSystemRequest{
		DataKeyList: dataKeyList,
		TenantID:    request.TenantID,
		TraceID:     traceLogger.TraceID,
		NeedEncrypt: false,
	}
	err := DeleteFromDataSystemWithTrace(req, traceLogger)
	if err != nil {
		traceLogger.Logger.Errorf("failed to delete from data system %s", err.Error())
		return nil, assembler.NewCommonErrorResponseByError(err, request.TraceID)
	}
	traceLogger.Logger.Info("delete success")
	return assembler.NewMultiDelSuccessResponse(request.TraceID), nil
}

// DeleteFromDataSystemWithTrace -
func DeleteFromDataSystemWithTrace(req *DeleteFromDataSystemRequest,
	trace *models.TraceLogger) error {
	return deleteFromDataSystem(req, trace)
}

func deleteFromDataSystem(req *DeleteFromDataSystemRequest, trace *models.TraceLogger) error {
	trace.With("deletekeys", strings.Join(req.DataKeyList, "|"))
	if len(req.DataKeyList) == 0 {
		return nil
	}
	dataKeyList := req.DataKeyList
	trace.Logger.Info("start delete")
	config := &datasystemclient.Config{
		TenantID:    req.TenantID,
		NeedEncrypt: false,
	}
	failedKeys, err := datasystemclient.DeleteArrayRetry(dataKeyList, config, trace.TraceID)
	if err != nil {
		trace.Logger.Errorf("delete data from data system failed failed keys %s ,err: %s",
			urnutils.AnonymizeKeys(failedKeys), err)
		return snerror.New(statuscode.DsDeleteFailed, "internal delete failed")
	}
	return nil
}
