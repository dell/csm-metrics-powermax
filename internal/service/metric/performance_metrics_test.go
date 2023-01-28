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
	"errors"
	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	"github.com/dell/gopowermax/v2/types/v100"
	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

var mockVolumes = []k8s.VolumeInfo{
	{
		Namespace:              "karavi",
		PersistentVolumeClaim:  "pvc-uid1",
		PersistentVolumeStatus: "Bound",
		VolumeClaimName:        "pvc-name",
		PersistentVolume:       "pv-1",
		StorageClass:           "powermax-iscsi",
		Driver:                 "csi-powermax.dellemc.com",
		ProvisionedSize:        "16Gi",
		VolumeHandle:           "csi-k8s-mock-398993ad1b-powermaxtest-000197902599-00822",
		StorageGroup:           "csi-BYM-Gold-SRP_1-SG",
	},
	{
		Namespace:              "karavi",
		PersistentVolumeClaim:  "pvc-uid2",
		PersistentVolumeStatus: "Bound",
		VolumeClaimName:        "pvc-name",
		PersistentVolume:       "pv-2",
		StorageClass:           "powermax-iscsi",
		Driver:                 "csi-powermax.dellemc.com",
		ProvisionedSize:        "8Gi",
		VolumeHandle:           "csi-PTR-ptrpmax-33ef905ff1-testpowermax-000197902599-002C",
		StorageGroup:           "csi-PTR-Gold-SRP_1-SG",
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

var volume002C = &v100.Volume{
	VolumeID:         "002C",
	Type:             "TDEV",
	AllocatedPercent: 10,
	CapacityGB:       8.0,
	FloatCapacityMB:  8194.0,
	VolumeIdentifier: "csi-PTR-ptrpmax-33ef905ff1-testpowermax",
	WWN:              "60000970000197902573533030334537",
	StorageGroups: []v100.StorageGroupName{
		{StorageGroupName: "csi-PTR-Gold-SRP_1-SG"},
		{StorageGroupName: "csi-no-srp-sg-PTR-peter-worker-1"}},
}

var arrayTimeResult = v100.ArrayKeysResult{
	ArrayInfos: []v100.ArrayInfo{
		{
			SymmetrixID:        "000197902599",
			FirstAvailableDate: 0,
			LastAvailableDate:  1,
		},
		{
			SymmetrixID:        "000197902572",
			FirstAvailableDate: 0,
			LastAvailableDate:  1,
		},
	},
}

var storageGroupTimeResult = v100.StorageGroupKeysResult{
	StorageGroupInfos: []v100.StorageGroupInfo{
		{
			StorageGroupID:     "csi-BYM-Gold-SRP_1-SG",
			FirstAvailableDate: 0,
			LastAvailableDate:  1,
		},
		{
			StorageGroupID:     "csi-PTR-Gold-SRP_1-SG",
			FirstAvailableDate: 0,
			LastAvailableDate:  1,
		},
		{
			StorageGroupID:     "csi-no-srp-sg-BYM-worker-2-zegnx4zktvbph",
			FirstAvailableDate: 0,
			LastAvailableDate:  1,
		},
		{
			StorageGroupID:     "csi-no-srp-sg-PTR-peter-worker-1",
			FirstAvailableDate: 0,
			LastAvailableDate:  1,
		},
	},
}

var volumePerfMetricsResult = v100.VolumeMetricsIterator{
	ResultList: v100.VolumeMetricsResultList{
		Result: []v100.VolumeResult{
			{
				VolumeResult: []v100.VolumeMetric{{
					MBRead:            0,
					MBWritten:         0,
					Reads:             0,
					Writes:            0,
					ReadResponseTime:  0,
					WriteResponseTime: 0,
					Timestamp:         0,
				}},
				VolumeID:      "00822",
				StorageGroups: "csi-BYM-Gold-SRP_1-SG,csi-no-srp-sg-BYM-worker-2-zegnx4zktvbph",
			},
			{
				VolumeResult: []v100.VolumeMetric{{
					MBRead:            0,
					MBWritten:         0,
					Reads:             0,
					Writes:            0,
					ReadResponseTime:  0,
					WriteResponseTime: 0,
					Timestamp:         0,
				}},
				VolumeID:      "002C",
				StorageGroups: "csi-PTR-Gold-SRP_1-SG,csi-no-srp-sg-PTR-peter-worker-1",
			},
		},
		From: 0,
		To:   1,
	},
	ID:             "",
	Count:          1,
	ExpirationTime: 1,
	MaxPageSize:    1,
}

// sample storage group performance metrics
var storageGroupPerfMetricsResult = v100.StorageGroupMetricsIterator{
	ResultList: v100.StorageGroupMetricsResultList{
		Result: []v100.StorageGroupMetric{{
			HostReads:         0,
			HostWrites:        0,
			HostMBReads:       0,
			HostMBWritten:     0,
			ReadResponseTime:  0,
			WriteResponseTime: 0,
			AllocatedCapacity: 0,
			AvgIOSize:         0,
			Timestamp:         0,
		}},
		From: 0,
		To:   1,
	},
	ID:             "",
	Count:          1,
	ExpirationTime: 1,
	MaxPageSize:    1,
}

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
	tests := map[string]func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error){
		"success": func(t *testing.T) (metric.PerformanceMetrics, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			metrics := mocks.NewMockMetricsRecorder(ctrl)
			volFinder := mocks.NewMockVolumeFinder(ctrl)

			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Times(4)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).Times(2)
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

			clients := make(map[string]types.PowerMaxClient)

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

			clients := make(map[string]types.PowerMaxClient)
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

			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			err := errors.New("failed to get perf keys")
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(nil, err).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(nil, err).Times(1)
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

			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			err := errors.New("failed to get metric")
			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, err).Times(2)
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
			metrics.EXPECT().RecordNumericMetrics(gomock.Any(), gomock.Any()).Return(err).Times(4)
			volFinder.EXPECT().GetPersistentVolumes(gomock.Any()).Return(mockVolumes, nil).Times(1)

			clients := make(map[string]types.PowerMaxClient)
			c := mocks.NewMockPowerMaxClient(ctrl)
			clients["000197902599"] = c

			c.EXPECT().GetArrayPerfKeys(gomock.Any()).Return(&arrayTimeResult, nil).Times(1)
			c.EXPECT().GetStorageGroupPerfKeys(gomock.Any(), gomock.Any()).Return(&storageGroupTimeResult, nil).Times(1)
			c.EXPECT().GetVolumesMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&volumePerfMetricsResult, nil).Times(1)
			c.EXPECT().GetStorageGroupMetrics(gomock.Any(), gomock.Any(), gomock.Any(),
				gomock.Any(), gomock.Any(), gomock.Any()).Return(&storageGroupPerfMetricsResult, nil).Times(2)
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
