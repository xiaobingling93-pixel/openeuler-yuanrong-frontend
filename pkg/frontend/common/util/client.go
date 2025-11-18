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

// Package util -
package util

import (
	"fmt"

	"yuanrong.org/kernel/runtime/libruntime/api"

	"frontend/pkg/common/faas_common/types"
	"frontend/pkg/common/faas_common/utils"
	"frontend/pkg/frontend/common/httpconstant"
)

const (
	maxInvokeRetries = 5
)

type invokerLibruntime interface {
	CreateInstance(funcMeta api.FunctionMeta, args []api.Arg,
		invokeOpt api.InvokeOptions) (instanceID string, err error)
	InvokeByInstanceId(funcMeta api.FunctionMeta, instanceID string, args []api.Arg,
		invokeOpt api.InvokeOptions) (returnObjectID string, err error)
	InvokeByFunctionName(funcMeta api.FunctionMeta, args []api.Arg,
		invokeOpt api.InvokeOptions) (returnObjectID string, err error)
	AcquireInstance(state string, funcMeta api.FunctionMeta,
		acquireOpt api.InvokeOptions) (api.InstanceAllocation, error)

	ReleaseInstance(allocation api.InstanceAllocation, stateID string, abnormal bool, option api.InvokeOptions)
	Kill(instanceID string, signal int, payload []byte) (err error)

	CreateInstanceRaw(createReqRaw []byte) (createRespRaw []byte, err error)
	InvokeByInstanceIdRaw(invokeReqRaw []byte) (resultRaw []byte, err error)
	KillRaw(killReqRaw []byte) (killRespRaw []byte, err error)

	SaveState(state []byte) (stateID string, err error)
	LoadState(checkpointID string) (state []byte, err error)

	Exit(code int, message string)

	KVSet(key string, value []byte, param api.SetParam) (err error)
	KVSetWithoutKey(value []byte, param api.SetParam) (key string, err error)
	KVGet(key string, timeoutms uint) (value []byte, err error)
	KVGetMulti(keys []string, timeoutms uint) (values [][]byte, err error)
	KVDel(key string) (err error)
	KVDelMulti(keys []string) (failedKeys []string, err error)

	SetTraceID(traceID string)

	Put(objectID string, value []byte, param api.PutParam, nestedObjectIDs ...string) (err error)
	Get(objectIDs []string, timeoutMs int) (data [][]byte, err error)
	GIncreaseRef(objectIDs []string, remoteClientID ...string) (failedIDs []string, err error)
	GDecreaseRef(objectIDs []string, remoteClientID ...string) (failedIDs []string, err error)
	GetAsync(objectID string, cb api.GetAsyncCallback)

	GetFormatLogger() api.FormatLogger
	GetCredential() api.Credential
	SetTenantID(tenantID string) error
	IsHealth() bool
	IsDsHealth() bool
}

var clientLibruntime invokerLibruntime

// SetAPIClientLibruntime set the client provided by the runtime
func SetAPIClientLibruntime(rt invokerLibruntime) {
	clientLibruntime = rt
}

// AcquireOption holds the options for acquireInstance
type AcquireOption struct {
	DesignateInstanceID string
	SchedulerFuncKey    string
	SchedulerID         string
	RequestID           string
	TraceID             string
	FuncSig             string
	ResourceSpecs       map[string]int64
	Timeout             int64
	TrafficLimited      bool
}

// InvokeRequest -
type InvokeRequest struct {
	Function         string
	InstanceID       string
	TraceID          string
	Args             []*api.Arg
	SchedulerID      string
	SchedulerFuncKey string
	RequestID        string
	FuncSig          string
	PoolLabel        string
	InstanceSession  *types.InstanceSessionConfig
	InvokeTag        map[string]string
	InstanceLabel    string
	ReturnObjectIDs  []string
	ResourceSpecs    map[string]int64
	AcquireTimeout   int64
	InvokeTimeout    int64
	TrafficLimited   bool
	RetryTimes       int
	BusinessType     string
	TenantID         string
}

// Client is used to invoke an instance and wait for its response
type Client interface {
	AcquireInstance(functionKey string, req AcquireOption) (*types.InstanceAllocationInfo, error)
	ReleaseInstance(allocation *types.InstanceAllocationInfo, abnormal bool)
	Invoke(req InvokeRequest) ([]byte, error)
	InvokeByName(req InvokeRequest) ([]byte, error)
	CreateInstanceRaw(createReq []byte) ([]byte, error)
	InvokeInstanceRaw(invokeReq []byte) ([]byte, error)
	KillRaw(killReq []byte) ([]byte, error)
	CreateInstanceByLibRt(funcMeta api.FunctionMeta, args []api.Arg,
		invokeOpt api.InvokeOptions) (instanceID string, err error)
	KillByLibRt(instanceID string, signal int, payload []byte) (err error)
	IsHealth() bool
	IsDsHealth() bool
}

// NewClient return a client used to invoke other functions
func NewClient() Client {
	return newDefaultClientLibruntime(clientLibruntime)
}

func newDefaultClientLibruntime(librtcli invokerLibruntime) *defaultClient {
	return &defaultClient{clientLibruntime: librtcli}
}

type defaultClient struct {
	clientLibruntime invokerLibruntime
}

func (c *defaultClient) AcquireInstance(functionKey string, req AcquireOption) (*types.InstanceAllocationInfo, error) {
	var err error
	var instanceAllocation api.InstanceAllocation
	functionMeta := api.FunctionMeta{
		FuncID: functionKey,
		Sig:    req.FuncSig,
		Name:   &req.DesignateInstanceID,
		Api:    api.FaaSApi,
	}
	option := convertAcquireOption(req)
	if instanceAllocation, err = c.clientLibruntime.AcquireInstance("", functionMeta, option); err != nil {
		return nil, err
	}
	return &types.InstanceAllocationInfo{
		FuncKey:       instanceAllocation.FuncKey,
		FuncSig:       instanceAllocation.FuncSig,
		InstanceID:    instanceAllocation.InstanceID,
		ThreadID:      instanceAllocation.LeaseID,
		LeaseInterval: instanceAllocation.LeaseInterval,
	}, nil
}

func (c *defaultClient) ReleaseInstance(allocation *types.InstanceAllocationInfo, abnormal bool) {
	instanceAllocation := api.InstanceAllocation{
		FuncKey:       allocation.FuncKey,
		FuncSig:       allocation.FuncSig,
		InstanceID:    allocation.InstanceID,
		LeaseID:       allocation.ThreadID,
		LeaseInterval: allocation.LeaseInterval,
	}
	c.clientLibruntime.ReleaseInstance(instanceAllocation, "", abnormal, api.InvokeOptions{})
}

func deepCopyArgs(args []*api.Arg, tenantID string) []api.Arg {
	rtArgs := make([]api.Arg, len(args))
	for idx, val := range args {
		rtArgs[idx] = api.Arg{
			Type:     val.Type,
			Data:     val.Data,
			TenantID: tenantID,
		}
	}
	return rtArgs
}

func (c *defaultClient) Invoke(req InvokeRequest) ([]byte, error) {
	wait := make(chan struct{}, 1)
	var (
		res         []byte
		resErr, err error
	)
	funcMeta := api.FunctionMeta{FuncID: req.Function, Api: api.FaaSApi}
	funcArgs := deepCopyArgs(req.Args, "")
	invokeOpts := api.InvokeOptions{TraceID: req.TraceID, CustomExtensions: req.InvokeTag, RetryTimes: req.RetryTimes,
		Timeout: int(req.InvokeTimeout)}
	var objID string
	objID, err = c.clientLibruntime.InvokeByInstanceId(funcMeta, req.InstanceID, funcArgs, invokeOpts)

	c.clientLibruntime.GetAsync(objID, func(result []byte, err error) {
		res = result
		resErr = err
		wait <- struct{}{}
		if _, err := c.clientLibruntime.GDecreaseRef([]string{objID}); err != nil {
			fmt.Printf("failed to decrease object ref,err: %s", err.Error())
		}
	})
	if err != nil {
		return res, fmt.Errorf("failed to invoke by instance id request, req: %#v, err: %s", req, err.Error())
	}
	<-wait
	return res, resErr
}

func convertInvokeOption(req InvokeRequest) api.InvokeOptions {
	cpu, mem, customRes := LibruntimeCustomResources(req.ResourceSpecs)
	invokeLabels := map[string]string{}
	if req.InstanceLabel != "" {
		invokeLabels[httpconstant.HeaderInstanceLabel] = req.InstanceLabel
	}
	invokeOpt := api.InvokeOptions{
		Cpu:                  cpu,
		Memory:               mem,
		InvokeLabels:         invokeLabels,
		CustomResources:      customRes,
		CustomExtensions:     req.InvokeTag,
		SchedulerFunctionID:  req.SchedulerFuncKey,
		SchedulerInstanceIDs: []string{req.SchedulerID},
		TraceID:              req.TraceID,
		Timeout:              int(req.InvokeTimeout),
		AcquireTimeout:       int(req.AcquireTimeout),
	}
	if req.InstanceSession != nil {
		invokeOpt.InstanceSession = &api.InstanceSessionConfig{
			SessionID:   req.InstanceSession.SessionID,
			SessionTTL:  req.InstanceSession.SessionTTL,
			Concurrency: req.InstanceSession.Concurrency,
		}
	}
	return invokeOpt
}

func convertAcquireOption(req AcquireOption) api.InvokeOptions {
	cpu, mem, customRes := LibruntimeCustomResources(req.ResourceSpecs)
	invokeOpt := api.InvokeOptions{
		Cpu:                  cpu,
		Memory:               mem,
		CustomResources:      customRes,
		SchedulerFunctionID:  req.SchedulerFuncKey,
		SchedulerInstanceIDs: []string{req.SchedulerID},
		TraceID:              req.TraceID,
		RetryTimes:           maxInvokeRetries,
		Timeout:              int(req.Timeout),
		AcquireTimeout:       int(req.Timeout),
		TrafficLimited:       req.TrafficLimited,
	}
	return invokeOpt
}

// InvokeByName -
func (c *defaultClient) InvokeByName(req InvokeRequest) ([]byte, error) {
	wait := make(chan struct{}, 1)
	var (
		res         []byte
		resErr, err error
	)
	funcMeta := api.FunctionMeta{
		FuncID:    req.Function,
		Name:      &req.InstanceID,
		Sig:       req.FuncSig,
		Api:       utils.GetAPIType(req.BusinessType),
		PoolLabel: req.PoolLabel,
	}
	funcArgs := deepCopyArgs(req.Args, req.TenantID)
	invokeOpt := convertInvokeOption(req)
	var objID string
	objID, err = c.clientLibruntime.InvokeByFunctionName(funcMeta, funcArgs, invokeOpt)
	if err != nil {
		return nil, err
	}
	c.clientLibruntime.GetAsync(objID, func(result []byte, err error) {
		res = result
		resErr = err
		wait <- struct{}{}
		if _, err := c.clientLibruntime.GDecreaseRef([]string{objID}); err != nil {
			fmt.Printf("failed to decrease object ref,err: %s", err.Error())
		}
	})
	<-wait
	return res, resErr
}

func (c *defaultClient) CreateInstanceRaw(createReq []byte) ([]byte, error) {
	resp, err := c.clientLibruntime.CreateInstanceRaw(createReq)
	return resp, err
}

func (c *defaultClient) InvokeInstanceRaw(invokeReq []byte) ([]byte, error) {
	notify, err := c.clientLibruntime.InvokeByInstanceIdRaw(invokeReq)
	return notify, err
}

func (c *defaultClient) KillByLibRt(instanceID string, signal int, payload []byte) error {
	return c.clientLibruntime.Kill(instanceID, signal, payload)
}

func (c *defaultClient) CreateInstanceByLibRt(
	funcMeta api.FunctionMeta,
	args []api.Arg,
	invokeOpt api.InvokeOptions,
) (string, error) {
	return c.clientLibruntime.CreateInstance(funcMeta, args, invokeOpt)
}

func (c *defaultClient) KillRaw(killReq []byte) ([]byte, error) {
	resp, err := c.clientLibruntime.KillRaw(killReq)
	return resp, err
}

func (c *defaultClient) IsHealth() bool {
	return c.clientLibruntime.IsHealth()
}

func (c *defaultClient) IsDsHealth() bool {
	return c.clientLibruntime.IsHealth()
}
