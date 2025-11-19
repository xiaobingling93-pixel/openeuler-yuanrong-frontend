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

package clusterhealth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/common/faas_common/tls"
	"frontend/pkg/frontend/common/httputil"
	config2 "frontend/pkg/frontend/config"
	"frontend/pkg/frontend/functiontask"
	"frontend/pkg/frontend/types"
)

func clearGetProxies() {
	clearList := []string{}
	functiontask.GetBusProxies().DoRange(func(nodeID, nodeIP string) bool {
		clearList = append(clearList, nodeID)
		return true
	})
	for _, nodeID := range clearList {
		functiontask.GetBusProxies().Delete(nodeID)
	}
}

func TestClusterHealth(t *testing.T) {
	clearGetProxies()
	c := &types.Config{
		AuthenticationEnable: true,
		MemoryEvaluatorConfig: &types.MemoryEvaluatorConfig{
			RequestMemoryEvaluator: 2,
		},
		DefaultTenantLimitQuota: 1800,
		HTTPConfig: &types.FrontendHTTP{
			WorkerInstanceReadTimeOut: 60,
			MaxRequestBodySize:        100,
		},
		HTTPSConfig:     &tls.InternalHTTPSConfig{},
		E2EMaxDelayTime: 60,
		LocalAuth: &localauth.AuthConfig{
			AKey:     "ak",
			SKey:     "sk",
			Duration: 5,
		},
		InvokeMaxRetryTimes: 3,
		RetryConfig:         &types.RetryConfig{},
		HeartbeatConfig: &types.HeartbeatConfig{
			HeartbeatTimeout:          1,
			HeartbeatInterval:         2,
			HeartbeatTimeoutThreshold: 3,
		},
	}

	patched := []*gomonkey.Patches{
		gomonkey.ApplyFunc(localauth.AuthCheckLocally, func(ak string, sk string, requestSign string, timestamp string, duration int) error {
			if timestamp == "" {
				return errors.New("no auth check info")
			}
			return nil
		}),
		gomonkey.ApplyPrivateMethod(reflect.TypeOf(&functiontask.BusProxy{}), "startMonitor", func(_ *functiontask.BusProxy, ch chan struct{}, status *int32) {
			return
		}),
		gomonkey.ApplyFunc(config2.GetConfig, func() *types.Config {
			return c
		}),
	}
	defer func() {
		for i := range patched {
			patched[i].Reset()
		}
	}()

	router := gin.New()
	router.GET("/serverless/v1/componentshealth", func(context *gin.Context) {
		CheckClusterHealth(context.Writer, context.Request)
	})

	httpClient := httputil.GetGlobalClient()

	tests := []struct {
		name            string
		proxyStatusMap  map[string]int
		headerMap       map[string]string
		timestamp       string
		exceptedCode    int
		expectResultMap map[string]string
	}{
		{
			name:            "no proxy",
			proxyStatusMap:  map[string]int{},
			timestamp:       strconv.Itoa(int(time.Now().Unix())),
			exceptedCode:    http.StatusInternalServerError,
			expectResultMap: map[string]string{task: unhealthy, instanceManager: unknown, functionAccessor: healthy},
		},
		{
			name:            "frontend lost router etcd contact",
			proxyStatusMap:  map[string]int{},
			timestamp:       strconv.Itoa(int(time.Now().Unix())),
			headerMap:       map[string]string{functionAccessor: "false"},
			exceptedCode:    http.StatusInternalServerError,
			expectResultMap: map[string]string{task: unknown, instanceManager: unknown, functionAccessor: subhealthy},
		},
		{
			name:           "proxy all ok",
			timestamp:      strconv.Itoa(int(time.Now().Unix())),
			proxyStatusMap: map[string]int{"127.0.0.1": http.StatusOK, "127.0.0.2": http.StatusOK},
			exceptedCode:   http.StatusOK,
		},
		{
			name:            "instance manager is not ok",
			timestamp:       strconv.Itoa(int(time.Now().Unix())),
			proxyStatusMap:  map[string]int{"127.0.0.1": http.StatusInternalServerError, "127.0.0.2": http.StatusInternalServerError},
			exceptedCode:    http.StatusInternalServerError,
			expectResultMap: map[string]string{task: healthy, instanceManager: unhealthy, functionAccessor: healthy},
		},
		{
			name:            "instance manager lost router etcd contact",
			timestamp:       strconv.Itoa(int(time.Now().Unix())),
			proxyStatusMap:  map[string]int{"127.0.0.1": http.StatusInternalServerError, "127.0.0.2": http.StatusInternalServerError},
			headerMap:       map[string]string{functionAccessor: "true", instanceManager: "false"},
			exceptedCode:    http.StatusInternalServerError,
			expectResultMap: map[string]string{task: healthy, instanceManager: subhealthy, functionAccessor: healthy},
		},
		{
			name:            "proxy all not ok",
			timestamp:       strconv.Itoa(int(time.Now().Unix())),
			proxyStatusMap:  map[string]int{"127.0.0.1": http.StatusBadRequest, "127.0.0.2": http.StatusBadRequest},
			exceptedCode:    http.StatusInternalServerError,
			expectResultMap: map[string]string{task: unhealthy, instanceManager: unknown, functionAccessor: healthy},
		},
		{
			name:           "auth check failed",
			proxyStatusMap: map[string]int{"127.0.0.1": http.StatusOK, "127.0.0.2": http.StatusOK},
			exceptedCode:   http.StatusUnauthorized,
		},
	}
	for _, test := range tests {
		for nodeIP, _ := range test.proxyStatusMap {
			nodeKey := "/sn/workers/business/yrk/tenant/0/function/function-task/version/$latest/defaultaz/" + nodeIP
			functiontask.GetBusProxies().Add(nodeKey, nodeIP)
		}
		patches := []*gomonkey.Patches{
			gomonkey.ApplyMethod(reflect.TypeOf(httpClient), "DoTimeout",
				func(_ *fasthttp.Client, req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error {
					nodeIP := strings.Split(string(req.Host()), ":")[0]
					if test.headerMap != nil && test.headerMap[instanceManager] == "false" {
						resp.Header.Set(headerRouterEtcdState, "false")
					}
					resp.SetStatusCode(test.proxyStatusMap[nodeIP])
					return nil
				}),
			gomonkey.ApplyFunc(etcd3.GetRouterEtcdClient, func() *etcd3.EtcdClient {
				return &etcd3.EtcdClient{}
			}),
			gomonkey.ApplyFunc((*etcd3.EtcdClient).GetEtcdStatusLostContact, func(_ *etcd3.EtcdClient) bool {
				if test.headerMap != nil && test.headerMap[functionAccessor] == "false" {
					return false
				}
				return true
			}),
		}
		req, _ := http.NewRequest(http.MethodGet, "/serverless/v1/componentshealth", nil)

		req.Header.Set(constant.HeaderAuthTimestamp, test.timestamp)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		// check response status code
		assert.Equal(t, test.exceptedCode, w.Result().StatusCode, test.name)
		t.Logf("after check expect code")
		// cluster is healthy only response http.StatusOK without responseBody
		if test.exceptedCode == http.StatusInternalServerError {
			expectResponseBody, _ := json.Marshal(test.expectResultMap)
			actualResponseBody, _ := ioutil.ReadAll(w.Result().Body)
			assert.Equal(t, string(expectResponseBody), string(actualResponseBody), test.name)
		}
		// 清理函数
		func() {
			clearGetProxies()
			for _, p := range patches {
				p.Reset()
			}
		}()
	}
}
