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

	"frontend/pkg/common/faas_common/aliasroute"
	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/logger/log"
)

const (
	aliasEtcdKeyLen   = 10
	aliasedIndex      = 2
	defaultAliasSign  = "aliases"
	defaultTenantSign = "tenant"
	defaultFuncSign   = "function"
)

func startWatchAlias(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetMetaEtcdClient()
	watcher := etcd3.NewEtcdWatcher(constant.AliasPrefix, aliasFilter,
		aliasHandler, stopCh, etcdClient)
	watcher.StartWatch()
}

// key: /sn/aliases/business/<businessID>/tenant/<tenantID>/function/<functionName>/<aliasName>
func aliasFilter(event *etcd3.Event) bool {
	etcdKey := event.Key
	keyParts := strings.Split(etcdKey, constant.ETCDEventKeySeparator)
	if len(keyParts) != aliasEtcdKeyLen {
		return true
	}
	if keyParts[aliasedIndex] != defaultAliasSign || keyParts[tenantsIndex] != defaultTenantSign ||
		keyParts[functionIndex] != defaultFuncSign {
		return true
	}

	return false
}

func aliasHandler(event *etcd3.Event) {
	log.GetLogger().Infof("handling alias event type %d, key:%s", event.Type, event.Key)
	switch event.Type {
	case etcd3.PUT:
		_, err := aliasroute.ProcessUpdate(event)
		if err != nil {
			return
		}
	case etcd3.DELETE:
		aliasroute.ProcessDelete(event)
	case etcd3.ERROR:
		log.GetLogger().Warnf("etcd error event: %s", event.Value)
	default:
		log.GetLogger().Warnf("unsupported event, key: %s", event.Key)
	}
}
