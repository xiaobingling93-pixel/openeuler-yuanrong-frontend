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
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/types"
)

// graceExitFlag flag for grace exit , true means grace exit
var graceExitFlag = false

// Wg is waitGroup for graceful shutdown
var Wg sync.WaitGroup

// GraceExitFilter -
func GraceExitFilter(next Handler) Handler {
	return func(ctx *types.InvokeProcessContext) error {
		if graceExitFlag {
			ctx.StatusCode = fasthttp.StatusBadRequest
			ctx.RespBody = []byte("frontend exiting")
			return errors.New("exiting")
		}

		Wg.Add(1)
		defer Wg.Done()
		return next(ctx)
	}
}

// ErrorWriter -
type ErrorWriter interface {
	WriteErrorToGinResponse(ctx *gin.Context, err error)
}

// GraceExitGinFilter -
func GraceExitGinFilter(errWriter ErrorWriter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if graceExitFlag {
			err := snerror.New(statuscode.FrontendStatusBadRequest, "frontend exiting")
			errWriter.WriteErrorToGinResponse(ctx, err)
			ctx.Abort()
			return
		}
		ctx.Next()
	}
}

// GraceExit is used to exit from the system gracefully,
// and after the received message is processed, the interface returns the following information.
// Sleep operation is added in preStop. Before the SIGTERM signal is sent, the k8s removes the endpoint.
// Add a preStop lifecycle hook that exec's sleep 60 or something similar.
// That will be triggered BEFORE you get SIGTERM
// LBs will be able to observe the deletionTimestamp
// see https://github.com/kubernetes/kubernetes/issues/88236
func GraceExit() {
	log.GetLogger().Infof("begin to call GraceExit")
	graceExitFlag = true
	// wait until all received messages are processed
	Wg.Wait()
	log.GetLogger().Infof("end to call GraceExit")
}
