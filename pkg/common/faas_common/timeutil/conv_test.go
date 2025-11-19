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

package timeutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestUnixMillisecond(t *testing.T) {
	assert.Equal(t, time.Second.Nanoseconds()/time.Second.Milliseconds(), int64(NanosecondToMillisecond))
	assert.Equal(t, int64(time.Millisecond/time.Nanosecond), int64(NanosecondToMillisecond))
}

func TestNowUnixMillisecond(t *testing.T) {
	millisecond := NowUnixMillisecond()
	assert.Equal(t, NowUnixMillisecond() >= millisecond, true)
}

func TestNowUnixNanoseconds(t *testing.T) {
	unixNanoseconds := NowUnixNanoseconds()
	assert.Equal(t, NowUnixNanoseconds() >= unixNanoseconds, true)
}
