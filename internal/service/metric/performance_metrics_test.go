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

package metric_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

const mockDir = "mockdata"

func TestCreatePerformanceMetricsInstance(t *testing.T) {
	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"init success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			powerMaxService := service.PowerMaxService{}
			return powerMaxService, ctrl
		},
		// due to the singleton instance, this call will enter another branch
		"reuse success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			powerMaxService := service.PowerMaxService{}
			return powerMaxService, ctrl
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			powerMaxService, ctrl := tc(t)
			powerMaxService.Logger = logrus.New()
			metric.CreatePerformanceMetricsInstance(&powerMaxService)
			ctrl.Finish()
		})
	}
}

func TestPerformanceMetrics_Collect(t *testing.T) {
	var mockVolumes []k8s.VolumeInfo
	var arrayKeysResult v100.ArrayKeysResult
	var storageGroupTimeResult v100.StorageGroupKeysResult
	var volumePerfMetricsResult v100.VolumeMetricsIterator
	var storageGroupPerfMetricsResult v100.StorageGroupMetricsIterator

	mockVolBytes, err := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	err = json.Unmarshal(mockVolBytes, &mockVolumes)
	arrayKeyBytes, err := os.ReadFile(filepath.Join(mockDir, "array_perf_key.json"))
	err = json.Unmarshal(arrayKeyBytes, &arrayKeysResult)
	sgKeyBytes, err := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_key.json"))
	err = json.Unmarshal(sgKeyBytes, &storageGroupTimeResult)
	sgMetricBytes, err := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_metrics.json"))
	err = json.Unmarshal(sgMetricBytes, &storageGroupPerfMetricsResult)
	volMetricBytes, err := os.ReadFile(filepath.Join(mockDir, "vol_perf_metrics.json"))
	err = json.Unmarshal(volMetricBytes, &volumePerfMetricsResult)
	assert.Nil(t, err)

	tests := map[string]func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error){
		"success": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(3)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, nil
		},
		"failed to get pvs": func(*testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			err := errors.New("find no PVs, will do nothing")
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(nil, err).Times(1)

			clients := make(map[string][]types.PowerMaxArray)

			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, err
		},
		"get 0 pv": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(2)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(nil, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, nil
		},
		"failed to get client": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        nil,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, nil
		},
		"failed to get perf keys": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			err := errors.New("failed to get perf keys")
			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(nil, err).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(nil, err).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, nil
		},
		"failed to get metrics": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			err := errors.New("failed to get metric")

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, nil
		},
		"failed to record metrics": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			err := errors.New("failed to record metric")
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Return(err).Times(3)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.PerformanceMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return performanceMetric, ctrl, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			performanceMetric, ctrl, err := tc(t)
			performanceMetric.Logger = logrus.New()
			assert.Equal(t, err, performanceMetric.Collect(context.Background()))
			ctrl.Finish()
		})
	}
}
