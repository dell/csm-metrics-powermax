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
	"github.com/dell/csm-metrics-powermax/utils"
	otelmetric "go.opentelemetry.io/otel/metric"
	"sync"

	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"go.opentelemetry.io/otel/attribute"
)

// MetricsRecorderWrapper contains data used for pushing metrics data
type MetricsRecorderWrapper struct {
	Meter        otelmetric.Meter
	QuotaMetrics sync.Map
}

// RecordNumericMetrics record metrics using Otel's InstrumentProvider
func (mrw *MetricsRecorderWrapper) RecordNumericMetrics(prefix string, labels []attribute.KeyValue, metric types.VolumeCapacityMetricsRecord) error {

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

	done := make(chan struct{})

	reg, err := mrw.Meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		observer.ObserveFloat64(totalCapacity, metric.Total, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(usedCapacity, metric.Used, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(usedCapacityPercentage, metric.Used*100/metric.Total, otelmetric.WithAttributes(labels...))
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

	readBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_bw_megabytes_per_second")
	if err != nil {
		return err
	}

	writeBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_bw_megabytes_per_second")
	if err != nil {
		return err
	}

	readLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_latency_milliseconds")
	if err != nil {
		return err
	}

	writeLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_latency_milliseconds")
	if err != nil {
		return err
	}

	readIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_io_per_second")
	if err != nil {
		return err
	}

	writeIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_io_per_second")
	if err != nil {
		return err
	}

	done := make(chan struct{})
	reg, err := mrw.Meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		observer.ObserveFloat64(readBWMegabytes, metric.MBRead, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeBWMegabytes, metric.MBWritten, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(readLatency, metric.ReadResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeLatency, metric.WriteResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(readIOPS, metric.Reads, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeIOPS, metric.Writes, otelmetric.WithAttributes(labels...))
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

func (mrw *MetricsRecorderWrapper) RecordStorageGroupPerfMetrics(prefix string, metric types.StorageGroupPerfMetricsRecord) error {

	labels := []attribute.KeyValue{
		attribute.String("ArrayID", metric.ArrayID),
		attribute.String("StorageGroupID", metric.StorageGroupID),
		attribute.String("PlotWithMean", "No"),
	}

	readBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_bw_megabytes_per_second")
	if err != nil {
		return err
	}

	writeBWMegabytes, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_bw_megabytes_per_second")
	if err != nil {
		return err
	}

	readLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_latency_milliseconds")
	if err != nil {
		return err
	}

	writeLatency, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_latency_milliseconds")
	if err != nil {
		return err
	}

	readIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_read_io_per_second")
	if err != nil {
		return err
	}

	writeIOPS, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_write_io_per_second")
	if err != nil {
		return err
	}

	averageIOSize, err := mrw.Meter.Float64ObservableUpDownCounter(prefix + "_average_io_size_megabytes_per_second")
	if err != nil {
		return err
	}

	done := make(chan struct{})
	reg, err := mrw.Meter.RegisterCallback(func(ctx context.Context, observer otelmetric.Observer) error {
		observer.ObserveFloat64(readBWMegabytes, metric.HostMBReads, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeBWMegabytes, metric.HostMBWritten, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(readLatency, metric.ReadResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeLatency, metric.WriteResponseTime, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(readIOPS, metric.HostReads, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(writeIOPS, metric.HostWrites, otelmetric.WithAttributes(labels...))
		observer.ObserveFloat64(averageIOSize, utils.UnitsConvert(metric.AvgIOSize, utils.KB, utils.MB), otelmetric.WithAttributes(labels...))
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
