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

import (
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestIsEmptyString(t *testing.T) {
	convey.Convey("test something is Empty String", t, func() {
		x := ""
		convey.So(isEmptyString(x), convey.ShouldEqual, true)
	})

	convey.Convey("test something is not Empty String", t, func() {
		x := "some"
		convey.So(isEmptyString(x), convey.ShouldEqual, false)
	})
}

func TestTransformToNumber(t *testing.T) {
	convey.Convey("test something is positive num string", t, func() {
		x := "1"
		isNum, res := transformToNumber(x)
		convey.So(isNum, convey.ShouldEqual, true)
		convey.So(res, convey.ShouldEqual, 1)
	})

	convey.Convey("test something is empty string", t, func() {
		x := ""
		isNum, res := transformToNumber(x)
		convey.So(isNum, convey.ShouldEqual, false)
		convey.So(res, convey.ShouldEqual, -1)
	})

	convey.Convey("test something is not num string", t, func() {
		x := "a"
		isNum, res := transformToNumber(x)
		convey.So(isNum, convey.ShouldEqual, false)
		convey.So(res, convey.ShouldEqual, -1)
	})

	convey.Convey("test something is negative num string", t, func() {
		x := "-2"
		isNum, res := transformToNumber(x)
		convey.So(isNum, convey.ShouldEqual, false)
		convey.So(res, convey.ShouldEqual, -2)
	})
}
