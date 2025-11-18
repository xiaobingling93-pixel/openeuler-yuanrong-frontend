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

package watcher

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"frontend/pkg/common/faas_common/etcd3"
)

func TestProcessTenantEvent(t *testing.T) {
	// put error
	event := &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant",
		Value: []byte{},
	}
	processTenantEvent(event)
	// put error
	event = &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant/tenantID",
		Value: []byte{},
	}
	processTenantEvent(event)
	event = &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant/tenantID",
		Value: []byte(`{"quo":10}`),
	}
	processTenantEvent(event)
	// delete error
	event = &etcd3.Event{
		Type:  etcd3.DELETE,
		Key:   "/sn/qos/business/yrk/tenant",
		Value: []byte{},
	}
	processTenantEvent(event)

	// error
	event = &etcd3.Event{
		Type:  etcd3.ERROR,
		Key:   "/sn/qos/business/yrk/tenant",
		Value: []byte{},
	}
	processTenantEvent(event)

	// error
	event = &etcd3.Event{
		Type:  etcd3.SYNCED,
		Key:   "/sn/qos/business/yrk/tenant",
		Value: []byte{},
	}
	processTenantEvent(event)
}

func TestTenantQOSFilter(t *testing.T) {
	event := &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant",
		Value: []byte{},
	}
	ok := tenantQOSFilter(event)
	assert.Equal(t, true, ok)

	event = &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/tenant/yrk/TenantID",
		Value: []byte{},
	}
	ok = tenantQOSFilter(event)
	assert.Equal(t, true, ok)

	event = &etcd3.Event{
		Type:  etcd3.PUT,
		Key:   "/sn/qos/business/yrk/tenant/TenantID",
		Value: []byte{},
	}
	ok = tenantQOSFilter(event)
	assert.Equal(t, false, ok)
}
