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

package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/common"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/functionmeta"
)

// ProxyHandler -
func ProxyHandler(ctx *gin.Context) {
	traceID := httputil.InitTraceID(ctx)
	logger := log.GetLogger().With(zap.Any("traceId", traceID))
	logger.Infof("route proxy handler receives one request")
	path := ctx.Request.URL.Path
	funcSpc, ok := functionmeta.LoadFuncSpecWithPath(path, traceID)
	if !ok {
		logger.Infof("load funcSpec with path failed, path: %s,traceID %s", path, traceID)
		ctx.Writer.WriteHeader(http.StatusNotFound)
		ctx.String(http.StatusNotFound, "404 page not found")
		return
	}
	ctx.AddParam(common.FunctionUrnParam, funcSpc.FuncMetaData.FunctionVersionURN)
	ctx.Request.Header.Set(constant.HeaderRequestID, traceID)
	InvokeHandler(ctx)
}
