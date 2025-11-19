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

	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/schedulerproxy"
)

// start to watch the schedulers by the etcd
func startWatchScheduler(stopCh <-chan struct{}) {
	switch config.GetConfig().SchedulerKeyPrefixType {
	case constant.SchedulerKeyTypeFunction:
		startWatchInstanceScheduler(stopCh)
	case constant.SchedulerKeyTypeModule:
		startWatchModuleScheduler(stopCh)
	default:
		startWatchInstanceScheduler(stopCh)
	}
}

// start to watch the instance faas schedulers by the etcd
func startWatchInstanceScheduler(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetRouterEtcdClient()
	watcher := etcd3.NewEtcdWatcher(constant.InstancePathPrefix, instanceSchedulerFilter, instanceSchedulerHandler,
		stopCh, etcdClient)
	watcher.StartWatch()
}

func isFaaSScheduler(etcdPath string) bool {
	info, err := utils.GetFunctionInstanceInfoFromEtcdKey(etcdPath)
	if err != nil {
		return false
	}
	return strings.Contains(info.FunctionName, "faasscheduler")
}

func instanceSchedulerFilter(event *etcd3.Event) bool {
	return !isFaaSScheduler(event.Key)
}

func instanceSchedulerHandler(event *etcd3.Event) {
	logger := log.GetLogger().With(zap.Any("eventType", event.Type), zap.Any("eventKey", event.Key),
		zap.Any("revisionId", event.Rev), zap.Any("schedulerType", "function"))
	logger.Infof("recv scheduler event type")
	if event.Type == etcd3.SYNCED {
		logger.Infof("faaSFrontend scheduler ready to receive etcd kv")
		return
	}
	info, err := utils.GetFunctionInstanceInfoFromEtcdKey(event.Key)
	if err != nil {
		logger.Errorf("failed to parse event key: %s", err.Error())
		return
	}
	handleEvent(event, info, logger)
}

// start to watch the module schedulers by the etcd
func startWatchModuleScheduler(stopCh <-chan struct{}) {
	etcdClient := etcd3.GetRouterEtcdClient()
	watcher := etcd3.NewEtcdWatcher(constant.ModuleSchedulerPrefix, moduleSchedulerFilter, moduleSchedulerHandler,
		stopCh, etcdClient)
	watcher.StartWatch()
}

func moduleSchedulerFilter(event *etcd3.Event) bool {
	return !strings.Contains(event.Key, constant.ModuleSchedulerPrefix)
}

func moduleSchedulerHandler(event *etcd3.Event) {
	logger := log.GetLogger().With(zap.Any("eventType", event.Type), zap.Any("eventKey", event.Key),
		zap.Any("revisionId", event.Rev), zap.Any("schedulerType", "module"))
	logger.Infof("recv module scheduler event type")
	if event.Type == etcd3.SYNCED {
		logger.Infof("faaSFrontend scheduler ready to receive etcd kv")
		return
	}
	info, err := utils.GetModuleSchedulerInfoFromEtcdKey(event.Key)
	if err != nil {
		logger.Errorf("failed to parse event key: %s", err.Error())
		return
	}
	handleEvent(event, info, logger)
}

func handleEvent(event *etcd3.Event, info *types.InstanceInfo, logger api.FormatLogger) {
	switch event.Type {
	case etcd3.PUT:
		schedulerproxy.ProcessUpdate(event, info, logger)
	case etcd3.DELETE:
		schedulerproxy.ProcessDelete(info, logger)
	default:
		logger.Warnf("unsupported event, type is %d", event.Type)
	}
}
