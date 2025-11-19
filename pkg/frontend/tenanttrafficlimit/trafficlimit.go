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

// Package tenanttrafficlimit is for trigger traffic limitation
package tenanttrafficlimit

import (
	"errors"
	"fmt"
	"sync"

	"golang.org/x/time/rate"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/urnutils"
	"frontend/pkg/frontend/config"
)

// TenantContainer tenant information and tenant's Limiter
type TenantContainer struct {
	tenantInfo *sync.Map
}

// TenantInfo -
type TenantInfo struct {
	Quota   int
	Limiter *rate.Limiter
}

var (
	// tenantBuf -
	tenantBuf = &TenantContainer{
		tenantInfo: &sync.Map{},
	}
)

// Limit is the main function of tenant traffic limitation
func Limit(tenantID string) error {
	allow, quota := tenantBuf.tenantTakeOneToken(tenantID)
	if !allow {
		errMsg := fmt.Sprintf("Requests reached the max quota %d of the tenant %s",
			quota, urnutils.Anonymize(tenantID))
		return errors.New(errMsg)
	}
	return nil
}

func (t *TenantContainer) tenantTakeOneToken(tenantID string) (bool, int) {
	tenantInfo := t.getTenantInfo(tenantID)
	if tenantInfo.Limiter == nil {
		log.GetLogger().Infof("traffic limiter is invalid")
		tenantInfo.Limiter = getDefaultLimiter()
		t.tenantInfo.Store(tenantID, tenantInfo)
		return tenantInfo.Limiter.Allow(), tenantInfo.Quota
	}
	return tenantInfo.Limiter.Allow(), tenantInfo.Quota
}

// getTenantInfo  to generator the tenant limiter
func (t *TenantContainer) getTenantInfo(tenantID string) TenantInfo {
	getInfo, ok := t.tenantInfo.Load(tenantID)
	if !ok {
		log.GetLogger().Warnf("failed to get tenant info for tenant %s.", urnutils.Anonymize(tenantID))
		return TenantInfo{Limiter: nil, Quota: config.GetConfig().DefaultTenantLimitQuota}
	}
	tenantInfo, ok := getInfo.(TenantInfo)
	if !ok {
		log.GetLogger().Warnf("invalid tenant info type for tenant %s.", urnutils.Anonymize(tenantID))
		return TenantInfo{Limiter: nil, Quota: config.GetConfig().DefaultTenantLimitQuota}
	}
	return tenantInfo
}

func (t *TenantContainer) syncLimiter(tenantID string, tenantInfo TenantInfo) {
	tenantInfo.Limiter = t.getTenantLimiter(tenantInfo)
	t.tenantInfo.Store(tenantID, tenantInfo)
}

// ProcessUpdate -
func ProcessUpdate(event *etcd3.Event) error {
	return tenantBuf.ProcessUpdate(event)
}

// ProcessDelete -
func ProcessDelete(event *etcd3.Event) {
	tenantBuf.ProcessDelete(event)
}
