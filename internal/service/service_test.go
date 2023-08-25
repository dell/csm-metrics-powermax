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
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

const mockDir = "metric/mockdata"

func Test_ExportCapacityMetrics(t *testing.T) {
	var mockVolumes []k8s.VolumeInfo
	var volume00833 v100.Volume
	var volume00834 v100.Volume

	mockVolBytes, err := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	err = json.Unmarshal(mockVolBytes, &mockVolumes)
	vol00833Bytes, err := os.ReadFile(filepath.Join(mockDir, "pmax_vol_00833.json"))
	err = json.Unmarshal(vol00833Bytes, &volume00833)
	vol00834Bytes, err := os.ReadFile(filepath.Join(mockDir, "pmax_vol_00834.json"))
	err = json.Unmarshal(vol00834Bytes, &volume00834)
	assert.Nil(t, err)

	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(6)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), "00833").Return(&volume00833, nil).Times(1)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), "00834").Return(&volume00834, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
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

	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(3)

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
