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

// Package parser
package parser

import (
	"encoding/json"
	"fmt"

	"frontend/pkg/frontend/frontendsdkadapter/models"
)

// ReadPayloadData -
func ReadPayloadData(body []byte, payloadInfo *models.DataSystemPayloadInfo) (*models.PayloadData,
	error) {
	return parsePayloadData(payloadInfo, body)
}

// ParsePayloadHeaderJSON -
func ParsePayloadHeaderJSON(payloadInfoStr string) (*models.DataSystemPayloadInfo, error) {
	var payloadInfoList []*models.PayloadInfo
	err := json.Unmarshal([]byte(payloadInfoStr), &payloadInfoList)
	if err != nil {
		return nil, fmt.Errorf("payloadInfo json invalid")
	}
	payloadInfo := &models.DataSystemPayloadInfo{
		Data: payloadInfoList,
	}
	return payloadInfo, nil
}

// parsePayloadData -
func parsePayloadData(payloadInfo *models.DataSystemPayloadInfo, body []byte) (*models.PayloadData,
	error) {
	payload := &models.PayloadData{}
	for _, imgData := range payloadInfo.Data {
		if imgData.Offset+imgData.Length > len(body) {
			return nil, fmt.Errorf("payload len invalid")
		}

		imageBytes := body[imgData.Offset : imgData.Offset+imgData.Length]

		payload.DataList = append(payload.DataList, &models.MultipartData{
			DataPrefix:  imgData.DataPrefix,
			Size:        imgData.Length,
			Data:        imageBytes,
			NeedEncrypt: imgData.NeedEncrypt,
		})
		payload.Size += len(imageBytes)
	}
	return payload, nil
}
