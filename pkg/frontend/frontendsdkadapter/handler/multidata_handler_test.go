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

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

func TestDataSystemMultiDelHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", MultiDelHandler)
	tests := []struct {
		name                string
		header              map[string]string
		clusterMap          map[string]string
		payloadData         string
		dataKeys            string
		expectedStatusCode  int
		expectedBody        string
		expectedInnerCode   string
		expectedPayloadInfo string
		maxBodySize         int
		maxKeySize          int
		mockDataSystemFunc  func() *gomonkey.Patches
	}{
		{name: "del_success",
			dataKeys:           "key1|key2",
			header:             dataCommonHeader(),
			payloadData:        `[{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false},{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "{\"code\":0,\"message\":\"success\"}",
			expectedInnerCode:  "0",
			maxBodySize:        1024,
			maxKeySize:         5,
			mockDataSystemFunc: mockDataSystemDeleteSuccess,
		},
		{name: "del_request_body_too_large",
			header:              dataCommonHeader(),
			dataKeys:            "key1|key2",
			payloadData:         "[]",
			expectedStatusCode:  http.StatusOK,
			expectedBody:        "{\"code\":200400,\"message\":\"deserialize request failed, err: body is beyond maximum: 0\"}",
			expectedInnerCode:   "200400",
			expectedPayloadInfo: "",
			maxBodySize:         0,
			maxKeySize:          5,
			mockDataSystemFunc:  mockDataSystemDeleteSuccess,
		},
		{name: "del_empty_keys",
			header:             dataCommonHeader(),
			payloadData:        "[]",
			dataKeys:           "",
			expectedStatusCode: http.StatusOK,
			expectedBody:       "{\"code\":0,\"message\":\"success\"}",
			expectedInnerCode:  "0",
			maxKeySize:         5,
			mockDataSystemFunc: mockDataSystemDeleteSuccess,
		},
		{name: "del_dataSystem_fail",
			header:             dataCommonHeader(),
			dataKeys:           "key1|key2",
			payloadData:        `[{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false},{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			expectedStatusCode: http.StatusOK,
			expectedBody:       "{\"code\":200703,\"message\":\"internal delete failed\"}",
			expectedInnerCode:  "200703",
			maxBodySize:        1024,
			maxKeySize:         5,

			mockDataSystemFunc: mockDataSystemDeleteFail,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p3 := test.mockDataSystemFunc()
			defer p3.Reset()
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(test.dataKeys)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(constant.HeaderDataSystemPayloadInfo, test.payloadData)
			for key, value := range test.header {
				req.Header.Set(key, value)
			}
			config.GetConfig().HTTPConfig = &types.FrontendHTTP{
				MaxDataSystemMultiDataBodySize: test.maxBodySize,
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, test.expectedStatusCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())
			assert.Equal(t, test.expectedInnerCode, w.Header().Get(constant.HeaderInnerCode))
			assert.NotEmpty(t, w.Header().Get(constant.HeaderTraceID))
			assert.Equal(t, test.expectedPayloadInfo, w.Header().Get(constant.HeaderDataSystemPayloadInfo))
		})
	}
}

func dataCommonHeader() map[string]string {
	return map[string]string{
		"Authorization":   "value2",
		"X-Tenant-Id":     "tenantId",
		"X-Client-Id":     "clientId",
		"X-Trace-Id":      "traceId",
		"X-User-Id":       "userId",
		"X-Function-Name": "functionname",
	}
}

func mockDataSystemDeleteSuccess() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.DeleteArrayRetry,
		func(keys []string, config *datasystemclient.Config, traceID string) ([]string, error) {
			return nil, nil
		})
}

func mockDataSystemDeleteFail() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.DeleteArrayRetry,
		func(keys []string, config *datasystemclient.Config, traceID string) ([]string, error) {
			return []string{"key1"}, fmt.Errorf("some keys failed to delete")
		})
}

func TestDataSystemMultiGetHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", MultiGetHandler)
	tests := []struct {
		name                string
		dataKeys            string
		header              map[string]string
		clusterMap          map[string]string
		expectedStatusCode  int
		expectedBody        string
		expectedInnerCode   string
		expectedPayloadInfo string
		maxDataSize         int
		maxKeySize          int
		payloadInfo         string
		mockFunc            func() *gomonkey.Patches
	}{
		{name: "download_success",
			dataKeys:            "key1|key2",
			expectedStatusCode:  http.StatusOK,
			header:              dataCommonHeader(),
			expectedBody:        "a",
			expectedInnerCode:   "0",
			maxDataSize:         1024,
			maxKeySize:          5,
			payloadInfo:         `[{"dataKey": "key1", "dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			expectedPayloadInfo: "[{\"dataKey\":\"key1\",\"offset\":0,\"len\":1}]",
			mockFunc:            mockDataSystemDownloadSuccess,
		},
		{name: "download_multiple_success",
			dataKeys:            "key1|key2",
			expectedStatusCode:  http.StatusOK,
			header:              dataCommonHeader(),
			expectedBody:        "a",
			expectedInnerCode:   "0",
			maxDataSize:         1024,
			maxKeySize:          5,
			payloadInfo:         `[{"dataKey": "key1", "dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false},{"dataKey": "key2", "dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			expectedPayloadInfo: "[{\"dataKey\":\"key1\",\"offset\":0,\"len\":1},{\"dataKey\":\"key2\",\"offset\":1,\"len\":0}]",
			mockFunc:            mockDataSystemDownloadMultipleSuccess,
		},
		{name: "download_size_not_match",
			dataKeys:            "key1|key2",
			expectedStatusCode:  http.StatusOK,
			header:              dataCommonHeader(),
			expectedBody:        "{\"code\":200500,\"message\":\"keylen = 2 is not equals to data len = 1\"}",
			expectedInnerCode:   "200500",
			maxDataSize:         1024,
			maxKeySize:          5,
			payloadInfo:         `[{"dataKey": "key1", "dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false},{"dataKey": "key2", "dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			expectedPayloadInfo: "",
			mockFunc:            mockDataSystemDownloadSuccess,
		},
		{name: "download_request_body_too_large",
			dataKeys:            string(createByteArray(1<<10 + 1)),
			header:              dataCommonHeader(),
			maxDataSize:         0,
			payloadInfo:         "[]",
			expectedStatusCode:  http.StatusOK,
			expectedBody:        "{\"code\":200400,\"message\":\"deserialize request failed, err: body is beyond maximum: 0\"}",
			expectedInnerCode:   "200400",
			expectedPayloadInfo: "",
			mockFunc:            mockDataSystemDownloadSuccess,
		},
		{name: "download_empty_keys",
			dataKeys:            "",
			header:              dataCommonHeader(),
			payloadInfo:         "[]",
			maxDataSize:         1024,
			maxKeySize:          5,
			expectedStatusCode:  http.StatusOK,
			expectedBody:        "",
			expectedInnerCode:   "0",
			expectedPayloadInfo: "",
			mockFunc:            mockDataSystemDownloadSuccess,
		},
		{name: "download_failed",
			dataKeys:            "key1|key2",
			payloadInfo:         `[{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false},{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			header:              dataCommonHeader(),
			maxDataSize:         1024,
			maxKeySize:          5,
			expectedStatusCode:  http.StatusOK,
			expectedBody:        "{\"code\":200702,\"message\":\"internal download failed\"}",
			expectedInnerCode:   "200702",
			expectedPayloadInfo: "",
			mockFunc:            mockDataSystemDownloadFail,
		},
		{name: "download_failed_parse_err",
			dataKeys:            "key1|key2",
			payloadInfo:         `[{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false},{"dataPrefix": "prefix", "offset": 0, "len": 0, "needEncrypt":false}]`,
			header:              dataCommonHeader(),
			maxKeySize:          5,
			maxDataSize:         1024,
			expectedStatusCode:  http.StatusOK,
			expectedBody:        "{\"code\":200702,\"message\":\"internal download failed\"}",
			expectedInnerCode:   "200702",
			expectedPayloadInfo: "",
			mockFunc:            mockDataSystemDownloadFail,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte(test.dataKeys)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set(constant.HeaderDataSystemPayloadInfo, test.payloadInfo)
			for key, value := range test.header {
				req.Header.Set(key, value)
			}
			config.GetConfig().HTTPConfig = &types.FrontendHTTP{
				MaxDataSystemMultiDataBodySize: test.maxDataSize,
			}
			p3 := test.mockFunc()
			defer p3.Reset()
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, test.expectedStatusCode, w.Code)
			assert.Equal(t, test.expectedBody, w.Body.String())
			assert.Equal(t, test.expectedInnerCode, w.Header().Get(constant.HeaderInnerCode))
			assert.NotEmpty(t, w.Header().Get(constant.HeaderTraceID))
			assert.Equal(t, test.expectedPayloadInfo, w.Header().Get(constant.HeaderDataSystemPayloadInfo))
		})
	}
}

func mockDataSystemDownloadSuccess() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.DownloadArrayRetry, func(keys []string,
		config *datasystemclient.Config, traceID string) ([][]byte, error) {
		return [][]byte{[]byte("a")}, nil
	})
}

func mockDataSystemDownloadMultipleSuccess() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.DownloadArrayRetry, func(keys []string,
		config *datasystemclient.Config, traceID string) ([][]byte, error) {
		return [][]byte{[]byte("a"), []byte("")}, nil
	})
}

func mockDataSystemDownloadFail() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.DownloadArrayRetry, func(keys []string,
		config *datasystemclient.Config, traceID string) ([][]byte, error) {
		return [][]byte{[]byte("a")}, fmt.Errorf("err")
	})
}

func TestDataSystemMultiSetHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", MultiSetHandler)
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
		mockFunc           func() *gomonkey.Patches
	}{
		{
			name:               "empty_request_success",
			header:             dataCommonHeader(),
			maxKeySize:         5,
			payloadInfo:        "[{\"dataPrefix\": \"prefix\", \"offset\": 0, \"len\": 0, \"needEncrypt\": false}]",
			body:               nil,
			expectedResponse:   "{\"dataKeys\":null}",
			expectedInnerCode:  "0",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockFunc:           mockDataSystemUploadSuccess,
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
			mockFunc:           mockDataSystemUploadSuccess,
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
			mockFunc:           mockDataSystemUploadSuccess,
		},
		{
			name:               "body_len_too_large",
			payloadInfo:        `[]`,
			maxKeySize:         5,
			body:               append(body, 'a'),
			expectedResponse:   "{\"code\":200400,\"message\":\"deserialize request failed, err: body is beyond maximum: 0\"}",
			expectedInnerCode:  "200400",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        0,
			mockFunc:           mockDataSystemUploadSuccess,
		},
		{
			name:               "upload_success",
			payloadInfo:        `[{"dataPrefix": "prefix", "offset": 0, "len": 512, "needEncrypt": false }]`,
			header:             dataCommonHeader(),
			maxKeySize:         5,
			body:               body,
			expectedResponse:   "{\"dataKeys\":[\"key1\"]}",
			expectedInnerCode:  "0",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockFunc:           mockDataSystemUploadSuccess,
		},
		{
			name:               "upload_multiple_success",
			payloadInfo:        `[{"dataPrefix": "prefix", "offset": 0, "len": 512, "needEncrypt": false }, {"dataPrefix": "prefix", "offset": 512, "len": 512, "needEncrypt": false}]`,
			header:             dataCommonHeader(),
			maxKeySize:         5,
			body:               body,
			expectedResponse:   "{\"dataKeys\":[\"key1\",\"key1\"]}",
			expectedInnerCode:  "0",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockFunc:           mockDataSystemUploadSuccess,
		},
		{
			name:               "upload_fail",
			payloadInfo:        `[{"dataPrefix": "prefix","offset": 0,"len": 1024,"needEncrypt": false}]`,
			header:             dataCommonHeader(),
			maxKeySize:         5,
			body:               body,
			expectedResponse:   "{\"code\":200701,\"message\":\"internal upload failed\"}",
			expectedInnerCode:  "200701",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockFunc:           mockDataSystemUploadFail,
		},
		{
			name:               "upload_execeed_key_size",
			payloadInfo:        `[{"dataPrefix": "prefix","offset": 0,"len": 1024,"needEncrypt": false}]`,
			header:             dataCommonHeader(),
			maxKeySize:         1,
			body:               body,
			expectedResponse:   "{\"code\":200701,\"message\":\"internal upload failed\"}",
			expectedInnerCode:  "200701",
			expectedStatusCode: http.StatusOK,
			maxDataSize:        1024,
			mockFunc:           mockDataSystemUploadFail,
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
			p3 := test.mockFunc()
			defer p3.Reset()
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, test.expectedStatusCode, w.Code)
			assert.Equal(t, test.expectedResponse, w.Body.String())
			assert.Equal(t, test.expectedInnerCode, w.Header().Get(constant.HeaderInnerCode))
			assert.NotEmpty(t, w.Header().Get(constant.HeaderTraceID))
		})
	}
}

func createByteArray(byteSize int) []byte {
	byteArray := make([]byte, byteSize)
	for i := 0; i < byteSize; i++ {
		byteArray[i] = 'A'
	}
	return byteArray
}

func mockDataSystemUploadSuccess() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.UploadWithKeyRetry, func(value []byte,
		config *datasystemclient.Config, param api.SetParam, traceID string) (string, error) {
		return "key1", nil
	})
}

func mockDataSystemUploadFail() *gomonkey.Patches {
	return gomonkey.ApplyFunc(datasystemclient.UploadWithKeyRetry, func(value []byte,
		config *datasystemclient.Config, param api.SetParam, traceID string) (string, error) {
		return "", fmt.Errorf("dsclient is nil")
	})
}
