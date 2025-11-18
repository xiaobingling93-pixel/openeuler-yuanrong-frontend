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
	"go.uber.org/zap"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/instanceconfig"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/instanceconfigmanager"
)

func startWatchInstanceConfig(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetRouterEtcdClient()
	watcher := etcd3.NewEtcdWatcher(instanceconfig.InsConfigEtcdPrefix,
		instanceconfig.GetWatcherFilter(config.GetConfig().ClusterID),
		instanceConfigHandler, stopCh, etcdClient)
	watcher.StartWatch()
}

func instanceConfigHandler(event *etcd3.Event) {
	logger := log.GetLogger().With(zap.Any("eventType", event.Type), zap.Any("rev", event.Rev))
	log.GetLogger().Infof("handling instance config, key: %s", event.Key)
	switch event.Type {
	case etcd3.PUT:
		instanceconfigmanager.ProcessUpdate(event, logger)
	case etcd3.DELETE:
		instanceconfigmanager.ProcessDelete(event, logger)
	case etcd3.ERROR:
		logger.Warnf("etcd error event: %s", event.Value)
	default:
		logger.Warnf("unsupported event")
	}
}
