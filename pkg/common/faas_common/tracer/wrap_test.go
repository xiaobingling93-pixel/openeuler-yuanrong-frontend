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

// Package tracer for init trace provider
package tracer

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/valyala/fasthttp"

	mockUtils "frontend/pkg/common/faas_common/utils"
)

func TestWrapGinHandler(t *testing.T) {
	type args struct {
		originHandlerFunc func(c *gin.Context)
	}
	actualMocked := false
	tests := []struct {
		name         string
		args         args
		patchesFunc  mockUtils.PatchesFunc
		expectMocked bool
	}{
		{
			name: "test success",
			args: args{
				originHandlerFunc: func(c *gin.Context) {
					fmt.Println("mock gin origin handler func")
				},
			},
			patchesFunc: func() mockUtils.PatchSlice {
				patches := mockUtils.InitPatchSlice()
				gomonkey.ApplyFunc(EnableCommonTracer, func() bool {
					actualMocked = true
					return true
				})
				return patches
			},
			expectMocked: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualMocked = false
			patches := tt.patchesFunc()
			defer patches.ResetAll()
			handlerFunc := WrapGinHandler(tt.args.originHandlerFunc)
			if handlerFunc == nil {
				t.Errorf("expect handler func is not nil")
				return
			}
			handlerFunc(&gin.Context{
				Request: &http.Request{
					URL: &url.URL{
						Path: "mockURLPath",
					},
					Header: make(http.Header),
				},
			})
			if actualMocked != tt.expectMocked {
				t.Errorf("expect %v but found %v", tt.expectMocked, actualMocked)
				return
			}
		})
	}
}

func TestWrapFastHTTPHandler(t *testing.T) {
	type args struct {
		originHandlerFunc func(ctx *fasthttp.RequestCtx)
	}
	actualMocked := false
	tests := []struct {
		name         string
		args         args
		patchesFunc  mockUtils.PatchesFunc
		expectMocked bool
	}{
		{
			name: "test success",
			args: args{
				originHandlerFunc: func(ctx *fasthttp.RequestCtx) {
					fmt.Println("mock fast http origin handler func")
				},
			},
			patchesFunc: func() mockUtils.PatchSlice {
				patches := mockUtils.InitPatchSlice()
				gomonkey.ApplyFunc(EnableCommonTracer, func() bool {
					actualMocked = true
					return true
				})
				return patches
			},
			expectMocked: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualMocked = false
			patches := tt.patchesFunc()
			defer patches.ResetAll()
			handlerFunc := WrapFastHTTPHandler(tt.args.originHandlerFunc)
			if handlerFunc == nil {
				t.Errorf("expect handler func is not nil")
				return
			}
			handlerFunc(&fasthttp.RequestCtx{})
			if actualMocked != tt.expectMocked {
				t.Errorf("expect %v but found %v", tt.expectMocked, actualMocked)
				return
			}
		})
	}
}
