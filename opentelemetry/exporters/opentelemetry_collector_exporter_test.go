package otlexporters

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
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
				tt.collector.exporter.Shutdown(context.Background())
			}

			err = tt.collector.StopExporter()
			if err != nil && tt.ExpectedError == nil {
				t.Fatal(err)
			}
		})
	}
}
