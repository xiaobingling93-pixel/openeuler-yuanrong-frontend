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

// Package wisecloud -
package wisecloud

import (
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/instanceconfig"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/queue"
	"frontend/pkg/common/faas_common/resspeckey"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/statuscode"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/functionmeta"
	"frontend/pkg/frontend/instanceconfigmanager"
	"frontend/pkg/frontend/instancemanager"
)

// queueManager -
var queueManager = &QueueManager{
	RWMutex:   sync.RWMutex{},
	queuesMap: make(map[string]map[string]*reqQueue),
	logger:    log.GetLogger(),
}

// GetQueueManager -
func GetQueueManager() *QueueManager {
	return queueManager
}

// QueueManager -
type QueueManager struct {
	sync.RWMutex
	queuesMap map[string]map[string]*reqQueue
	logger    api.FormatLogger
}

// ProcessFunctionDelete -
func (m *QueueManager) ProcessFunctionDelete(funcMeta *types.FuncSpec) {
	m.Lock()
	queues, ok := m.queuesMap[funcMeta.FunctionKey]
	if !ok {
		m.Unlock()
		return
	}
	delete(m.queuesMap, funcMeta.FunctionKey)
	m.Unlock()

	resKeySum := ""
	for resKey, q := range queues {
		q.destroy()
		resKeySum += resKey + ","
	}
	m.logger.Infof("recv function delete event, delete queues success, funcKey: %s, resKeysum: %s",
		funcMeta.FunctionKey, resKeySum)
}

// ProcessInsConfigDelete -
func (m *QueueManager) ProcessInsConfigDelete(insConfig *instanceconfig.Configuration) {
	m.Lock()
	queues, ok := m.queuesMap[insConfig.FuncKey]
	if !ok {
		m.Unlock()
		return
	}
	deleteQueue := make(map[string]*reqQueue)

	for resSpecKey, q := range queues {
		if q.resSpec.InvokeLabel == insConfig.InstanceLabel {
			deleteQueue[resSpecKey] = q
		}
	}
	for k, _ := range deleteQueue {
		delete(queues, k)
	}

	if len(queues) == 0 {
		delete(m.queuesMap, insConfig.FuncKey)
	}
	m.Unlock()

	resKeySum := ""
	for _, q := range deleteQueue {
		q.destroy()
		resKeySum += q.resSpec.String()
	}
	m.logger.Infof("recv insConfig delete event, delete queue success, funcKey: %s, reskey: %s",
		insConfig.FuncKey, resKeySum)
}

// AddPendingRequest -
func (m *QueueManager) AddPendingRequest(funcKey string, resSpec *resspeckey.ResSpecKey, pendingReq *PendingRequest) {
	_, ok := functionmeta.LoadFuncSpec(funcKey)
	if !ok {
		pendingReq.ResultChan <- &PendingResponse{
			Error: snerror.New(statuscode.FuncMetaNotFoundErrCode, statuscode.FuncMetaNotFoundErrMsg),
		}
		return
	}

	insConfig, ok := instanceconfigmanager.Load(funcKey, resSpec.InvokeLabel)
	if !ok {
		pendingReq.ResultChan <- &PendingResponse{
			Error: snerror.New(statuscode.FuncMetaNotFoundErrCode, "instance label not found"),
		}
		return
	}
	m.RLock()
	queues, ok := m.queuesMap[funcKey]
	if !ok {
		m.RUnlock()
		m.Lock()
		queues, ok = m.queuesMap[funcKey]
		if !ok {
			queues = make(map[string]*reqQueue)
			m.queuesMap[funcKey] = queues
		}
		m.Unlock()
		m.RLock()
	}

	queue, ok := queues[resSpec.String()]
	if !ok {
		m.RUnlock()
		m.Lock()
		queue, ok = queues[resSpec.String()]
		if !ok {
			queue = newQueue(funcKey, resSpec, insConfig)
			queues[resSpec.String()] = queue
		}
		m.Unlock()
		m.RLock()
	}

	queue.addPendingRequest(pendingReq)
	m.RUnlock()
}

// ProcessInstanceUpdate -
func (m *QueueManager) ProcessInstanceUpdate(instance *types.InstanceSpecification) {
	funcKey, ok := instance.CreateOptions[constant.FunctionKeyNote]
	if !ok {
		return
	}
	resSpec, err := resspeckey.GetResKeyFromStr(instance.CreateOptions[constant.ResourceSpecNote])
	if err != nil {
		return
	}

	m.RLock()
	queues, ok := m.queuesMap[funcKey]
	if !ok {
		m.RUnlock()
		return
	}

	queue, ok := queues[resSpec.String()]
	if !ok {
		m.RUnlock()
		return
	}
	queue.handleInstanceUpdate(instance)
	if queue.Len() == 0 {
		m.RUnlock()
		m.Lock()
		if queue.Len() == 0 {
			queue.destroy()
			delete(queues, resSpec.String())
			if len(queues) == 0 {
				delete(m.queuesMap, funcKey)
			}
		}
		m.Unlock()
		m.RLock()
	}

	m.RUnlock()
}

// ProcessQueueEmpty -
func (m *QueueManager) ProcessQueueEmpty(funcKey string, resSpec *resspeckey.ResSpecKey) {
	m.Lock()
	defer m.Unlock()
	queues, ok := m.queuesMap[funcKey]
	if !ok {
		return
	}

	queue, ok := queues[resSpec.String()]
	if !ok {
		return
	}
	queue.Lock()
	defer queue.Unlock()
	if queue.Len() != 0 {
		return
	}
	go queue.destroy()
	delete(queues, resSpec.String())
	if len(queues) == 0 {
		delete(m.queuesMap, funcKey)
	}
}

type reqQueue struct {
	funcKey   string
	resSpec   *resspeckey.ResSpecKey
	insConfig *instanceconfig.Configuration
	sync.RWMutex
	*queue.FifoQueue
	logger  api.FormatLogger
	stopCh  chan struct{}
	running atomic.Bool
}

func newQueue(funcKey string, resSpec *resspeckey.ResSpecKey,
	insConfig *instanceconfig.Configuration) *reqQueue {
	q := &reqQueue{
		funcKey:   funcKey,
		resSpec:   resSpec,
		RWMutex:   sync.RWMutex{},
		FifoQueue: queue.NewFifoQueue(nil),
		insConfig: insConfig,
		logger:    log.GetLogger().With(zap.Any("funcKey", funcKey), zap.Any("resSpecKey", resSpec.String())),
		stopCh:    make(chan struct{}),
	}

	q.running.Store(true)
	go q.timeoutLoop()
	return q
}

// PendingRequest -
type PendingRequest struct {
	CreatedTime     time.Time
	ScheduleTimeout time.Duration
	ResultChan      chan *PendingResponse
}

// PendingResponse -
type PendingResponse struct {
	Instance *types.InstanceSpecification
	Error    error
}

func (q *reqQueue) destroy() {
	q.Lock()
	if !q.running.Load() {
		q.Unlock()
		return
	}
	q.running.Store(false)
	utils.SafeCloseChannel(q.stopCh)
	q.Unlock()
}

func (q *reqQueue) addPendingRequest(req *PendingRequest) {
	q.Lock()
	defer q.Unlock()
	if q.Len() >= 100 { // magic number
		req.ResultChan <- &PendingResponse{
			Instance: nil,
			Error:    snerror.New(statuscode.FrontendStatusTooManyRequests, "queue has too many requests"),
		}
	}
	err := q.PushBack(req)
	if err != nil {
		return
	}
	if q.Len() == 1 && coldStartProvider != nil {
		go func() {
			err := coldStartProvider.ColdStart(q.funcKey, *q.resSpec, &q.insConfig.NuwaRuntimeInfo)
			if err != nil {
				q.clearQueueWithError(err)
			}
		}()
	}
}

func (q *reqQueue) handleInstanceUpdate(_ *types.InstanceSpecification) {
	q.Lock()
	defer q.Unlock()
	for {
		if q.Len() == 0 {
			break
		}
		pendingReq, ok := q.Front().(*PendingRequest)
		if !ok {
			q.PopFront()
			continue
		}
		instance := instancemanager.GetGlobalInstanceScheduler().GetRandomInstanceWithoutUnexpectedInstance(q.funcKey,
			q.resSpec.String(), nil, q.logger)
		if instance == nil {
			break
		}
		pendingReq.ResultChan <- &PendingResponse{
			Instance: instance,
			Error:    nil,
		}
		q.PopFront()
	}

	if q.Len() == 0 {
		go queueManager.ProcessQueueEmpty(q.funcKey, q.resSpec)
	}
}

func (q *reqQueue) clearQueueWithError(err error) {
	q.Lock()
	defer q.Unlock()
	for q.Len() != 0 {
		pendingReq, ok := q.PopFront().(*PendingRequest)
		if !ok {
			continue
		}
		pendingReq.ResultChan <- &PendingResponse{
			Instance: nil,
			Error:    err,
		}
	}
	go queueManager.ProcessQueueEmpty(q.funcKey, q.resSpec)
}

// timeoutLoop -
func (q *reqQueue) timeoutLoop() {
	timeoutTicker := time.NewTicker(1 * time.Second)
	defer func() {
		timeoutTicker.Stop()
		q.logger.Infof("exit queue")
	}()

	for {
		select {
		case <-q.stopCh:
			q.logger.Infof("recv stop event")
			err := snerror.New(statuscode.FuncMetaNotFoundErrCode, statuscode.FuncMetaNotFoundErrMsg)
			q.clearQueueWithError(err)
			return
		case <-timeoutTicker.C:
		}
		q.checkQueueTimeout()
	}
}

func (q *reqQueue) checkQueueTimeout() {
	q.Lock()
	defer q.Unlock()
	for q.Len() != 0 {
		pendingReq, ok := q.Front().(*PendingRequest)
		if !ok {
			q.PopFront()
			continue
		}
		waitTime := time.Now().Sub(pendingReq.CreatedTime)
		if waitTime.Milliseconds() > pendingReq.ScheduleTimeout.Milliseconds() {
			err := snerror.New(statuscode.ErrAcquireTimeoutCode, statuscode.InsThdReqTimeoutErrMsg)
			pendingReq.ResultChan <- &PendingResponse{
				Instance: nil,
				Error:    err,
			}
			q.PopFront()
			continue
		}
		break
	}
}
