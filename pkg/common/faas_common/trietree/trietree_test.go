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

package trietree

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"frontend/pkg/common/faas_common/constant"
)

func TestTrie(t *testing.T) {
	s1 := "/hello"
	s2 := "/hello/server"
	prefixTrie := NewTrie()
	prefixTrie.Insert(strings.Split(s1, constant.URLSeparator))
	prefixTrie.Insert(strings.Split(s2, constant.URLSeparator))

	input := "/hello"
	longestMatch := prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "/hello")

	input = "/helloe"
	longestMatch = prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "")

	input = "/hello/se"
	longestMatch = prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "/hello")

	input = "/hello/server"
	longestMatch = prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "/hello/server")

	input = "/hello/server/aaa"
	longestMatch = prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "/hello/server")

	prefixTrie.Delete(strings.Split(s2, constant.URLSeparator))

	input = "/hello/server"
	longestMatch = prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "/hello")

	prefixTrie.Delete(strings.Split(s1, constant.URLSeparator))

	ok := prefixTrie.Search(strings.Split(s1, constant.URLSeparator))
	assert.False(t, ok)

	input = "/hello"
	longestMatch = prefixTrie.LongestMatch(strings.Split(input, constant.URLSeparator))
	assert.Equal(t, longestMatch, "")
}
