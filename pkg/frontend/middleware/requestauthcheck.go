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
	"fmt"
	"strings"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/responsehandler"
	"frontend/pkg/frontend/types"
	"frontend/pkg/frontend/wisecloud"
)

// RequestAuthCheck  check auth for request
func RequestAuthCheck(next Handler) Handler {
	return func(ctx *types.InvokeProcessContext) error {
		err := authCheck(ctx)
		if err != nil {
			responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusUnAuthorized, err.Error())
			log.GetLogger().Errorf("failed to authenticate request, traceID %s: %s", ctx.TraceID, err.Error())
			return err
		}
		return next(ctx)
	}
}

func authCheck(c *types.InvokeProcessContext) error {
	if !config.GetConfig().AuthenticationEnable {
		return nil
	}
	requestSign := c.ReqHeader[constant.HeaderAuthorization]
	if strings.HasPrefix(requestSign, "HMAC-SHA256 ") {
		if !wisecloud.AuthDownGradeFunctionCall(c.ReqPath, c.RespHeader, c.ReqBody, config.GetConfig().LocalAuth.AKey,
			[]byte(config.GetConfig().LocalAuth.SKey)) {
			log.GetLogger().Errorf("failed to check authorization for downgrade functioncall")
			return fmt.Errorf("auth check failed")
		}
		return nil
	}
	timestamp := c.ReqHeader[constant.HeaderAuthTimestamp]
	err := localauth.AuthCheckLocally(config.GetConfig().LocalAuth.AKey, config.GetConfig().LocalAuth.SKey,
		requestSign, timestamp, config.GetConfig().LocalAuth.Duration)
	if err != nil {
		log.GetLogger().Errorf("failed to check authorization of URL locally, error: %s", err.Error())
		return err
	}
	return nil
}
