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

// Package functiontask manage Function LB and deal with worker instance
package functiontask

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/loadbalance"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

const (
	busProxyHealthy         = 0
	busProxyUnhealthy       = 1
	busHeartBeatLogInternal = 20
)

var (
	once sync.Once

	busproxys *BusProxies
)

// GetBusProxies - test用例使用之前最好使用clearGetProxies清理下
func GetBusProxies() *BusProxies {
	once.Do(func() {
		busproxys = &BusProxies{
			logger:         log.GetLogger().With(zap.Any("BusProxies", "")),
			loadBalance:    loadbalance.NewCHGeneric(),
			list:           make(map[string]*BusProxy),
			nodeIPToNodeID: make(map[string]string),
		}
	})
	return busproxys
}

// BusProxies -
type BusProxies struct {
	list           map[string]*BusProxy
	nodeIPToNodeID map[string]string
	sync.RWMutex
	logger      api.FormatLogger
	loadBalance loadbalance.LoadBalance
}

// BusProxy -
type BusProxy struct {
	ch     chan struct{}
	NodeID string
	NodeIP string

	url                   string
	status                *int32 // busProxyHealthy: 0, busProxyUnhealthy: 1
	types.HeartbeatConfig `json:"-"`
	m                     sync.RWMutex
	healthyCB             func(nodeIP string)
	unhealthyCB           func(nodeIP string)
	logger                api.FormatLogger
}

// Add -
func (fts *BusProxies) Add(nodeID, nodeIP string) {
	fts.Lock()
	defer fts.Unlock()
	if _, ok := fts.list[nodeID]; ok {
		fts.logger.Infof("no need add duplicate busproxy: %s", nodeID)
		return
	}
	f := newBusProxy(nodeIP, fts.lbAdd, fts.lbDel)
	fts.list[nodeID] = f
	fts.nodeIPToNodeID[nodeIP] = nodeID
}

// Delete -
func (fts *BusProxies) Delete(nodeID string) {
	fts.Lock()
	defer fts.Unlock()
	f, ok := fts.list[nodeID]
	if !ok {
		fts.logger.Infof("no need Delete BusProxy: %s, not exist", nodeID)
		return
	}
	f.stopMonitor()
	fts.logger.Infof("Delete BusProxy: %s", nodeID)
	delete(fts.list, nodeID)
	delete(fts.nodeIPToNodeID, f.NodeIP)
}

func (fts *BusProxies) lbAdd(nodeIP string) {
	fts.logger.Infof("add busproxy: %s into loadbalance", nodeIP)
	fts.loadBalance.Add(nodeIP, 0)
}

func (fts *BusProxies) lbDel(nodeIP string) {
	fts.logger.Infof("del busproxy: %s from loadbalance", nodeIP)
	fts.loadBalance.Remove(nodeIP)
}

// UpdateConfig -
func (fts *BusProxies) UpdateConfig() {
	fts.Lock()
	defer fts.Unlock()
	fts.logger.Infof("BusProxies update config")
	defer fts.logger.Infof("BusProxies update config over")
	for _, b := range fts.list {
		b.updateConfig()
	}
}

// GetNum -
func (fts *BusProxies) GetNum() int {
	fts.RLock()
	num := len(fts.list)
	fts.RUnlock()
	return num
}

// DoRange -
func (fts *BusProxies) DoRange(fn func(nodeID string, nodeIP string) bool) {
	fts.RLock()
	defer fts.RUnlock()
	for nodeID, ft := range fts.list {
		if !fn(nodeID, ft.NodeIP) {
			return
		}
	}
}

// NextWithName -
func (fts *BusProxies) NextWithName(name string, move bool) string {
	raw := fts.loadBalance.Next(name, move)
	if raw == nil {
		return ""
	}
	nodeIP, ok := raw.(string)
	if !ok {
		fts.logger.Warnf("node is not string type: %v", raw)
		return ""
	}
	return nodeIP
}

// IsBusProxyHealthy -
func (fts *BusProxies) IsBusProxyHealthy(nodeIP string, traceID string) bool {
	fts.RLock()
	defer fts.RUnlock()
	nodeID, ok := fts.nodeIPToNodeID[nodeIP]
	if !ok {
		fts.logger.Warnf("not found the busproxy: %s, traceID: %s", nodeIP, traceID)
		return false
	}
	b, ok := fts.list[nodeID]
	if !ok {
		fts.logger.Warnf("not found the busproxy: %s, traceID: %s", nodeIP, traceID)
		return false
	}
	return b.IsHealthy()
}

func newBusProxy(nodeIP string, healthyCB func(nodeIP string), unhealthyCB func(nodeIP string)) *BusProxy {
	url := ""
	if config.GetConfig().HTTPSConfig.HTTPSEnable {
		url = fmt.Sprintf("https://%s:%s/heartbeat", nodeIP, constant.BusProxyHTTPPort)
	} else {
		url = fmt.Sprintf("http://%s:%s/heartbeat", nodeIP, constant.BusProxyHTTPPort)
	}

	status := new(int32)
	*status = busProxyUnhealthy
	b := &BusProxy{
		ch:              make(chan struct{}),
		NodeIP:          nodeIP,
		url:             url,
		HeartbeatConfig: *config.GetConfig().HeartbeatConfig,
		status:          status,
		m:               sync.RWMutex{},
		healthyCB:       healthyCB,
		unhealthyCB:     unhealthyCB,
		logger: log.GetLogger().With(zap.Any("busproxy", ""), zap.Any("nodeIP", nodeIP),
			zap.Any("url", url), zap.Any("heartbeatConfig", *config.GetConfig().HeartbeatConfig)),
	}
	go b.startMonitor(b.ch, status)
	return b
}

func (b *BusProxy) updateConfig() {
	url := ""
	if config.GetConfig().HTTPSConfig.HTTPSEnable {
		url = fmt.Sprintf("https://%s:%s/heartbeat", b.NodeIP, constant.BusProxyHTTPPort)
	} else {
		url = fmt.Sprintf("http://%s:%s/heartbeat", b.NodeIP, constant.BusProxyHTTPPort)
	}
	heartbeatConfig := config.GetConfig().HeartbeatConfig
	bytesNew, err1 := json.Marshal(heartbeatConfig)
	bytesOld, err2 := json.Marshal(b.HeartbeatConfig)
	if err1 != nil || err2 != nil {
		b.logger.Warnf("unmarshal heartbeatConfig failed")
		return
	}

	if url != b.url || string(bytesOld) != string(bytesNew) {
		b.logger.Infof("update heartbeatConfig, newUrl: %s, new heartbeatConfig: %s", url, string(bytesNew))
		defer b.logger.Infof("update heartbeatConfig over")
		b.stopMonitor()
		b.m.Lock()
		b.url = url
		b.HeartbeatConfig = *heartbeatConfig
		status := new(int32)
		*status = busProxyUnhealthy
		b.status = status // 我认为心跳检测热更新不是一个太好的功能，因为可能会导致在热更新期间流量受损
		b.ch = make(chan struct{})
		b.logger = log.GetLogger().With(zap.Any("busproxy", ""), zap.Any("nodeIP", b.NodeIP), zap.Any("url", url),
			zap.Any("heartbeatConfig", *config.GetConfig().HeartbeatConfig))
		go b.startMonitor(b.ch, status)
		b.m.Unlock()
	} else {
		b.logger.Infof("config not need update")
	}
}

func (b *BusProxy) stopMonitor() {
	utils.SafeCloseChannel(b.ch)
	b.logger.Warnf("stop monitor")
	b.unhealthyCB(b.NodeIP)
}

func (b *BusProxy) startMonitor(ch chan struct{}, status *int32) {
	b.m.RLock()
	url := b.url
	heartConfig := b.HeartbeatConfig
	b.m.RUnlock()
	ticker := time.NewTicker(time.Second * time.Duration(b.HeartbeatInterval))
	defer ticker.Stop()
	count := new(int32)
	*count = 0
	for {
		select {
		case <-ch:
			b.logger.Infof("heartbeat exit")
			return
		case <-ticker.C:
			go b.doHeartBeat(url, time.Duration(heartConfig.HeartbeatTimeout), count, status,
				int32(heartConfig.HeartbeatTimeoutThreshold))
		}
	}
}

func (b *BusProxy) doHeartBeat(url string, timeout time.Duration, count *int32, status *int32, threshold int32) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	req.SetRequestURI(url)
	req.Header.SetMethod(fasthttp.MethodGet)
	httputil.AddAuthorizationHeaderForFG(req)
	err := httputil.GetHeartbeatClient().DoTimeout(req, resp, time.Second*timeout)
	code := resp.StatusCode()
	if err != nil || code != statuscode.FrontendStatusOk {
		if atomic.LoadInt32(count)%busHeartBeatLogInternal == 0 {
			errMsg := fmt.Sprintf("code: %d", code)
			if err != nil {
				errMsg = fmt.Sprintf("err: %v, %s", err, errMsg)
			}
			b.logger.Warnf("heartbeat not ok, errMsg: %s, count: %d", errMsg, atomic.LoadInt32(count))
		}

		curCount := atomic.AddInt32(count, 1)
		if curCount >= threshold && atomic.LoadInt32(status) == busProxyHealthy {
			atomic.StoreInt32(status, busProxyUnhealthy)
			b.unhealthyCB(b.NodeIP)
			b.logger.Warnf("status from healthy to unhealthy")
		}
	} else {
		atomic.StoreInt32(count, 0)
		if atomic.LoadInt32(status) == busProxyUnhealthy {
			atomic.StoreInt32(status, busProxyHealthy)
			b.healthyCB(b.NodeIP)
			b.logger.Warnf("status from unhealthy to healthy")
		}
	}
}

// IsHealthy -
func (b *BusProxy) IsHealthy() bool {
	return atomic.LoadInt32(b.status) == busProxyHealthy
}
