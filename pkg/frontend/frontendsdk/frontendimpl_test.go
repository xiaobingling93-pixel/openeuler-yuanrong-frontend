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

package frontendsdk

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/magiconair/properties"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/logger/config"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/sts/raw"
	commonType "frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/wisecloudtool"
	wisecloudtypes "frontend/pkg/common/faas_common/wisecloudtool/types"
	frontendConfig "frontend/pkg/frontend/config"
	"frontend/pkg/frontend/frontendsdk/posixsdk"
	"frontend/pkg/frontend/types"
	"frontend/pkg/frontend/watcher"
	"frontend/pkg/frontend/wisecloud"
	"yuanrong.org/kernel/runtime/libruntime/api"
	"yuanrong.org/kernel/runtime/libruntime/common"
	"yuanrong.org/kernel/runtime/libruntime/execution"
)

func TestInit(t *testing.T) {
	convey.Convey("Test frontend init", t, func() {
		frontend := NewFrontend()
		cfg := &types.Config{
			Runtime: types.RuntimeConfig{
				LogConfig: config.CoreInfo{
					Level: "DEBUG",
				},
				SystemAuthConfig: types.SystemAuthConfig{
					Enable:    true,
					AccessKey: "ak",
					SecretKey: "sk",
				},
			},
			RawStsConfig: raw.StsConfig{
				StsEnable: true,
			},
		}
		data, _ := json.Marshal(cfg)
		convey.Convey("config file is empty", func() {
			err := frontend.Init("")
			convey.So(err.Error(), convey.ShouldContainSubstring, "config file path is empty")
		})
		convey.Convey("read config file error", func() {
			err := frontend.Init("/home/sn/config.json")
			convey.So(err.Error(), convey.ShouldContainSubstring, "read config failed")
		})
		convey.Convey("Unmarshal config error", func() {
			defer gomonkey.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
				return []byte{}, nil
			}).Reset()
			err := frontend.Init("/home/sn/config.json")
			convey.So(err.Error(), convey.ShouldContainSubstring, "unmarshal config failed")
		})
		convey.Convey("sts init failed", func() {
			defer gomonkey.ApplyFunc(os.ReadFile, func(name string) ([]byte, error) {
				return data, nil
			}).Reset()
			err := frontend.Init("/home/sn/config.json")
			convey.So(err.Error(), convey.ShouldContainSubstring, "failed to init sts sdk")
		})
	})
}

func TestInitSDKHandler(t *testing.T) {
	convey.Convey("Test initSDKHandler", t, func() {
		args := []api.Arg{{
			Data: []byte("hello"),
		}}
		convey.Convey("init frontend config fail", func() {
			defer gomonkey.ApplyFunc(frontendConfig.InitFunctionConfig, func(data []byte) error {
				return errors.New("failed to parse the config data")
			}).Reset()
			_, err := initSDKHandler(args, nil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "failed to parse the config data")
		})

		convey.Convey("failed to init etcd", func() {
			defer gomonkey.ApplyFunc(frontendConfig.InitFunctionConfig, func(data []byte) error {
				return nil
			}).Reset()
			defer gomonkey.ApplyFunc(frontendConfig.InitEtcd, func(stopCh <-chan struct{}) error {
				return errors.New("failed to init etcd")
			}).Reset()
			_, err := initSDKHandler(args, nil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "failed to init etcd")
		})

		convey.Convey("failed to watch etcd", func() {
			defer gomonkey.ApplyFunc(frontendConfig.InitFunctionConfig, func(data []byte) error {
				return nil
			}).Reset()
			defer gomonkey.ApplyFunc(frontendConfig.InitEtcd, func(stopCh <-chan struct{}) error {
				return nil
			}).Reset()
			defer gomonkey.ApplyFunc(watcher.StartWatch, func(stopCh <-chan struct{}) error {
				return errors.New("failed to watch etcd")
			}).Reset()
			_, err := initSDKHandler(args, nil)
			convey.So(err.Error(), convey.ShouldContainSubstring, "failed to watch etcd")
		})

		convey.Convey("success", func() {
			defer gomonkey.ApplyFunc(frontendConfig.InitFunctionConfig, func(data []byte) error {
				return nil
			}).Reset()
			defer gomonkey.ApplyFunc(frontendConfig.InitEtcd, func(stopCh <-chan struct{}) error {
				return nil
			}).Reset()
			defer gomonkey.ApplyFunc(watcher.StartWatch, func(stopCh <-chan struct{}) error {
				return nil
			}).Reset()
			defer gomonkey.ApplyFunc(datasystemclient.InitDataSystemLibruntime, func(cfg *commonType.DataSystemConfig,
				rt api.LibruntimeAPI, stopCh <-chan struct{}) {
			}).Reset()
			defer gomonkey.ApplyFunc(wisecloud.NewColdStartProvider, func(serviceAccountJwt *wisecloudtypes.ServiceAccountJwt) *wisecloudtool.PodOperator {
				return nil
			}).Reset()
			_, err := initSDKHandler(args, nil)
			convey.So(err, convey.ShouldBeNil)
		})
	})
}

func TestSetEnvError(t *testing.T) {
	convey.Convey("Test SetEnv Error", t, func() {
		cfg := &types.Config{
			Runtime: types.RuntimeConfig{
				LogConfig: config.CoreInfo{
					Level: "INFO",
				},
			},
		}
		configFilePath := "/home/sn/config"
		convey.Convey("json Marshal error ", func() {
			defer gomonkey.ApplyFunc(json.Marshal, func(v interface{}) ([]byte, error) {
				return nil, errors.New("json marshal error")
			}).Reset()
			err := setEnv(configFilePath, cfg)
			convey.So(err.Error(), convey.ShouldContainSubstring, "json marshal error")
		})
		convey.Convey("set logConfigKey error ", func() {
			defer gomonkey.ApplyFunc(os.Setenv, func(key, value string) error {
				if key == logConfigKey {
					return errors.New("set env logConfigKey error")
				}
				return nil
			}).Reset()
			err := setEnv(configFilePath, cfg)
			convey.So(err.Error(), convey.ShouldContainSubstring, "set env logConfigKey error")
		})
		convey.Convey("set initArgsFilePathEnvKey error ", func() {
			defer gomonkey.ApplyFunc(os.Setenv, func(key, value string) error {
				if key == initArgsFilePathEnvKey {
					return errors.New("set env initArgsFilePathEnvKey error")
				}
				return nil
			}).Reset()
			err := setEnv(configFilePath, cfg)
			convey.So(err.Error(), convey.ShouldContainSubstring, "set env initArgsFilePathEnvKey error")
		})
		convey.Convey("set dataSystemAddr error ", func() {
			defer gomonkey.ApplyFunc(os.Setenv, func(key, value string) error {
				if key == dataSystemAddr {
					return errors.New("set env dataSystemAddr error")
				}
				return nil
			}).Reset()
			err := setEnv(configFilePath, cfg)
			convey.So(err.Error(), convey.ShouldContainSubstring, "set env dataSystemAddr error")
		})
	})
}

func TestCheckLocalDataSystemStatusReady(t *testing.T) {
	convey.Convey("test check local dataSystem status ready", t, func() {
		frontend := NewFrontend()
		convey.Convey("test check local dataSystem status, when streamEnable is false", func() {
			defer gomonkey.ApplyFunc(frontendConfig.GetConfig, func() *types.Config {
				return &types.Config{
					StreamEnable: false,
				}
			}).Reset()
			result := frontend.CheckLocalDataSystemStatusReady()
			convey.So(result, convey.ShouldBeTrue)
		})

		convey.Convey("test check local dataSystem status, when it's not local dataSystem or not ready", func() {
			defer gomonkey.ApplyFunc(frontendConfig.GetConfig, func() *types.Config {
				return &types.Config{
					StreamEnable: true,
				}
			}).Reset()
			defer gomonkey.ApplyFunc(datasystemclient.IsLocalDataSystemStatusReady, func() bool {
				return false
			}).Reset()
			result := frontend.CheckLocalDataSystemStatusReady()
			convey.So(result, convey.ShouldBeFalse)
		})

		convey.Convey("test check local dataSystem status, when it's local dataSystem and ready", func() {
			defer gomonkey.ApplyFunc(frontendConfig.GetConfig, func() *types.Config {
				return &types.Config{
					StreamEnable: true,
				}
			}).Reset()
			defer gomonkey.ApplyFunc(datasystemclient.IsLocalDataSystemStatusReady, func() bool {
				return true
			}).Reset()
			result := frontend.CheckLocalDataSystemStatusReady()
			convey.So(result, convey.ShouldBeTrue)
		})
	})
}
