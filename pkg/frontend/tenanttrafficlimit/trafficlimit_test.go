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

package tenanttrafficlimit

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/types"
)

// TestTenantTraffic test for tenant traffic limiter
func TestTenantTraffic(t *testing.T) {
	defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
		return &types.Config{
			InstanceNum:             3,
			DefaultTenantLimitQuota: 1800,
		}
	}).Reset()
	tenantID01 := "tenantID1"
	tenantID02 := "tenantID2"
	tenantID03 := "tenantID3"
	tenanting1 := TenantInfo{Quota: 17, Limiter: &rate.Limiter{}}
	tenanting2 := TenantInfo{Quota: 13, Limiter: &rate.Limiter{}}
	tenanting3 := TenantInfo{Quota: 20, Limiter: &rate.Limiter{}}
	tenantBuf.tenantInfo.Store(tenantID01, tenanting1)
	tenantBuf.tenantInfo.Store(tenantID02, tenanting2)
	tenantBuf.tenantInfo.Store(tenantID03, tenanting3)
	tenantBuf.syncLimiter(tenantID01, tenanting1)
	tenantBuf.syncLimiter(tenantID02, tenanting2)
	tenantBuf.syncLimiter(tenantID03, tenanting3)
	for i := 0; i < 6; i++ {
		assert.Equal(t, nil, Limit(tenantID01))
	}
	for i := 7; i < 20; i++ {
		assert.Error(t, Limit(tenantID01))
	}
	for i := 0; i < 4; i++ {
		assert.Equal(t, nil, Limit(tenantID02))
	}
	for i := 5; i < 20; i++ {
		assert.Error(t, Limit(tenantID02))
	}
	for i := 0; i < 7; i++ {
		assert.Equal(t, nil, Limit(tenantID03))
	}
	for i := 8; i < 20; i++ {
		assert.Error(t, Limit(tenantID03))
	}
	fmt.Println("test update")
	time.Sleep(5 * time.Second)
	//update tenant quota
	event1 := &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant/tenantID1",
		Value: []byte(`{"quota":10}`),
	}
	tenantBuf.ProcessUpdate(event1)
	for i := 0; i < 3; i++ {
		assert.Equal(t, nil, Limit(tenantID01))
	}
	for i := 4; i < 20; i++ {
		assert.Error(t, Limit(tenantID01))
	}
	time.Sleep(5 * time.Second)
	//delete tenant quota
	fmt.Println("test delete")
	event2 := &etcd3.Event{
		Type: etcd3.DELETE,
		Key:  "/sn/qos/business/yrk/tenant/tenantID2",
	}
	tenantBuf.ProcessDelete(event2)
	for i := 0; i < 40; i++ {
		assert.Equal(t, nil, Limit(tenantID02))
	}
	fmt.Println("test large scale")
	event3 := &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant/tenantID3",
		Value: []byte(`{"quota":200000}`),
	}
	tenantBuf.ProcessUpdate(event3)
	for i := 0; i < 40000; i++ {
		assert.Equal(t, nil, Limit(tenantID03))
	}
}

func TestTenantTakeOneToken(t *testing.T) {
	defer gomonkey.ApplyFunc(config.GetConfig, func() *types.Config {
		return &types.Config{
			InstanceNum:             3,
			DefaultTenantLimitQuota: 1800,
		}
	}).Reset()
	tenantBuf.tenantInfo = &sync.Map{}
	allow, quota := tenantBuf.tenantTakeOneToken("test")
	assert.Equal(t, allow, true)
	assert.Equal(t, quota, 1800)
	tenantBuf.tenantInfo.Store("test", 18)
	allow, quota = tenantBuf.tenantTakeOneToken("test")
	assert.Equal(t, allow, true)
	assert.Equal(t, quota, 1800)
}
