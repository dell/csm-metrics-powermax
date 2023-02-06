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

	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/sirupsen/logrus"
)

const (
	// DefaultMaxPowerMaxConnections is the number of workers that can query PowerMax at a time
	DefaultMaxPowerMaxConnections = 10
)

// PowerMaxService contains configuration stuff and represents the service for getting metrics data for a PowerMax system
type PowerMaxService struct {
	MetricsRecorder        types.MetricsRecorder
	MaxPowerMaxConnections int
	Logger                 *logrus.Logger
	PowerMaxClients        map[string]types.PowerMaxClient
	VolumeFinder           types.VolumeFinder
	StorageClassFinder     types.StorageClassFinder
}

// GetLogger return logger
func (s *PowerMaxService) GetLogger() *logrus.Logger {
	return s.Logger
}

// GetPowerMaxClients return PowerMaxClients
func (s *PowerMaxService) GetPowerMaxClients() map[string]types.PowerMaxClient {
	return s.PowerMaxClients
}

// GetMetricsRecorder return MetricsRecorder
func (s *PowerMaxService) GetMetricsRecorder() types.MetricsRecorder {
	return s.MetricsRecorder
}

// GetMaxPowerMaxConnections return MaxPowerMaxConnections
func (s *PowerMaxService) GetMaxPowerMaxConnections() int {
	return s.MaxPowerMaxConnections
}

// GetVolumeFinder return VolumeFinder
func (s *PowerMaxService) GetVolumeFinder() types.VolumeFinder {
	return s.VolumeFinder
}

// ExportCapacityMetrics collect capacity for array, storageclass, srp, storagegroup and volume, and export to Otel
func (s *PowerMaxService) ExportCapacityMetrics(ctx context.Context) {
	metric.CreateCapacityMetricsInstance(s).ExportMetrics(ctx)
}

// ExportPerformanceMetrics collect performance and export to Otel
func (s *PowerMaxService) ExportPerformanceMetrics(ctx context.Context) {
	metric.CreatePerformanceMetricsInstance(s).ExportMetrics(ctx)
}
