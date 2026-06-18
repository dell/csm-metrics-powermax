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
	"os"
	"path/filepath"
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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
	var storageGroupPerfMetricsBulkResult v100.StorageGroupPerfCategoryResult
	var storageGroupIDList v100.StorageGroupIDList

	mockVolBytes, _ := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	_ = json.Unmarshal(mockVolBytes, &mockVolumes)
	arrayKeyBytes, _ := os.ReadFile(filepath.Join(mockDir, "array_perf_key.json"))
	_ = json.Unmarshal(arrayKeyBytes, &arrayKeysResult)
	sgKeyBytes, _ := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_key.json"))
	_ = json.Unmarshal(sgKeyBytes, &storageGroupTimeResult)
	sgMetricBytes, _ := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_metrics.json"))
	_ = json.Unmarshal(sgMetricBytes, &storageGroupPerfMetricsResult)
	sgMetricBulkBytes, _ := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_metrics_bulk.json"))
	_ = json.Unmarshal(sgMetricBulkBytes, &storageGroupPerfMetricsBulkResult)
	volMetricBytes, _ := os.ReadFile(filepath.Join(mockDir, "vol_perf_metrics.json"))
	err := json.Unmarshal(volMetricBytes, &volumePerfMetricsResult)
	assert.Nil(t, err)

	// Create a mock storage group ID list with few total SGs to test bulk API path
	// (1 requested SG out of 2 total = 50% > bulkThresholdRatio, so bulk API will be used)
	storageGroupIDList = v100.StorageGroupIDList{
		StorageGroupIDs: []string{"csi-TAO-Gold-SRP_1-SG", "csi-TAO-Gold-SRP_2-SG"},
	}

	// Empty bulk result – no metric instances at all
	emptyBulkResult := v100.StorageGroupPerfCategoryResult{
		ID:              "StorageGroup",
		ResourceType:    "performance-categories",
		System:          "000197902599",
		MetricInstances: nil,
	}

	tests := map[string]func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error){
		"success - bulk returns empty, fallback to per-SG": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			// 2 total SGs -> 50% > bulkThresholdRatio, so bulk path is chosen
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupIDList, nil).Times(1)
			// Bulk returns empty result -> fallback to legacy
			c.EXPECT().GetStorageGroupMetricsBulk(gomock.Any(), gomock.Any()).Return(&emptyBulkResult, nil).Times(1)
			// Legacy fallback calls
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"success - bulk API": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			// metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(3)
			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupIDList, nil).Times(1)
			c.EXPECT().GetStorageGroupMetricsBulk(gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsBulkResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"success - bulk fails, fallback to per-SG": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			// Create a storage group ID list with many total SGs to test individual API path
			// (1 requested SG out of 10 total = 10% < bulkThresholdRatio, so individual API will be used)
			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG",
					"csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG",
					"csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG",
					"csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG",
					"csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG",
					"csi-TAO-Gold-SRP_10-SG",
				},
			}

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to get pvs": func(*testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			err := errors.New("find no PVs, will do nothing")
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(nil, err).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, err
		},
		"get 0 pv": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			// metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(nil, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to get client": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        nil,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to get perf keys": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			// Create a storage group ID list with many total SGs to test individual API path
			// (1 requested SG out of 10 total = 10% < bulkThresholdRatio, so individual API will be used)
			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG",
					"csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG",
					"csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG",
					"csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG",
					"csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG",
					"csi-TAO-Gold-SRP_10-SG",
				},
			}

			err := errors.New("failed to get perf keys")
			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(nil, err).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(nil, err).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to get metrics": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			// Create a storage group ID list with many total SGs to test individual API path
			// (1 requested SG out of 10 total = 10% < bulkThresholdRatio, so individual API will be used)
			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG",
					"csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG",
					"csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG",
					"csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG",
					"csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG",
					"csi-TAO-Gold-SRP_10-SG",
				},
			}

			err := errors.New("failed to get metric")

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).AnyTimes()

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to record metrics": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			err := errors.New("failed to record metric")
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Return(err).Times(0)
			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			// Create a storage group ID list with many total SGs to test individual API path
			// (1 requested SG out of 10 total = 10% < bulkThresholdRatio, so individual API will be used)
			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG",
					"csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG",
					"csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG",
					"csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG",
					"csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG",
					"csi-TAO-Gold-SRP_10-SG",
				},
			}

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"volume with short handle": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			// Return a volume whose VolumeHandle has only one segment (no dashes),
			// which should trigger the len(volumeProperties) < 2 warning path.
			shortHandleVolumes := []k8s.VolumeInfo{
				{VolumeHandle: "noDash"},
			}

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(shortHandleVolumes, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"success - bulk API returns error": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			// 2 total SGs -> 50% > bulkThresholdRatio, so bulk path is chosen
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupIDList, nil).Times(1)
			// Bulk returns an error -> fall back to legacy
			c.EXPECT().GetStorageGroupMetricsBulk(gomock.Any(), gomock.Any()).Return(nil, errors.New("bulk error")).Times(1)
			// Legacy fallback calls
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"success - bulk API returns nil result": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			// 2 total SGs -> 50% > bulkThresholdRatio, so bulk path is chosen
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupIDList, nil).Times(1)
			// Bulk returns nil (no error) -> fall back to legacy
			c.EXPECT().GetStorageGroupMetricsBulk(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
			// Legacy fallback calls
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to record vol perf metrics": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			recErr := errors.New("failed to record vol perf metric")
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Return(recErr).Times(2)
			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG", "csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG", "csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG", "csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG", "csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG", "csi-TAO-Gold-SRP_10-SG",
				},
			}

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to record sg perf metrics": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			recErr := errors.New("failed to record sg perf metric")
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)
			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Return(recErr).Times(1)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG", "csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG", "csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG", "csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG", "csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG", "csi-TAO-Gold-SRP_10-SG",
				},
			}

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"volume with empty result": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			// No vol perf metric recorded because volumeResult is empty
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			emptyVolResult := v100.VolumeMetricsIterator{
				ResultList: v100.VolumeMetricsResultList{
					Result: []v100.VolumeResult{
						{
							VolumeID:      "00833",
							StorageGroups: "csi-TAO-Gold-SRP_1-SG",
							VolumeResult:  nil, // empty result
						},
						{
							VolumeID:      "00834",
							StorageGroups: "csi-TAO-Gold-SRP_1-SG",
							VolumeResult:  nil, // empty result
						},
					},
				},
			}

			manySGsList := v100.StorageGroupIDList{
				StorageGroupIDs: []string{
					"csi-TAO-Gold-SRP_1-SG", "csi-TAO-Gold-SRP_2-SG",
					"csi-TAO-Gold-SRP_3-SG", "csi-TAO-Gold-SRP_4-SG",
					"csi-TAO-Gold-SRP_5-SG", "csi-TAO-Gold-SRP_6-SG",
					"csi-TAO-Gold-SRP_7-SG", "csi-TAO-Gold-SRP_8-SG",
					"csi-TAO-Gold-SRP_9-SG", "csi-TAO-Gold-SRP_10-SG",
				},
			}

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&manySGsList, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&emptyVolResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
			return performanceMetric, ctrl, nil
		},
		"failed to get sg count - fallback to legacy": func(t *testing.T) (*metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			// GetStorageGroupIDList fails -> getTotalSGCount returns error -> fallback to legacy
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("sg list error")).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).AnyTimes()
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			performanceMetric := metric.NewPerformanceMetrics(&metric.BaseMetrics{
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			})
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
