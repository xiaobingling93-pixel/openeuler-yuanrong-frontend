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

// Package selfregister -
package selfregister

import (
	"fmt"
	"os"

	"frontend/pkg/common/faas_common/etcd3"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/frontend/config"
)

// RegisterFrontendInstanceToEtcd -
func RegisterFrontendInstanceToEtcd(stopCh <-chan struct{}) error {
	instanceKey, err := getInstanceKeyWithClusterID()
	if err != nil {
		return err
	}
	register := etcd3.EtcdRegister{
		EtcdClient:  etcd3.GetMetaEtcdClient(),
		InstanceKey: instanceKey,
		Value:       "active",
		StopCh:      stopCh,
	}
	err = register.Register()
	if err != nil {
		return err
	}
	return nil
}

func getInstanceKeyWithClusterID() (string, error) {
	clusterID, err := getClusterID()
	if err != nil {
		return "", err
	}
	nodeIP := getNodeIP()
	podName := os.Getenv("POD_NAME")
	err = validateEnvs(nodeIP, podName)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("/sn/frontend/instances/%s/%s/%s", clusterID, nodeIP, podName)
	return key, nil
}

func getClusterID() (string, error) {
	clusterID := config.GetConfig().ClusterID
	if clusterID == "" {
		clusterID = os.Getenv("CLUSTER_ID")
	}
	if clusterID == "" {
		log.GetLogger().Errorf("get cluster failed, can not register frontend info to etcd")
		return "", fmt.Errorf("get cluster failed")
	}
	return clusterID, nil
}

func getNodeIP() string {
	nodeIP := os.Getenv("HOST_IP")
	if nodeIP == "" {
		nodeIP = os.Getenv("NODE_IP")
	}
	return nodeIP
}

func getInstanceKey() (string, error) {
	nodeIP := getNodeIP()
	podName := os.Getenv("POD_NAME")
	err := validateEnvs(nodeIP, podName)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("/sn/frontend/instances/%s/%s", nodeIP, podName)
	return key, nil
}

func validateEnvs(nodeIP, podName string) error {
	if nodeIP == "" {
		log.GetLogger().Errorf("can not find NODE_IP env, can not register frontend info to etcd")
		return fmt.Errorf("NODE_IP env not found")
	}
	if podName == "" {
		log.GetLogger().Errorf("can not find POD_NAME env, can not register frontend info to etcd")
		return fmt.Errorf("POD_NAME env not found")
	}
	return nil
}
