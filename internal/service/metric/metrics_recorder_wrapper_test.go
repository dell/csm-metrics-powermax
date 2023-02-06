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
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	"github.com/dell/csm-metrics-powermax/internal/service/types/mocks/asyncfloat64mock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/metric/global"
)

func Test_RecordNumericMetrics(t *testing.T) {
	tests := map[string]func(t *testing.T) (*metric.MetricsRecorderWrapper, []types.NumericMetric, *gomock.Controller, error){
		"success": func(*testing.T) (*metric.MetricsRecorderWrapper, []types.NumericMetric, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			otMeter := global.Meter("powermax_test")
			meter := mocks.NewMockAsyncMetricCreator(ctrl)
			recorder := &metric.MetricsRecorderWrapper{meter}
			provider := asyncfloat64mock.NewMockInstrumentProvider(ctrl)

			total, err := otMeter.AsyncFloat64().UpDownCounter("powermax_srp_total_capacity_gigabytes")
			if err != nil {
				t.Fatal(err)
			}
			used, err := otMeter.AsyncFloat64().UpDownCounter("powermax_srp_used_capacity_gigabytes")
			if err != nil {
				t.Fatal(err)
			}

			meter.EXPECT().AsyncFloat64().Return(provider).Times(2)
			provider.EXPECT().UpDownCounter(gomock.Any(), gomock.Any()).Return(total, nil).Times(1)
			provider.EXPECT().UpDownCounter(gomock.Any(), gomock.Any()).Return(used, nil).Times(1)

			var metrics []types.NumericMetric
			metrics = append(metrics, types.NumericMetric{Name: "powermax_srp_total_capacity_gigabytes", Value: 16})
			metrics = append(metrics, types.NumericMetric{Name: "powermax_srp_used_capacity_gigabytes", Value: 8})
			return recorder, metrics, ctrl, nil
		},
		"failed": func(*testing.T) (*metric.MetricsRecorderWrapper, []types.NumericMetric, *gomock.Controller, error) {
			ctrl := gomock.NewController(t)
			meter := mocks.NewMockAsyncMetricCreator(ctrl)
			recorder := &metric.MetricsRecorderWrapper{meter}
			provider := asyncfloat64mock.NewMockInstrumentProvider(ctrl)

			err := errors.New("error")
			meter.EXPECT().AsyncFloat64().Return(provider).Times(1)
			provider.EXPECT().UpDownCounter(gomock.Any(), gomock.Any()).Return(nil, err).Times(1)

			var metrics []types.NumericMetric
			metrics = append(metrics, types.NumericMetric{Name: "powermax_srp_total_capacity_gigabytes", Value: 16})
			metrics = append(metrics, types.NumericMetric{Name: "powermax_srp_used_capacity_gigabytes", Value: 8})
			return recorder, metrics, ctrl, err
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			recorder, metrics, ctrl, err := tc(t)
			assert.Equal(t, err, recorder.RecordNumericMetrics(context.Background(), metrics))
			ctrl.Finish()
		})
	}
}
