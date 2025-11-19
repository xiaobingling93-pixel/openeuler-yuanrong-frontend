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

// Package register -
package selfregister

import (
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

func TestPrepareKey(t *testing.T) {
	prepareEnv()
	defer cleanEnv()
	defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
		return &types.Config{
			ClusterID: "cluster1",
		}
	}).Reset()
	convey.Convey("test prepareKey", t, func() {
		convey.Convey("getInstanceKey ok", func() {
			_ = os.Setenv("HOST_IP", "127.0.0.1")
			key, err := getInstanceKey()
			_ = os.Setenv("HOST_IP", "")
			convey.So(err, convey.ShouldBeNil)
			convey.So(key, convey.ShouldEqual, "/sn/frontend/instances/127.0.0.1/frontend_****")
		})
		convey.Convey("getInstanceKeyWithClusterID ok", func() {
			key, err := getInstanceKeyWithClusterID()
			convey.So(err, convey.ShouldBeNil)
			convey.So(key, convey.ShouldEqual, "/sn/frontend/instances/cluster1/127.0.0.1/frontend_****")
		})
		convey.Convey("validate env", func() {
			err := validateEnvs("", "podname")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldEqual, "NODE_IP env not found")
			err = validateEnvs("ip", "")
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldEqual, "POD_NAME env not found")
		})
		convey.Convey("getInstanceKeyWithClusterID cluster in evn", func() {
			defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
				return &types.Config{
					ClusterID: "",
				}
			}).Reset()
			_ = os.Setenv("CLUSTER_ID", "cluster1")
			key, err := getInstanceKeyWithClusterID()
			convey.So(err, convey.ShouldBeNil)
			convey.So(key, convey.ShouldEqual, "/sn/frontend/instances/cluster1/127.0.0.1/frontend_****")
		})
		convey.Convey("getInstanceKeyWithClusterID get cluster failed", func() {
			defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
				return &types.Config{
					ClusterID: "",
				}
			}).Reset()
			_ = os.Setenv("CLUSTER_ID", "")
			_, err := getInstanceKeyWithClusterID()
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(err.Error(), convey.ShouldEqual, "get cluster failed")
		})
	})
}

func prepareEnv() {
	_ = os.Setenv("NODE_IP", "127.0.0.1")
	_ = os.Setenv("POD_NAME", "frontend_****")
}

func cleanEnv() {
	_ = os.Setenv("NODE_IP", "")
	_ = os.Setenv("POD_NAME", "")
}
