/*
 * Copyright 2017-2018 IBM Corporation
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
 
 package metricsmon

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {

	counter := NewCounter("method_call_counter", "test counter", []string{"count"})
	latency := NewSummary("method_latency_seconds", "test latency/summary", []string{"latency"})

	finish := make(chan bool)

	//after 5 seconds get all the metrics
	time.AfterFunc(10*time.Second, func() {
		req, _ := http.NewRequest("GET", "/metrics", nil)
		rr := httptest.NewRecorder()
		promhttp.Handler().ServeHTTP(rr, req)
		log.Debugf("Body is %v", rr.Body)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.True(t, strings.Contains(rr.Body.String(), "method_call_counter{count=\"total\"}"), "couldn't find the metrics info published for counter")
		assert.True(t, strings.Contains(rr.Body.String(), "method_latency_seconds_sum{latency=\"invocation_duration\"}"), "couldn't find the metrics info published for latency sum")
		assert.True(t, strings.Contains(rr.Body.String(), "method_latency_seconds_count{latency=\"invocation_duration\"}"), "couldn't find the metrics info published for latency count")
		finish <- true
	})

	//keep looping and send metrics every 2 seconds unless someone sends a done message
	for {
		select {
		case done := <-finish:
			if done {
				return
			}
		default:
			begin := time.Now()
			time.Sleep(2 * time.Second)
			counter.With("count", "total").Add(1)
			latency.With("latency", "invocation_duration").Observe(time.Since(begin).Seconds())
		}

	}

}
