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

package types

import (
	"context"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	pmax "github.com/dell/gopowermax/v2"
	types "github.com/dell/gopowermax/v2/types/v100"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/asyncint64"
	v1 "k8s.io/api/storage/v1"
)

// PowerMaxClient contains operations for accessing the PowerMax API
//
//go:generate mockgen -destination=mocks/powermax_client_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/types PowerMaxClient
type PowerMaxClient interface {
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
//go:generate mockgen -destination=mocks/volume_finder_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/types VolumeFinder
type VolumeFinder interface {
	GetPersistentVolumes(context.Context) ([]k8s.VolumeInfo, error)
}

// StorageClassFinder is used to find storage classes in kubernetes
//
//go:generate mockgen -destination=mocks/storage_class_finder_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/types StorageClassFinder
type StorageClassFinder interface {
	GetStorageClasses(context.Context) ([]v1.StorageClass, error)
}

// LeaderElector will elect a leader
//
//go:generate mockgen -destination=mocks/leader_elector_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/types LeaderElector
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
//go:generate mockgen -destination=mocks/service_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/types Service
type Service interface {
	GetLogger() *logrus.Logger
	GetPowerMaxClients() map[string]PowerMaxClient
	GetMetricsRecorder() MetricsRecorder
	GetMaxPowerMaxConnections() int
	GetVolumeFinder() VolumeFinder
	ExportCapacityMetrics(ctx context.Context)
	ExportPerformanceMetrics(ctx context.Context)
}

// AsyncMetricCreator to create AsyncInt64/AsyncFloat64 InstrumentProvider
//
//go:generate mockgen -destination=mocks/asyncint64mock/instrument_asyncint64_provider_mocks.go -package=asyncint64mock go.opentelemetry.io/otel/metric/instrument/asyncint64 InstrumentProvider
//go:generate mockgen -destination=mocks/asyncfloat64mock/instrument_asyncfloat64_provider_mocks.go -package=asyncfloat64mock go.opentelemetry.io/otel/metric/instrument/asyncfloat64 InstrumentProvider
type AsyncMetricCreator interface {
	AsyncInt64() asyncint64.InstrumentProvider
	AsyncFloat64() asyncfloat64.InstrumentProvider
}

// MetricsRecorder supports recording storage resources metrics
//
//go:generate mockgen -destination=mocks/types_mocks.go -package=mocks github.com/dell/csm-metrics-powermax/internal/service/types MetricsRecorder,AsyncMetricCreator
type MetricsRecorder interface {
	RecordNumericMetrics(ctx context.Context, metric []NumericMetric) error
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
	Client         pmax.Pmax
}

// VolumeCapacityMetricsRecord struct for volume capacity
type VolumeCapacityMetricsRecord struct {
	ArrayID, VolumeID, SrpID, StorageGroupID                                         string
	StorageClass, Driver, PersistentVolumeName, PersistentVolumeClaimName, NameSpace string
	Total, Used, UsedPercent                                                         float64
}

// StorageGroupPerfMetricsRecord struct for storage group performance
type StorageGroupPerfMetricsRecord struct {
	ArrayID, StorageGroupID                                                                           string
	HostMBReads, HostMBWritten, ReadResponseTime, WriteResponseTime, HostReads, HostWrites, AvgIOSize float64
}

// VolumePerfMetricsRecord struct for volume performance
type VolumePerfMetricsRecord struct {
	ArrayID, VolumeID, StorageGroupID                                                string
	StorageClass, Driver, PersistentVolumeName, PersistentVolumeClaimName, NameSpace string
	MBRead, MBWritten, ReadResponseTime, WriteResponseTime, Reads, Writes            float64
}
