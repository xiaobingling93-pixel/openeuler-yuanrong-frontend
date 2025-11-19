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

// Package stream -
package stream

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/common/httpconstant"
	"frontend/pkg/frontend/responsehandler"
	"frontend/pkg/frontend/types"
)

// SubscribeHandler -
func SubscribeHandler(ctx *types.InvokeProcessContext) error {
	streamName := ctx.ReqHeader[httpconstant.HeaderStreamName]
	if isEmptyString(streamName) {
		responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusBadRequest, "stream name is invalid")
		log.GetLogger().Errorf("stream name with is missing")
		return errors.New("stream name missing")
	}

	if isEmptyString(ctx.ReqHeader[httpconstant.HeaderTimeoutMs]) {
		responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusBadRequest,
			"parameter: "+httpconstant.HeaderTimeoutMs+" is missing in header")
		log.GetLogger().Errorf("parameter: '%s' is missing in header", httpconstant.HeaderTimeoutMs)
		return errors.New("parameter missing in header")
	}

	statusTransTimeout, timeoutMs := transformToNumber(ctx.ReqHeader[httpconstant.HeaderTimeoutMs])
	if !statusTransTimeout {
		responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusBadRequest,
			"parameter: "+httpconstant.HeaderTimeoutMs+" is invalid")
		log.GetLogger().Errorf("parameter: '%s' is invalid", httpconstant.HeaderTimeoutMs)
		return errors.New("parameter invalid")
	}

	statusExpectNum, expectReceiveNum := transformToNumber(ctx.ReqHeader[httpconstant.HeaderExpectNum])
	if !isEmptyString(ctx.ReqHeader[httpconstant.HeaderExpectNum]) && !statusExpectNum {
		responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusBadRequest,
			"parameter: "+httpconstant.HeaderExpectNum+" is invalid")
		log.GetLogger().Errorf("parameter: '%s' is invalid", httpconstant.HeaderExpectNum)
		return errors.New("parameter invalid")
	}

	streamCtx := &types.StreamContext{
		StreamName: streamName,
		ExpectNum:  int32(expectReceiveNum),
		TimeoutMs:  uint32(timeoutMs),
	}
	ctx.StreamCtx = streamCtx
	return nil
}

// StartSubscribeStream -
func StartSubscribeStream(ctx *types.InvokeProcessContext, httpCtx *gin.Context) error {
	log.GetLogger().Infof("start subscribe stream with name: %s, timeout_ms: %d, expect_num: %d",
		ctx.StreamCtx.StreamName, ctx.StreamCtx.TimeoutMs, ctx.StreamCtx.ExpectNum)
	resultError := datasystemclient.SubscribeStream(datasystemclient.SubscribeParam{
		StreamName:       ctx.StreamCtx.StreamName,
		TimeoutMs:        ctx.StreamCtx.TimeoutMs,
		ExpectReceiveNum: ctx.StreamCtx.ExpectNum,
	}, &datasystemclient.GinCtxAdapter{Context: httpCtx})
	if resultError != nil {
		errInfo, ok := resultError.(api.ErrorInfo)
		if !ok {
			responsehandler.SetErrorInContext(ctx, statuscode.InternalErrorCode,
				"subscribeStream return invalid error type")
		}
		responsehandler.SetErrorInContext(ctx, errInfo.Code, utils.MessageTruncation(errInfo.Error()))
	}
	return resultError
}

func isEmptyString(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func transformToNumber(s string) (bool, int) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return false, -1
	}
	if n <= 0 {
		return false, n
	}
	return true, n
}
