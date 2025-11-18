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

package datasystem

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/datasystemclient"
	"frontend/pkg/common/faas_common/grpc/pb/commonargs"
	"frontend/pkg/common/faas_common/grpc/pb/data"
)

func TestPutHandler(t *testing.T) {
	errMsg := &data.PutRequest{
		WriteMode:       0,
		ConsistencyType: 0,
		NestedObjectIds: nil,
	}
	body, _ := proto.Marshal(errMsg)
	convey.Convey("parse failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		PutHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})

	msg := &data.PutRequest{
		ObjectData:      []byte("123"),
		ObjectId:        "objectId",
		WriteMode:       0,
		ConsistencyType: 0,
		NestedObjectIds: nil,
	}
	body, _ = proto.Marshal(msg)
	convey.Convey("put failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		objPut := gomonkey.ApplyFunc(datasystemclient.ObjPut,
			func(req *data.PutRequest, config *datasystemclient.Config, traceID string) api.ErrorInfo {
				return api.ErrorInfo{Err: errors.New("put failed"),
					Code: int(commonargs.ErrorCode_ERR_INNER_SYSTEM_ERROR)}
			})
		defer objPut.Reset()
		PutHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
		response := &data.PutResponse{}
		err := proto.Unmarshal(rw.Body.Bytes(), response)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(commonargs.ErrorCode_ERR_INNER_SYSTEM_ERROR), response.Code)
		assert.Equal(t, "put failed", response.Message)

		rw = httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
			return errors.New("proto unmarshal error")
		}).Reset()
		PutHandler(ctx)
		convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
	})
	convey.Convey("put succeed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		objPut := gomonkey.ApplyFunc(datasystemclient.ObjPut,
			func(req *data.PutRequest, config *datasystemclient.Config, traceID string) api.ErrorInfo {
				return api.ErrorInfo{Code: 0, Err: errors.New("")}
			})
		defer objPut.Reset()
		PutHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
		response := &data.PutResponse{}
		err := proto.Unmarshal(rw.Body.Bytes(), response)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(commonargs.ErrorCode_ERR_NONE), response.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		PutHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		ctx.Request.Header.Add(constant.HeaderTenantID, "tenantId")
		ctx.Request.Header.Add(constant.HeaderTraceID, "traceId")
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		PutHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestGetHandler(t *testing.T) {
	msg := &data.GetRequest{
		ObjectIds: []string{"objectId"},
		TimeoutMs: 0,
	}
	body, _ := proto.Marshal(msg)
	convey.Convey("get failed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		objGet := gomonkey.ApplyFunc(datasystemclient.ObjGet,
			func(req *data.GetRequest, config *datasystemclient.Config, traceID string) ([][]byte, api.ErrorInfo) {
				return nil, api.ErrorInfo{Err: errors.New("get failed"),
					Code: int(commonargs.ErrorCode_ERR_INNER_SYSTEM_ERROR)}
			})
		defer objGet.Reset()
		GetHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
		response := &data.GetResponse{}
		err := proto.Unmarshal(rw.Body.Bytes(), response)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(commonargs.ErrorCode_ERR_INNER_SYSTEM_ERROR), response.Code)
		assert.Equal(t, "get failed", response.Message)

		rw = httptest.NewRecorder()
		ctx, _ = gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
			return errors.New("proto unmarshal error")
		}).Reset()
		GetHandler(ctx)
		convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
	})
	convey.Convey("get succeed", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		objGet := gomonkey.ApplyFunc(datasystemclient.ObjGet,
			func(req *data.GetRequest, config *datasystemclient.Config, traceID string) ([][]byte, api.ErrorInfo) {
				return nil, api.ErrorInfo{Err: errors.New(""), Code: int(commonargs.ErrorCode_ERR_NONE)}
			})
		defer objGet.Reset()
		GetHandler(ctx)
		assert.Equal(t, http.StatusOK, rw.Code)
		response := &data.GetResponse{}
		err := proto.Unmarshal(rw.Body.Bytes(), response)
		assert.Equal(t, nil, err)
		assert.Equal(t, int32(commonargs.ErrorCode_ERR_NONE), response.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		GetHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		GetHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestIncreaseRefHandler(t *testing.T) {
	convey.Convey("IncreaseRefHandler", t, func() {
		convey.Convey("failed to parse increase ref request message, empty ids", func() {
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.IncreaseRefRequest{}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			IncreaseRefHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)

			defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
				return errors.New("proto unmarshal error")
			}).Reset()
			IncreaseRefHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("OK", func() {
			defer gomonkey.ApplyFunc(datasystemclient.GIncreaseRef, func(req *data.IncreaseRefRequest, config *datasystemclient.Config, traceID string) ([]string, api.ErrorInfo) {
				return []string{}, api.ErrorInfo{
					Code: 0,
					Err:  fmt.Errorf("nil err"),
				}
			}).Reset()
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.IncreaseRefRequest{
				RemoteClientId: "123456",
				ObjectIds:      []string{"123", "345"},
			}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			IncreaseRefHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
		})
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.IncreaseRefRequest{
			RemoteClientId: "123456",
			ObjectIds:      []string{"123", "345"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		IncreaseRefHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.IncreaseRefRequest{
			RemoteClientId: "123456",
			ObjectIds:      []string{"123", "345"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		IncreaseRefHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestDecreaseRefHandler(t *testing.T) {
	convey.Convey("DecreaseRefHandler", t, func() {
		convey.Convey("failed to parse decrease ref request message", func() {
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.DecreaseRefRequest{}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			DecreaseRefHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)

			defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
				return errors.New("proto unmarshal error")
			}).Reset()
			DecreaseRefHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("OK", func() {
			defer gomonkey.ApplyFunc(datasystemclient.GDecreaseRef, func(req *data.DecreaseRefRequest, config *datasystemclient.Config, traceID string) ([]string, api.ErrorInfo) {
				return []string{}, api.ErrorInfo{
					Code: 0,
					Err:  fmt.Errorf("nil err"),
				}
			}).Reset()
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.DecreaseRefRequest{
				RemoteClientId: "123456",
				ObjectIds:      []string{"123", "345"},
			}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			DecreaseRefHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
		})
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.DecreaseRefRequest{
			RemoteClientId: "123456",
			ObjectIds:      []string{"123", "345"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		DecreaseRefHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.DecreaseRefRequest{
			RemoteClientId: "123456",
			ObjectIds:      []string{"123", "345"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		DecreaseRefHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestKvSetHandler(t *testing.T) {
	convey.Convey("KvSetHandler", t, func() {
		convey.Convey("failed to parse kv set request message", func() {
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvSetRequest{}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvSetHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)

			defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
				return errors.New("proto unmarshal error")
			}).Reset()
			KvSetHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("OK", func() {
			defer gomonkey.ApplyFunc(datasystemclient.Set, func(req *data.KvSetRequest, config *datasystemclient.Config, traceID string) api.ErrorInfo {
				return api.ErrorInfo{
					Code: 0,
					Err:  fmt.Errorf("nil err"),
				}
			}).Reset()
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvSetRequest{
				Key:   "test",
				Value: []byte("VALUE"),
			}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvSetHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
		})
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvSetRequest{
			Key:   "test",
			Value: []byte("VALUE"),
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvSetHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvSetRequest{
			Key:   "test",
			Value: []byte("VALUE"),
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvSetHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestKvMSetTxHandler(t *testing.T) {
	convey.Convey("KvMSetTxHandler", t, func() {
		convey.Convey("failed to parse kv mset tx request message", func() {
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvMSetTxRequest{}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvMSetTxHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)

			defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
				return errors.New("proto unmarshal error")
			}).Reset()
			KvMSetTxHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("OK", func() {
			defer gomonkey.ApplyFunc(datasystemclient.MSetTx, func(req *data.KvMSetTxRequest,
				config *datasystemclient.Config, traceID string) api.ErrorInfo {
				return api.ErrorInfo{
					Code: 0,
					Err:  fmt.Errorf("nil err"),
				}
			}).Reset()
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvMSetTxRequest{
				Keys:   []string{"test"},
				Values: [][]byte{[]byte("VALUE")},
			}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvMSetTxHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
		})
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvMSetTxRequest{
			Keys:   []string{"test"},
			Values: [][]byte{[]byte("VALUE")},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvMSetTxHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvMSetTxRequest{
			Keys:   []string{"test"},
			Values: [][]byte{[]byte("VALUE")},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvMSetTxHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvMSetTxRequest{
			Keys:   []string{"test", "test2"},
			Values: [][]byte{[]byte("VALUE")},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		KvMSetTxHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestKvGetHandler(t *testing.T) {
	convey.Convey("KvGetHandler", t, func() {
		convey.Convey("failed to parse kv get request message", func() {
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvGetRequest{}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvGetHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)

			defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
				return errors.New("proto unmarshal error")
			}).Reset()
			KvGetHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("OK", func() {
			defer gomonkey.ApplyFunc(datasystemclient.Get, func(req *data.KvGetRequest, config *datasystemclient.Config, traceID string) ([][]byte, api.ErrorInfo) {
				return [][]byte{}, api.ErrorInfo{
					Code: 0,
					Err:  fmt.Errorf("nil err"),
				}
			}).Reset()
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvGetRequest{
				Keys: []string{"test"},
			}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvGetHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
		})
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvGetRequest{
			Keys: []string{"test"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvGetHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvGetRequest{
			Keys: []string{"test"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvGetHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}

func TestKvDelHandler(t *testing.T) {
	convey.Convey("KvDelHandler", t, func() {
		convey.Convey("failed to parse kv set request message", func() {
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvDelRequest{}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvDelHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)

			defer gomonkey.ApplyFunc(proto.Unmarshal, func(b []byte, m proto.Message) error {
				return errors.New("proto unmarshal error")
			}).Reset()
			KvDelHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusBadRequest)
		})

		convey.Convey("OK", func() {
			defer gomonkey.ApplyFunc(datasystemclient.Del, func(req *data.KvDelRequest, config *datasystemclient.Config, traceID string) ([][]byte, api.ErrorInfo) {
				return [][]byte{}, api.ErrorInfo{
					Code: 0,
					Err:  fmt.Errorf("nil err"),
				}
			}).Reset()
			rw := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rw)
			errMsg := &data.KvDelRequest{
				Keys: []string{"test"},
			}
			body, _ := proto.Marshal(errMsg)
			ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
			KvDelHandler(ctx)
			convey.So(rw.Code, convey.ShouldEqual, http.StatusOK)
		})
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvDelRequest{
			Keys: []string{"test"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(io.ReadAll,
			func(r io.Reader) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvDelHandler(ctx)
		assert.Equal(t, http.StatusInternalServerError, rw.Code)
	})
	convey.Convey("internal error", t, func() {
		rw := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(rw)
		body, _ := proto.Marshal(&data.KvDelRequest{
			Keys: []string{"test"},
		})
		ctx.Request, _ = http.NewRequest("", "", bytes.NewBuffer(body))
		patch := gomonkey.ApplyFunc(proto.Marshal,
			func(m proto.Message) ([]byte, error) {
				return nil, errors.New("some error")
			})
		defer patch.Reset()
		KvDelHandler(ctx)
		assert.Equal(t, http.StatusBadRequest, rw.Code)
	})
}
