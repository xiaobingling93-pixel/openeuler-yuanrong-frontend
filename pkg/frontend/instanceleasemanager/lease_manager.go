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

// Package instanceleasemanager for message process
package instanceleasemanager

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/constant"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/snerror"
	"frontend/pkg/common/faas_common/tls"
	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/frontend/common/httputil"
	"frontend/pkg/frontend/common/util"
	"frontend/pkg/frontend/config"
	"frontend/pkg/frontend/schedulerproxy"
)

const (
	renewAction                = "retain"
	releaseAction              = "release"
	idleHoldTime               = 100 // millisecond
	defaultAcquireLeaseTimeout = 120 // second
	beforeRetainTime           = 100 // millisecond
	defaultMapSize             = 16
	callSchedulerPath          = "/invoke"
)

var (
	instanceManager *Manager
	once            sync.Once
)

// InstanceLease holds a lease of an invokable instanceID acquired from instance scheduler
type InstanceLease struct {
	types.InstanceAllocationInfo
	stateID       string
	reportRecord  *ReportRecord
	acquireOption util.AcquireOption
	claimTime     time.Time
	available     bool
	releaseCh     chan struct{}
	releaseTimer  *time.Timer
	sync.RWMutex
}

func (il *InstanceLease) report(reset bool) InstanceReport {
	return il.reportRecord.report(reset)
}

func (il *InstanceLease) claim() bool {
	il.Lock()
	if !il.available {
		il.Unlock()
		return false
	}
	il.available = false
	il.claimTime = time.Now()
	if il.releaseTimer != nil {
		il.releaseTimer.Stop()
		il.releaseTimer = nil
	}
	il.Unlock()
	return true
}

func (il *InstanceLease) free(abnormal bool, record bool) {
	if abnormal {
		il.reportRecord.recordAbnormal()
		il.releaseCh <- struct{}{}
		return
	}
	var claimTime time.Time
	il.Lock()
	// if claimTime is zero then it's already freed and should not record request
	if !il.claimTime.IsZero() {
		claimTime = il.claimTime
		il.claimTime = time.Time{}
	}
	il.available = true
	il.Unlock()
	if record && !claimTime.IsZero() {
		il.reportRecord.recordRequest(time.Now().Sub(claimTime))
	}
	il.Lock()
	if il.releaseTimer == nil {
		il.releaseTimer = time.AfterFunc(idleHoldTime*time.Millisecond, func() {
			if il.claim() {
				il.releaseCh <- struct{}{}
			}
		})
	}
	il.Unlock()
}

type instanceLeaseRecord struct {
	lease   *InstanceLease
	element *list.Element
}

// InstanceLeasePool stores instance leases
type InstanceLeasePool struct {
	funcKey       string
	idleLeaseList *list.List
	leaseRecord   map[string]*instanceLeaseRecord
	sync.RWMutex
}

// GetInstanceManager creates Manager
func GetInstanceManager() *Manager {
	once.Do(func() {
		instanceManager = &Manager{
			leasePools: make(map[string]*InstanceLeasePool, defaultMapSize),
		}
	})
	return instanceManager
}

func newInstanceLeasePool(funcKey string) *InstanceLeasePool {
	return &InstanceLeasePool{
		funcKey:       funcKey,
		idleLeaseList: list.New(),
		leaseRecord:   make(map[string]*instanceLeaseRecord, defaultMapSize),
	}
}

func (ip *InstanceLeasePool) invokeHandler(schedulerID, traceID string,
	args []*api.Arg, timeout int64) (string, error) {
	scheduler, err := schedulerproxy.Proxy.GetSchedulerByInstanceName(schedulerID, traceID)
	if err != nil {
		return "", err
	}
	if len(scheduler.Address) == 0 {
		return "", errors.New("scheduler address is empty")
	}
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)
	err = prepareSchedulerRequest(req, scheduler.Address, args, traceID)
	if err != nil {
		return "", err
	}
	err = httputil.GetSchedulerClient().DoTimeout(req, resp, time.Duration(timeout)*time.Second)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("call scheduler failed,http code %d", resp.StatusCode())
	}
	return string(resp.Body()), nil
}
func prepareSchedulerRequest(schedulerReq *fasthttp.Request, dstHost string,
	args []*api.Arg, traceID string) error {
	schedulerReq.SetRequestURI(callSchedulerPath)
	schedulerReq.Header.SetMethod(http.MethodPost)
	schedulerReq.Header.ResetConnectionClose()
	schedulerReq.SetHost(dstHost)
	schedulerReq.URI().SetScheme(tls.GetURLScheme(config.GetConfig().HTTPSConfig.HTTPSEnable))
	schedulerReq.Header.Set(constant.HeaderTraceID, traceID)
	httputil.AddAuthorizationHeaderForFG(schedulerReq)
	argsData, err := json.Marshal(args)
	if err != nil {
		return err
	}
	schedulerReq.SetBody(argsData)
	return nil
}

func (ip *InstanceLeasePool) invokeScheduler(schedulerID, traceID string, args []*api.Arg,
	timeout int64) (*types.InstanceResponse, error) {
	if timeout <= 0 {
		timeout = defaultAcquireLeaseTimeout
	}
	responseData, err := ip.invokeHandler(schedulerID, traceID, args, timeout)
	if err != nil {
		log.GetLogger().Errorf("invoke to instance scheduler %s encounters error %s "+
			"traceID %s", schedulerID, err.Error(), traceID)
		return nil, err
	}
	instanceResponse := &types.InstanceResponse{}
	err = json.Unmarshal([]byte(responseData), instanceResponse)
	if err != nil {
		log.GetLogger().Errorf("failed to marshal instance response from scheduler %s error %s traceID %s",
			schedulerID, err.Error(), traceID)
		return nil, err
	}
	return instanceResponse, nil
}

func (ip *InstanceLeasePool) acquireHandler(funcKey string, stateID string, option util.AcquireOption) (*InstanceLease,
	snerror.SNError) {
	logger := log.GetLogger().With(zap.Any("traceID", option.TraceID), zap.Any("stateID", stateID))
	logger.Infof("acquireHandler for %s, stateID %s, faasscheduler: %s-%s",
		funcKey, stateID, option.SchedulerFuncKey, option.SchedulerID)
	args := createInvokeArgs(option, funcKey, stateID)
	response, err := ip.invokeScheduler(option.SchedulerID, option.TraceID, args, option.Timeout)
	if err != nil {
		logger.Errorf("failed to acquire instance for function %s from scheduler %s error %s traceID %s",
			funcKey, option.SchedulerID, err.Error(), option.TraceID)
		return nil, snerror.NewWithError(constant.InsAcquireFailedErrorCode, err)
	}
	if response.ErrorCode != constant.InsReqSuccessCode {
		logger.Errorf("failed to acquire instance for function %s from scheduler %s error %s traceID %s",
			funcKey, option.SchedulerID, response.ErrorMessage, option.TraceID)
		return nil, snerror.New(response.ErrorCode, response.ErrorMessage)
	}
	return &InstanceLease{
		InstanceAllocationInfo: response.InstanceAllocationInfo,
		stateID:                stateID,
		reportRecord:           &ReportRecord{},
		acquireOption:          option,
		available:              true,
		releaseCh:              make(chan struct{}, 1),
	}, nil
}

func createInvokeArgs(option util.AcquireOption, funcKey string, stateID string) []*api.Arg {
	var invokeArgs []*api.Arg
	var acquireOps []byte
	instanceRequirement := make(map[string][]byte, 3)
	if stateID == "" {
		acquireOps = []byte(fmt.Sprintf("acquire#%s", funcKey))
	} else {
		acquireOps = []byte(fmt.Sprintf("acquire#%s;%s", funcKey, stateID))
	}

	if option.DesignateInstanceID == "" {
		resourcesData, err := json.Marshal(option.ResourceSpecs)
		instanceRequirement[constant.InstanceRequirementResourcesKey] = resourcesData
		if err != nil {
			log.GetLogger().Errorf("failed to marshal resource when acquire %s instance, error %s",
				funcKey, err.Error())
		}
	} else {
		log.GetLogger().Infof("acquire specified instance[%s] lease, traceID %s", option.DesignateInstanceID,
			option.TraceID)
		instanceRequirement[constant.InstanceRequirementInsIDKey] = []byte(option.DesignateInstanceID)
	}

	callerPodName := getPodName()
	log.GetLogger().Infof("caller pod name is %s", callerPodName)
	if callerPodName != "" {
		instanceRequirement[constant.InstanceCallerPodName] = []byte(callerPodName)
	}

	if option.TrafficLimited {
		instanceRequirement[constant.InstanceTrafficLimited] = []byte("true")
	}

	insRequirementBytes, err := json.Marshal(instanceRequirement)
	if err != nil {
		log.GetLogger().Errorf("failed to marshal resource when acquire %s instance, error %s",
			funcKey, err.Error())
	}
	acquireArg := &api.Arg{Type: api.Value, Data: acquireOps}
	instanceArg := &api.Arg{Type: api.Value, Data: insRequirementBytes}
	traceID := &api.Arg{Type: api.Value, Data: []byte(option.TraceID)}
	invokeArgs = []*api.Arg{acquireArg, instanceArg, traceID}
	return invokeArgs
}

func getPodName() string {
	podName := os.Getenv(constant.HostNameEnvKey)
	if os.Getenv(constant.PodNameEnvKey) != "" {
		podName = os.Getenv(constant.PodNameEnvKey)
	}
	return podName
}

func (ip *InstanceLeasePool) processRenewAndRelease(action, leaseID string, option util.AcquireOption,
	report InstanceReport) (*types.InstanceResponse, error) {
	reportData, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	actionArg := &api.Arg{
		Type: api.Value,
		Data: []byte(fmt.Sprintf("%s#%s", action, leaseID)),
	}
	reportArg := &api.Arg{
		Type: api.Value,
		Data: reportData,
	}
	traceID := &api.Arg{Type: api.Value, Data: []byte(option.TraceID)}
	response, err := ip.invokeScheduler(option.SchedulerID, option.TraceID,
		[]*api.Arg{actionArg, reportArg, traceID}, option.Timeout)
	if err == nil && response.ErrorCode != constant.InsReqSuccessCode {
		err = fmt.Errorf("code %d, message %s", response.ErrorCode, response.ErrorMessage)
	}
	return response, err
}

func (ip *InstanceLeasePool) renewHandler(leaseID string, option util.AcquireOption,
	report InstanceReport) (*types.InstanceResponse, error) {
	rsp, err := ip.processRenewAndRelease(renewAction, leaseID, option, report)
	if err != nil {
		log.GetLogger().Errorf("failed to renew instance lease %s from scheduler %s for function %s error %s "+
			"traceID %s", leaseID, option.SchedulerID, ip.funcKey, err.Error(), option.TraceID)
		return nil, err
	}
	return rsp, nil
}

func (ip *InstanceLeasePool) releaseHandler(leaseID string, option util.AcquireOption,
	report InstanceReport) error {
	_, err := ip.processRenewAndRelease(releaseAction, leaseID, option, report)
	if err != nil {
		log.GetLogger().Errorf("failed to release instance lease %s from scheduler %s for function %s error %s "+
			"traceID %s", leaseID, option.SchedulerID, ip.funcKey, err.Error(), option.TraceID)
		return err
	}
	return nil
}

func (ip *InstanceLeasePool) removeLease(leaseID string) {
	ip.Lock()
	record, exist := ip.leaseRecord[leaseID]
	if !exist {
		ip.Unlock()
		return
	}
	delete(ip.leaseRecord, leaseID)
	if ip.idleLeaseList.Len() != 0 && record.element != nil {
		ip.idleLeaseList.Remove(record.element)
	}
	ip.Unlock()
}

func (ip *InstanceLeasePool) handleLeaseLifeCycle(lease *InstanceLease, timer *time.Timer) {
	if timer == nil {
		log.GetLogger().Warnf("timer is nil,not need to start Lease life cycle for lease %s", lease.ThreadID)
		return
	}
	defer func() {
		timer.Stop()
	}()
	for {
		release := false
		select {
		case _, ok := <-timer.C:
			if !ok {
				log.GetLogger().Warnf("release timer is closed for lease %s", lease.ThreadID)
			}
			timer.Reset(time.Duration(lease.LeaseInterval)*time.Millisecond - beforeRetainTime*time.Millisecond)
			if !ok {
				log.GetLogger().Warnf("timer is closed for lease %s", lease.ThreadID)
			}
			if lease.claim() {
				release = true
			}
		case _, ok := <-lease.releaseCh:
			if !ok {
				log.GetLogger().Warnf("release channel is closed for lease %s", lease.ThreadID)
			}
			release = true
		}
		if release {
			ip.removeLease(lease.ThreadID)
			if err := ip.releaseHandler(lease.ThreadID, lease.acquireOption, lease.report(true)); err != nil {
				log.GetLogger().Errorf("failed to release lease %s for function %s", lease.ThreadID, ip.funcKey)
			}
			return
		}
		rsp, err := ip.renewHandler(lease.ThreadID, lease.acquireOption, lease.report(false))
		if err != nil {
			log.GetLogger().Warnf("renew failed lease %s for function %s", lease.ThreadID, ip.funcKey)
			ip.removeLease(lease.ThreadID)
			return
		}
		lease.LeaseInterval = rsp.LeaseInterval
	}
}

func (ip *InstanceLeasePool) traverseIdleLeaseList(stateID string, option util.AcquireOption) *InstanceLease {
	ip.RLock()
	idleLeaseList := ip.idleLeaseList
	ip.RUnlock()
	var lease *InstanceLease
	for i := idleLeaseList.Front(); i != nil; i = i.Next() {
		l, ok := i.Value.(*InstanceLease)
		if !ok {
			ip.Lock()
			ip.idleLeaseList.Remove(i)
			ip.Unlock()
			continue
		}
		// If the function signature has changed, the lease is not reused.
		if l.acquireOption.FuncSig != option.FuncSig {
			log.GetLogger().Warnf("lease %s has a different signature %s which should be %s for function %s,traceID %s",
				l.ThreadID, l.acquireOption.FuncSig, option.FuncSig, ip.funcKey, option.TraceID)
			ip.Lock()
			ip.idleLeaseList = list.New()
			ip.Unlock()
			break
		}
		if stateID != l.stateID {
			continue
		}
		if option.DesignateInstanceID != "" && l.InstanceID != option.DesignateInstanceID {
			continue
		}
		if l.claim() {
			lease = l
			ip.Lock()
			ip.idleLeaseList.Remove(i)
			ip.Unlock()
			break
		}
	}
	return lease
}

func (ip *InstanceLeasePool) acquireInstanceLease(stateID string, option util.AcquireOption) (*InstanceLease,
	snerror.SNError) {
	lease := ip.traverseIdleLeaseList(stateID, option)
	if lease != nil {
		return lease, nil
	}
	lease, snError := ip.acquireHandler(ip.funcKey, stateID, option)
	if snError != nil {
		return nil, snError
	}
	lease.claim()
	ip.RLock()
	_, exist := ip.leaseRecord[lease.ThreadID]
	ip.RUnlock()
	if exist {
		log.GetLogger().Errorf("acquired lease %s already exist for function %s traceID %s", lease.ThreadID,
			ip.funcKey, option.TraceID)
		// acquired a repeated lease, should acquire a new lease
		return nil, snerror.New(constant.InsAcquireLeaseExistErrorCode, "lease already exist")
	}
	ip.Lock()
	ip.leaseRecord[lease.ThreadID] = &instanceLeaseRecord{
		lease: lease,
	}
	ip.Unlock()
	timer := time.NewTimer(time.Duration(lease.LeaseInterval)*time.Millisecond - beforeRetainTime*time.Millisecond)
	go ip.handleLeaseLifeCycle(lease, timer)
	log.GetLogger().Infof("succeed to acquire lease %s for function %s from scheduler %s",
		lease.ThreadID, ip.funcKey, option.SchedulerID)
	return lease, nil
}

func (ip *InstanceLeasePool) releaseInstanceLease(leaseID string, abnormal bool) {
	ip.Lock()
	record, exist := ip.leaseRecord[leaseID]
	if !exist {
		ip.Unlock()
		return
	}
	if !abnormal {
		record.element = ip.idleLeaseList.PushBack(record.lease)
	}
	ip.Unlock()
	record.lease.free(abnormal, true)
}

// Manager manges
type Manager struct {
	leasePools map[string]*InstanceLeasePool
	sync.RWMutex
}

// ClearFuncLeasePools -
func (im *Manager) ClearFuncLeasePools(funcKey string) {
	log.GetLogger().Infof("function %s is delete,clean lease pools", funcKey)
	im.Lock()
	defer im.Unlock()
	leasePool, exist := im.leasePools[funcKey]
	if !exist {
		log.GetLogger().Infof("function %s leasePool is not exist,no need to delete", funcKey)
		return
	}
	leasePool.Lock()
	for _, leaseRecord := range leasePool.leaseRecord {
		leaseRecord.lease.releaseCh <- struct{}{}
	}
	leasePool.Unlock()
	delete(im.leasePools, funcKey)
}

// AcquireInstanceLease -
func (im *Manager) AcquireInstanceLease(funcKey, stateID string,
	option util.AcquireOption) (*InstanceLease, snerror.SNError) {
	log.GetLogger().Infof("acquire instance lease for function %s state %s from instance "+
		"scheduler %s traceID %s", funcKey, stateID, option.SchedulerID, option.TraceID)
	im.Lock()
	leasePool, exist := im.leasePools[funcKey]
	if !exist {
		leasePool = newInstanceLeasePool(funcKey)
		im.leasePools[funcKey] = leasePool
	}
	im.Unlock()
	return leasePool.acquireInstanceLease(stateID, option)
}

// AcquireInstanceAllocation -
func (im *Manager) AcquireInstanceAllocation(funcKey string, stateID string,
	option util.AcquireOption) (*types.InstanceAllocationInfo, snerror.SNError) {
	lease, err := im.AcquireInstanceLease(funcKey, stateID, option)
	if err != nil {
		return &types.InstanceAllocationInfo{}, err
	}
	return &lease.InstanceAllocationInfo, nil
}

// ReleaseInstanceAllocation -
func (im *Manager) ReleaseInstanceAllocation(allocation *types.InstanceAllocationInfo, abnormal bool, traceID string) {
	if allocation == nil || allocation.ThreadID == "" {
		return
	}
	log.GetLogger().Infof("release instance lease %s for function %s,abnormal %t,traceID %s",
		allocation.ThreadID, allocation.FuncKey, abnormal, traceID)
	im.RLock()
	leasePool, exist := im.leasePools[allocation.FuncKey]
	if !exist {
		log.GetLogger().Errorf("funcKey %s is not in lease pools!", allocation.FuncKey)
		im.RUnlock()
		return
	}
	im.RUnlock()
	leasePool.releaseInstanceLease(allocation.ThreadID, abnormal)
}
