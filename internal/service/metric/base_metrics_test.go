/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func Test_ExportMetrics(t *testing.T) {
	var mockVolumes []k8s.VolumeInfo
	var volume00833 v100.Volume
	var volume00834 v100.Volume

	mockVolBytes, _ := os.ReadFile(filepath.Join(mockDir, "persistent_volumes.json"))
	_ = json.Unmarshal(mockVolBytes, &mockVolumes)
	vol00833Bytes, _ := os.ReadFile(filepath.Join(mockDir, "pmax_vol_00833.json"))
	_ = json.Unmarshal(vol00833Bytes, &volume00833)
	vol00834Bytes, _ := os.ReadFile(filepath.Join(mockDir, "pmax_vol_00834.json"))
	err := json.Unmarshal(vol00834Bytes, &volume00834)
	assert.Nil(t, err)

	tests := map[string]func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller){
		"success": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any(), gomock.Any()).Times(6)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(&volume00833, nil).Times(1)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(&volume00834, nil).Times(1)

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
			capacityMetric := metric.CreateCapacityMetricsInstance(&service)

			return capacityMetric.BaseMetrics, ctrl
		},
		"success - no collector ": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			base := &metric.BaseMetrics{
				Logger:                 logrus.New(),
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			}

			return base, ctrl
		},
		"error - not active": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			// No call to RecordNumericMetrics since not active

			c := mocks.NewMockPowerMaxClient(ctrl)
			// Not active so no calls to GetVolumeByID

			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: false,
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
			base := metric.NewBaseMetrics(&service)
			myCapacityInstance := &metric.CapacityMetrics{base}
			base.Collector = myCapacityInstance

			return base, ctrl
		},
		"success - no recorder ": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			c := mocks.NewMockPowerMaxClient(ctrl)
			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			base := &metric.BaseMetrics{
				Logger:                 logrus.New(),
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
				Collector:              &metric.CapacityMetrics{},
				MetricsRecorder:        nil,
			}

			return base, ctrl
		},
		"success - no client ": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			metrics := mocks.NewMockMetricsRecorder(ctrl)

			c := mocks.NewMockPowerMaxClient(ctrl)
			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			base := &metric.BaseMetrics{
				Logger:                 logrus.New(),
				MetricsRecorder:        metrics,
				VolumeFinder:           volFinder,
				Collector:              &metric.CapacityMetrics{},
				PowerMaxClients:        nil,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
			}

			return base, ctrl
		},
		"success - no connections ": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			metrics := mocks.NewMockMetricsRecorder(ctrl)

			c := mocks.NewMockPowerMaxClient(ctrl)
			clients := make(map[string][]metrictypes.PowerMaxArray)
			array := metrictypes.PowerMaxArray{
				Client:   c,
				IsActive: true,
			}
			clients["000197902599"] = append(clients["000197902599"], array)

			base := &metric.BaseMetrics{
				Logger:                 logrus.New(),
				MetricsRecorder:        metrics,
				VolumeFinder:           volFinder,
				Collector:              &metric.CapacityMetrics{},
				PowerMaxClients:        clients,
				MaxPowerMaxConnections: 0,
			}

			return base, ctrl
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			baseMetric, ctrl := tc(t)
			baseMetric.ExportMetrics(context.Background())
			ctrl.Finish()
		})
	}
}
