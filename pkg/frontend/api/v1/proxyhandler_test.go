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

	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/common"
	"frontend/pkg/frontend/functionmeta"
)

func TestProxyHandler(t *testing.T) {
	convey.Convey("invoke path not found", t, func() {
		defer gomonkey.ApplyFunc(functionmeta.LoadFuncSpecWithPath,
			func(path string, traceID string) (*types.FuncSpec, bool) {
				return &types.FuncSpec{}, false
			}).Reset()
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("POST", "/hello", bytes.NewBuffer([]byte("hello")))
		ProxyHandler(ctx)
		convey.So(rw.Body.String(), convey.ShouldEqual, "404 page not found")
		convey.So(rw.Code, convey.ShouldEqual, http.StatusNotFound)
	})
	convey.Convey("proxy success", t, func() {
		defer gomonkey.ApplyFunc(functionmeta.LoadFuncSpecWithPath,
			func(path string, traceID string) (*types.FuncSpec, bool) {
				return &types.FuncSpec{
					FuncMetaData: types.FuncMetaData{
						Name:               "testFunc",
						FunctionVersionURN: "123/testFunc/latest",
					},
				}, true
			}).Reset()
		defer gomonkey.ApplyFunc(InvokeHandler, func(ctx *gin.Context) {
			ctx.Writer.WriteHeader(http.StatusOK)
			ctx.String(http.StatusOK, "ok")
		}).Reset()
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("POST", "/hello", bytes.NewBuffer([]byte("hello")))
		ProxyHandler(ctx)
		convey.So(ctx.Param(common.FunctionUrnParam), convey.ShouldEqual, "123/testFunc/latest")
		convey.So(rw.Body.String(), convey.ShouldEqual, "ok")
		convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
	})
}
