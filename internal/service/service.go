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

package service

import (
	"context"
	"maps"
	"sync"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
)

const (
	// DefaultMaxPowerMaxConnections is the number of workers that can query PowerMax at a time
	DefaultMaxPowerMaxConnections = 10
)

// PowerMaxService contains configuration stuff and represents the service for getting metrics data for a PowerMax system
type PowerMaxService struct {
	MetricsRecorder        metrictypes.MetricsRecorder
	ObsInstrumenter        *PMAXObsInstrumenter
	MaxPowerMaxConnections int
	PowerMaxClients        map[string][]metrictypes.PowerMaxArray
	VolumeFinder           metrictypes.VolumeFinder
	StorageClassFinder     metrictypes.StorageClassFinder
	mu                     sync.RWMutex
}

// GetPowerMaxClients return a snapshot of PowerMaxClients.
func (s *PowerMaxService) GetPowerMaxClients() map[string][]metrictypes.PowerMaxArray {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.PowerMaxClients)
}

// SetPowerMaxClients replaces the PowerMaxClients map with a defensive copy.
func (s *PowerMaxService) SetPowerMaxClients(clients map[string][]metrictypes.PowerMaxArray) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PowerMaxClients = maps.Clone(clients)
}

// GetMetricsRecorder return MetricsRecorder
func (s *PowerMaxService) GetMetricsRecorder() metrictypes.MetricsRecorder {
	return s.MetricsRecorder
}

// GetObsInstrumenter returns the observability instrumenter
func (s *PowerMaxService) GetObsInstrumenter() interface{} {
	return s.ObsInstrumenter
}

// GetMaxPowerMaxConnections return MaxPowerMaxConnections
func (s *PowerMaxService) GetMaxPowerMaxConnections() int {
	return s.MaxPowerMaxConnections
}

// GetVolumeFinder return VolumeFinder
func (s *PowerMaxService) GetVolumeFinder() metrictypes.VolumeFinder {
	return s.VolumeFinder
}

// ExportCapacityMetrics collect capacity for array, storageclass, srp, storagegroup and volume, and export to Otel
func (s *PowerMaxService) ExportCapacityMetrics(ctx context.Context) {
	start := time.Now()
	err := metric.CreateCapacityMetricsInstance(s).ExportMetrics(ctx)
	s.recordObsMetrics(time.Since(start), err)
}

// ExportPerformanceMetrics collect performance and export to Otel
func (s *PowerMaxService) ExportPerformanceMetrics(ctx context.Context) {
	start := time.Now()
	err := metric.CreatePerformanceMetricsInstance(s).ExportMetrics(ctx)
	s.recordObsMetrics(time.Since(start), err)
}

// ExportTopologyMetrics collect topology metrics and export to Otel
func (s *PowerMaxService) ExportTopologyMetrics(ctx context.Context) {
	start := time.Now()
	err := metric.CreateTopologyMetricsInstance(s).ExportMetrics(ctx)
	s.recordObsMetrics(time.Since(start), err)
}

// recordObsMetrics records observability self-metrics for each known PowerMax array.
func (s *PowerMaxService) recordObsMetrics(elapsed time.Duration, exportErr error) {
	if s.ObsInstrumenter == nil {
		return
	}
	clients := s.GetPowerMaxClients()
	for arrayID, arrays := range clients {
		isActive := false
		for _, a := range arrays {
			if a.IsActive {
				isActive = true
				break
			}
		}
		s.ObsInstrumenter.SetArrayConnectivity(arrayID, isActive)
		s.ObsInstrumenter.RecordCollectionRate(arrayID, float64(len(arrays)))
		s.ObsInstrumenter.RecordProcessingLatency(arrayID, elapsed.Seconds())
		status := "success"
		if !isActive {
			status = "error"
		} else if exportErr != nil {
			status = "failure"
		}
		s.ObsInstrumenter.RecordExportSuccess(arrayID, status)
	}
}
