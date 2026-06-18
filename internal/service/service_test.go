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

package service_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

const mockDir = "metric/mockdata"

func Test_ExportCapacityMetrics(t *testing.T) {
	var mockVolumes []k8s.VolumeInfo
	var bulkCapacity v100.Volumev1

	mockVolBytes, _ := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	_ = json.Unmarshal(mockVolBytes, &mockVolumes)
	bulkBytes, _ := os.ReadFile(filepath.Join(mockDir, "pmax_vol_capacity_bulk.json"))
	err := json.Unmarshal(bulkBytes, &bulkCapacity)
	assert.Nil(t, err)

	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(6)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetVolumesCapacityBulk(gomock.Any(), gomock.Any()).Return(&bulkCapacity, nil).Times(1)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			service := service.PowerMaxService{
				Logger:                 logrus.New(),
				MetricsRecorder:        metrics,
				VolumeFinder:           volFinder,
				StorageClassFinder:     scFinder,
				PowerMaxClients:        clients,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			}
			return service, ctrl
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			service, ctrl := tc(t)
			service.ExportCapacityMetrics(context.Background())
			ctrl.Finish()
		})
	}
}

func Test_ExportPerformanceMetrics(t *testing.T) {
	var mockVolumes []k8s.VolumeInfo
	var arrayKeysResult v100.ArrayKeysResult
	var storageGroupTimeResult v100.StorageGroupKeysResult
	var volumePerfMetricsResult v100.VolumeMetricsIterator
	var storageGroupPerfMetricsResult v100.StorageGroupMetricsIterator
	var storageGroupIDList v100.StorageGroupIDList

	mockVolBytes, _ := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	_ = json.Unmarshal(mockVolBytes, &mockVolumes)
	arrayKeyBytes, _ := os.ReadFile(filepath.Join(mockDir, "array_perf_key.json"))
	_ = json.Unmarshal(arrayKeyBytes, &arrayKeysResult)
	sgKeyBytes, _ := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_key.json"))
	_ = json.Unmarshal(sgKeyBytes, &storageGroupTimeResult)
	sgMetricBytes, _ := os.ReadFile(filepath.Join(mockDir, "storage_group_perf_metrics.json"))
	_ = json.Unmarshal(sgMetricBytes, &storageGroupPerfMetricsResult)
	volMetricBytes, _ := os.ReadFile(filepath.Join(mockDir, "vol_perf_metrics.json"))
	err := json.Unmarshal(volMetricBytes, &volumePerfMetricsResult)
	assert.Nil(t, err)

	// Create a mock storage group ID list with more total SGs to test individual API path
	// (2 requested SGs out of 10 total = 20% < 25%, so individual API will be used)
	storageGroupIDList = v100.StorageGroupIDList{
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

	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			metrics.EXPECT().RecordStorageGroupPerfMetrics(gomock.Any(), gomock.Any()).Times(1)
			metrics.EXPECT().RecordVolPerfMetrics(gomock.Any(), gomock.Any()).Times(2)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayKeysResult, nil).Times(1)
			c.EXPECT().GetStorageGroupIDList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupIDList, nil).Times(1)
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

			service := service.PowerMaxService{
				Logger:                 logrus.New(),
				MetricsRecorder:        metrics,
				VolumeFinder:           volFinder,
				StorageClassFinder:     scFinder,
				PowerMaxClients:        clients,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			}
			return service, ctrl
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			service, ctrl := tc(t)
			service.ExportPerformanceMetrics(context.Background())
			ctrl.Finish()
		})
	}
}

func Test_ExportTopologyMetrics(t *testing.T) {
	var mockVolumes []k8s.VolumeInfo

	mockVolBytes, _ := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	err := json.Unmarshal(mockVolBytes, &mockVolumes)

	assert.Nil(t, err)

	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			metrics.EXPECT().RecordTopologyMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)

			c := mocks.NewMockPowerMaxClient(ctrl)

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			service := service.PowerMaxService{
				Logger:                 logrus.New(),
				MetricsRecorder:        metrics,
				VolumeFinder:           volFinder,
				StorageClassFinder:     scFinder,
				PowerMaxClients:        clients,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			}
			return service, ctrl
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			service, ctrl := tc(t)
			service.ExportTopologyMetrics(context.Background())
			ctrl.Finish()
		})
	}
}
