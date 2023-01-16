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

package entrypoint

import (
	"context"
	"fmt"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	"os"
	"runtime"
	"time"

	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"google.golang.org/grpc/credentials"
)

const (
	// MaximumTickInterval is the maximum allowed interval when querying metrics
	MaximumTickInterval = 10 * time.Minute
	// MinimumTickInterval is the minimum allowed interval when querying metrics
	MinimumTickInterval = 5 * time.Second
	// DefaultEndPoint for leader election path
	DefaultEndPoint = "karavi-metrics-powermax"
	// DefaultNameSpace for PowerMax pod running metrics collection
	DefaultNameSpace = "karavi"
)

var (
	// ConfigValidatorFunc is used to override config validation in testing
	ConfigValidatorFunc func(*Config) error = ValidateConfig
)

// Config holds data that will be used by the service
type Config struct {
	LeaderElector             types.LeaderElector
	ArrayCapacityTickInterval time.Duration
	CapacityMetricsEnabled    bool
	PerformanceMetricsEnabled bool
	CollectorAddress          string
	CollectorCertPath         string
	Logger                    *logrus.Logger
}

// Run is the entry point for starting the service
func Run(ctx context.Context, config *Config, exporter otlexporters.Otlexporter, powerMaxSvc types.Service) error {
	err := ConfigValidatorFunc(config)
	if err != nil {
		return err
	}
	logger := config.Logger

	errCh := make(chan error, 1)

	go func() {
		powerMaxEndpoint := os.Getenv("POWERMAX_METRICS_ENDPOINT")
		if powerMaxEndpoint == "" {
			powerMaxEndpoint = DefaultEndPoint
		}
		powerMaxNamespace := os.Getenv("POWERMAX_METRICS_NAMESPACE")
		if powerMaxNamespace == "" {
			powerMaxNamespace = DefaultNameSpace
		}
		errCh <- config.LeaderElector.InitLeaderElection(powerMaxEndpoint, powerMaxNamespace)
	}()

	go func() {
		options := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(config.CollectorAddress),
		}

		if config.CollectorCertPath != "" {

			transportCreds, err := credentials.NewClientTLSFromFile(config.CollectorCertPath, "")
			if err != nil {
				errCh <- err
			}
			options = append(options, otlpmetricgrpc.WithTLSCredentials(transportCreds))
		} else {
			options = append(options, otlpmetricgrpc.WithInsecure())
		}

		errCh <- exporter.InitExporter(options...)
	}()

	defer exporter.StopExporter()

	runtime.GOMAXPROCS(runtime.NumCPU())

	// set initial tick intervals
	arrayCapacityTickInterval := config.ArrayCapacityTickInterval
	arrayCapacityTicker := time.NewTicker(arrayCapacityTickInterval)

	for {
		select {
		case <-arrayCapacityTicker.C:
			if !config.LeaderElector.IsLeader() {
				logger.Info("not leader pod to collect metrics")
				continue
			}
			if !config.CapacityMetricsEnabled {
				logger.Info("powerMax array capacity metrics collection is disabled")
				continue
			}
			powerMaxSvc.ExportArrayCapacityMetrics(ctx)
		case err := <-errCh:
			if err == nil {
				continue
			}
			return err
		case <-ctx.Done():
			return nil
		}

		// check if tick interval config settings have changed
		if arrayCapacityTickInterval != config.ArrayCapacityTickInterval {
			arrayCapacityTickInterval = config.ArrayCapacityTickInterval
			arrayCapacityTicker = time.NewTicker(arrayCapacityTickInterval)
		}
	}
}

// ValidateConfig will validate the configuration and return any errors
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("no config provided")
	}

	if config.ArrayCapacityTickInterval > MaximumTickInterval || config.ArrayCapacityTickInterval < MinimumTickInterval {
		return fmt.Errorf("quota capacity polling frequency not within allowed range of %v and %v", MinimumTickInterval.String(), MaximumTickInterval.String())
	}

	return nil
}
