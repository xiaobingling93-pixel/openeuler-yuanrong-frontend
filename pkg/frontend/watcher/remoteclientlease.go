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
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/remoteclientlease"
)

func startWatchRemoteClientLease(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetRouterEtcdClient()
	watcher := etcd3.NewEtcdWatcher(constant.InstancePathPrefix, remoteClientLeaseFilter,
		remoteClientLeaseHandler, stopCh, etcdClient)
	watcher.StartWatch()
}

func isFaaSManager(etcdPath string) bool {
	info, err := utils.GetFunctionInstanceInfoFromEtcdKey(etcdPath)
	if err != nil {
		return false
	}
	return strings.Contains(info.FunctionName, "faasmanager")
}

func remoteClientLeaseFilter(event *etcd3.Event) bool {
	return !isFaaSManager(event.Key)
}

func remoteClientLeaseHandler(event *etcd3.Event) {
	log.GetLogger().Infof("handling faas manager event type %d, key:%s", event.Type, event.Key)
	if event.Type == etcd3.SYNCED {
		log.GetLogger().Infof("faaSManager ready to receive etcd kv")
		return
	}
	info, err := utils.GetFunctionInstanceInfoFromEtcdKey(event.Key)
	if err != nil {
		log.GetLogger().Errorf("failed to parse event key of %+v: %s", event, err)
		return
	}
	switch event.Type {
	case etcd3.PUT:
		remoteclientlease.UpdateFaasManager(event, info)
	case etcd3.DELETE:
		remoteclientlease.DeleteFaasManager(info)
	default:
		log.GetLogger().Warnf("unsupported event: %#v", event)
	}
}
