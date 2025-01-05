package otlexporters

import (
	"errors"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"testing"
)

func TestInitExporter(t *testing.T) {
	tests := [] struct {
		name string
		collector *OtlCollectorExporter
		opts []otlpmetricgrpc.Option
		ExpectedError error
	} {
		{
			name: "Successful Exporter Initialization",
			collector: &OtlCollectorExporter{
				CollectorAddr: "localhost:0000",
			},
			opts: []otlpmetricgrpc.Option{
				otlpmetricgrpc.WithInsecure(),
			},
			ExpectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.collector.InitExporter(tt.opts...)
			if !errors.Is(err, tt.ExpectedError) {
				t.Errorf("OtlCollectorExporter.InitExporter() error = %v, wantErr %v", err, tt.ExpectedError)
			}
		})
	}
}