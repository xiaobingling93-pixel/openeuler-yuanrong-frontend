//go:build module

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
	"bufio"
	"bytes"
	"frontend/pkg/frontend/types"
	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTTPUploadStream(t *testing.T) {
	convey.Convey("Test IsHTTPUploadStream", t, func() {
		req, err := http.NewRequest("POST", "http://example.com", nil)
		convey.So(err, convey.ShouldBeNil)
		// when no content-type
		convey.So(IsHTTPUploadStream(req), convey.ShouldBeFalse)

		// when is stream req
		req.Header.Set("Content-Type", "multipart/form-data")
		convey.So(IsHTTPUploadStream(req), convey.ShouldBeTrue)

		// when not stream req
		req.Header.Set("Content-Type", "json")
		convey.So(IsHTTPUploadStream(req), convey.ShouldBeFalse)
	})
}

func TestBuildStreamContext(t *testing.T) {
	convey.Convey("Test BuildStreamContext", t, func() {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		reqBody := bytes.NewBufferString("test body")
		ctx.Request, _ = http.NewRequest("POST", "/", reqBody)

		processCtx := &types.InvokeProcessContext{}
		BuildStreamContext(ctx, processCtx)

		convey.So(processCtx.StreamInfo.ReqStream, convey.ShouldResemble, ctx.Request.Body)
		convey.So(processCtx.StreamInfo.RspStream, convey.ShouldResemble, ctx.Writer)
	})
}

type MockReadCloser struct{}

func (m MockReadCloser) Close() error {
	return nil
}

func (m MockReadCloser) Read([]byte) (int, error) {
	return 0, nil
}

type fakeResponseWriter struct {
	close      bool
	header     http.Header
	body       []byte
	statusCode int
}

func (f fakeResponseWriter) Header() http.Header {
	return f.header
}

func (f fakeResponseWriter) Write(i []byte) (int, error) {
	panic("implement me")
}

func (f *fakeResponseWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

func (f fakeResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	panic("implement me")
}

func (f fakeResponseWriter) Flush() {
	panic("implement me")
}

func (f fakeResponseWriter) CloseNotify() <-chan bool {
	panic("implement me")
}

func (f fakeResponseWriter) Status() int {
	panic("implement me")
}

func (f fakeResponseWriter) Size() int {
	panic("implement me")
}

func (f fakeResponseWriter) WriteString(s string) (int, error) {
	panic("implement me")
}

func (f fakeResponseWriter) Written() bool {
	panic("implement me")
}

func (f fakeResponseWriter) WriteHeaderNow() {
	panic("implement me")
}

func (f fakeResponseWriter) Pusher() http.Pusher {
	panic("implement me")
}
