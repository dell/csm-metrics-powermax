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
	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	"github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"testing"
)

var mockVolumes = []k8s.VolumeInfo{
	{
		Namespace:              "karavi",
		PersistentVolumeClaim:  "pvc-uid",
		PersistentVolumeStatus: "Bound",
		VolumeClaimName:        "pvc-name",
		PersistentVolume:       "pv-1",
		StorageClass:           "powermax-iscsi",
		Driver:                 "csi-powermax.dellemc.com",
		ProvisionedSize:        "16Gi",
		VolumeHandle:           "csi-k8s-mock-398993ad1b-powermaxtest-000197902599-00822",
	},
}
var volume00822 = &v100.Volume{
	VolumeID:         "00822",
	Type:             "TDEV",
	AllocatedPercent: 10,
	CapacityGB:       8.0,
	FloatCapacityMB:  8194.0,
	VolumeIdentifier: "csi-k8s-mock-398993ad1b-powermaxtest",
	WWN:              "60000970000197902273533030383532",
	StorageGroups: []v100.StorageGroupName{
		{StorageGroupName: "csi-BYM-Gold-SRP_1-SG"},
		{StorageGroupName: "csi-no-srp-sg-BYM-worker-2-zegnx4zktvbph"}},
}

func Test_ExportArrayCapacityMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (service.PowerMaxService, *gomock.Controller){
		"success": func(*testing.T) (service.PowerMaxService, *gomock.Controller) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)
			scFinder := mocks.NewMockStorageClassFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(2)

			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			c.EXPECT().GetVolumeByID(gomock.Any(), gomock.Any(), gomock.Any()).Return(volume00822, nil).Times(1)

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
			service.ExportArrayCapacityMetrics(context.Background())
			ctrl.Finish()
		})
	}
}
