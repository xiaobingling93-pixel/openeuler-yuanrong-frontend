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

// Package util -
package util

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"

	"frontend/pkg/common/faas_common/constant"
	commontype "frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/types"
)

func TestConvertResourceSpecs(t *testing.T) {
	type args struct {
		ctx      *types.InvokeProcessContext
		funcSpec *commontype.FuncSpec
	}
	ctx := &types.InvokeProcessContext{
		ReqHeader: map[string]string{},
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]int64
		wantErr bool
	}{
		{"case1 succeed to convert custom resource",
			args{
				ctx: ctx,
				funcSpec: &commontype.FuncSpec{
					ResourceMetaData: commontype.ResourceMetaData{CPU: 300, Memory: 128, CustomResources: "{\"nvidia.com/gpu\":8}"},
				}},
			map[string]int64{"CPU": 300, "Memory": 128, "nvidia.com/gpu": 8}, false},
		{"case2 failed to unmarshal",
			args{
				ctx: ctx,
				funcSpec: &commontype.FuncSpec{
					ResourceMetaData: commontype.ResourceMetaData{CPU: 300, Memory: 128, CustomResources: "b"},
				}},
			nil, true},
		{"case3 failed to un",
			args{
				ctx: ctx,
				funcSpec: &commontype.FuncSpec{
					ResourceMetaData: commontype.ResourceMetaData{CPU: 300, Memory: 128, CustomResources: "{\"nvidia.com/gpu\":0}"}},
			},
			map[string]int64{"CPU": 300, "Memory": 128}, false},
		{"case4 succeed to convert custom resource from header",
			args{
				ctx: &types.InvokeProcessContext{
					ReqHeader: map[string]string{constant.HeaderCPUSize: "400", constant.HeaderMemorySize: "256"},
				},
				funcSpec: &commontype.FuncSpec{
					ResourceMetaData: commontype.ResourceMetaData{CPU: 300, Memory: 128, CustomResources: "{\"nvidia.com/gpu\":8}"},
				}},
			map[string]int64{"CPU": 400, "Memory": 256, "nvidia.com/gpu": 8}, false},
		{"case5 failed to convert custom resource from header",
			args{
				ctx: &types.InvokeProcessContext{
					ReqHeader: map[string]string{constant.HeaderCPUSize: "aaa", constant.HeaderMemorySize: "bbb"},
				},
				funcSpec: &commontype.FuncSpec{
					ResourceMetaData: commontype.ResourceMetaData{CPU: 300, Memory: 128, CustomResources: "{\"nvidia.com/gpu\":8}"},
				}},
			map[string]int64{"CPU": 300, "Memory": 128, "nvidia.com/gpu": 8}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertResourceSpecs(tt.args.ctx, tt.args.funcSpec)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertResourceSpecs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertResourceSpecs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetry(t *testing.T) {
	convey.Convey("Retry", t, func() {
		retryCount := 0
		do := func() error {
			if retryCount > 2 {
				return nil
			}
			return errors.New("should retry")
		}
		ifRetry := func() bool {
			retryCount++
			if retryCount > 2 {
				return false
			}
			return true
		}
		err := Retry(do, ifRetry, 3, 500*time.Millisecond)
		convey.So(err.Error(), convey.ShouldEqual, "should retry")

		retryCount = 0
		do = func() error {
			if retryCount > 1 {
				return nil
			}
			return errors.New("should retry")
		}
		err = Retry(do, ifRetry, 3, 500*time.Millisecond)
		convey.So(err, convey.ShouldBeNil)
	})
}

func TestPeekIgnoreCase(t *testing.T) {
	tests := []struct {
		reqHeader map[string]string
		name      string
		want      string
	}{
		{map[string]string{"Content-Type": "application/json"}, "Content-Type", "application/json"},
		{map[string]string{"content-type": "application/json"}, "Content-Type", "application/json"},
		{map[string]string{"CONTENT-TYPE": "application/json"}, "content-type", "application/json"},
		{map[string]string{}, "Content-Type", ""},
		{map[string]string{"X-Custom-Header": "value"}, "Content-Type", ""},
	}

	for _, tt := range tests {
		got := PeekIgnoreCase(tt.reqHeader, tt.name)
		if got != tt.want {
			t.Errorf("PeekIgnoreCase(%v, %v) = %v, want %v", tt.reqHeader, tt.name, got, tt.want)
		}
	}
}
