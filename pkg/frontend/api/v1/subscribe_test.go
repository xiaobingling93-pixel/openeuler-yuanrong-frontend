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

// Package v1 -
package v1

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"

	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/frontend/common/httpconstant"
	"frontend/pkg/frontend/common/util"
	"frontend/pkg/frontend/invocation"
	"frontend/pkg/frontend/middleware"
	"frontend/pkg/frontend/responsehandler"
)

func constructFakeSubscribeRequest(streamName string, timeoutMs string, expectReceiveNum string) (*gin.Context, *httptest.ResponseRecorder) {
	rw := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rw)
	ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer([]byte("")))
	ctx.Request.Header.Set(httpconstant.HeaderStreamName, streamName)
	ctx.Request.Header.Set(httpconstant.HeaderTimeoutMs, timeoutMs)
	ctx.Request.Header.Set(httpconstant.HeaderExpectNum, expectReceiveNum)
	return ctx, rw
}

func Test_Subscribe(t *testing.T) {
	patches := []*gomonkey.Patches{
		gomonkey.ApplyFunc(util.NewClient, func() util.Client {
			return &fakeClient{}
		}),
		gomonkey.ApplyFunc(datasystemclient.SubscribeStream, func(param datasystemclient.SubscribeParam,
			ctx datasystemclient.StreamCtx) error {
			t.Logf("MockSubscribeStream")
			return nil
		}),
	}
	defer func() {
		for _, patch := range patches {
			patch.Reset()
		}
	}()
	fgAdapter := &invocation.FGAdapter{}
	responsehandler.Handler = fgAdapter.MakeResponseHandler()
	middleware.Invoker = fgAdapter.MakeInvoker()
	convey.Convey("subscribe success", t, func() {
		ctx, rw := constructFakeSubscribeRequest("test_stream_name", "100", "1")
		SubscribeHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
	})

	convey.Convey("subscribe failed by invalid stream name", t, func() {
		ctx, rw := constructFakeSubscribeRequest("", "100", "1")
		SubscribeHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	convey.Convey("subscribe failed by invalid timoutMs", t, func() {
		ctx, rw := constructFakeSubscribeRequest("test_stream_name", "", "1")
		SubscribeHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	convey.Convey("subscribe failed by invalid expectReceiveNum", t, func() {
		ctx, rw := constructFakeSubscribeRequest("test_stream_name", "100", "-1")
		SubscribeHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	convey.Convey("subscribe failed by invalid timeout", t, func() {
		ctx, rw := constructFakeSubscribeRequest("test_stream_name", "-1", "1")
		SubscribeHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}
