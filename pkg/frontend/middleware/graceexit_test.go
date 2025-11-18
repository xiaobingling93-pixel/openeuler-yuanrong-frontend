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

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func SetGraceExit(flag bool) {
	graceExitFlag = flag
}

func TestGraceExitFilter(t *testing.T) {
	router := gin.New()
	router.POST("/serverless/caas/v1/execute", GraceExitGinFilter(NewCommonErrorWriter()),
		nil)
	SetGraceExit(true)
	defer func() {
		SetGraceExit(false)
	}()
	req, err := http.NewRequest(http.MethodPost, "/serverless/caas/v1/execute", nil)
	assert.NoError(t, err)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	assert.Equal(t, "frontend exiting", w.Body.String())
}

type CommonErrorWriter struct{}

func NewCommonErrorWriter() *CommonErrorWriter {
	return &CommonErrorWriter{}
}

func (d *CommonErrorWriter) WriteErrorToGinResponse(ctx *gin.Context, err error) {
	ctx.Writer.Write([]byte(err.Error()))
}
