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

package utils

import (
	"errors"
	"net"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey"
	"github.com/stretchr/testify/assert"
)

func TestIsErrorAddressAlreadyInUse(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:55555")
	defer func(listener net.Listener) {
		_ = listener.Close()
	}(listener)
	assert.Nil(t, err)
	listener2, err := net.Listen("tcp", "127.0.0.1:55555")
	assert.True(t, isErrorAddressAlreadyInUse(err))
	if err == nil {
		_ = listener2.Close()
	}
}

func TestProcessBindErrorAndExit(t *testing.T) {
	flag := false
	patches := gomonkey.ApplyFunc(isErrorAddressAlreadyInUse, func(err error) bool {
		return flag
	}).ApplyFunc(os.Exit, func(code int) {
		return
	})
	defer patches.Reset()
	ProcessBindErrorAndExit(errors.New("mock err"))
	flag = true
	ProcessBindErrorAndExit(errors.New("mock err2"))
}

func TestCheckAddress(t *testing.T) {
	test := []struct {
		addr   string
		wanted bool
	}{
		{
			addr:   "111",
			wanted: false,
		},
		{
			addr:   "asdasd:asdasd",
			wanted: false,
		},
		{
			addr:   "127.0.0.1:asd",
			wanted: false,
		},
		{
			addr:   "127.0.0.1:994651",
			wanted: false,
		},
		{
			addr:   "127.0.0.1:6379",
			wanted: true,
		},
	}
	for _, tt := range test {
		res := CheckAddress(tt.addr)
		if tt.wanted {
			assert.True(t, res)
		} else {
			assert.False(t, res)
		}
	}
}
