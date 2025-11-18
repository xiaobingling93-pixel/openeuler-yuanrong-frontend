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
	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/frontend/types"
)

// InvokeProcessContext -
type InvokeProcessContext = types.InvokeProcessContext

// Config -
type Config = datasystemclient.Config

// SetParam -
type SetParam = api.SetParam

// WriteModeEnum -
type WriteModeEnum = api.WriteModeEnum

// RequestTraceInfo -
type RequestTraceInfo = types.RequestTraceInfo

// StreamCtx -
type StreamCtx = datasystemclient.StreamCtx

// FastHttpCtxAdapter -
type FastHttpCtxAdapter = datasystemclient.FastHttpCtxAdapter

// SubscribeParam -
type SubscribeParam = datasystemclient.SubscribeParam
