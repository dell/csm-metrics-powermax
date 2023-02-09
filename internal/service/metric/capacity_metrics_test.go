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
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func Test_CreateCapacityMetricsInstance(t *testing.T) {
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
			metric.CreateCapacityMetricsInstance(&powerMaxService)
			ctrl.Finish()
		})
	}
}

func Test_CapacityMetricsCollect(t *testing.T) {
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

	tests := map[string]func(t *testing.T) (metric.CapacityMetrics, *gomock.Controller, error){
		"success": func(t *testing.T) (metric.CapacityMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(6)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(&volume00833, nil).Times(1)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(&volume00834, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			capacityMetric := metric.CapacityMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return capacityMetric, ctrl, nil
		},
		"failed to get pvs": func(*testing.T) (metric.CapacityMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			err := errors.New("find no PVs, will do nothing")
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(nil, err).Times(1)

			clients := make(map[string][]types.PowerMaxArray)

			capacityMetric := metric.CapacityMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return capacityMetric, ctrl, err
		},
		"get 0 pv": func(t *testing.T) (metric.CapacityMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(5)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(nil, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			capacityMetric := metric.CapacityMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return capacityMetric, ctrl, nil
		},
		"failed to get client": func(t *testing.T) (metric.CapacityMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			capacityMetric := metric.CapacityMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        nil,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return capacityMetric, ctrl, nil
		},
		"failed to get volume": func(t *testing.T) (metric.CapacityMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(0)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			err := errors.New("failed to get volume")

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).Times(2)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			capacityMetric := metric.CapacityMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return capacityMetric, ctrl, nil
		},
		"failed to record metrics": func(t *testing.T) (metric.CapacityMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			err := errors.New("failed to record metric")
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Return(err).Times(6)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(&volume00833, nil).Times(1)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(&volume00834, nil).Times(1)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			capacityMetric := metric.CapacityMetrics{
				BaseMetrics: &metric.BaseMetrics{
					VolumeFinder:           volFinder,
					PowerMaxClients:        clients,
					MetricsRecorder:        metrics,
					MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				},
			}
			return capacityMetric, ctrl, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			capacityMetric, ctrl, err := tc(t)
			capacityMetric.Logger = logrus.New()
			assert.Equal(t, err, capacityMetric.Collect(context.Background()))
			ctrl.Finish()
		})
	}
}
