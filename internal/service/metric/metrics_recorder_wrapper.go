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

	"github.com/dell/csm-metrics-powermax/internal/service/types"
)

// MetricsRecorderWrapper contains data used for pushing metrics data
type MetricsRecorderWrapper struct {
	Meter types.AsyncMetricCreator
}

// RecordNumericMetrics record metrics using Otel's InstrumentProvider
func (mrw *MetricsRecorderWrapper) RecordNumericMetrics(ctx context.Context, metrics []types.NumericMetric) error {
	for _, metric := range metrics {
		counter, err := mrw.Meter.AsyncFloat64().UpDownCounter(metric.Name)
		if err != nil {
			return err
		}

		counter.Observe(ctx, metric.Value, metric.Labels...)
	}
	return nil
}
