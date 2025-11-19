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

	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/monitor"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/urnutils"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/responsehandler"
	"frontend/pkg/frontend/tenanttrafficlimit"
	"frontend/pkg/frontend/types"
)

// TrafficLimiter -
func TrafficLimiter(next Handler) Handler {
	return func(ctx *types.InvokeProcessContext) error {
		if err := tenanttrafficlimit.Limit(urnutils.GetTenantFromFuncKey(ctx.FuncKey)); err != nil {
			responsehandler.SetErrorInContext(ctx, statuscode.FrontendStatusTrafficLimitEffective, err.Error())
			log.GetLogger().Errorf("tenant traffic limit err:%s,traceID%s", err.Error(), ctx.TraceID)
			return err
		}
		memoryWant := uint64(float64(len(ctx.ReqBody)) * config.GetConfig().
			MemoryEvaluatorConfig.RequestMemoryEvaluator)
		if !monitor.IsAllowByMemory(ctx.FuncKey, memoryWant, ctx.TraceID) {
			ErrHeavyLoad := errors.New("http server is under heavy load")
			responsehandler.SetErrorInContext(ctx, statuscode.HeavyLoadCode, ErrHeavyLoad.Error())
			return ErrHeavyLoad
		}
		defer monitor.GetMemInstance().ReleaseFunctionMem(ctx.FuncKey, memoryWant)
		return next(ctx)
	}
}
