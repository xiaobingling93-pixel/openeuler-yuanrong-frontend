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

// Package middleware -
package middleware

import (
	"errors"
	"fmt"
	"strconv"

	"go.uber.org/zap"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/responsehandler"
	"frontend/pkg/frontend/types"
)

const (
	maxRequestBodySizeMsg = "the size of request body is beyond maximum:%d, body size:%d"
	contentLengthInvalid  = "Content-Length is invalid"
	megabytes             = 1024 * 1024

	baseTen = 10
	bitSize = 64
)

// BodySizeChecker -
func BodySizeChecker(next Handler) Handler {
	return func(ctx *types.InvokeProcessContext) error {
		var err error
		if ctx.IsHTTPUploadStream {
			err = checkStreamRequestContentLength(ctx)
		} else {
			err = checkRequestBodySize(ctx)
		}
		if err != nil {
			responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusMaxRequestBodySize, err.Error())
			log.GetLogger().With(zap.Any("traceID", ctx.TraceID)).Errorf("the size of request body is out of range %s",
				err.Error())
			return err
		}
		return next(ctx)
	}
}

// checkRequestBodySize function check whether the body of the function is greater than the configured value.
func checkRequestBodySize(ctx *types.InvokeProcessContext) error {
	bodyLength := len(ctx.ReqBody)
	maxRequestBodySize := config.GetConfig().HTTPConfig.MaxRequestBodySize * megabytes
	if bodyLength > maxRequestBodySize {
		errMsg := fmt.Sprintf(maxRequestBodySizeMsg, maxRequestBodySize, bodyLength)
		log.GetLogger().Errorf(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

func checkStreamRequestContentLength(ctx *types.InvokeProcessContext) error {
	bodyLength, err := getContentLength(ctx)
	if err != nil {
		return err
	}

	maxRequestBodySize := config.GetConfig().HTTPConfig.MaxStreamRequestBodySize * megabytes

	if bodyLength > int64(maxRequestBodySize) {
		errMsg := fmt.Sprintf(maxRequestBodySizeMsg, maxRequestBodySize, bodyLength)
		log.GetLogger().Errorf(errMsg)
		return errors.New(errMsg)
	}
	return nil
}

func getContentLength(ctx *types.InvokeProcessContext) (int64, error) {
	logger := log.GetLogger().With(zap.Any("traceID", ctx.TraceID))
	contentLengthStr, ok := ctx.ReqHeader[constant.HeaderContentLength]
	if !ok {
		logger.Errorf("Content-Length header not found")
		return 0, errors.New(contentLengthInvalid)
	}

	contentLength, err := strconv.ParseInt(contentLengthStr, baseTen, bitSize)
	if err != nil || contentLength < 0 {
		logger.Errorf("Content-Length is invalid")
		return 0, errors.New(contentLengthInvalid)
	}
	return contentLength, nil
}
