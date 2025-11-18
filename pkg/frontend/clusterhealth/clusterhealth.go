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

// Package clusterhealth -
package clusterhealth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/tls"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/functiontask"
)

const (
	healthy    = "healthy"
	unhealthy  = "unhealthy"
	subhealthy = "subhealthy"
	unknown    = "unknown"

	task             = "functiontask"
	instanceManager  = "instancemanager"
	functionAccessor = "functionaccessor"

	defaultHealthTimeOut = 5 * time.Second

	headerRouterEtcdState = "X-Router-Etcd-State"
)

var components = [...]string{task, instanceManager, functionAccessor}

type healthyStatus string

// CheckClusterHealth -
func CheckClusterHealth(w http.ResponseWriter, r *http.Request) {
	resultMap := make(map[string]healthyStatus, len(components))

	initComponentsStatus(resultMap)
	if !etcd3.GetRouterEtcdClient().GetEtcdStatusLostContact() {
		resultMap[functionAccessor] = subhealthy
		resultMap[task] = unknown
		resultMap[instanceManager] = unknown
	}
	if authCheckReq(r) != nil {
		w.WriteHeader(http.StatusUnauthorized)
		log.GetLogger().Errorf("failed to return auth error")
		return
	}

	// check task and instance manager
	err := checkCoreComponentsHealth(resultMap, r)
	if err != nil {
		log.GetLogger().Errorf("failed to check component health %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		bytes, err := json.Marshal(resultMap)
		if err != nil {
			_, err := w.Write([]byte(err.Error()))
			if err != nil {
				log.GetLogger().Errorf("failed to write rsp err %s", err.Error())
			}
		}
		_, err = w.Write(bytes)
		if err != nil {
			log.GetLogger().Errorf("failed to write rsp err %s", err.Error())
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	return
}

func initComponentsStatus(resultMap map[string]healthyStatus) {
	if resultMap != nil {
		for _, component := range components {
			resultMap[component] = healthy
		}
	}
}

func checkCoreComponentsHealth(resultMap map[string]healthyStatus, r *http.Request) error {
	if resultMap == nil {
		return fmt.Errorf("failed to check, result map is nil")
	}

	if resultMap[functionAccessor] == subhealthy {
		return errors.New("frontend health status is subhealthy")
	}

	if functiontask.GetBusProxies().GetNum() == 0 {
		resultMap[task] = unhealthy
		resultMap[instanceManager] = unknown
		return errors.New("no available proxy to request")
	}
	return sendClusterHealthCheckToTask(resultMap, r)
}

func sendClusterHealthCheckToTask(resultMap map[string]healthyStatus, r *http.Request) error {
	if resultMap == nil {
		log.GetLogger().Errorf("healthyStatus resultMap is nil")
		return fmt.Errorf("failed to check instance manager healthy")
	}
	// Traverse all nodes. If one node is available, the node is available.
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	var err error
	needReturn := false
	functiontask.GetBusProxies().DoRange(func(nodeID string, nodeIP string) bool {
		setComponentHealthReq(req, nodeIP, r.Header.Get(constant.HeaderAuthTimestamp),
			r.Header.Get(constant.HeaderAuthorization))
		proxyClient := httputil.GetGlobalClient()
		err = proxyClient.DoTimeout(req, resp, defaultHealthTimeOut)
		if err != nil {
			log.GetLogger().Errorf("failed to request proxy %s, err %s", req.URI().String(), err.Error())
			return true
		}
		if resp == nil {
			return true
		}
		if resp.StatusCode() == http.StatusInternalServerError {
			log.GetLogger().Errorf("failed to request proxy, err %s ", string(resp.Body()))
			if string(resp.Header.Peek(headerRouterEtcdState)) == "false" {
				resultMap[instanceManager] = subhealthy
			} else {
				resultMap[instanceManager] = unhealthy
			}
			err = fmt.Errorf("failed to check instance manager healthy")
			needReturn = true
			return false
		}
		if resp.StatusCode() == http.StatusOK {
			needReturn = true
			return false
		}
		err = fmt.Errorf("failed to request proxy")
		log.GetLogger().Errorf("failed to request proxy err code %d uri %s", resp.StatusCode(), req.URI().String())
		return true
	})
	if needReturn {
		return err
	}
	// all tasks are unhealthy
	resultMap[task] = unhealthy
	resultMap[instanceManager] = unknown
	return err
}

func setComponentHealthReq(req *fasthttp.Request, nodeIP string, timeStamp string, authorization string) {
	req.Header.SetMethod(http.MethodGet)
	req.Header.Set(constant.HeaderAuthorization, authorization)
	req.Header.Set(constant.HeaderAuthTimestamp, timeStamp)
	req.Header.ResetConnectionClose()
	req.SetRequestURI("/componenthealth")
	req.Header.SetHost(fmt.Sprintf("%s:%s", nodeIP, constant.BusProxyHTTPPort))
	req.URI().SetScheme(tls.GetURLScheme(config.GetConfig().HTTPSConfig.HTTPSEnable))
}

func authCheckReq(r *http.Request) error {
	functionConfig := config.GetConfig()
	if !functionConfig.AuthenticationEnable {
		return nil
	}
	requestSign := r.Header.Get(constant.HeaderAuthorization)
	timestamp := r.Header.Get(constant.HeaderAuthTimestamp)
	err := localauth.AuthCheckLocally(functionConfig.LocalAuth.AKey, config.GetConfig().LocalAuth.SKey,
		requestSign, timestamp, functionConfig.LocalAuth.Duration)
	if err != nil {
		log.GetLogger().Errorf("failed to check authorization of URL locally, error: %s", err.Error())
		return err
	}
	return nil
}
