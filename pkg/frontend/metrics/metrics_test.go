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

package metrics

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"frontend/pkg/common/faas_common/localauth"
	"frontend/pkg/common/faas_common/redisclient"
)

func TestMetrics(t *testing.T) {
	t.Run("TestStartReportMetrics", testStartReportMetrics)
	resetMetrics()
	t.Run("TestMetricsChan", testMetricsChan)
}
func testStartReportMetrics(t *testing.T) {
	var redisValue string
	now := time.Now()
	patches := [...]*gomonkey.Patches{
		gomonkey.ApplyFunc(localauth.Decrypt,
			func(src string) ([]byte, error) {
				return []byte("test"), nil
			}),
		gomonkey.ApplyFunc(redisclient.New, func(param redisclient.NewRedisClientParam, stopCh <-chan struct{},
			options ...redisclient.Option) (*redisclient.Client, error) {
			return &redisclient.Client{}, nil
		}),
		gomonkey.ApplyFunc(redisclient.ZADDMetricsToRedis,
			func(key string, metrics interface{}, limit int64, expireTime time.Duration) error {
				redisValue = string(metrics.([]byte))
				return nil
			}),
		gomonkey.ApplyFunc(time.NewTicker, func(d time.Duration) *time.Ticker {
			c := make(chan time.Time, 1)
			t := &time.Ticker{
				C: c,
			}
			time.Sleep(10 * time.Millisecond)
			c <- time.Now()
			return t
		}),
		gomonkey.ApplyFunc(redisclient.CheckRedisConnectivity, func(clientRedisConfig *redisclient.NewRedisClientParam,
			client *redisclient.Client, stopCh <-chan struct{}) {
		}),
		gomonkey.ApplyFunc(time.Now, func() time.Time {
			return now
		}),
	}
	defer func() {
		for idx := range patches {
			patches[idx].Reset()
		}
	}()
	mockMetrics := RequestStatistics{
		TotalDelay:    0.1,
		BusDelay:      0.02,
		FrontendDelay: 0.01,
		RuntimeDelay:  0.07,
	}
	PublishMetrics(mockMetrics)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, int64(0), statistics.count)
	//
	ch := make(chan struct{})
	go StartReportMetrics(ch)
	time.Sleep(100 * time.Millisecond)
	timestamp := now.Unix() - int64(now.Second())
	expect := fmt.Sprintf(`{"timeStamp":%d,"count":1,"totalDelay":0.1,"frontendDelay":0.01,"busDelay":0.02,"runtimeDelay":0.07}`, timestamp)
	assert.Equal(t, expect, redisValue)

	mockMetrics = RequestStatistics{
		ErrorFlag:     true,
		TotalDelay:    0.1,
		BusDelay:      0.02,
		FrontendDelay: 0.05,
		RuntimeDelay:  0.03,
	}
	PublishMetrics(mockMetrics)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, statistics.count, int64(1))
	assert.Equal(t, statistics.errCount, int64(1))
	assert.Equal(t, statistics.totalDelay, 0.1)
	assert.Equal(t, statistics.runtimeDelay, 0.03)

	close(ch)
	time.Sleep(10 * time.Millisecond)
	StartReportMetrics(nil)
	subscribeMetrics(nil)
}

func testMetricsChan(t *testing.T) {
	go func() {
		for i := 0; i < 100; i++ {
			PublishMetrics(RequestStatistics{
				TotalDelay: float64(i) / 100.0,
			})
		}
	}()
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, 100, len(metricsChan))

	go func() {
		for i := 0; i < bufferSize+50; i++ {
			PublishMetrics(RequestStatistics{
				TotalDelay: float64(i) / 100.0,
			})
		}
	}()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, bufferSize, len(metricsChan))
	ch := make(chan struct{})
	go subscribeMetrics(ch)
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, int64(bufferSize), statistics.count)
	assert.Equal(t, 0, len(metricsChan))
	resetMetrics()
	go func() {
		for i := 0; i < bufferSize; i++ {
			PublishMetrics(RequestStatistics{
				TotalDelay: float64(i) / 100.0,
			})
		}
		time.Sleep(10 * time.Millisecond)
		for i := 0; i < bufferSize/2; i++ {
			PublishMetrics(RequestStatistics{
				TotalDelay: float64(i) / 100.0,
			})
		}
	}()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, int64(bufferSize/2+bufferSize), statistics.count)

	go func() {
		for i := 0; i < bufferSize/2; i++ {
			PublishMetrics(RequestStatistics{
				TotalDelay: float64(i) / 100.0,
			})
		}
		time.Sleep(50 * time.Millisecond)
		for i := 0; i < bufferSize; i++ {
			PublishMetrics(RequestStatistics{
				TotalDelay: float64(i) / 100.0,
			})
		}
	}()
	time.Sleep(1 * time.Millisecond)
	b := resetMetrics()
	time.Sleep(100 * time.Millisecond)

	var request RequestInfoMetrics
	err := json.Unmarshal(b, &request)
	assert.Nil(t, err)
	assert.Equal(t, int64(3*bufferSize), statistics.count+request.Count)
}
