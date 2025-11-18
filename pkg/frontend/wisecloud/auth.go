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

// Package wisecloud -
package wisecloud

import (
	"crypto/hmac"
	"crypto/sha256"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// Auth -
func Auth(ctx *fasthttp.RequestCtx, ak string, sk []byte) bool {

	headers := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		headers[strings.ToLower(string(key))] = string(value)
	})

	return AuthDownGradeFunctionCall(string(ctx.Request.URI().Path()), headers, ctx.Request.Body(), ak, sk)
}

// AuthDownGradeFunctionCall -
func AuthDownGradeFunctionCall(url string, headers map[string]string, body []byte, ak string, sk []byte) bool {
	headerAuthorization := headers["authorization"]
	layout := "20060102T150405Z"
	info, ok := parseAuthorization(headerAuthorization)
	if !ok {
		return false
	}
	t, err := time.Parse(layout, info.timeStamp)
	if err != nil {
		return false
	}
	if time.Now().Sub(t) > 5*time.Minute { // 时间戳有效期5分钟
		return false
	}
	signature := generateSignature(url, info.timeStamp, body, ak, sk)
	if encodeHex(signature) != info.signature {
		return false
	}
	return true
}

type authorizationStruct struct {
	timeStamp string
	ak        string
	signature string
}

func parseAuthorization(authorization string) (*authorizationStruct, bool) {
	suffix, flag := strings.CutPrefix(authorization, "HMAC-SHA256 ")
	if !flag {
		return nil, false
	}
	splits := strings.Split(suffix, ",")
	if len(splits) != 3 { // 固定格式
		return nil, false
	}
	timeStamp, flag0 := strings.CutPrefix(splits[0], "timestamp=")
	ak, flag1 := strings.CutPrefix(splits[1], "access_key=")
	signature, flag2 := strings.CutPrefix(splits[2], "signature=")
	if flag0 && flag1 && flag2 {
		return &authorizationStruct{
			timeStamp: timeStamp,
			ak:        ak,
			signature: signature,
		}, true
	}
	return nil, false
}

func generateSignature(url string, timeStamp string, body []byte, ak string, sk []byte) []byte {
	digestBytes := buildDigest(url, timeStamp, body, ak)
	digestHex := sha256AndHex(digestBytes)
	return sign(sk, []byte(digestHex))
}

func buildDigest(url string, timeStamp string, body []byte, ak string) []byte {
	var builder strings.Builder
	builder.WriteString(url)
	builder.WriteString("\n")
	builder.WriteString("X-Timestamp: ")
	builder.WriteString(timeStamp)
	builder.WriteString("\n")
	builder.WriteString("X-Access-Key: ")
	builder.WriteString(ak)
	builder.WriteString("\n")
	builder.Write(body)
	return []byte(builder.String())
}

func sign(sk, content []byte) []byte {
	h := hmac.New(sha256.New, sk)
	_, err := h.Write(content)
	if err != nil {
		return nil
	}
	return h.Sum(nil)
}

const (
	firstFourBitShift = 4
)

// EncodeHex encode hex
func encodeHex(data []byte) string {
	if data == nil || len(data) == 0 {
		return ""
	}
	l := len(data)
	out := make([]byte, l<<1)
	j := 0
	for i := 0; i < l; i++ {
		if j >= l<<1 {
			return ""
		}
		out[j] = hexDigits[(data[i]>>4)&0xF] // magic number
		j++
		if j >= l<<1 {
			return ""
		}
		out[j] = hexDigits[(data[i] & 0xF)] // magic number
		j++
	}
	return string(out)
}

func sha256AndHex(input []byte) string {
	// 计算SHA256哈希
	hash := sha256.Sum256(input)

	// 将哈希值转换为16进制字符串
	var builder strings.Builder
	for _, b := range hash {
		// 获取高4位和低4位，并转换为对应的16进制字符
		builder.WriteByte(hexDigits[b>>firstFourBitShift])
		builder.WriteByte(hexDigits[b&0x0f]) // 取后四位
	}
	builder.WriteByte('\n') // 与C++版本一致添加换行符

	return builder.String()
}

// 16进制字符集，使用小写字母与C++版本保持一致
var hexDigits = []byte{
	'0', '1', '2', '3', '4', '5', '6', '7',
	'8', '9', 'a', 'b', 'c', 'd', 'e', 'f',
}
