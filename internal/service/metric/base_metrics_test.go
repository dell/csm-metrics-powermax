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
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
)

func Test_ExportMetrics(t *testing.T) {
	mockVolumes := []k8s.VolumeInfo{
		{
			Namespace:              "karavi",
			PersistentVolumeClaim:  "pvc-uid1",
			PersistentVolumeStatus: "Bound",
			VolumeClaimName:        "pvc-name1",
			PersistentVolume:       "pv-1",
			StorageClass:           "powermax-iscsi",
			Driver:                 "csi-powermax.dellemc.com",
			ProvisionedSize:        "16Gi",
			VolumeHandle:           "csi-k8s-mock-398993ad1b-powermaxtest-000197902599-00833",
			SRP:                    "SRP_1",
			StorageGroup:           "csi-TAO-Gold-SRP_1-SG",
		},
		{
			Namespace:              "karavi",
			PersistentVolumeClaim:  "pvc-uid2",
			PersistentVolumeStatus: "Bound",
			VolumeClaimName:        "pvc-name2",
			PersistentVolume:       "pv-2",
			StorageClass:           "powermax-iscsi",
			Driver:                 "csi-powermax.dellemc.com",
			ProvisionedSize:        "8Gi",
			VolumeHandle:           "csi-k8s-mock-398993ad1d-powermaxtest-000197902599-00834",
			SRP:                    "SRP_1",
			StorageGroup:           "csi-TAO-Gold-SRP_1-SG",
		},
	}
	volume00833 := &v100.Volume{
		VolumeID:         "00833",
		Type:             "TDEV",
		AllocatedPercent: 10,
		CapacityGB:       16.0,
		FloatCapacityMB:  8194.0,
		VolumeIdentifier: "csi-k8s-mock-398993ad1b-powermaxtest",
		WWN:              "60000970000197902273533030383543",
		StorageGroups: []v100.StorageGroupName{
			{StorageGroupName: "csi-TAO-Gold-SRP_1-SG"},
			{StorageGroupName: "csi-no-srp-sg-TAO-worker-2-zegnx4zktvbph"}},
	}
	volume00834 := &v100.Volume{
		VolumeID:         "00834",
		Type:             "TDEV",
		AllocatedPercent: 10,
		CapacityGB:       8.0,
		FloatCapacityMB:  8194.0,
		VolumeIdentifier: "csi-k8s-mock-398993ad1d-powermaxtest",
		WWN:              "60000970000197902273533030383544",
		StorageGroups: []v100.StorageGroupName{
			{StorageGroupName: "csi-TAO-Gold-SRP_1-SG"},
			{StorageGroupName: "csi-no-srp-sg-TAO-worker-2-zegnx4zktvbph"}},
	}

	tests := map[string]func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller){
		"success": func(t *testing.T) (*metric.BaseMetrics, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(6)

			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(volume00833, nil).Times(1)
			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(volume00834, nil).Times(1)

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
			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			base := &metric.BaseMetrics{
				Logger:                 logrus.New(),
				VolumeFinder:           volFinder,
				PowerMaxClients:        clients,
				MetricsRecorder:        metrics,
				MaxPowerMaxConnections: service.DefaultMaxPowerMaxConnections,
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
