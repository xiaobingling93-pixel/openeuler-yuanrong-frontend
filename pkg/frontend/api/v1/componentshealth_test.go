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

package v1

import (
	"encoding/json"
	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"
	"testing"
)

func TestComponentsHealthHandler(t *testing.T) {
	convey.Convey("ComponentsHealthHandler", t, func() {
		ctx := &fasthttp.RequestCtx{}
		ComponentsHealthHandler(ctx)
		convey.So(ctx.Response.StatusCode(), convey.ShouldEqual, fasthttp.StatusOK)
		resultMap := make(map[string]string)
		err := json.Unmarshal(ctx.Response.Body(), &resultMap)
		convey.So(err, convey.ShouldBeNil)
		convey.So(resultMap["functiontask"], convey.ShouldEqual, "healthy")
		convey.So(resultMap["instancemanager"], convey.ShouldEqual, "healthy")
		convey.So(resultMap["functionaccessor"], convey.ShouldEqual, "healthy")
	})
}
