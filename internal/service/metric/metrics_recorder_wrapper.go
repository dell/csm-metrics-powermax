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
	"log"
	"sync"

	"github.com/dell/csm-metrics-powermax/utils"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"go.opentelemetry.io/otel/attribute"
)

// MetricsRecorderWrapper contains data used for pushing metrics data
type MetricsRecorderWrapper struct {
	Meter               otelmetric.Meter
	NumericMetrics      sync.Map
	VolumeMetrics       sync.Map
	StorageGroupMetrics sync.Map
	Labels              sync.Map
}

type NumericMetrics struct {
	TotalCapacity          otelmetric.Float64ObservableUpDownCounter
	UsedCapacity           otelmetric.Float64ObservableUpDownCounter
	UsedCapacityPercentage otelmetric.Float64ObservableUpDownCounter
}

type VolumeMetrics struct {
	ReadBW       otelmetric.Float64ObservableUpDownCounter
	WriteBW      otelmetric.Float64ObservableUpDownCounter
	ReadIOPS     otelmetric.Float64ObservableUpDownCounter
	WriteIOPS    otelmetric.Float64ObservableUpDownCounter
	ReadLatency  otelmetric.Float64ObservableUpDownCounter
	WriteLatency otelmetric.Float64ObservableUpDownCounter
}

type StorageGroupMetrics struct {
	VolumeMetrics
	AverageIOSize otelmetric.Float64ObservableUpDownCounter
}

func (mrw *MetricsRecorderWrapper) initNumbericMetrics(prefix string, metaID string) *NumericMetrics {
	totalCapacity, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "total_capacity_gigabytes")
	usedCapacity, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "used_capacity_gigabytes")
	usedCapacityPercentage, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "used_capacity_percentage")

	metrics := &NumericMetrics{TotalCapacity: totalCapacity, UsedCapacity: usedCapacity, UsedCapacityPercentage: usedCapacityPercentage}

	mrw.NumericMetrics.Store(metaID, metrics)

	return metrics
}

func (mrw *MetricsRecorderWrapper) initVolumeMetrics(prefix string, metaID string) *VolumeMetrics {
	readBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_bw_megabytes_per_second")
	writeBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_bw_megabytes_per_second")
	readLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_latency_milliseconds")
	writeLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_latency_milliseconds")
	readIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_io_per_second")
	writeIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_io_per_second")

	metrics := &VolumeMetrics{ReadBW: readBWMegabytes, WriteBW: writeBWMegabytes, ReadIOPS: readIOPS, WriteIOPS: writeIOPS, ReadLatency: readLatency, WriteLatency: writeLatency}

	mrw.VolumeMetrics.Store(metaID, metrics)

	return metrics
}

func (mrw *MetricsRecorderWrapper) initStorageGroupMetrics(prefix string, metaID string) *StorageGroupMetrics {
	readBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_bw_megabytes_per_second")
	writeBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_bw_megabytes_per_second")
	readLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_latency_milliseconds")
	writeLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_latency_milliseconds")
	readIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_io_per_second")
	writeIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_io_per_second")
	averageIOSize, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_average_io_size_megabytes_per_second")

	metrics := &StorageGroupMetrics{
		VolumeMetrics: VolumeMetrics{
			ReadBW:       readBWMegabytes,
			WriteBW:      writeBWMegabytes,
			ReadIOPS:     readIOPS,
			WriteIOPS:    writeIOPS,
			ReadLatency:  readLatency,
			WriteLatency: writeLatency,
		},
		AverageIOSize: averageIOSize,
	}

	mrw.StorageGroupMetrics.Store(metaID, metrics)

	return metrics
}

// RecordNumericMetrics record metrics using Otel's InstrumentProvider
func (mrw *MetricsRecorderWrapper) RecordNumericMetrics(prefix string, labels []attribute.KeyValue, metric types.VolumeCapacityMetricsRecord) error {
	// metricsMapValue, ok := mrw.NumericMetrics.Load(metric.ArrayID)
	// if !ok {
	// 	metricsMapValue = mrw.initNumbericMetrics(prefix, metric.ArrayID)
	// 	mrw.Labels.Store(metric.ArrayID, labels)
	// } else {
	// 	currentLabels, ok := mrw.Labels.Load(metric.ArrayID)
	// 	if ok {
	// 		currentLabels := currentLabels.([]attribute.KeyValue)
	// 		updatedLabels := currentLabels
	// 		haveLabelsChanged := false
	// 		for i, current := range currentLabels {
	// 			for _, new := range labels {
	// 				if current.Key == new.Key {
	// 					if current.Value != new.Value {
	// 						updatedLabels[i].Value = new.Value
	// 						haveLabelsChanged = true
	// 					}
	// 				}
	// 			}
	// 		}
	// 		if haveLabelsChanged {
	// 			metricsMapValue = mrw.initNumbericMetrics(prefix, metric.ArrayID)
	// 			mrw.Labels.Store(metric.ArrayID, labels)
	// 		}
	// 	}
	// }

	// _ = metricsMapValue.(*NumericMetrics)

	// log.Printf("Testing Synchronous Callback: RecordNumericMetrics prefix: %s, labels: %+v, metric: %+v", prefix, labels, metric)

	// counter, _ := mrw.Meter.Float64Gauge(prefix + "total_capacity_gigabytes")
	// counter.Record(context.Background(), metric.Total, otelmetric.WithAttributes(labels...))

	// counter, _ = mrw.Meter.Float64Gauge((prefix + "used_capacity_gigabytes"))
	// counter.Record(context.Background(), metric.Used, otelmetric.WithAttributes(labels...))

	// counter, _ = mrw.Meter.Float64Gauge(prefix + "used_capacity_percentage")
	// counter.Record(context.Background(), metric.Used*100/metric.Total, otelmetric.WithAttributes(labels...))

	totalCapacity, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "total_capacity_gigabytes")
	if err != nil {
		return err
	}

	usedCapacity, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "used_capacity_gigabytes")
	if err != nil {
		return err
	}

	usedCapacityPercentage, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "used_capacity_percentage")
	if err != nil {
		return err
	}

	_, _ = mrw.Meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		// log.Printf("Calling RecordNumericMetrics RegisterCallback, metrics: %+v, labels: %+v", metric, labels)

		observer.ObserveFloat64(totalCapacity, metric.Total, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(usedCapacity, metric.Used, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(usedCapacityPercentage, metric.Used*100/metric.Total, otelmetric.WithAttributes(labels...))
		return nil
	},
		totalCapacity,
		usedCapacity,
		usedCapacityPercentage)
	return nil
}

func (mrw *MetricsRecorderWrapper) RecordVolPerfMetrics(prefix string, metric types.VolumePerfMetricsRecord) error {
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

	metricsMapValue, ok := mrw.VolumeMetrics.Load(metric.ArrayID)
	if !ok {
		log.Println("[RecordVolPerfMetrics] initVolumeMetrics")
		metricsMapValue = mrw.initVolumeMetrics(prefix, metric.ArrayID)
		mrw.Labels.Store(metric.ArrayID, labels)
	}

	metrics := metricsMapValue.(*VolumeMetrics)

	// readBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_bw_megabytes_per_second")
	// writeBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_bw_megabytes_per_second")
	// readLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_latency_milliseconds")
	// writeLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_latency_milliseconds")
	// readIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_io_per_second")
	// writeIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_io_per_second")

	_, _ = mrw.Meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		// log.Println("Calling RecordVolPerfMetrics RegisterCallback")
		observer.ObserveFloat64(metrics.ReadBW, metric.MBRead, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.WriteBW, metric.MBWritten, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.ReadLatency, metric.ReadResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.WriteLatency, metric.WriteResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.ReadIOPS, metric.Reads, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.WriteIOPS, metric.Writes, otelmetric.WithAttributes(labels...))
		return nil
	},
		metrics.ReadBW,
		metrics.WriteBW,
		metrics.ReadLatency,
		metrics.WriteLatency,
		metrics.ReadIOPS,
		metrics.WriteIOPS)
	return nil
}

func (mrw *MetricsRecorderWrapper) RecordStorageGroupPerfMetrics(prefix string, metric types.StorageGroupPerfMetricsRecord) error {
	labels := []attribute.KeyValue{
		attribute.String("ArrayID", metric.ArrayID),
		attribute.String("StorageGroupID", metric.StorageGroupID),
		attribute.String("PlotWithMean", "No"),
	}

	log.Printf("RecordStorageGroupPerfMetrics arrayID: %s, storageGroupID: %s", metric.ArrayID, metric.StorageGroupID)

	metricsMapValue, ok := mrw.StorageGroupMetrics.Load(metric.ArrayID)
	if !ok {
		log.Println("[RecordVolPerfMetrics] initStorageGroupMetrics")
		metricsMapValue = mrw.initStorageGroupMetrics(prefix, metric.ArrayID)
		mrw.Labels.Store(metric.ArrayID, labels)
	}

	metrics := metricsMapValue.(*StorageGroupMetrics)

	// readBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_bw_megabytes_per_second")
	// writeBWMegabytes, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_bw_megabytes_per_second")
	// readLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_latency_milliseconds")
	// writeLatency, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_latency_milliseconds")
	// readIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_io_per_second")
	// writeIOPS, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_io_per_second")
	// averageIOSize, _ := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_average_io_size_megabytes_per_second")

	_, _ = mrw.Meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		// log.Println("Calling RecordStorageGroupPerfMetrics RegisterCallback")
		observer.ObserveFloat64(metrics.ReadBW, metric.HostMBReads, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.WriteBW, metric.HostMBWritten, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.ReadLatency, metric.ReadResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.WriteLatency, metric.WriteResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.ReadIOPS, metric.HostReads, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.WriteIOPS, metric.HostWrites, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(metrics.AverageIOSize, utils.UnitsConvert(metric.AvgIOSize, utils.KB, utils.MB), otelmetric.WithAttributes(labels...))
		return nil
	}, metrics.ReadBW,
		metrics.WriteBW,
		metrics.ReadLatency,
		metrics.WriteLatency,
		metrics.ReadIOPS,
		metrics.WriteIOPS,
		metrics.AverageIOSize)

	return nil
}
