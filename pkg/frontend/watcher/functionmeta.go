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
	"frontend/pkg/frontend/functionmeta"
)

const (
	functionEtcdKeyLen = 11
	functionsIndex     = 2
	tenantsIndex       = 5
	functionIndex      = 7
)

func startWatchFunctionMeta(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetMetaEtcdClient()
	watcher := etcd3.NewEtcdWatcher(constant.FunctionPrefix, functionMetaFilter,
		functionMetaHandler, stopCh, etcdClient)
	watcher.StartWatch()
}

// key: /sn/functions/business/<businessID>/tenant/<tenantID>/function/<functionName>/version/<version>
func functionMetaFilter(event *etcd3.Event) bool {
	etcdKey := event.Key
	keyParts := strings.Split(etcdKey, constant.ETCDEventKeySeparator)

	if len(keyParts) != functionEtcdKeyLen {
		return true
	}
	if keyParts[functionsIndex] != "functions" || keyParts[tenantsIndex] != "tenant" ||
		keyParts[functionIndex] != "function" {
		return true
	}

	return false
}

func functionMetaHandler(event *etcd3.Event) {
	log.GetLogger().Infof("handling function meta event type %d, key:%s", event.Type, event.Key)
	switch event.Type {
	case etcd3.PUT:
		if err := functionmeta.ProcessUpdate(event.Key, event.Value, event.ETCDType); err != nil {
			return
		}
		return
	case etcd3.DELETE:
		if err := functionmeta.ProcessDelete(event.Key, event.ETCDType); err != nil {
			log.GetLogger().Errorf("failed to process delete event, err:%s", err)
			return
		}
		return
	case etcd3.SYNCED:
		log.GetLogger().Infof("frontend function ready to receive etcd kv")
	default:
		log.GetLogger().Errorf("undefined etcd event")
		return
	}
}

func startWatchCAEFunctionMeta(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetCAEMetaEtcdClient()
	if etcdClient == nil {
		return
	}
	watcher := etcd3.NewEtcdWatcher(constant.FunctionPrefix, functionMetaFilter,
		functionMetaHandler, stopCh, etcdClient)
	watcher.StartWatch()
}
