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

// Package v1 -
package v1

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/common/httpconstant"
	"frontend/pkg/frontend/stream"
	"frontend/pkg/frontend/types"
)

// SubscribeHandler -
func SubscribeHandler(ctx *gin.Context) {
	processCtx := types.CreateInvokeProcessContext()
	setStreamProcessCtxReq(ctx, processCtx)
	streamName := processCtx.ReqHeader[httpconstant.HeaderStreamName]
	log.GetLogger().Infof("subscribe handler receives one request with stream name: %s", streamName)
	err := stream.SubscribeHandler(processCtx)
	if err != nil {
		setStreamProcessCtxResp(ctx, processCtx)
		log.GetLogger().Errorf("failed to handle request stream %s, error: %s", streamName, err.Error())
	} else {
		errorSubscribe := stream.StartSubscribeStream(processCtx, ctx)
		if errorSubscribe != nil {
			log.GetLogger().Errorf("failed to subscribe stream %s, error: %s",
				streamName, errorSubscribe.Error())
			setStreamProcessCtxResp(ctx, processCtx)
		}
	}
}

func setStreamProcessCtxReq(ctx *gin.Context, processCtx *types.InvokeProcessContext) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("get request body failed, err: %v", err)
		ctx.Writer.WriteHeader(http.StatusUnauthorized)
		ctx.Writer.WriteString("get request body")
		return
	}
	processCtx.ReqBody = body
	processCtx.ReqHeader = readStreamReqHeaders(ctx)
}

func setStreamProcessCtxResp(ctx *gin.Context, processCtx *types.InvokeProcessContext) {
	ctx.Writer.WriteHeader(processCtx.StatusCode)
	ctx.Writer.Write(processCtx.RespBody)
	writeHeadersToStreamResponse(processCtx.RespHeader, ctx)
}

func readStreamReqHeaders(ctx *gin.Context) map[string]string {
	headerMap := make(map[string]string)
	for key := range ctx.Request.Header {
		headerMap[key] = ctx.Request.Header.Get(key)
	}
	return headerMap
}

func writeHeadersToStreamResponse(headers map[string]string, ctx *gin.Context) {
	for key, value := range headers {
		ctx.Writer.Header().Set(key, value)
	}
}
