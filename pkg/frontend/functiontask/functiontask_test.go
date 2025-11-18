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

package functiontask

import (
	"fmt"
	"frontend/pkg/common/faas_common/statuscode"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/tls"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

func shieldGetConfig() *gomonkey.Patches {
	mockConfig := &types.Config{
		HTTPConfig: &types.FrontendHTTP{
			RespTimeOut:               0,
			WorkerInstanceReadTimeOut: 1,
			MaxRequestBodySize:        0,
		},
		HTTPSConfig: &tls.InternalHTTPSConfig{
			HTTPSEnable: false,
		},
		HeartbeatConfig: &types.HeartbeatConfig{
			HeartbeatTimeout:          3,
			HeartbeatInterval:         1,
			HeartbeatTimeoutThreshold: 2,
		},
	}
	return gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
		return mockConfig
	})
}

func clearGetProxies() {
	clearList := []string{}
	GetBusProxies().DoRange(func(nodeID, nodeIP string) bool {
		clearList = append(clearList, nodeID)
		return true
	})
	for _, nodeID := range clearList {
		GetBusProxies().Delete(nodeID)
	}
}

func TestBusProxys(t *testing.T) {
	defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(&BusProxy{}), "startMonitor", func(_ *BusProxy, _ chan struct{}, _ *int32) {
	}).Reset()
	defer shieldGetConfig().Reset()
	convey.Convey("TestBusProxys_base", t, func() {
		clearGetProxies()
		defer clearGetProxies()
		nodeID1, nodeIP1 := "1", "1.1.1.1"
		nodeID2, nodeIP2 := "2", "2.2.2.2"
		nodeID3, nodeIP3 := "3", "3.3.3.3"
		GetBusProxies().Add(nodeID1, nodeIP1)
		GetBusProxies().Add(nodeID2, nodeIP2)
		GetBusProxies().Add(nodeID3, nodeIP3)
		convey.So(GetBusProxies().GetNum(), convey.ShouldEqual, 3)

		GetBusProxies().Add(nodeID1, "4.4.4.4") // 这个添加失败
		convey.So(GetBusProxies().GetNum(), convey.ShouldEqual, 3)
		convey.So(GetBusProxies().list[nodeID1].NodeIP, convey.ShouldEqual, "1.1.1.1")

		GetBusProxies().Delete(nodeID1)
		GetBusProxies().Delete(nodeID2)
		convey.So(GetBusProxies().GetNum(), convey.ShouldEqual, 1)

		count := 0
		flag := false
		f := func(nodeID string, nodeIP string) bool {
			if nodeID == "3" {
				flag = true
			}
			count++
			return true
		}

		for i := 0; i < 100; i++ {
			count = 0
			flag = false
			GetBusProxies().DoRange(f)
			convey.So(flag, convey.ShouldBeTrue)
			convey.So(count == 1, convey.ShouldBeTrue)
		}
	})

	convey.Convey("TestBusProxys_parallel", t, func() {
		clearGetProxies()
		defer clearGetProxies()
		nodes := make([]struct {
			nodeID string
			nodeIP string
		}, 150, 150)
		for i := 0; i < 150; i++ {
			nodes[i].nodeID, nodes[i].nodeIP = strconv.Itoa(i), fmt.Sprintf("%s.%s.%s.%s", strconv.Itoa(i), strconv.Itoa(i), strconv.Itoa(i), strconv.Itoa(i))
		}

		wg := sync.WaitGroup{}
		addf := func(index int) {
			for i := index; i < index+10; i++ {
				GetBusProxies().Add(nodes[i].nodeID, nodes[i].nodeIP)
			}
			go func() {
				for i := index + 10; i < index+20; i++ {
					GetBusProxies().Add(nodes[i].nodeID, nodes[i].nodeIP)
				}
				wg.Done()
			}()
		}
		delf := func(index int) {
			for i := index; i < index+12; i++ {
				GetBusProxies().Delete(nodes[i].nodeID)
			}
			go func() {
				for i := index + 12; i < index+25; i++ {
					GetBusProxies().Delete(nodes[i].nodeID)
				}
				wg.Done()
			}()
		}

		for i := 0; i < 100; i += 20 {
			wg.Add(1)
			go addf(i)
		}
		wg.Wait()

		convey.So(GetBusProxies().GetNum(), convey.ShouldEqual, 100)
		for i := 0; i < 100; i++ {
			convey.So(GetBusProxies().list[strconv.Itoa(i)], convey.ShouldNotBeNil)
		}

		for i := 100; i < 140; i += 20 {
			wg.Add(1)
			addf(i)
		}
		for i := 0; i < 100; i += 25 {
			wg.Add(1)
			delf(i)
		}
		wg.Wait()
		convey.So(GetBusProxies().GetNum(), convey.ShouldEqual, 40)
		for i := 100; i < 140; i++ {
			convey.So(GetBusProxies().list[strconv.Itoa(i)], convey.ShouldNotBeNil)
		}
	})
}

func TestBusProxy_startMonitor(t *testing.T) {
	defer shieldGetConfig().Reset()
	convey.Convey("TestBusProxy_startMonitor", t, func() {
		clearGetProxies()
		defer clearGetProxies()
		//	var status *int32
		status := new(int32)
		*status = busProxyUnhealthy
		healthyCount := 0
		unhealthyCount := 0
		b := &BusProxy{
			ch:     make(chan struct{}),
			NodeIP: "",
			url:    "",
			status: status,
			HeartbeatConfig: types.HeartbeatConfig{
				HeartbeatTimeoutThreshold: 3,
				HeartbeatTimeout:          3,
				HeartbeatInterval:         1,
			},
			m: sync.RWMutex{},
			healthyCB: func(nodeIP string) {
				healthyCount++
				unhealthyCount = 0
			},
			unhealthyCB: func(nodeIP string) {
				healthyCount = 0
				unhealthyCount++
			},
			logger: log.GetLogger(),
		}

		count := 0
		f := func(response *fasthttp.Response) error {
			count++
			if count < 4 {
				if count%2 == 0 {
					response.SetStatusCode(fasthttp.StatusInternalServerError)
					return nil
				}
				return fmt.Errorf("error")
			}
			response.SetStatusCode(statuscode.FrontendStatusOk)
			return nil
		}
		defer gomonkey.ApplyFunc(httputil.AddAuthorizationHeaderForFG, func(req *fasthttp.Request) {
			return
		}).Reset()
		defer gomonkey.ApplyMethod(reflect.TypeOf(httputil.GetHeartbeatClient()), "DoTimeout",
			func(_ *fasthttp.Client, req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error {
				return f(resp)
			}).Reset()
		go b.startMonitor(b.ch, b.status)
		time.Sleep(3 * time.Second)
		convey.So(b.IsHealthy(), convey.ShouldBeFalse)

		time.Sleep(1*time.Second + 100*time.Millisecond)
		convey.So(healthyCount, convey.ShouldEqual, 1)
		convey.So(unhealthyCount, convey.ShouldEqual, 0)
		convey.So(b.IsHealthy(), convey.ShouldBeTrue)
		b.stopMonitor()
		convey.So(unhealthyCount, convey.ShouldEqual, 1)
		convey.So(healthyCount, convey.ShouldEqual, 0)
	})
}

func TestBusProxies_UpdateConfig(t *testing.T) {
	convey.Convey("TestBusProxies_UpdateConfig", t, func() {

		count := 0
		defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(&BusProxy{}), "startMonitor", func(_ *BusProxy, _ chan struct{}, _ *int32) {
			count++
		}).Reset()

		mockConfig := &types.Config{
			HTTPConfig: &types.FrontendHTTP{
				RespTimeOut:               0,
				WorkerInstanceReadTimeOut: 1,
				MaxRequestBodySize:        0,
			},
			HTTPSConfig: &tls.InternalHTTPSConfig{
				HTTPSEnable: false,
			},
			HeartbeatConfig: &types.HeartbeatConfig{
				HeartbeatTimeout:          3,
				HeartbeatInterval:         1,
				HeartbeatTimeoutThreshold: 2,
			},
		}
		defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
			return mockConfig
		}).Reset()

		mockCB := func(_ string) {
			return
		}
		b := newBusProxy("1.1.1.1", mockCB, mockCB)
		time.Sleep(100 * time.Millisecond)
		convey.So(count == 1, convey.ShouldBeTrue)

		b.updateConfig()
		time.Sleep(100 * time.Millisecond)
		convey.So(count == 1, convey.ShouldBeTrue)

		mockConfig.HTTPSConfig.HTTPSEnable = true
		b.updateConfig()
		time.Sleep(100 * time.Millisecond)
		convey.So(count == 2, convey.ShouldBeTrue)
		b.updateConfig()
		time.Sleep(100 * time.Millisecond)
		convey.So(count == 2, convey.ShouldBeTrue)

		mockConfig.HeartbeatConfig.HeartbeatTimeout = 6
		b.updateConfig()
		time.Sleep(100 * time.Millisecond)
		convey.So(count == 3, convey.ShouldBeTrue)
		b.stopMonitor()
	})
}

func TestBusProxies_complex(t *testing.T) {
	convey.Convey("TestBusProxies_complex", t, func() {
		defer gomonkey.ApplyFunc(httputil.AddAuthorizationHeaderForFG, func(req *fasthttp.Request) {
			return
		}).Reset()

		defer shieldGetConfig().Reset()
		clearGetProxies()
		defer clearGetProxies()

		IS1111Ok := false
		defer gomonkey.ApplyMethod(reflect.TypeOf(httputil.GetHeartbeatClient()), "DoTimeout",
			func(_ *fasthttp.Client, req *fasthttp.Request, resp *fasthttp.Response, timeout time.Duration) error {
				if strings.Contains(req.URI().String(), "1.1.1.1") {
					if IS1111Ok {
						resp.SetStatusCode(statuscode.FrontendStatusOk)
						return nil
					} else {
						resp.SetStatusCode(fasthttp.StatusInternalServerError)
						return nil
					}

				}
				resp.SetStatusCode(statuscode.FrontendStatusOk)
				return nil
			}).Reset()

		nodes := make([]struct {
			nodeID string
			nodeIP string
		}, 11, 11)
		for i := 1; i <= 10; i++ {
			nodes[i].nodeID, nodes[i].nodeIP = strconv.Itoa(i), strconv.Itoa(i)+"."+strconv.Itoa(i)+"."+strconv.Itoa(i)+"."+strconv.Itoa(i)
			GetBusProxies().Add(nodes[i].nodeID, nodes[i].nodeIP)
		}
		time.Sleep(1*time.Second + time.Millisecond*100)

		m := map[string]int{
			"2.2.2.2":     0,
			"3.3.3.3":     0,
			"4.4.4.4":     0,
			"5.5.5.5":     0,
			"6.6.6.6":     0,
			"7.7.7.7":     0,
			"8.8.8.8":     0,
			"9.9.9.9":     0,
			"10.10.10.10": 0,
		}
		for i := 0; i < 10; i++ {
			nodeIP := GetBusProxies().NextWithName("test", true)
			_, ok := m[nodeIP]
			convey.So(ok, convey.ShouldBeTrue)
			m[nodeIP]++
		}
		count1 := 0
		count2 := 0
		for k, v := range m {
			t.Logf("choose node: %s, count: %d\n", k, v)
			if v == 1 {
				count1++
			}
			if v == 2 {
				count2++
			}
			convey.So(v == 1 || v == 2, convey.ShouldBeTrue)
		}
		convey.So(count1, convey.ShouldEqual, 8)
		convey.So(count2, convey.ShouldEqual, 1)
		convey.So(GetBusProxies().list["1"], convey.ShouldNotBeNil)
		convey.So(GetBusProxies().IsBusProxyHealthy("1.1.1.1", ""), convey.ShouldBeFalse)
		IS1111Ok = true

		time.Sleep(1*time.Second + 500*time.Millisecond)
		convey.So(GetBusProxies().IsBusProxyHealthy("1.1.1.1", ""), convey.ShouldBeTrue)
		m = map[string]int{
			"1.1.1.1":     0,
			"2.2.2.2":     0,
			"3.3.3.3":     0,
			"4.4.4.4":     0,
			"5.5.5.5":     0,
			"6.6.6.6":     0,
			"7.7.7.7":     0,
			"8.8.8.8":     0,
			"9.9.9.9":     0,
			"10.10.10.10": 0,
		}
		for i := 0; i < 10; i++ {
			nodeIP := GetBusProxies().NextWithName("test", true)
			_, ok := m[nodeIP]
			convey.So(ok, convey.ShouldBeTrue)
			m[nodeIP]++
		}
		for k, v := range m {
			t.Logf("choose node: %s, count: %d\n", k, v)
			convey.So(v == 1, convey.ShouldBeTrue)
		}

		GetBusProxies().Delete(nodes[10].nodeID)
		GetBusProxies().Delete(nodes[8].nodeID)
		for k, v := range GetBusProxies().list {
			t.Logf("print proxy in map, k: %s, v: %s", k, v.NodeIP) // TODO
		}
		convey.So(len(GetBusProxies().list) == 8, convey.ShouldBeTrue)
		m = map[string]int{
			"1.1.1.1": 0,
			"2.2.2.2": 0,
			"3.3.3.3": 0,
			"4.4.4.4": 0,
			"5.5.5.5": 0,
			"6.6.6.6": 0,
			"7.7.7.7": 0,
			"9.9.9.9": 0,
		}

		for i := 0; i < 8; i++ {
			nodeIP := GetBusProxies().NextWithName("test", true)
			_, ok := m[nodeIP]
			convey.So(ok, convey.ShouldBeTrue)
			m[nodeIP]++
		}
		for k, v := range m {
			t.Logf("choose node: %s, count: %d\n", k, v)
			convey.So(v == 1, convey.ShouldBeTrue)
		}

		// 清理
		for i := 1; i <= 10; i++ {
			GetBusProxies().Delete(nodes[i].nodeID)
		}
	})
}

func TestBusProxies_UpdateConfig2(t *testing.T) {
	convey.Convey("TestBusProxies_UpdateConfig2", t, func() {
		clearGetProxies()
		defer clearGetProxies()
		m := make(map[string]struct{})
		defer gomonkey.ApplyPrivateMethod(reflect.TypeOf(&BusProxy{}), "updateConfig", func(b *BusProxy) {
			m[b.NodeIP] = struct{}{}
		}).Reset()

		mockCB := func(nodeIP string) {}
		GetBusProxies().list["1.1.1.1"] = &BusProxy{
			NodeIP:      "1.1.1.1",
			logger:      log.GetLogger(),
			unhealthyCB: mockCB,
		}
		GetBusProxies().list["2.2.2.2"] = &BusProxy{
			NodeIP:      "2.2.2.2",
			logger:      log.GetLogger(),
			unhealthyCB: mockCB,
		}
		GetBusProxies().list["3.3.3.3"] = &BusProxy{
			NodeIP:      "3.3.3.3",
			logger:      log.GetLogger(),
			unhealthyCB: mockCB,
		}
		mockConfig := &types.Config{
			HTTPConfig: &types.FrontendHTTP{
				RespTimeOut:               0,
				WorkerInstanceReadTimeOut: 1,
				MaxRequestBodySize:        0,
			},
			HTTPSConfig: &tls.InternalHTTPSConfig{
				HTTPSEnable: false,
			},
			HeartbeatConfig: &types.HeartbeatConfig{
				HeartbeatTimeout:          3,
				HeartbeatInterval:         1,
				HeartbeatTimeoutThreshold: 2,
			},
		}
		defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
			return mockConfig
		}).Reset()
		GetBusProxies().UpdateConfig()

		_, ok1 := m["1.1.1.1"]
		_, ok2 := m["2.2.2.2"]
		_, ok3 := m["3.3.3.3"]

		convey.So(len(m) == 3, convey.ShouldBeTrue)
		convey.So(ok1 && ok2 && ok3, convey.ShouldBeTrue)
	})
}
