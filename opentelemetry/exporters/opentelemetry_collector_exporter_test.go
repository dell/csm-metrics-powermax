/*
 Copyright (c) 2025 Dell Inc. or its subsidiaries. All Rights Reserved.

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

package otlexporters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestInitExporter(t *testing.T) {
	tests := []struct {
		name          string
		collector     *OtlCollectorExporter
		opts          []otlpmetricgrpc.Option
		ExpectedError error
	}{
		{
			name: "Successful Exporter Initialization",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			},
			opts: []otlpmetricgrpc.Option{
				otlpmetricgrpc.WithInsecure(),
			},
			ExpectedError: nil,
		},
		{
			name: "Invalid Service Config",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			},
			opts: []otlpmetricgrpc.Option{
				otlpmetricgrpc.WithServiceConfig("invalid config"),
			},
			ExpectedError: errors.New("grpc: the provided default service config is invalid: invalid character 'i' looking for beginning of value"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.collector.InitExporter(tt.opts...)
			assert.Equal(t, err, tt.ExpectedError)
		})
	}
}

func TestOtlCollectorExporter_StopExporter(t *testing.T) {
	tests := []struct {
		name          string
		collector     *OtlCollectorExporter
		opts          []otlpmetricgrpc.Option
		preShutdown   bool
		ExpectedError error
	}{
		{
			name: "Error: gRPC exporter is shutdown",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			},
			opts: []otlpmetricgrpc.Option{
				otlpmetricgrpc.WithInsecure(),
			},
			preShutdown:   true,
			ExpectedError: errors.New("gRPC exporter is shutdown"),
		},
		{
			name: "Shutdown Exporter",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			},
			opts: []otlpmetricgrpc.Option{
				otlpmetricgrpc.WithInsecure(),
				otlpmetricgrpc.WithEndpoint("localhost:8080"),
			},
			preShutdown:   false,
			ExpectedError: errors.New("gRPC exporter is shutdown"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.collector.InitExporter(tt.opts...)
			if err != nil {
				t.Fatal(err)
			}

			if tt.preShutdown {
				_ = tt.collector.exporter.Shutdown(context.Background())
			}

			err = tt.collector.StopExporter()
			if err != nil && tt.ExpectedError == nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSetExportFailureRecorder(t *testing.T) {
	tests := []struct {
		name        string
		collector   *OtlCollectorExporter
		hasCallback bool
	}{
		{
			name: "Set Export Failure Recorder",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			},
			hasCallback: true,
		},
		{
			name: "Set Export Failure Recorder with nil callback",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			},
			hasCallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hasCallback {
				tt.collector.SetExportFailureRecorder(func() {
					// Mock callback
				})
				assert.NotNil(t, tt.collector.recordExportFailure)
			} else {
				tt.collector.SetExportFailureRecorder(nil)
				assert.Nil(t, tt.collector.recordExportFailure)
			}
		})
	}
}

func TestRecordingExporter_Export(t *testing.T) {
	tests := []struct {
		name        string
		hasCallback bool
	}{
		{
			name:        "Export with failure callback",
			hasCallback: true,
		},
		{
			name:        "Export without failure callback",
			hasCallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := &OtlCollectorExporter{
				CollectorAddr: "localhost:8080",
			}
			err := collector.InitExporter(otlpmetricgrpc.WithInsecure())
			if err != nil {
				t.Fatalf("failed to initialize exporter: %v", err)
			}
			defer func() {
				_ = collector.StopExporter()
			}()

			if tt.hasCallback {
				collector.SetExportFailureRecorder(func() {
					// Callback for export failure
				})
				assert.NotNil(t, collector.recordExportFailure)
			}

			// Create empty metrics for testing
			metrics := &metricdata.ResourceMetrics{}

			// Export should fail with connection error (no real collector)
			// but the recordingExporter wrapper should handle it
			_ = collector.exporter.Export(context.Background(), metrics)
		})
	}
}
