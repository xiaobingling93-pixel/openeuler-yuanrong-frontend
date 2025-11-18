//go:build module

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

package stream

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

type MockResponseWriter struct{}

func (m MockResponseWriter) Header() http.Header {
	return http.Header{}
}

func (m MockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m MockResponseWriter) WriteHeader(int) {
	return
}

func TestRegisterResponse(t *testing.T) {
	t.Run("not enable dataSystem", func(t *testing.T) {
		config.GetConfig().StreamEnable = false
		reqInfo := &ResponseStream{
			processContext: &types.InvokeProcessContext{
				RequestTraceInfo: &types.RequestTraceInfo{
					Deadline: time.Now().Add(-1 * time.Second),
				},
				StreamInfo: &types.StreamInvokeInfo{
					RspStream: MockResponseWriter{}},
			},
		}
		RegisterResponse(reqInfo.processContext)
		assert.Nil(t, reqInfo.processContext.StreamInfo.ResponseStopChan)
		_, ok := responseStreamMap.Load(reqInfo.processContext.StreamInfo.ResponseStreamName)
		assert.False(t, ok)
	})
	t.Run("enable dataSystem", func(t *testing.T) {
		config.GetConfig().StreamEnable = true
		reqInfo := &ResponseStream{
			processContext: &types.InvokeProcessContext{
				RequestTraceInfo: &types.RequestTraceInfo{
					Deadline: time.Now().Add(-1 * time.Second),
				},
				StreamInfo: &types.StreamInvokeInfo{
					RspStream: MockResponseWriter{}},
			},
		}
		RegisterResponse(reqInfo.processContext)
		assert.NotNil(t, reqInfo.processContext.StreamInfo.ResponseStopChan)
		_, ok := responseStreamMap.Load(reqInfo.processContext.StreamInfo.ResponseStreamName)
		assert.True(t, ok)
	})
}

func TestCheckIsResponseStream(t *testing.T) {
	t.Run("not enable datasystem", func(t *testing.T) {
		config.GetConfig().StreamEnable = false
		result := CheckIsResponseStream("xxx")
		assert.False(t, result)
	})
	t.Run("not exist", func(t *testing.T) {
		config.GetConfig().StreamEnable = true
		traceID := "not exist"
		result := CheckIsResponseStream(traceID)
		assert.False(t, result)
	})
	t.Run("not_response_stream_type", func(t *testing.T) {
		config.GetConfig().StreamEnable = true
		traceID := "test_trace_id"
		responseStreamMap.Store(traceID, "not_response_stream_type")
		result := CheckIsResponseStream(traceID)
		assert.False(t, result)
	})
	t.Run("not stream", func(t *testing.T) {
		config.GetConfig().StreamEnable = true
		traceID := "test_trace_id"
		responseStreamMap.Store(traceID, &ResponseStream{isStream: false})
		defer responseStreamMap.Delete(traceID)
		result := CheckIsResponseStream(traceID)
		assert.False(t, result)
	})
	t.Run("is stream", func(t *testing.T) {
		config.GetConfig().StreamEnable = true
		traceID := "test_trace_id"
		responseStreamMap.Store(traceID, &ResponseStream{isStream: true})
		defer responseStreamMap.Delete(traceID)
		result := CheckIsResponseStream(traceID)
		assert.True(t, result)
	})
}
