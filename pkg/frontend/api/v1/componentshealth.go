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
	"encoding/json"

	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/logger/log"
)

const (
	healthy = "healthy"

	task             = "functiontask"
	instanceManager  = "instancemanager"
	functionAccessor = "functionaccessor"
)

var components = [...]string{task, instanceManager, functionAccessor}

type healthyStatus string

// ComponentsHealthHandler - handler components health check request
func ComponentsHealthHandler(ctx *fasthttp.RequestCtx) {
	// Temporary realization. return all healthy response same with fg
	resultMap := make(map[string]healthyStatus, len(components))
	resultMap[functionAccessor] = healthy
	resultMap[task] = healthy
	resultMap[instanceManager] = healthy
	bytes, err := json.Marshal(resultMap)
	if err != nil {
		log.GetLogger().Errorf("failed to marshal resultMap, error: %s", err.Error())
		ctx.Response.SetBody([]byte(err.Error()))
	} else {
		ctx.Response.SetBody(bytes)
	}
	ctx.Response.SetStatusCode(fasthttp.StatusOK)
	return
}
