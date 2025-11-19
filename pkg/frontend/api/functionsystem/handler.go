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

// Package frontend the api of frontend
package frontend

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/common/util"
	"frontend/pkg/frontend/serverstatus"
)

// CreateHandler the handler of create
func CreateHandler(ctx *gin.Context) {
	remoteClientID, traceID := getHeaderPrams(ctx)
	log.GetLogger().Infof("%s|receive instance create request, remoteClientID: %s", traceID, remoteClientID)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	resp, err := util.NewClient().CreateInstanceRaw(body)
	log.GetLogger().Debugf("receive instance create response, msg: %s", resp)
	if err != nil {
		SetCtxResponse(ctx, []byte(err.Error()), http.StatusBadRequest)
	}
	SetCtxResponse(ctx, resp, http.StatusOK)
}

// InvokeHandler the handler of invoke
func InvokeHandler(ctx *gin.Context) {
	remoteClientID, traceID := getHeaderPrams(ctx)
	log.GetLogger().Infof("%s|receive instance invoke request, remoteClientID: %s", traceID, remoteClientID)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	notify, err := util.NewClient().InvokeInstanceRaw(body)
	log.GetLogger().Debugf("receive instance invoke response, msg: %s", notify)
	if err != nil {
		SetCtxResponse(ctx, []byte(err.Error()), http.StatusBadRequest)
	}
	SetCtxResponse(ctx, notify, http.StatusOK)
}

// KillHandler the handler of kill
func KillHandler(ctx *gin.Context) {
	remoteClientID, traceID := getHeaderPrams(ctx)
	log.GetLogger().Infof("%s|receives instance kill request, remoteClientID: %s", traceID, remoteClientID)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	resp, err := util.NewClient().KillRaw(body)
	log.GetLogger().Debugf("receive instance kill response, msg: %s", resp)
	if err != nil {
		SetCtxResponse(ctx, []byte(err.Error()), http.StatusBadRequest)
	}
	SetCtxResponse(ctx, resp, http.StatusOK)
}

func getHeaderPrams(ctx *gin.Context) (string, string) {
	remoteClientID := httputil.GetCompatibleGinHeader(ctx.Request, constant.HeaderRemoteClientId, "remoteClientId")
	traceID := httputil.GetCompatibleGinHeader(ctx.Request, constant.HeaderTraceID, "traceId")
	return remoteClientID, traceID
}

// SetCtxResponse set ctx response
func SetCtxResponse(ctx *gin.Context, body []byte, statusCode int) {
	if len(body) == 0 {
		log.GetLogger().Warnf("the body of ctx response is empty")
	}
	ctx.Writer.WriteHeader(statusCode)
	if serverstatus.IsShutdown() {
		ctx.Writer.Header().Set("Connection", "close")
	}
	if _, err := ctx.Writer.Write(body); err != nil {
		log.GetLogger().Errorf("failed to set response body in context error %s", err.Error())
	}
}
