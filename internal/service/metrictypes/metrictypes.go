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

package metrictypes

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	pmax "github.com/dell/gopowermax/v2"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	types "github.com/dell/gopowermax/v2/types/v100"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	v1 "k8s.io/api/storage/v1"
)

// PowerMaxClient contains operations for accessing the PowerMax API
//
//go:generate mockgen -destination=mocks/powermax_client_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/metrictypes PowerMaxClient
type PowerMaxClient interface {
	Authenticate(ctx context.Context, configConnect *pmax.ConfigConnect) error
	GetStorageGroup(ctx context.Context, symID string, storageGroupID string) (*types.StorageGroup, error)
	GetVolumeByID(ctx context.Context, symID string, volumeID string) (*types.Volume, error)
	GetArrayPerfKeys(ctx context.Context) (*types.ArrayKeysResult, error)
	GetVolumesMetrics(ctx context.Context, symID string, storageGroups string, metricsQuery []string, firstAvailableTime,
		lastAvailableTime int64) (*types.VolumeMetricsIterator, error)
	GetStorageGroupPerfKeys(ctx context.Context, symID string) (*types.StorageGroupKeysResult, error)
	GetStorageGroupMetrics(ctx context.Context, symID string, storageGroupID string, metricsQuery []string,
		firstAvailableTime, lastAvailableTime int64) (*types.StorageGroupMetricsIterator, error)
}

// VolumeFinder is used to find volume information in kubernetes
//
//go:generate mockgen -destination=mocks/volume_finder_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/metrictypes VolumeFinder
type VolumeFinder interface {
	GetPersistentVolumes(context.Context) ([]k8s.VolumeInfo, error)
}

// StorageClassFinder is used to find storage classes in kubernetes
//
//go:generate mockgen -destination=mocks/storage_class_finder_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/metrictypes StorageClassFinder
type StorageClassFinder interface {
	GetStorageClasses(context.Context) ([]v1.StorageClass, error)
}

// LeaderElector will elect a leader
//
//go:generate mockgen -destination=mocks/leader_elector_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/metrictypes LeaderElector
type LeaderElector interface {
	InitLeaderElection(string, string) error
	IsLeader() bool
}

// NumericMetric numeric metric struct
type NumericMetric struct {
	Labels []attribute.KeyValue
	Name   string
	Value  float64
}

// Service aggregate necessary info and define export metrics methods
//
//go:generate mockgen -destination=mocks/service_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/metrictypes Service
type Service interface {
	GetLogger() *logrus.Logger
	GetPowerMaxClients() map[string][]PowerMaxArray
	GetMetricsRecorder() MetricsRecorder
	GetMaxPowerMaxConnections() int
	GetVolumeFinder() VolumeFinder
	ExportCapacityMetrics(ctx context.Context)
	ExportPerformanceMetrics(ctx context.Context)
	ExportTopologyMetrics(ctx context.Context)
}

// MeterCreator interface is used to create and provide Meter instances, which are used to report measurements
//
//go:generate mockgen -destination=mocks/meter_mocks.go -package=mocks go.opentelemetry.io/otel/metric Meter
type MeterCreator interface {
	MetricProvider() metric.Meter
}

// MetricsRecorder supports recording storage resources metrics
//
//go:generate mockgen -destination=mocks/types_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/metrictypes MetricsRecorder,MeterCreator
type MetricsRecorder interface {
	RecordNumericMetrics(prefix string, labels []attribute.KeyValue, metric VolumeCapacityMetricsRecord) error
	RecordVolPerfMetrics(prefix string, metric VolumePerfMetricsRecord) error
	RecordStorageGroupPerfMetrics(prefix string, metric StorageGroupPerfMetricsRecord) error
	RecordTopologyMetrics(ctx context.Context, meta interface{}, metric *TopologyMetricsRecord) error
}

// PowerMaxArray is a struct that stores all PowerMax connection information.
// It stores gopowermax client that can be directly used to invoke PowerSale API calls.
// This structure is supposed to be parsed from config and mainly is created by GetPowerScaleArrays function.
type PowerMaxArray struct {
	Endpoint       string
	StorageArrayID string
	Username       string
	Password       string
	Insecure       bool
	IsPrimary      bool
	IsActive       bool
	Client         PowerMaxClient
}

// VolumeCapacityMetricsRecord struct for volume capacity
type VolumeCapacityMetricsRecord struct {
	ArrayID, VolumeID, SrpID, StorageGroupID                                                                 string
	StorageClass, Driver, PersistentVolumeName, PersistentVolumeStatus, PersistentVolumeClaimName, Namespace string
	Total, Used, UsedPercent                                                                                 float64
}

type TopologyMeta struct {
	Namespace               string
	PersistentVolumeClaim   string
	PersistentVolumeStatus  string
	VolumeClaimName         string
	PersistentVolume        string
	StorageClass            string
	Driver                  string
	ProvisionedSize         string
	StorageSystemVolumeName string
	StoragePoolName         string
	StorageSystem           string
	Protocol                string
	CreatedTime             string
}

type TopologyMetricsRecord struct {
	TopologyMeta *TopologyMeta
	PVAvailable  int64
}

// StorageGroupPerfMetricsRecord struct for storage group performance
type StorageGroupPerfMetricsRecord struct {
	ArrayID, StorageGroupID                                                                           string
	HostMBReads, HostMBWritten, ReadResponseTime, WriteResponseTime, HostReads, HostWrites, AvgIOSize float64
}

// VolumePerfMetricsRecord struct for volume performance
type VolumePerfMetricsRecord struct {
	ArrayID, VolumeID, StorageGroupID                                                string
	StorageClass, Driver, PersistentVolumeName, PersistentVolumeClaimName, Namespace string
	MBRead, MBWritten, ReadResponseTime, WriteResponseTime, Reads, Writes            float64
}
