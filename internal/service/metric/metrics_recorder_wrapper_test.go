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
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func Test_RecordNumericMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (*metric.MetricsRecorderWrapper, types.VolumeCapacityMetricsRecord, []attribute.KeyValue, *gomock.Controller, error){
		"success": func(*testing.T) (*metric.MetricsRecorderWrapper, types.VolumeCapacityMetricsRecord, []attribute.KeyValue, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			otMeter := otel.Meter("powermax_test")
			recorder := &metric.MetricsRecorderWrapper{
				Meter: otMeter,
			}

			metrics := types.VolumeCapacityMetricsRecord{
				Total:       10,
				Used:        5,
				UsedPercent: 50,
			}
			labels := []attribute.KeyValue{
				attribute.String("ArrayID", uuid.NewString()),
				attribute.String("Driver", "powermax"),
			}

			return recorder, metrics, labels, ctrl, nil
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder, metrics, labels, ctrl, err := tc(t)
			assert.Equal(t, err, recorder.RecordNumericMetrics("powermax_numeric_", labels, metrics))
			ctrl.Finish()
		})
	}
}
