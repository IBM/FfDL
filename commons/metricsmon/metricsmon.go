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
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	"github.com/go-kit/kit/metrics/statsd"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

//NewCounter ... new prometheus counter
func NewCounter(name string, help string, labelsForPartitioningMetrics []string) metrics.Counter {
	counter := prometheus.NewCounterFrom(stdprometheus.CounterOpts{Name: name, Help: help}, labelsForPartitioningMetrics)
	return counter
}

//NewGauge ... new prometheus gauge
func NewGauge(name string, help string, labelsForPartitioningMetrics []string) metrics.Gauge {
	gauge := prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{Name: name, Help: help}, labelsForPartitioningMetrics)
	return gauge
}

//NewSummary ... new prometheus gauge
func NewSummary(name string, help string, labelsForPartitioningMetrics []string) metrics.Histogram {
	hist := prometheus.NewSummaryFrom(stdprometheus.SummaryOpts{Name: name, Help: help}, labelsForPartitioningMetrics)
	return hist
}

//NewStatsdClient ...
func NewStatsdClient(prefix string) *statsd.Statsd {
	stats := statsd.New(fmt.Sprintf("%s.", prefix), log.NewNopLogger())
	return stats
}

//use the statsd client code to create the metrics, no wrapper required
