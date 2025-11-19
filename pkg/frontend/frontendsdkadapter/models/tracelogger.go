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
	"github.com/google/uuid"
	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/logger/log"
)

// TraceInfo -

// TraceLogger -
type TraceLogger struct {
	TenantID string
	TraceID  string
	DataKey  string
	Logger   api.FormatLogger
}

// NewTraceLogger -
func NewTraceLogger(reqType string, traceID string) *TraceLogger {
	if traceID == "" {
		traceID = uuid.NewString()
	}
	return &TraceLogger{
		Logger:  log.GetLogger().With(zap.Any("type", reqType)),
		TraceID: traceID,
	}
}

// AppendDataKey -
func (t *TraceLogger) AppendDataKey(prefix string, key string) {
	if t.DataKey != "" {
		t.DataKey = t.DataKey + "&" + prefix + key
		return
	}
	t.DataKey = prefix + key
}

// With -
func (t *TraceLogger) With(key string, value string) {
	t.Logger = t.Logger.With(zap.Any(key, value))
}
