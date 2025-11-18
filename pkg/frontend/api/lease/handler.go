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

// Package lease the api of lease scene
package lease

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/grpc/pb/lease"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/api/functionsystem"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/remoteclientlease"
)

// NewLeaseHandler the handler of put
func NewLeaseHandler(ctx *gin.Context) {
	traceID := getHeaderPrams(ctx)
	msg, isOk := getLeaseRequest(ctx)
	if !isOk {
		return
	}
	log.GetLogger().Infof("receive new lease request, traceID: %s", traceID)
	response := remoteclientlease.NewLease(msg.RemoteClientId, traceID)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	log.GetLogger().Debugf("new lease response:%v, traceID: %s", response, traceID)
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// DelLeaseHandler the handler of del
func DelLeaseHandler(ctx *gin.Context) {
	traceID := getHeaderPrams(ctx)
	msg, isOk := getLeaseRequest(ctx)
	if !isOk {
		return
	}
	log.GetLogger().Infof("receive delete lease request, traceId: %s", traceID)
	response := remoteclientlease.DelLease(msg.RemoteClientId, traceID)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	log.GetLogger().Debugf("delete lease response:%v, traceID: %s", response, traceID)
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// KeepAliveHandler the handler of keep-alive
func KeepAliveHandler(ctx *gin.Context) {
	traceID := getHeaderPrams(ctx)
	msg, isOk := getLeaseRequest(ctx)
	if !isOk {
		return
	}
	log.GetLogger().Infof("receive keep-alive lease request, traceID: %s", traceID)
	response := remoteclientlease.KeepAlive(msg.RemoteClientId, traceID)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	log.GetLogger().Debugf("keep alive lease response:%v, traceID: %s", response, traceID)
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

func getLeaseRequest(ctx *gin.Context) (*lease.LeaseRequest, bool) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		return nil, false
	}
	msg := &lease.LeaseRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse lease request, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return nil, false
	}
	if msg.RemoteClientId == "" {
		log.GetLogger().Errorf("failed to parse lease request, empty remote client id")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return nil, false
	}
	return msg, true
}

func getHeaderPrams(ctx *gin.Context) string {
	traceID := httputil.GetCompatibleGinHeader(ctx.Request, constant.HeaderTraceID, "traceId")
	log.GetLogger().Debugf("check traceID in request header: %s", traceID)
	return traceID
}
