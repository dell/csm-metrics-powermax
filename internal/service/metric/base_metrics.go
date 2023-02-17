/*
 Copyright (c) 2022-2023 Dell Inc. or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package metric

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/sirupsen/logrus"
)

var lock = &sync.Mutex{}

// MetricsCollector interface for metric collect
type MetricsCollector interface {
	Collect(ctx context.Context) error
}

// BaseMetrics presents the base class of concrete metric classes
type BaseMetrics struct {
	Collector              MetricsCollector
	Logger                 *logrus.Logger
	VolumeFinder           types.VolumeFinder
	PowerMaxClients        map[string][]types.PowerMaxArray
	MetricsRecorder        types.MetricsRecorder
	MaxPowerMaxConnections int
}

// NewBaseMetrics return BaseMetrics instance.
func NewBaseMetrics(service types.Service) *BaseMetrics {
	return &BaseMetrics{
		Logger:                 service.GetLogger(),
		VolumeFinder:           service.GetVolumeFinder(),
		PowerMaxClients:        service.GetPowerMaxClients(),
		MetricsRecorder:        service.GetMetricsRecorder(),
		MaxPowerMaxConnections: service.GetMaxPowerMaxConnections(),
	}
}

// TimeSince will log the amount of time spent in a given function
func (m *BaseMetrics) TimeSince(start time.Time, fName string) {
	if m.Logger != nil {
		m.Logger.WithFields(logrus.Fields{
			"duration": fmt.Sprintf("%v", time.Since(start)),
			"function": fName,
		}).Info("function duration")
	}
}

// GetPowerMaxClient return the first live PowerMaxClient based on the given arrayID
func (m *BaseMetrics) GetPowerMaxClient(arrayID string) (types.PowerMaxClient, error) {
	if m.PowerMaxClients == nil {
		return nil, fmt.Errorf("PowerMaxClients is empty")
	}
	if arrays, ok := m.PowerMaxClients[arrayID]; ok {
		for _, array := range arrays {
			if array.IsActive {
				m.Logger.WithFields(logrus.Fields{"arrayID": arrayID, "endpoint": array.Endpoint}).Debug("connection is active")
				return array.Client, nil
			}
			m.Logger.WithFields(logrus.Fields{"arrayID": arrayID, "endpoint": array.Endpoint}).Warn("connection is inactive")
		}
	}
	return nil, fmt.Errorf("unable to find active gopowermax client for array %s", arrayID)
}

// ExportMetrics collect and export metrics to Otel
func (m *BaseMetrics) ExportMetrics(ctx context.Context) {
	if m.Collector == nil {
		m.Logger.Errorf("no MetricsCollector provided")
		return
	}

	if m.MetricsRecorder == nil {
		m.Logger.Errorf("no MetricsRecorder provided for %T", m.Collector)
		return
	}

	if m.PowerMaxClients == nil {
		m.Logger.Errorf("no PowerMaxClients provided for %T", m.Collector)
		return
	}

	if m.MaxPowerMaxConnections == 0 {
		m.Logger.Errorf("no MaxPowerMaxConnections provided for %T", m.Collector)
		return
	}

	start := time.Now()
	defer m.TimeSince(start, fmt.Sprintf("ExportMetrics for %T", m.Collector))

	err := m.Collector.Collect(ctx)
	if err != nil {
		m.Logger.WithError(err).Warn("failed to collect metrics")
	}
}
