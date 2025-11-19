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

// Package handler
package handler

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/invocation"
	"frontend/pkg/frontend/types"
)

func TestExecuteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", ExecuteHandler)
	body := createByteArray(1 << 10)
	tests := []struct {
		name               string
		header             map[string]string
		clusterMap         map[string]string
		payloadInfo        string
		body               []byte
		maxDataSize        int
		maxKeySize         int
		expectedResponse   string
		expectedInnerCode  string
		expectedStatusCode int
		expectedTraceID    string
		mockUploadFunc     func() *gomonkey.Patches
		mockInvokeFunc     func() *gomonkey.Patches
		mockDownloadFunc   func() *gomonkey.Patches
		mockDeleteFunc     func() *gomonkey.Patches
	}{
		{
			name:               "execute_success",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 512, \"needEncrypt\": false}]",
			body:               body,
			expectedResponse:   "a",
			expectedInnerCode:  "0",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadSuccess,
			mockInvokeFunc:     mockInvokeHandlerSuccess,
			mockDownloadFunc:   mockDataSystemDownloadSuccess,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
		{
			name:               "payloadInfo_large_than_body",
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 1048276, \"needEncrypt\": false}]",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			body:               nil,
			expectedResponse:   "{\"code\":200400,\"message\":\"deserialize request failed, err: payload len invalid\"}",
			expectedInnerCode:  "200400",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadSuccess,
			mockInvokeFunc:     mockInvokeHandlerSuccess,
			mockDownloadFunc:   mockDataSystemDownloadSuccess,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
		{
			name:               "payloadInfo-header-json-invalid",
			payloadInfo:        `{}`,
			header:             dataCommonHeader(),
			maxKeySize:         5,
			body:               nil,
			expectedResponse:   "{\"code\":200400,\"message\":\"deserialize request failed, err: payloadInfo json invalid\"}",
			expectedInnerCode:  "200400",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadSuccess,
			mockInvokeFunc:     mockInvokeHandlerSuccess,
			mockDownloadFunc:   mockDataSystemDownloadSuccess,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
		{
			name:               "execute_upload_failed",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 512, \"needEncrypt\": false}]",
			body:               body,
			expectedResponse:   "{\"code\":200701,\"message\":\"internal upload failed\"}",
			expectedInnerCode:  "200701",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadFail,
			mockInvokeFunc:     mockInvokeHandlerSuccess,
			mockDownloadFunc:   mockDataSystemDownloadSuccess,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
		{
			name:               "execute_invoke_fail",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 512, \"needEncrypt\": false}]",
			body:               body,
			expectedResponse:   "{\"code\":200705,\"message\":\"internal invoke failed\"}",
			expectedInnerCode:  "200705",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadSuccess,
			mockInvokeFunc:     mockInvokeHandlerFail,
			mockDownloadFunc:   mockDataSystemDownloadSuccess,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
		{
			name:               "execute_download_fail",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 512, \"needEncrypt\": false}]",
			body:               body,
			expectedResponse:   "{\"code\":200702,\"message\":\"internal download failed\"}",
			expectedInnerCode:  "200702",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadSuccess,
			mockInvokeFunc:     mockInvokeHandlerSuccess,
			mockDownloadFunc:   mockDataSystemDownloadFail,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
		{
			name:               "execute_key_not_match",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 512, \"needEncrypt\": false}]",
			body:               body,
			expectedResponse:   "{\"code\":200500,\"message\":\"keylen = 1 is not equals to data len = 2\"}",
			expectedInnerCode:  "200500",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockUploadFunc:     mockDataSystemUploadSuccess,
			mockInvokeFunc:     mockInvokeHandlerSuccess,
			mockDownloadFunc:   mockDataSystemDownloadMultipleSuccess,
			mockDeleteFunc:     mockDataSystemDeleteSuccess,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader(test.body))
			req.Header.Set("Content-Type", "application/octet-stream")
			req.Header.Set(constant.HeaderDataSystemPayloadInfo, test.payloadInfo)
			for key, value := range test.header {
				req.Header.Set(key, value)
			}

			config.GetConfig().HTTPConfig = &types.FrontendHTTP{
				MaxDataSystemMultiDataBodySize: test.maxDataSize,
			}
			p3 := test.mockUploadFunc()
			defer p3.Reset()
			p4 := test.mockInvokeFunc()
			defer p4.Reset()
			p5 := test.mockDownloadFunc()
			defer p5.Reset()
			p6 := test.mockDeleteFunc()
			defer p6.Reset()

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, test.expectedStatusCode, w.Code)
			assert.Equal(t, test.expectedResponse, w.Body.String())
			assert.Equal(t, test.expectedInnerCode, w.Header().Get(constant.HeaderInnerCode))
			assert.NotEmpty(t, w.Header().Get(constant.HeaderTraceID))
		})
	}
}
func mockInvokeHandlerSuccess() *gomonkey.Patches {
	return gomonkey.ApplyFunc(invocation.InvokeHandler, func(ctx *types.InvokeProcessContext) error {
		ctx.RespHeader[constant.HeaderInnerCode] = "0"
		ctx.RespHeader["X-Caas-Data-System-Key"] = "key1"
		ctx.StatusCode = http.StatusOK
		ctx.RespBody = []byte("internal invoke success")
		return nil
	})
}

func mockInvokeHandlerFail() *gomonkey.Patches {
	return gomonkey.ApplyFunc(invocation.InvokeHandler, func(ctx *types.InvokeProcessContext) error {
		ctx.RespHeader[constant.HeaderInnerCode] = "400"
		ctx.RespBody = []byte("internal invoke failed")
		return fmt.Errorf("internal invoke failed")
	})
}
