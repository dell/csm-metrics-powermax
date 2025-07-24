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

package metric

import (
	"context"
	"errors"
	"sync"

	"github.com/dell/csm-metrics-powermax/utilsconverter"
	otelMetric "go.opentelemetry.io/otel/metric"

	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"go.opentelemetry.io/otel/attribute"
)

// MetricsRecorderWrapper contains data used for pushing metrics data
type MetricsRecorderWrapper struct {
	Meter           otelMetric.Meter
	Labels          sync.Map
	QuotaMetrics    sync.Map
	TopologyMetrics sync.Map
}

type (
	loadMetricsFunc func(metaID string) (value any, ok bool)
	initMetricsFunc func(prefix, metaID string, labels []attribute.KeyValue) (any, error)
)

// TopologyMetricsData contains topology metrics when PV is available on cluster
type TopologyMetricsData struct {
	PvAvailable otelMetric.Float64ObservableUpDownCounter
}

const (
	TotalCapacityGigabytes          = "total_capacity_gigabytes"
	UsedCapacityGigabytes           = "used_capacity_gigabytes"
	UsedCapacityPercentage          = "used_capacity_percentage"
	ReadBWMegabytesPerSecond        = "_read_bw_megabytes_per_second"
	WriteBWMegabytesPerSecond       = "_write_bw_megabytes_per_second"
	ReadLatencyMilliseconds         = "_read_latency_milliseconds"
	WriteLatencyMilliseconds        = "_write_latency_milliseconds"
	ReadIOPerSecond                 = "_read_io_per_second"
	WriteIOPerSecond                = "_write_io_per_second"
	AverageIOSizeMegabytesPerSecond = "_average_io_size_megabytes_per_second"
)

// RecordNumericMetrics record metrics using Otel's InstrumentProvider
func (mrw *MetricsRecorderWrapper) RecordNumericMetrics(prefix string, labels []attribute.KeyValue, metric metrictypes.VolumeCapacityMetricsRecord) error {
	totalCapacity, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + TotalCapacityGigabytes)
	if err != nil {
		return err
	}

	usedCapacity, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + UsedCapacityGigabytes)
	if err != nil {
		return err
	}

	usedCapacityPercentage, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + UsedCapacityPercentage)
	if err != nil {
		return err
	}

	done := make(chan struct{})

	reg, err := mrw.Meter.RegisterCallback(func(_ context.Context, observer otelMetric.Observer) error {
		observer.ObserveFloat64(totalCapacity, metric.Total, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(usedCapacity, metric.Used, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(usedCapacityPercentage, metric.Used*100/metric.Total, otelMetric.WithAttributes(labels...))
		go func() {
			done <- struct{}{}
		}()
		return nil
	},
		totalCapacity,
		usedCapacity,
		usedCapacityPercentage)
	if err != nil {
		return err
	}
	<-done
	_ = reg.Unregister()

	return nil
}

func (mrw *MetricsRecorderWrapper) RecordVolPerfMetrics(prefix string, metric metrictypes.VolumePerfMetricsRecord) error {
	labels := []attribute.KeyValue{
		attribute.String("VolumeID", metric.VolumeID),
		attribute.String("ArrayID", metric.ArrayID),
		attribute.String("Driver", metric.Driver),
		attribute.String("StorageClass", metric.StorageClass),
		attribute.String("PersistentVolumeName", metric.PersistentVolumeName),
		attribute.String("PersistentVolumeClaimName", metric.PersistentVolumeClaimName),
		attribute.String("Namespace", metric.Namespace),
		attribute.String("PlotWithMean", "No"),
	}

	readBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + ReadBWMegabytesPerSecond)
	if err != nil {
		return err
	}

	writeBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + WriteBWMegabytesPerSecond)
	if err != nil {
		return err
	}

	readLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + ReadLatencyMilliseconds)
	if err != nil {
		return err
	}

	writeLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + WriteLatencyMilliseconds)
	if err != nil {
		return err
	}

	readIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + ReadIOPerSecond)
	if err != nil {
		return err
	}

	writeIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + WriteIOPerSecond)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	reg, err := mrw.Meter.RegisterCallback(func(_ context.Context, observer otelMetric.Observer) error {
		observer.ObserveFloat64(readBWMegabytes, metric.MBRead, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeBWMegabytes, metric.MBWritten, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(readLatency, metric.ReadResponseTime, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeLatency, metric.WriteResponseTime, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(readIOPS, metric.Reads, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeIOPS, metric.Writes, otelMetric.WithAttributes(labels...))
		go func() {
			done <- struct{}{}
		}()
		return nil
	},
		readBWMegabytes,
		writeBWMegabytes,
		readLatency,
		writeLatency,
		readIOPS,
		writeIOPS)
	if err != nil {
		return err
	}
	<-done
	_ = reg.Unregister()

	return nil
}

func (mrw *MetricsRecorderWrapper) RecordStorageGroupPerfMetrics(prefix string, metric metrictypes.StorageGroupPerfMetricsRecord) error {
	labels := []attribute.KeyValue{
		attribute.String("ArrayID", metric.ArrayID),
		attribute.String("StorageGroupID", metric.StorageGroupID),
		attribute.String("PlotWithMean", "No"),
	}

	readBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + ReadBWMegabytesPerSecond)
	if err != nil {
		return err
	}

	writeBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + WriteBWMegabytesPerSecond)
	if err != nil {
		return err
	}

	readLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + ReadLatencyMilliseconds)
	if err != nil {
		return err
	}

	writeLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + WriteLatencyMilliseconds)
	if err != nil {
		return err
	}

	readIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + ReadIOPerSecond)
	if err != nil {
		return err
	}

	writeIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + WriteIOPerSecond)
	if err != nil {
		return err
	}

	averageIOSize, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + AverageIOSizeMegabytesPerSecond)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	reg, err := mrw.Meter.RegisterCallback(func(_ context.Context, observer otelMetric.Observer) error {
		observer.ObserveFloat64(readBWMegabytes, metric.HostMBReads, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeBWMegabytes, metric.HostMBWritten, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(readLatency, metric.ReadResponseTime, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeLatency, metric.WriteResponseTime, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(readIOPS, metric.HostReads, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeIOPS, metric.HostWrites, otelMetric.WithAttributes(labels...))
		observer.ObserveFloat64(averageIOSize, utilsconverter.UnitsConvert(metric.AvgIOSize, utilsconverter.KB, utilsconverter.MB), otelMetric.WithAttributes(labels...))
		go func() {
			done <- struct{}{}
		}()
		return nil
	},
		readBWMegabytes,
		writeBWMegabytes,
		readLatency,
		writeLatency,
		readIOPS,
		writeIOPS,
		averageIOSize)
	if err != nil {
		return err
	}
	<-done
	_ = reg.Unregister()
	return nil
}

// RecordTopologyMetrics will publish topology data to Otel
func (mrw *MetricsRecorderWrapper) RecordTopologyMetrics(_ context.Context, meta interface{}, metric *metrictypes.TopologyMetricsRecord) error {
	var metaID string
	var labels []attribute.KeyValue

	switch v := meta.(type) {
	case *metrictypes.TopologyMeta:
		metaID = v.PersistentVolume
		labels = []attribute.KeyValue{
			attribute.String("PersistentVolumeClaim", v.PersistentVolumeClaim),
			attribute.String("Driver", v.Driver),
			attribute.String("PersistentVolume", v.PersistentVolume),
			attribute.String("PersistentVolumeStatus", v.PersistentVolumeStatus),
			attribute.String("StorageClass", v.StorageClass),
			attribute.String("PlotWithMean", "No"),
			attribute.String("StorageClass", v.StorageClass),
			attribute.String("Namespace", v.Namespace),
			attribute.String("ProvisionedSize", v.ProvisionedSize),
			attribute.String("StorageSystemVolumeName", v.StorageSystemVolumeName),
			attribute.String("StorageSystem", v.StorageSystem),
			attribute.String("Protocol", v.Protocol),
			attribute.String("CreatedTime", v.CreatedTime),
		}
	default:
		return errors.New("unknown MetaData type")
	}

	loadMetricsFunc := func(metaID string) (any, bool) {
		return mrw.TopologyMetrics.Load(metaID)
	}

	initMetricsFunc := func(_, metaID string, labels []attribute.KeyValue) (any, error) {
		return mrw.initTopologyMetrics(metaID, labels)
	}

	metricsMapValue, err := updateLabels("", metaID, labels, mrw, loadMetricsFunc, initMetricsFunc)
	if err != nil {
		return err
	}

	metrics := metricsMapValue.(*TopologyMetricsData)

	done := make(chan struct{})
	reg, err := mrw.Meter.RegisterCallback(func(_ context.Context, obs otelMetric.Observer) error {
		obs.ObserveFloat64(metrics.PvAvailable, float64(metric.PVAvailable), otelMetric.WithAttributes(labels...))
		go func() {
			done <- struct{}{}
		}()
		return nil
	},
		metrics.PvAvailable)
	if err != nil {
		return err
	}

	<-done

	_ = reg.Unregister()

	return nil
}

func (mrw *MetricsRecorderWrapper) initTopologyMetrics(metaID string, labels []attribute.KeyValue) (*TopologyMetricsData, error) {
	pvAvailable, _ := mrw.Meter.Float64ObservableUpDownCounter("karavi_topology_metrics")

	metrics := &TopologyMetricsData{
		PvAvailable: pvAvailable,
	}

	mrw.TopologyMetrics.Store(metaID, metrics)
	mrw.Labels.Store(metaID, labels)

	return metrics, nil
}

func updateLabels(prefix, metaID string, labels []attribute.KeyValue, mrw *MetricsRecorderWrapper, loadMetrics loadMetricsFunc, initMetrics initMetricsFunc) (any, error) {
	metricsMapValue, ok := loadMetrics(metaID)

	if !ok {
		newMetrics, err := initMetrics(prefix, metaID, labels)
		if err != nil {
			return nil, err
		}
		metricsMapValue = newMetrics
	} else {
		// If VolumeSpaceMetrics for this MetricsWrapper exist, then update the labels
		currentLabels, ok := mrw.Labels.Load(metaID)
		if ok {
			haveLabelsChanged, updatedLabels := haveLabelsChanged(currentLabels.([]attribute.KeyValue), labels)
			if haveLabelsChanged {
				newMetrics, err := initMetrics(prefix, metaID, updatedLabels)
				if err != nil {
					return nil, err
				}
				metricsMapValue = newMetrics
			}
		}
	}

	done := make(chan struct{})
	defer close(done)

	return metricsMapValue, nil
}

// haveLabelsChanged checks if labels have been changed
func haveLabelsChanged(currentLabels []attribute.KeyValue, labels []attribute.KeyValue) (bool, []attribute.KeyValue) {
	updatedLabels := currentLabels
	haveLabelsChanged := false
	for i, current := range currentLabels {
		for _, new := range labels {
			if current.Key == new.Key {
				if current.Value != new.Value {
					updatedLabels[i].Value = new.Value
					haveLabelsChanged = true
				}
			}
		}
	}
	return haveLabelsChanged, updatedLabels
}
