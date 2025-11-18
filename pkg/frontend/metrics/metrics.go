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

// Package metrics report to redis for the monitor market
package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/common/faas_common/logger/log"
	"frontend/pkg/common/faas_common/redisclient"
	"frontend/pkg/frontend/config"
)

const (
	reportTickerTime = 1 * time.Minute
	metricsKeyTTL    = 60 * time.Minute

	redisRetryTimes    = 3
	redisRetryInterval = 100 * time.Millisecond

	bufferSize = 10000

	listLimit = 3

	logLimitFrequency = 1000000
)

var (
	metricsChan = make(chan RequestStatistics, bufferSize)

	requestInfoRedisKey = fmt.Sprintf("/sn/metrics/requestinfo/%s/%s",
		os.Getenv("CLUSTER_ID"), os.Getenv("POD_NAME"))

	statistics = &totalStatistics{}

	reachMaxChanCount = 0
)

// RequestStatistics -
type RequestStatistics struct {
	ErrorFlag     bool
	TotalDelay    float64
	FrontendDelay float64
	BusDelay      float64
	RuntimeDelay  float64
}

// totalStatistics -
type totalStatistics struct {
	lock          sync.RWMutex
	count         int64
	errCount      int64
	totalDelay    float64
	frontendDelay float64
	busDelay      float64
	runtimeDelay  float64
}

// RequestInfoMetrics request info metrics
type RequestInfoMetrics struct {
	TimeStamp     int64   `json:"timeStamp,omitempty"`
	Count         int64   `json:"count,omitempty"`
	ErrCount      int64   `json:"errCount,omitempty"`
	TotalDelay    float64 `json:"totalDelay,omitempty"`
	FrontendDelay float64 `json:"frontendDelay,omitempty"`
	BusDelay      float64 `json:"busDelay,omitempty"`
	RuntimeDelay  float64 `json:"runtimeDelay,omitempty"`
}

// PublishMetrics -
func PublishMetrics(metrics RequestStatistics) {
	if len(metricsChan) < bufferSize {
		metricsChan <- metrics
		return
	}
	if reachMaxChanCount%logLimitFrequency == 0 {
		reachMaxChanCount = 1
		log.GetLogger().Warnf("metricsChan reaches capacity and will discard the metric statistics :%v", metrics)
		return
	}
	reachMaxChanCount++
}

// subscribeMetrics -
func subscribeMetrics(stopChan <-chan struct{}) {
	if stopChan == nil {
		log.GetLogger().Warnf("stopChan is nil")
		return
	}
	for {
		select {
		case msg, ok := <-metricsChan:
			if !ok {
				log.GetLogger().Errorf("metrics channel is closed")
				return
			}
			statistics.lock.Lock()
			statistics.count = statistics.count + 1
			if msg.ErrorFlag {
				statistics.errCount = statistics.errCount + 1
			}
			statistics.totalDelay += msg.TotalDelay
			statistics.frontendDelay += msg.FrontendDelay
			statistics.busDelay += msg.BusDelay
			statistics.runtimeDelay += msg.RuntimeDelay
			statistics.lock.Unlock()
		case <-stopChan:
			log.GetLogger().Warnf("Received signal to quit")
			return
		}
	}
}

// StartReportMetrics -
func StartReportMetrics(stopChan <-chan struct{}) {
	if stopChan == nil {
		log.GetLogger().Warnf("stopChan is nil")
		return
	}
	err := initRedisClient(stopChan)
	if err != nil {
		log.GetLogger().Errorf("failed to new redis client and skip to report metrics, err: %s", err.Error())
		return
	}
	log.GetLogger().Infof("start to report metrics to server")
	go subscribeMetrics(stopChan)
	reportTick := time.NewTicker(reportTickerTime)
	for {
		select {
		case <-reportTick.C:
			if statistics.count <= 0 {
				log.GetLogger().Infof("no request info and skip reporting the metrics")
				continue
			}
			reportRequestInfoMetrics(requestInfoRedisKey, resetMetrics(), metricsKeyTTL)
		case _, ok := <-stopChan:
			if !ok {
				reportTick.Stop()
				log.GetLogger().Infof("stop report metrics to redis server")
				return
			}
		}
	}
}

func resetMetrics() []byte {
	now := time.Now()
	statistics.lock.Lock()
	metrics := RequestInfoMetrics{
		TimeStamp:     now.Unix() - int64(now.Second()),
		Count:         statistics.count,
		ErrCount:      statistics.errCount,
		TotalDelay:    statistics.totalDelay / float64(statistics.count),
		FrontendDelay: statistics.frontendDelay / float64(statistics.count),
		BusDelay:      statistics.busDelay / float64(statistics.count),
		RuntimeDelay:  statistics.runtimeDelay / float64(statistics.count),
	}
	statistics.count = 0
	statistics.errCount = 0
	statistics.totalDelay = 0
	statistics.frontendDelay = 0
	statistics.busDelay = 0
	statistics.runtimeDelay = 0
	statistics.lock.Unlock()
	value, err := json.Marshal(metrics)
	if err != nil {
		log.GetLogger().Errorf("failed to marshal requestInfo metrics, err: %s", err.Error())
	}
	return value
}

func initRedisClient(stopCh <-chan struct{}) error {
	var err error
	var pwd []byte
	pwd, err = localauth.Decrypt(config.GetConfig().RedisConfig.Password)
	if err != nil {
		log.GetLogger().Errorf("failed to decrypt redis password, %s", err.Error())
		return err
	}
	redisConf := redisclient.NewRedisClientParam{
		ServerMode: config.GetConfig().RedisConfig.ServerMode,
		ServerAddr: config.GetConfig().RedisConfig.ServerAddr,
		Password:   string(pwd),
		Timeout:    config.GetConfig().RedisConfig.TimeoutConf,
	}
	redisCmd, err := redisclient.New(redisclient.NewRedisClientParam{
		ServerMode: config.GetConfig().RedisConfig.ServerMode,
		ServerAddr: config.GetConfig().RedisConfig.ServerAddr,
		Password:   string(pwd),
		Timeout:    config.GetConfig().RedisConfig.TimeoutConf,
	}, stopCh, redisclient.SetEnableTLS(config.GetConfig().RedisConfig.EnableTLS))
	if err != nil {
		log.GetLogger().Errorf("failed to new a redis client and will "+
			"retry to reconnect later, err: %s", err.Error())
	} else {
		redisclient.SetRedisCmd(redisCmd)
	}
	go redisclient.CheckRedisConnectivity(&redisConf, redisclient.GetRedisCmd(), stopCh)
	return nil
}

func reportRequestInfoMetrics(redisKey string, redisValue []byte, expireTime time.Duration) {
	for i := 0; i < redisRetryTimes; i++ {
		if redisclient.GetRedisCmd() == nil {
			log.GetLogger().Warnf("[reportCPUFlagsMatchingRatio]redis is not ready")
			continue
		}
		err := redisclient.ZADDMetricsToRedis(redisKey, redisValue, listLimit, expireTime)
		if err == nil {
			log.GetLogger().Infof("succeed to report metrics key: %s, value:%s", redisKey, string(redisValue))
			return
		}
		log.GetLogger().Errorf("failed to set key: %s, err: %s, retry time %d", redisKey, err.Error(), i)
		time.Sleep(redisRetryInterval)
	}
}
