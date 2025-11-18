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

// Package frontendsdk is sdk
package frontendsdk

import "github.com/valyala/fasthttp"

// FrontendAPI - frontend SDK api
type FrontendAPI interface {
	Init(configFilePath string) error
	InvokeHandler(ctx *InvokeProcessContext) error
	UploadWithKeyRetry(value []byte, config *Config, param SetParam, traceID string) (string, error)
	DownloadArrayRetry(keys []string, config *Config, traceID string) ([][]byte, error)
	DeleteArrayRetry(keys []string, config *Config, traceID string) ([]string, error)
	SubscribeStream(param SubscribeParam, ctx StreamCtx) error
	ExecShutdownHandler(signum int)
	CheckLocalDataSystemStatusReady() bool
	CheckFrontendIsHealth() bool
	Auth(ctx *fasthttp.RequestCtx, ak string, sk []byte) bool
}
