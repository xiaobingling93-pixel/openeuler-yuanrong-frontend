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
	"errors"
	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

func Test_authCheck(t *testing.T) {
	config.GetConfig().LocalAuth = &localauth.AuthConfig{}
	// constant.FConfig.AuthenticationEnable = true
	config.GetConfig().AuthenticationEnable = true
	defer func() {
		config.GetConfig().AuthenticationEnable = false
	}()
	patched := []*gomonkey.Patches{
		gomonkey.ApplyFunc(localauth.AuthCheckLocally, func(ak string, sk string, requestSign string, timestamp string, duration int) error {
			if timestamp == "" {
				return errors.New("no auth check info")
			}
			return nil
		}),
	}
	defer func() {
		for i := range patched {
			patched[i].Reset()
		}
	}()

	tests := []struct {
		name      string
		timestamp string

		exceptedError bool
	}{
		{
			name:      "normal request",
			timestamp: strconv.Itoa(int(time.Now().Unix())),

			exceptedError: false,
		},
		{
			name: "no timestamp",

			exceptedError: true,
		},
	}

	for _, test := range tests {
		// req, _ := http.NewRequest(http.MethodPost, "http://127.0.0.1:8080", nil)
		processCtx := types.CreateInvokeProcessContext()
		processCtx.ReqHeader["X-Timestamp-Auth"] = test.timestamp
		// processCtx.ReqHeader["Method"] = http.MethodPost
		processCtx.ReqHeader["URL"] = "http://127.0.0.1:8080"
		// req.Header.Set("X-Timestamp-Auth", test.timestamp)
		err := authCheck(processCtx)
		if test.exceptedError {
			assert.Errorf(t, err, test.name)
		} else {
			assert.Nil(t, err, test.name)
		}
	}
}
