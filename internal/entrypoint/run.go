/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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
	"os"
	"runtime"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/k8spmax"

	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"

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
	// LivenessProbeInterval the interval to probe PowerMax connection liveness
	LivenessProbeInterval = 30 * time.Second
)

// ConfigValidatorFunc is used to override config validation in testing
var ConfigValidatorFunc = ValidateConfig

// Config holds data that will be used by the service
type Config struct {
	LeaderElector               metrictypes.LeaderElector
	CapacityTickInterval        time.Duration
	PerformanceTickInterval     time.Duration
	TopologyMetricsTickInterval time.Duration
	LivenessProbeTickInterval   time.Duration
	CapacityMetricsEnabled      bool
	PerformanceMetricsEnabled   bool
	TopologyMetricsEnabled      bool
	CollectorAddress            string
	CollectorCertPath           string
	Logger                      *logrus.Logger
}

// Run is the entry point for starting the service
func Run(ctx context.Context, config *Config, exporter otlexporters.Otlexporter, powerMaxSvc metrictypes.Service) error {
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
	capacityTickInterval := config.CapacityTickInterval
	capacityTicker := time.NewTicker(capacityTickInterval)
	performanceTickInterval := config.PerformanceTickInterval
	performanceTicker := time.NewTicker(performanceTickInterval)
	topologyMetricsTickInterval := config.TopologyMetricsTickInterval
	topologyMetricsTicker := time.NewTicker(topologyMetricsTickInterval)

	livenessProbeTickInterval := config.LivenessProbeTickInterval
	if livenessProbeTickInterval == 0 {
		livenessProbeTickInterval = LivenessProbeInterval
	}
	livenessProbeTick := time.NewTicker(livenessProbeTickInterval)

	for {
		select {
		case <-capacityTicker.C:
			if !config.LeaderElector.IsLeader() {
				logger.Info("not leader pod to collect metrics")
				continue
			}
			if !config.CapacityMetricsEnabled {
				logger.Info("powerMax capacity metrics collection is disabled")
				continue
			}
			powerMaxSvc.ExportCapacityMetrics(ctx)
		case <-performanceTicker.C:
			if !config.LeaderElector.IsLeader() {
				logger.Info("not leader pod to collect metrics")
				continue
			}
			if !config.PerformanceMetricsEnabled {
				logger.Info("powerMax performance metrics collection is disabled")
				continue
			}
			powerMaxSvc.ExportPerformanceMetrics(ctx)
		case <-topologyMetricsTicker.C:
			if !config.LeaderElector.IsLeader() {
				logger.Info("not leader pod to collect metrics")
				continue
			}
			if !config.TopologyMetricsEnabled {
				logger.Info("powermax topology metrics collection is disabled")
				continue
			}
			powerMaxSvc.ExportTopologyMetrics(ctx)
		case <-livenessProbeTick.C:
			logger.Info("validate powermax connection")
			validatePowerMaxArrays(ctx, powerMaxSvc)
		case err := <-errCh:
			if err == nil {
				continue
			}
			return err
		case <-ctx.Done():
			return nil
		}

		// check if tick interval config settings have changed
		if capacityTickInterval != config.CapacityTickInterval {
			capacityTickInterval = config.CapacityTickInterval
			capacityTicker = time.NewTicker(capacityTickInterval)
		}

		// check if tick interval config settings have changed
		if performanceTickInterval != config.PerformanceTickInterval {
			performanceTickInterval = config.PerformanceTickInterval
			performanceTicker = time.NewTicker(performanceTickInterval)
		}

		// check if tick interval config settings have changed for topology metrics
		if topologyMetricsTickInterval != config.TopologyMetricsTickInterval {
			topologyMetricsTickInterval = config.TopologyMetricsTickInterval
			topologyMetricsTicker = time.NewTicker(topologyMetricsTickInterval)
		}
	}
}

func validatePowerMaxArrays(ctx context.Context, powerMaxSvc metrictypes.Service) {
	for arrayID, powerMaxArrays := range powerMaxSvc.GetPowerMaxClients() {
		for _, array := range powerMaxArrays {
			err := k8spmax.Authenticate(ctx, array.Client, array)
			if err != nil {
				array.IsActive = false
				powerMaxSvc.GetLogger().WithError(err).Errorf("authentication failed to PowerMax array %s, %s", arrayID, array.Endpoint)
				continue
			}
			array.IsActive = true
			powerMaxSvc.GetLogger().Infof("authentication successful to PowerMax array %s, %s", arrayID, array.Endpoint)
		}
	}
}

// ValidateConfig will validate the configuration and return any errors
func ValidateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("no config provided")
	}

	if config.CapacityTickInterval > MaximumTickInterval || config.CapacityTickInterval < MinimumTickInterval {
		return fmt.Errorf("capacity polling frequency not within allowed range of %v and %v", MinimumTickInterval.String(), MaximumTickInterval.String())
	}

	if config.PerformanceTickInterval > MaximumTickInterval || config.PerformanceTickInterval < MinimumTickInterval {
		return fmt.Errorf("performance metrics polling frequency not within allowed range of %v and %v", MinimumTickInterval.String(), MaximumTickInterval.String())
	}

	if config.TopologyMetricsTickInterval > MaximumTickInterval || config.TopologyMetricsTickInterval < MinimumTickInterval {
		return fmt.Errorf("topology metrics polling frequency not within allowed range of %v and %v", MinimumTickInterval.String(), MaximumTickInterval.String())
	}

	return nil
}
