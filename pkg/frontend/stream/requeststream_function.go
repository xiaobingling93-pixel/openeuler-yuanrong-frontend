//go:build function

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

package stream

import "frontend/pkg/frontend/types"

// BuildStreamContext -
func BuildStreamContext(ctx interface{}, processCtx *types.InvokeProcessContext) {
	processCtx.StreamInfo = &types.StreamInvokeInfo{}
}

// HTTPStreamInvokeHandler -
func HTTPStreamInvokeHandler(ctx interface{}, timeout interface{}) error {
	return nil
}

// IsHTTPUploadStream -
func IsHTTPUploadStream(r interface{}) bool {
	return false
}
