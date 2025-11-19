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

// Package datasystem the api of datasystem scene
package datasystem

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/proto"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/grpc/pb/data"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/api/functionsystem"
	"frontend/pkg/frontend/common/httputil"
)

const (
	errPramInvalid = 2 // error code from datasystem: K_INVALID
	errNone        = 0
)

// PutHandler the handler of put
func PutHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.PutRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse object put request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.ObjectId == "" || msg.ObjectData == nil {
		log.GetLogger().Errorf("failed to parse object put request message, empty id or data")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receive object put request, traceID: %s", traceID)
	state := datasystemclient.ObjPut(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.PutResponse{Code: int32(state.Code)}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}
	if int32(state.Code) == errNone {
		response.Message = "object put success"
	}
	log.GetLogger().Debugf("receive object put response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// GetHandler the handler of get
func GetHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.GetRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse object get request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.ObjectIds == nil {
		log.GetLogger().Errorf("failed to parse object get request message, empty object ids")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receive object get request, traceID: %s", traceID)
	values, state := datasystemclient.ObjGet(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.GetResponse{
		Code:    int32(state.Code),
		Buffers: values,
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}
	if int32(state.Code) == errNone {
		response.Message = "object get success"
	}
	log.GetLogger().Debugf("receive object get response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// IncreaseRefHandler the handler of increaseref
func IncreaseRefHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.IncreaseRefRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse increase ref request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.ObjectIds == nil || msg.RemoteClientId == "" {
		log.GetLogger().Errorf("failed to parse increase ref request message, empty ids")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receive increase ref request, traceID: %s", traceID)
	values, state := datasystemclient.GIncreaseRef(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.IncreaseRefResponse{
		Code:            int32(state.Code),
		FailedObjectIds: values,
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}
	if int32(state.Code) == errNone {
		response.Message = "increase ref success"
	}
	log.GetLogger().Debugf("receive increase ref response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// DecreaseRefHandler the handler of decreaseref
func DecreaseRefHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.DecreaseRefRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse decrease ref request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.ObjectIds == nil || msg.RemoteClientId == "" {
		log.GetLogger().Errorf("failed to parse decrease ref request message, empty ids")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receive decrease ref request, traceID: %s", traceID)
	values, state := datasystemclient.GDecreaseRef(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.DecreaseRefResponse{
		Code:            int32(state.Code),
		FailedObjectIds: values,
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}

	if int32(state.Code) == errNone {
		response.Message = "decrease ref success"
	}
	log.GetLogger().Debugf("receive decrease ref response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// KvSetHandler the handler of kv-set
func KvSetHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.KvSetRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse kv set request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.Key == "" || msg.Value == nil {
		log.GetLogger().Errorf("failed to parse kv set request message, empty key or value")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receives kv set request, traceID: %s", traceID)
	state := datasystemclient.Set(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.KvSetResponse{
		Code: int32(state.Code),
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}

	if int32(state.Code) == errNone {
		response.Message = "kv set success"
	}
	log.GetLogger().Debugf("receive kv set response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// KvMSetTxHandler the handler of kv-mSetTx
func KvMSetTxHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.KvMSetTxRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse kv multi set tx request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.Keys == nil || msg.Values == nil {
		log.GetLogger().Errorf("failed to parse kv multi set tx request message, empty keys or values")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if len(msg.Keys) != len(msg.Values) {
		log.GetLogger().Errorf("failed to parse kv multi set tx request message, "+
			"keys size: %d isn't equal to values size: %d", len(msg.Keys), len(msg.Values))
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receives kv multi set tx request, traceID: %s", traceID)
	state := datasystemclient.MSetTx(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.KvMSetTxResponse{
		Code: int32(state.Code),
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}

	if int32(state.Code) == errNone {
		response.Message = "kv multi set tx success"
	}
	log.GetLogger().Debugf("receive kv multi set tx response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// KvGetHandler the handler of kv-get
func KvGetHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.KvGetRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse kv get request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.Keys == nil {
		log.GetLogger().Errorf("failed to parse kv get request message, empty keys")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receives kv get request, traceID: %s", traceID)
	values, state := datasystemclient.Get(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.KvGetResponse{
		Code:   int32(state.Code),
		Values: values,
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}

	if int32(state.Code) == errNone {
		response.Message = "kv get success"
	}
	log.GetLogger().Debugf("receive kv get response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

// KvDelHandler the handler of kv-del
func KvDelHandler(ctx *gin.Context) {
	tenantID, traceID := getHeaderPrams(ctx)
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		log.GetLogger().Errorf("failed to read request body error %s", err.Error())
		frontend.SetCtxResponse(ctx, nil, http.StatusInternalServerError)
		return
	}
	msg := &data.KvDelRequest{}
	err = proto.Unmarshal(body, msg)
	if err != nil {
		log.GetLogger().Errorf("failed to parse kv del request message, err: %s", err)
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	if msg.Keys == nil {
		log.GetLogger().Errorf("failed to parse kv del request message, empty keys")
		frontend.SetCtxResponse(ctx, nil, http.StatusBadRequest)
		return
	}
	log.GetLogger().Infof("receive kv del request, traceID: %s", traceID)
	values, state := datasystemclient.Del(msg, &datasystemclient.Config{TenantID: tenantID}, traceID)
	response := &data.KvDelResponse{
		Code:       int32(state.Code),
		FailedKeys: values,
	}
	if state.Err != nil {
		response.Message = state.Err.Error()
	}

	log.GetLogger().Debugf("receive kv del response, traceID: %s, code: %d", traceID, state.Code)
	if int32(state.Code) == errNone {
		response.Message = "kv del success"
	}
	log.GetLogger().Debugf("receive kv del response, traceID: %s, code: %d", traceID, state.Code)
	respBody, parseErr := proto.Marshal(response)
	if parseErr != nil {
		log.GetLogger().Errorf("proto Marshal failed after receive kv del response, err: %s", parseErr)
		frontend.SetCtxResponse(ctx, respBody, http.StatusBadRequest)
		return
	}
	frontend.SetCtxResponse(ctx, respBody, http.StatusOK)
}

func getHeaderPrams(ctx *gin.Context) (string, string) {
	tenantID := httputil.GetCompatibleGinHeader(ctx.Request, constant.HeaderTenantID, "tenantId")
	traceID := httputil.GetCompatibleGinHeader(ctx.Request, constant.HeaderTraceID, "traceId")
	log.GetLogger().Debugf("check tenantID and traceID in request header: %s, %s", tenantID, traceID)
	return tenantID, traceID
}
