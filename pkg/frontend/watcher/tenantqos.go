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
	"strings"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/tenanttrafficlimit"
)

const (
	// QOSEtcdPrefix is the etcd prefix for tenant qos limit
	QOSEtcdPrefix = "/sn/qos/business/yrk"
	// ValidTenantPathCount is the separator count of tenant Path
	ValidTenantPathCount = 7
	// tenantMark is mark of etcd of tenant
	tenantMark  = "tenant"
	tenantIndex = 5
)

// StartWatch -
func startWatchTenantQOS(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetMetaEtcdClient()
	watcher := etcd3.NewEtcdWatcher(QOSEtcdPrefix, tenantQOSFilter, processTenantEvent,
		stopCh, etcdClient)
	watcher.StartWatch()
}

func processTenantEvent(event *etcd3.Event) {
	log.GetLogger().Infof("handling tenant qos event type %d, key:%s", event.Type, event.Key)
	var err error
	switch event.Type {
	case etcd3.PUT:
		if err = tenanttrafficlimit.ProcessUpdate(event); err != nil {
			return
		}
	case etcd3.DELETE:
		tenanttrafficlimit.ProcessDelete(event)
	case etcd3.ERROR:
		log.GetLogger().Warnf("etcd error event: %s", event.Value)
	default:
		log.GetLogger().Warnf("unsupported event: %s", event.Value)
	}
}

func tenantQOSFilter(event *etcd3.Event) bool {
	etcdKey := event.Key
	s := strings.Split(etcdKey, constant.KeySeparator)
	if len(s) != ValidTenantPathCount {
		return true
	}
	if s[tenantIndex] != tenantMark {
		return true
	}
	return false
}
