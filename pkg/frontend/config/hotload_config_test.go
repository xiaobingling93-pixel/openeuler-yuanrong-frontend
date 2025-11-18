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

package config

import (
	"encoding/json"
	"fmt"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/monitor"
	commonType "frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/types"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"io/ioutil"
	"testing"
)

const (
	serverPort              = "8888"
	defaultMetaEtcdCafile   = "/home/sn/resource/ca/ca.pem"
	defaultMetaEtcdCertfile = "/home/sn/resource/ca/cert.pem"
	defaultMetaEtcdKeyfile  = "/home/sn/resource/ca/key.pem"

	defaultRouterEtcdCafile   = "/home/sn/resource/routerEtcd/ca.pem"
	defaultRouterEtcdCertfile = "/home/sn/resource/routerEtcd/cert.pem"
	defaultRouterEtcdKeyfile  = "/home/sn/resource/routerEtcd/key.pem"
)

var (
	watcher    *monitor.MockFileWatcher
	maxTimeout = 100*24*3600 + 1
	testConfig = &types.Config{
		CPU:      5,
		Memory:   100,
		SLAQuota: 10,
		HTTPConfig: &types.FrontendHTTP{
			RespTimeOut:               int64(maxTimeout),
			WorkerInstanceReadTimeOut: int64(maxTimeout),
			MaxRequestBodySize:        6,
		},
		MetaEtcd: etcd3.EtcdConfig{
			Servers:   []string{"127.0.0.1:2379"},
			User:      "root",
			Password:  "0000",
			CaFile:    defaultMetaEtcdCafile,
			CertFile:  defaultMetaEtcdCertfile,
			KeyFile:   defaultMetaEtcdKeyfile,
			SslEnable: true,
		},
		CAEMetaEtcd: etcd3.EtcdConfig{
			Servers:   []string{"127.0.0.1:2379"},
			User:      "root",
			Password:  "00001",
			CaFile:    defaultMetaEtcdCafile,
			CertFile:  defaultMetaEtcdCertfile,
			KeyFile:   defaultMetaEtcdKeyfile,
			SslEnable: true,
		},
		RouterEtcd: etcd3.EtcdConfig{
			Servers:   []string{"127.0.0.2:2379"},
			User:      "root",
			Password:  "1111",
			CaFile:    defaultRouterEtcdCafile,
			CertFile:  defaultRouterEtcdCertfile,
			KeyFile:   defaultRouterEtcdKeyfile,
			SslEnable: true,
		},
		MemoryControlConfig: &commonType.MemoryControlConfig{
			LowerMemoryPercent:  0.5,
			BodyThreshold:       10,
			HighMemoryPercent:   0.7,
			MemDetectIntervalMs: 100,
		},
		MemoryEvaluatorConfig: &types.MemoryEvaluatorConfig{
			RequestMemoryEvaluator: 2,
		},
		EtcdLeaseConfig: &types.EtcdLeaseConfig{
			LeaseTTL: 10,
			RenewTTL: 10,
		},
	}

	testConfig2 = &types.Config{
		CPU:      5,
		Memory:   100,
		SLAQuota: 10,
		HTTPConfig: &types.FrontendHTTP{
			RespTimeOut:               int64(maxTimeout),
			WorkerInstanceReadTimeOut: int64(maxTimeout),
			MaxRequestBodySize:        6,
		},
		MetaEtcd: etcd3.EtcdConfig{
			SslEnable: true,
		},
		CAEMetaEtcd: etcd3.EtcdConfig{
			SslEnable: true,
		},
		RouterEtcd: etcd3.EtcdConfig{
			SslEnable: true,
		},
	}
)

func createMockFileWatcher(stopCh <-chan struct{}) (monitor.FileWatcher, error) {
	watcher = &monitor.MockFileWatcher{
		Callbacks: map[string]monitor.FileChangedCallback{},
		StopCh:    stopCh,
		EventChan: make(chan string, 1),
	}
	return watcher, nil
}

func TestWatchConfig(t *testing.T) {
	convey.Convey("TestWatchConfig error", t, func() {
		defer gomonkey.ApplyFunc(monitor.CreateFileWatcher, func(stopCh <-chan struct{}) (monitor.FileWatcher, error) {
			return nil, fmt.Errorf("ioutil.ReadFile error")
		}).Reset()
		stopCh := make(chan struct{}, 1)
		err := WatchConfig(ConfigFilePath, stopCh, nil)
		if err != nil {
			return
		}
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestHotLoadConfig(t *testing.T) {
	convey.Convey("TestHotLoadConfig OK", t, func() {
		patches := gomonkey.NewPatches()
		data, _ := json.Marshal(testConfig)
		patches.ApplyFunc(ioutil.ReadFile, func() ([]byte, error) {
			fmt.Println(string(data))
			return data, nil
		})
		defer func() {
			patches.Reset()
		}()

		SetConfig(*testConfig)
		initDefaultMemoryControlConfig()
		initDefaultMemoryEvaluatorConfig()

		monitor.SetCreator(createMockFileWatcher)

		stopCh := make(chan struct{}, 1)
		callbackChan := make(chan int, 1)
		WatchConfig(ConfigFilePath, stopCh, func() {
			callbackChan <- 1
			fmt.Println("do call back")
		})

		watcher.EventChan <- ConfigFilePath

		<-callbackChan
		convey.So(GetConfig().MemoryControlConfig.LowerMemoryPercent, convey.ShouldEqual,
			testConfig.MemoryControlConfig.LowerMemoryPercent)
		convey.So(GetConfig().MemoryControlConfig.BodyThreshold, convey.ShouldEqual,
			testConfig.MemoryControlConfig.BodyThreshold)
		convey.So(GetConfig().MemoryControlConfig.HighMemoryPercent, convey.ShouldEqual,
			testConfig.MemoryControlConfig.HighMemoryPercent)
		convey.So(GetConfig().MemoryControlConfig.MemDetectIntervalMs, convey.ShouldEqual,
			testConfig.MemoryControlConfig.MemDetectIntervalMs)
		convey.So(GetConfig().MemoryEvaluatorConfig.RequestMemoryEvaluator, convey.ShouldEqual,
			testConfig.MemoryEvaluatorConfig.RequestMemoryEvaluator)

		convey.So(GetConfig().MetaEtcd.SslEnable, convey.ShouldEqual,
			testConfig.MetaEtcd.SslEnable)
		convey.So(GetConfig().CAEMetaEtcd.SslEnable, convey.ShouldEqual,
			testConfig.CAEMetaEtcd.SslEnable)
		convey.So(GetConfig().RouterEtcd.SslEnable, convey.ShouldEqual,
			testConfig.RouterEtcd.SslEnable)
		close(stopCh)
	})

	convey.Convey("TestHotLoadConfig OK 2", t, func() {
		patches := gomonkey.NewPatches()
		data, _ := json.Marshal(testConfig2)
		patches.ApplyFunc(ioutil.ReadFile, func() ([]byte, error) {
			fmt.Println(string(data))
			return data, nil
		})
		defer func() {
			patches.Reset()
		}()

		SetConfig(*testConfig)
		initDefaultMemoryControlConfig()
		initDefaultMemoryEvaluatorConfig()

		monitor.SetCreator(createMockFileWatcher)

		stopCh := make(chan struct{}, 1)
		callbackChan := make(chan int, 1)
		WatchConfig(ConfigFilePath, stopCh, func() {
			callbackChan <- 1
			fmt.Println("do call back")
		})

		watcher.EventChan <- ConfigFilePath

		<-callbackChan
		convey.So(GetConfig().MemoryControlConfig.LowerMemoryPercent, convey.ShouldEqual,
			testConfig.MemoryControlConfig.LowerMemoryPercent)
		convey.So(GetConfig().MemoryControlConfig.BodyThreshold, convey.ShouldEqual,
			testConfig.MemoryControlConfig.BodyThreshold)
		convey.So(GetConfig().MemoryControlConfig.HighMemoryPercent, convey.ShouldEqual,
			testConfig.MemoryControlConfig.HighMemoryPercent)
		convey.So(GetConfig().MemoryControlConfig.MemDetectIntervalMs, convey.ShouldEqual,
			testConfig.MemoryControlConfig.MemDetectIntervalMs)
		convey.So(GetConfig().MemoryEvaluatorConfig.RequestMemoryEvaluator, convey.ShouldEqual,
			testConfig.MemoryEvaluatorConfig.RequestMemoryEvaluator)

		convey.So(GetConfig().MetaEtcd.SslEnable, convey.ShouldEqual,
			testConfig.MetaEtcd.SslEnable)
		convey.So(GetConfig().CAEMetaEtcd.SslEnable, convey.ShouldEqual,
			testConfig.CAEMetaEtcd.SslEnable)
		convey.So(GetConfig().RouterEtcd.SslEnable, convey.ShouldEqual,
			testConfig.RouterEtcd.SslEnable)
		close(stopCh)
	})
}

func TestLoadConfig(t *testing.T) {
	convey.Convey("TestLoadConfig error 0", t, func() {
		defer gomonkey.ApplyFunc(ioutil.ReadFile, func() ([]byte, error) {
			return nil, fmt.Errorf("ioutil.ReadFile error")
		}).Reset()
		_, err := loadConfig(ConfigFilePath)
		convey.So(err, convey.ShouldNotBeNil)
	})

	convey.Convey("TestLoadConfig error 1", t, func() {
		patches := gomonkey.NewPatches()
		data, _ := json.Marshal(testConfig)
		patches.ApplyFunc(ioutil.ReadFile, func() ([]byte, error) {
			fmt.Println(string(data))
			return data, nil
		})
		patches.ApplyFunc(json.Unmarshal, func(data []byte, v any) error {
			return fmt.Errorf("json.Unmarshal error")
		})
		defer func() {
			patches.Reset()
		}()
		_, err := loadConfig(ConfigFilePath)
		convey.So(err, convey.ShouldNotBeNil)
	})

	convey.Convey("TestLoadConfig error 2", t, func() {
		patches := gomonkey.NewPatches()
		data, _ := json.Marshal(testConfig)
		patches.ApplyFunc(ioutil.ReadFile, func() ([]byte, error) {
			fmt.Println(string(data))
			return data, nil
		})
		patches.ApplyFunc(loadFunctionConfig, func(config *types.Config) error {
			return fmt.Errorf("loadFunctionConfig error")
		})
		defer func() {
			patches.Reset()
		}()
		_, err := loadConfig(ConfigFilePath)
		convey.So(err, convey.ShouldNotBeNil)
	})
}

func TestUpdateMemoryControlConfig(t *testing.T) {
	convey.Convey("TestUpdateMemoryControlConfig", t, func() {
		updateMemoryControlConfig(nil, nil)
		oldConfig := &commonType.MemoryControlConfig{}
		newConfig := &commonType.MemoryControlConfig{
			LowerMemoryPercent:     0.6,
			StatefulHighMemPercent: 0.85,
		}
		updateMemoryControlConfig(newConfig, oldConfig)
		convey.So(oldConfig.LowerMemoryPercent, convey.ShouldEqual, newConfig.LowerMemoryPercent)
		convey.So(oldConfig.StatefulHighMemPercent, convey.ShouldEqual, newConfig.StatefulHighMemPercent)
	})
	convey.Convey("TestUpdateMemoryControlConfig 2", t, func() {
		oldConfig := &commonType.MemoryControlConfig{}
		newConfig := &commonType.MemoryControlConfig{
			LowerMemoryPercent:     0.6,
			HighMemoryPercent:      0.8,
			StatefulHighMemPercent: 0.85,
			MemDetectIntervalMs:    1,
			BodyThreshold:          2,
		}
		updateMemoryControlConfig(newConfig, oldConfig)
		convey.So(oldConfig.LowerMemoryPercent, convey.ShouldEqual, newConfig.LowerMemoryPercent)
		convey.So(oldConfig.StatefulHighMemPercent, convey.ShouldEqual, newConfig.StatefulHighMemPercent)
	})
}
