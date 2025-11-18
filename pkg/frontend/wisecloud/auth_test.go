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

package wisecloud

import (
	"testing"
	"time"

	"github.com/smartystreets/goconvey/convey"
	"github.com/valyala/fasthttp"
)

func TestSign(t *testing.T) {
	convey.Convey("test authorization format", t, func() {
		tests := []struct {
			authorizaton string
			timestamp    string
			ak           string
			signature    string
			parseOk      bool
		}{
			{
				authorizaton: "HMAC-SHA256 timestamp=20250829T091626Z,access_key=access_key,signature=d334c14fd0d493c1f8d4490ff62d409ac8e9c4ffd35d4d96f5656aa02922c1a8",
				timestamp:    "20250829T091626Z",
				ak:           "access_key",
				signature:    "d334c14fd0d493c1f8d4490ff62d409ac8e9c4ffd35d4d96f5656aa02922c1a8",
				parseOk:      true,
			},
			{
				authorizaton: "HMAC-SHA25",
				parseOk:      false,
			},
			{
				authorizaton: "HMAC-SHA257 timestamp=20250829T091626Z,access_key=access_key,signature=d334c14fd0d493c1f8d4490ff62d409ac8e9c4ffd35d4d96f5656aa02922c1a8",
				parseOk:      false,
			},
			{
				authorizaton: "HMAC-SHA256 timestamp20250829T091626Z,access_keyaccess_key,signatured334c14fd0d493c1f8d4490ff62d409ac8e9c4ffd35d4d96f5656aa02922c1a8",
				parseOk:      false,
			},
		}

		for _, tt := range tests {
			info, ok := parseAuthorization(tt.authorizaton)
			convey.So(ok, convey.ShouldEqual, tt.parseOk)
			if !ok {
				convey.So(info, convey.ShouldBeNil)
			} else {
				convey.So(info.timeStamp, convey.ShouldEqual, tt.timestamp)
				convey.So(info.ak, convey.ShouldEqual, tt.ak)
				convey.So(info.signature, convey.ShouldEqual, tt.signature)
			}
		}
	})

	ctx := &fasthttp.RequestCtx{
		Request:  fasthttp.Request{},
		Response: fasthttp.Response{},
	}
	ctx.Request.SetRequestURI("http://127.0.0.1:8080/serverless/v2/functions/wisefunction:cn:iot:8d86c63b22e24d9ab650878b75408ea6:function:0@faas@python:latest/invocations")
	ctx.Request.Header.Set("Authorization", "HMAC-SHA256 timestamp=20250829T091626Z,access_key=access_key,signature=edc450dcccdc7f46e701dcbc03409d4c465915ef42cb8da51f8afbfc25818cd6")
	ctx.Request.SetBody([]byte("123"))
	ak := "access_key"
	sk := []byte("secret_key")
	url := "/serverless/v2/functions/wisefunction:cn:iot:8d86c63b22e24d9ab650878b75408ea6:function:0@faas@python:latest/invocations"
	convey.Convey("test buildDigest", t, func() {
		expectDigest := "/serverless/v2/functions/wisefunction:cn:iot:8d86c63b22e24d9ab650878b75408ea6:function:0@faas@python:latest/invocations\n" +
			"X-Timestamp: 20250830T034742Z\n" +
			"X-Access-Key: access_key\n" +
			"123"

		convey.So(expectDigest, convey.ShouldEqual, string(buildDigest(url, "20250830T034742Z", []byte("123"), "access_key")))
	})

	convey.Convey("Test signature", t, func() {
		signature := encodeHex(generateSignature(url, "20250830T034742Z", ctx.Request.Body(), ak, sk))
		convey.So(signature, convey.ShouldEqual, "050e8457b82a6c366ff800827df1a75c58fafa9f9319c99b01c4b23a6d9cbd09")
	})

	convey.Convey("test timeout", t, func() {
		tests := []struct {
			timeStamp string
			ok        bool
		}{
			{
				timeStamp: time.Now().Add(-6 * time.Minute).UTC().Format("20060102T150405Z"),
				ok:        false,
			},
			{
				timeStamp: time.Now().Add(-4*time.Minute - 55*time.Second).UTC().Format("20060102T150405Z"),
				ok:        true,
			},
		}
		for _, tt := range tests {
			signature := generateSignature(string(ctx.Request.URI().Path()), tt.timeStamp, ctx.Request.Body(), ak, sk)
			acutalAuthorization := "HMAC-SHA256 timestamp=" + tt.timeStamp + ",access_key=" + ak + ",signature=" + encodeHex(signature)
			ctx.Request.Header.Set("Authorization", acutalAuthorization)
			convey.So(Auth(ctx, ak, sk), convey.ShouldEqual, tt.ok)
		}
	})
}
