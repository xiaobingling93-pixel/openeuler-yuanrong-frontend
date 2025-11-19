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

// Package responsehandler -
package responsehandler

import (
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/frontend/types"
)

// SetErrorInContextWithDefault will set error in process context
func SetErrorInContextWithDefault(ctx *types.InvokeProcessContext, err error, defaultInnerErrorCode int,
	defaultMessage interface{}) {
	errorCode := defaultInnerErrorCode
	errorMessage := defaultMessage
	snErr, ok := err.(snerror.SNError)
	if ok {
		errorCode = snErr.Code()
		errorMessage = snErr.Error()
	}
	if ctx.RequestTraceInfo != nil {
		ctx.RequestTraceInfo.InnerCode = errorCode
	}
	Handler.SetResponseFromFrontend(ctx, errorCode, errorMessage)
}

// SetErrorInContext will set error in process context
func SetErrorInContext(ctx *types.InvokeProcessContext, innerCode int, message interface{}) {
	Handler.SetResponseFromFrontend(ctx, innerCode, message)
}

// SetResponseInContext will set response body in process context
func SetResponseInContext(ctx *types.InvokeProcessContext,
	message []byte) (*types.CallResp, snerror.SNError) {
	return Handler.SetResponseFromInvocation(ctx, message)
}

// HandlerInterface -
type HandlerInterface interface {
	SetResponseFromFrontend(ctx *types.InvokeProcessContext, innerCode int, message interface{})
	SetResponseFromInvocation(ctx *types.InvokeProcessContext,
		message []byte) (*types.CallResp, snerror.SNError)
}

var (
	// Handler -
	Handler HandlerInterface
)
