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

package types

import (
	"io"
	"net/http"
	"sync"
	"sync/atomic"
)

// StreamInvokeInfo -
type StreamInvokeInfo struct {
	RequestStreamName      string
	ResponseStreamName     string
	ReqStream              io.ReadCloser
	RspStream              http.ResponseWriter
	RequestStreamErrorCode int32
	ResponseStopChan       *StreamStopChan
}

// SetRequestStreamErrorCode -
func (r *StreamInvokeInfo) SetRequestStreamErrorCode(errorCode int32) {
	atomic.StoreInt32(&r.RequestStreamErrorCode, errorCode)
}

// GetRequestStreamErrorCode -
func (r *StreamInvokeInfo) GetRequestStreamErrorCode() int32 {
	return atomic.LoadInt32(&r.RequestStreamErrorCode)
}

// StreamStopChan -
type StreamStopChan struct {
	C    chan struct{}
	once sync.Once
}

// SafeClose -
func (mc *StreamStopChan) SafeClose() {
	mc.once.Do(func() {
		close(mc.C)
	})
}
