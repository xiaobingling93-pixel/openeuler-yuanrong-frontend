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

// Package invocation -
package invocation

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/responsehandler"
	"frontend/pkg/frontend/types"
)

func TestMakeInvoker(t *testing.T) {
	adapter := &FGAdapter{}
	invoker := adapter.MakeInvoker()
	assert.NotNil(t, invoker)
}

func TestSetResponseFromInvocation(t *testing.T) {
	message := []byte(`{"billingDuration":"xxx","innerCode": "0", "body": "test"}`)
	ctx := &types.InvokeProcessContext{
		ReqHeader:  map[string]string{},
		RespHeader: map[string]string{},
	}
	responsehandler.Handler = (&FGAdapter{}).MakeResponseHandler()
	want := []byte(`"test"`)
	type args struct {
		ctx     *types.InvokeProcessContext
		message []byte
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "TestSetResponseFromInvocation",
			args: args{
				ctx:     ctx,
				message: message,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			responsehandler.SetResponseInContext(tt.args.ctx, tt.args.message)
			assert.Equal(t, tt.args.ctx.RespBody, want)
		})
	}
	convey.Convey("TestSetResponseFromInvocation", t, func() {
		convey.Convey("do nothing and return", func() {
			ctx = &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			ctx.RespHeader["123"] = "zhangsan"
			responsehandler.SetResponseInContext(ctx, []byte("message"))
			convey.So(len(ctx.RespBody), convey.ShouldEqual, 0)
		})
		convey.Convey("failed to unmarshal call response data", func() {
			ctx = &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			responsehandler.SetResponseInContext(ctx, []byte("message"))
			convey.So(len(ctx.RespBody), convey.ShouldEqual, 0)
		})
		convey.Convey("failed to get the innerCode", func() {
			ctx = &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			responsehandler.SetResponseInContext(ctx, []byte(`{"billingDuration":"xxx","innerCode": "0xx", "body": "test"}`))
			convey.So(len(ctx.RespBody), convey.ShouldEqual, 0)
		})
	})
}

func TestHandleInvokeError(t *testing.T) {
	responsehandler.Handler = (&FGAdapter{}).MakeResponseHandler()
	convey.Convey("HandleInvokeError", t, func() {
		convey.Convey("false SetResponseFromFrontend", func() {
			err := errors.New("failed to create, code: \"4\", message: {\"errorCode\":\"6001\",\"message\":\"initError\"}")
			ctx := &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			httputil.HandleInvokeError(ctx, err)
			convey.So(ctx.StatusCode, convey.ShouldEqual, fasthttp.StatusInternalServerError)
		})
		convey.Convey("true", func() {
			err := errors.New(`failed to finish the request error failed to create, code: 2002, message: {"errorCode":"4211","message":"init failed bootstrap timed out after 5s"}`)
			ctx := &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			httputil.HandleInvokeError(ctx, err)
			convey.So(ctx.StatusCode, convey.ShouldEqual, fasthttp.StatusInternalServerError)
		})
		convey.Convey("temp true", func() {
			err := errors.New(`failed to finish the request error failed to create, code: 2002, message: [{"errorCode":"4211","message":"init failed bootstrap timed out after 5s"}]`)
			ctx := &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			httputil.HandleInvokeError(ctx, err)
			convey.So(ctx.StatusCode, convey.ShouldEqual, fasthttp.StatusInternalServerError)
		})
	})
}

func TestHandleCreateInstanceError(t *testing.T) {
	responsehandler.Handler = (&FGAdapter{}).MakeResponseHandler()
	convey.Convey("HandleCreateInstanceError", t, func() {
		convey.Convey("failed to handle create error", func() {
			ctx := &types.InvokeProcessContext{
				ReqHeader:  map[string]string{},
				RespHeader: map[string]string{},
			}
			err := snerror.New(4201, "")
			instanceError := httputil.HandleCreateInstanceError(ctx, err)
			convey.So(instanceError, convey.ShouldBeTrue)
			convey.So(ctx.StatusCode, convey.ShouldEqual, http.StatusInternalServerError)

			err = snerror.New(statuscode.InternalErrorCode, `[{"errorCode": "wrong code", "message":"failed to init function"}]`)
			httputil.HandleCreateInstanceError(ctx, err)
			convey.So(instanceError, convey.ShouldBeTrue)
			convey.So(ctx.StatusCode, convey.ShouldEqual, http.StatusInternalServerError)
		})
	})
}

func TestSetResponseFromFrontend(t *testing.T) {
	responsehandler.Handler = (&FGAdapter{}).MakeResponseHandler()
	convey.Convey("SetResponseFromFrontend", t, func() {
		defer gomonkey.ApplyFunc(json.Marshal, func(v any) ([]byte, error) {
			return nil, errors.New("marshal error")
		}).Reset()
		ctx := &types.InvokeProcessContext{
			ReqHeader:  map[string]string{},
			RespHeader: map[string]string{},
		}
		responsehandler.SetErrorInContext(ctx, 4211, "failed")
		convey.So(ctx.RespBody, convey.ShouldBeNil)
	})
}
