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

// Package models
package models

import (
	"encoding/json"
)

// PayloadInfo -
type PayloadInfo struct {
	DataPrefix  string `json:"dataPrefix,omitempty"`
	DataKey     string `json:"dataKey,omitempty"`
	Offset      int    `json:"offset"`
	Length      int    `json:"len"`
	NeedEncrypt bool   `json:"needEncrypt,omitempty"`
}

// DataSystemPayloadInfo -
type DataSystemPayloadInfo struct {
	Data []*PayloadInfo `json:"data"`
}

// ToJSON -
func (d *DataSystemPayloadInfo) ToJSON() string {
	headerStr, err := json.Marshal(d.Data)
	if err != nil {
		return "[]"
	}
	return string(headerStr)
}

// MultipartData -
type MultipartData struct {
	Data        []byte
	DataPrefix  string
	Size        int
	NeedEncrypt bool
}

// PayloadData -
type PayloadData struct {
	DataList []*MultipartData
	Size     int
}

// DataKeyPrefixJoin -
func DataKeyPrefixJoin(d *DataSystemPayloadInfo) []string {
	result := make([]string, len(d.Data))
	for i, part := range d.Data {
		result[i] = part.DataPrefix + part.DataKey
	}
	return result
}
