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
	"reflect"
	"testing"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func Test_RecordNumericMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (*metric.MetricsRecorderWrapper, metrictypes.VolumeCapacityMetricsRecord, []attribute.KeyValue, *gomock.Controller, *otlexporters.OtlCollectorExporter, error){
		"success": func(*testing.T) (*metric.MetricsRecorderWrapper, metrictypes.VolumeCapacityMetricsRecord, []attribute.KeyValue, *gomock.Controller, *otlexporters.OtlCollectorExporter, error) {
			exporter := &otlexporters.OtlCollectorExporter{}
			err := exporter.InitExporter()
			if err != nil {
				t.Fatal(err)
			}

			ctrl := gomock.NewController(t)
			otMeter := otel.Meter("powermax_test")
			recorder := &metric.MetricsRecorderWrapper{
				Meter: otMeter,
			}

			metrics := metrictypes.VolumeCapacityMetricsRecord{
				Total:       10,
				Used:        5,
				UsedPercent: 50,
			}
			labels := []attribute.KeyValue{
				attribute.String("ArrayID", uuid.NewString()),
				attribute.String("Driver", "powermax"),
			}

			return recorder, metrics, labels, ctrl, exporter, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder, metrics, labels, ctrl, _, err := tc(t)
			assert.Equal(t, err, recorder.RecordNumericMetrics("powermax_numeric_", labels, metrics))
			ctrl.Finish()
		})
	}
}

func Test_RecordVolPerfMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (*metric.MetricsRecorderWrapper, metrictypes.VolumePerfMetricsRecord, *gomock.Controller, *otlexporters.OtlCollectorExporter, error){
		"success": func(*testing.T) (*metric.MetricsRecorderWrapper, metrictypes.VolumePerfMetricsRecord, *gomock.Controller, *otlexporters.OtlCollectorExporter, error) {
			exporter := &otlexporters.OtlCollectorExporter{}
			err := exporter.InitExporter()
			if err != nil {
				t.Fatal(err)
			}

			ctrl := gomock.NewController(t)
			otMeter := otel.Meter("powermax_test")
			recorder := &metric.MetricsRecorderWrapper{
				Meter: otMeter,
			}

			metrics := metrictypes.VolumePerfMetricsRecord{
				ArrayID:                   uuid.NewString(),
				VolumeID:                  uuid.NewString(),
				Driver:                    "powermax",
				StorageClass:              "myStorageClass",
				PersistentVolumeName:      "myPersistentVolumeName",
				PersistentVolumeClaimName: "myPersistentVolumeClaimName",
				Namespace:                 "myNamespace",
			}

			return recorder, metrics, ctrl, exporter, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder, metrics, ctrl, _, err := tc(t)
			assert.Equal(t, err, recorder.RecordVolPerfMetrics("powermax_volume_", metrics))
			ctrl.Finish()
		})
	}
}

func Test_RecordStorageGroupPerfMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (*metric.MetricsRecorderWrapper, metrictypes.StorageGroupPerfMetricsRecord, *gomock.Controller, *otlexporters.OtlCollectorExporter, error){
		"success": func(*testing.T) (*metric.MetricsRecorderWrapper, metrictypes.StorageGroupPerfMetricsRecord, *gomock.Controller, *otlexporters.OtlCollectorExporter, error) {
			exporter := &otlexporters.OtlCollectorExporter{}
			err := exporter.InitExporter()
			if err != nil {
				t.Fatal(err)
			}

			ctrl := gomock.NewController(t)
			otMeter := otel.Meter("powermax_test")
			recorder := &metric.MetricsRecorderWrapper{
				Meter: otMeter,
			}

			metrics := metrictypes.StorageGroupPerfMetricsRecord{
				ArrayID:        uuid.NewString(),
				StorageGroupID: uuid.NewString(),
			}

			return recorder, metrics, ctrl, exporter, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder, metrics, ctrl, _, err := tc(t)
			assert.Equal(t, err, recorder.RecordStorageGroupPerfMetrics("powermax_storage_group_", metrics))
			ctrl.Finish()
		})
	}
}

func Test_RecordTopologyMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (*metric.MetricsRecorderWrapper, metrictypes.TopologyMetricsRecord, *gomock.Controller, *otlexporters.OtlCollectorExporter, error){
		"success": func(*testing.T) (*metric.MetricsRecorderWrapper, metrictypes.TopologyMetricsRecord, *gomock.Controller, *otlexporters.OtlCollectorExporter, error) {
			exporter := &otlexporters.OtlCollectorExporter{}
			err := exporter.InitExporter()
			if err != nil {
				t.Fatal(err)
			}

			t1, err := time.Parse(time.RFC3339, "2022-06-06T20:00:00+00:00")
			assert.Nil(t, err)

			ctrl := gomock.NewController(t)
			otMeter := otel.Meter("powermax_test")

			metrics := metrictypes.TopologyMetricsRecord{
				TopologyMeta: &metrictypes.TopologyMeta{
					Namespace:               "namespace-1",
					PersistentVolumeClaim:   "pvc-uid",
					VolumeClaimName:         "pvc-1",
					PersistentVolumeStatus:  "Bound",
					PersistentVolume:        "pv-1",
					StorageClass:            "sc-1",
					Driver:                  "csi-powermax.dellemc.com",
					ProvisionedSize:         "16Gi",
					StorageSystemVolumeName: "k8s-7242537ae1",
					StoragePoolName:         "pool-1",
					StorageSystem:           "000197902573",
					Protocol:                "",
					CreatedTime:             t1.String(),
				},
				PVAvailable: 10,
			}

			recorder := &metric.MetricsRecorderWrapper{
				Meter: otMeter,
			}

			return recorder, metrics, ctrl, exporter, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder, metrics, ctrl, _, err := tc(t)
			assert.Equal(t, err, recorder.RecordTopologyMetrics(context.Background(), metrics.TopologyMeta, &metrics))
			ctrl.Finish()
		})
	}
}

func TestHaveLabelsChanged(t *testing.T) {
	tests := []struct {
		name                  string
		currentLabels         []attribute.KeyValue
		newLabels             []attribute.KeyValue
		wantHaveLabelsChanged bool
		wantUpdatedLabels     []attribute.KeyValue
	}{
		{
			name: "no change",
			currentLabels: []attribute.KeyValue{
				{Key: "key1", Value: attribute.StringValue("value1")},
				{Key: "key2", Value: attribute.StringValue("value2")},
			},
			newLabels: []attribute.KeyValue{
				{Key: "key1", Value: attribute.StringValue("value1")},
				{Key: "key2", Value: attribute.StringValue("value2")},
			},
			wantHaveLabelsChanged: false,
			wantUpdatedLabels: []attribute.KeyValue{
				{Key: "key1", Value: attribute.StringValue("value1")},
				{Key: "key2", Value: attribute.StringValue("value2")},
			},
		},
		{
			name: "label value changed",
			currentLabels: []attribute.KeyValue{
				{Key: "key1", Value: attribute.StringValue("value1")},
				{Key: "key2", Value: attribute.StringValue("value2")},
			},
			newLabels: []attribute.KeyValue{
				{Key: "key1", Value: attribute.StringValue("value3")},
				{Key: "key2", Value: attribute.StringValue("value4")},
			},
			wantHaveLabelsChanged: true,
			wantUpdatedLabels: []attribute.KeyValue{
				{Key: "key1", Value: attribute.StringValue("value3")},
				{Key: "key2", Value: attribute.StringValue("value4")},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHaveLabelsChanged, gotUpdatedLabels := metric.HaveLabelsChanged(tt.currentLabels, tt.newLabels)
			if gotHaveLabelsChanged != tt.wantHaveLabelsChanged {
				t.Errorf("haveLabelsChanged() haveLabelsChanged = %v, want %v", gotHaveLabelsChanged, tt.wantHaveLabelsChanged)
			}
			if !reflect.DeepEqual(gotUpdatedLabels, tt.wantUpdatedLabels) {
				t.Errorf("haveLabelsChanged() updatedLabels = %v, want %v", gotUpdatedLabels, tt.wantUpdatedLabels)
			}
		})
	}
}
