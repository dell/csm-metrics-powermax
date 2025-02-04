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

package main

import (
	"testing"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/entrypoint"
	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestUpdateMetricsEnabled(t *testing.T) {
	tests := []struct {
		name                string
		capacityValue       string
		performanceValue    string
		expectedCapacity    bool
		expectedPerformance bool
	}{
		{
			name:                "default values",
			capacityValue:       "true",
			performanceValue:    "true",
			expectedCapacity:    true,
			expectedPerformance: true,
		},
		{
			name:                "capacity disabled",
			capacityValue:       "false",
			performanceValue:    "true",
			expectedCapacity:    false,
			expectedPerformance: true,
		},
		{
			name:                "performance disabled",
			capacityValue:       "true",
			performanceValue:    "false",
			expectedCapacity:    true,
			expectedPerformance: false,
		},
		{
			name:                "both disabled",
			capacityValue:       "false",
			performanceValue:    "false",
			expectedCapacity:    false,
			expectedPerformance: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger = logrus.New()

			config := &entrypoint.Config{}
			viper.Set("POWERMAX_CAPACITY_METRICS_ENABLED", tt.capacityValue)
			viper.Set("POWERMAX_PERFORMANCE_METRICS_ENABLED", tt.performanceValue)

			updateMetricsEnabled(config)

			assert.Equal(t, tt.expectedCapacity, config.CapacityMetricsEnabled)
			assert.Equal(t, tt.expectedPerformance, config.PerformanceMetricsEnabled)
		})
	}
}

func TestUpdateProvisionerNames(t *testing.T) {
	tests := []struct {
		name                  string
		provisionerNamesValue string
		expectedDriverNames   []string
	}{
		{
			name:                  "default value",
			provisionerNamesValue: "provisioner1,provisioner2",
			expectedDriverNames:   []string{"provisioner1", "provisioner2"},
		},
		{
			name:                  "empty value",
			provisionerNamesValue: "",
			expectedDriverNames:   []string{},
		},
		{
			name:                  "single value",
			provisionerNamesValue: "provisioner3",
			expectedDriverNames:   []string{"provisioner3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger = logrus.New()

			viper.Set("PROVISIONER_NAMES", tt.provisionerNamesValue)

			volumeFinder := &k8s.VolumeFinder{
				DriverNames: []string{},
			}

			storageClassFinder := &k8s.StorageClassFinder{
				StorageArrayID: []k8s.StorageArrayID{
					{
						DriverNames: []string{},
					},
				},
			}

			updateProvisionerNames(volumeFinder, storageClassFinder)

			assert.Equal(t, tt.expectedDriverNames, volumeFinder.DriverNames)
			assert.Equal(t, tt.expectedDriverNames, storageClassFinder.StorageArrayID[0].DriverNames)
		})
	}
}

func TestUpdateCollectorAddress(t *testing.T) {
	tests := []struct {
		name                    string
		collectorAddressValue   string
		expectedCollectorAddr   string
		expectedExporterAddress string
	}{
		{
			name:                    "default value",
			collectorAddressValue:   "localhost:8080",
			expectedCollectorAddr:   "localhost:8080",
			expectedExporterAddress: "localhost:8080",
		},
		{
			name:                    "empty value",
			collectorAddressValue:   "",
			expectedCollectorAddr:   "",
			expectedExporterAddress: "",
		},
		{
			name:                    "valid address",
			collectorAddressValue:   "127.0.0.1:9090",
			expectedCollectorAddr:   "127.0.0.1:9090",
			expectedExporterAddress: "127.0.0.1:9090",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger = logrus.New()

			config := &entrypoint.Config{}
			exporter := &otlexporters.OtlCollectorExporter{}

			viper.Set("COLLECTOR_ADDR", tt.collectorAddressValue)

			updateCollectorAddress(config, exporter)

			assert.Equal(t, tt.expectedCollectorAddr, config.CollectorAddress)
			assert.Equal(t, tt.expectedExporterAddress, exporter.CollectorAddr)
		})
	}
}

func TestUpdateTickIntervals(t *testing.T) {
	tests := []struct {
		name                     string
		capacityPollFrequency    string
		expectedCapacityTick     time.Duration
		performancePollFrequency string
		expectedPerformanceTick  time.Duration
	}{
		{
			name:                     "default values",
			capacityPollFrequency:    "",
			expectedCapacityTick:     defaultTickInterval,
			performancePollFrequency: "",
			expectedPerformanceTick:  defaultTickInterval,
		},
		{
			name:                     "valid capacity poll frequency",
			capacityPollFrequency:    "30",
			expectedCapacityTick:     30 * time.Second,
			performancePollFrequency: "",
			expectedPerformanceTick:  defaultTickInterval,
		},
		{
			name:                     "valid performance poll frequency",
			capacityPollFrequency:    "",
			expectedCapacityTick:     defaultTickInterval,
			performancePollFrequency: "15",
			expectedPerformanceTick:  15 * time.Second,
		},
		{
			name:                     "invalid capacity poll frequency",
			capacityPollFrequency:    "invalid",
			expectedCapacityTick:     defaultTickInterval,
			performancePollFrequency: "",
			expectedPerformanceTick:  defaultTickInterval,
		},
		{
			name:                     "invalid performance poll frequency",
			capacityPollFrequency:    "",
			expectedCapacityTick:     defaultTickInterval,
			performancePollFrequency: "invalid",
			expectedPerformanceTick:  defaultTickInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger = logrus.New()

			config := &entrypoint.Config{}

			viper.Set("POWERMAX_CAPACITY_POLL_FREQUENCY", tt.capacityPollFrequency)
			viper.Set("POWERMAX_PERFORMANCE_POLL_FREQUENCY", tt.performancePollFrequency)

			updateTickIntervals(config)

			assert.Equal(t, tt.expectedCapacityTick, config.CapacityTickInterval)
			assert.Equal(t, tt.expectedPerformanceTick, config.PerformanceTickInterval)
		})
	}
}

func TestUpdateMaxConnections(t *testing.T) {
	tests := []struct {
		name           string
		maxConnections string
		expected       int
	}{
		{
			name:           "default value",
			maxConnections: "10",
			expected:       service.DefaultMaxPowerMaxConnections,
		},
		{
			name:           "valid value",
			maxConnections: "5",
			expected:       5,
		},
		{
			name:           "invalid value (<= 0)",
			maxConnections: "-1",
			expected:       service.DefaultMaxPowerMaxConnections,
		},
		{
			name:           "invalid value (string)",
			maxConnections: "string",
			expected:       service.DefaultMaxPowerMaxConnections,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			logger = logrus.New()

			viper.Set("POWERMAX_MAX_CONCURRENT_QUERIES", tt.maxConnections)

			powerMaxSvc := &service.PowerMaxService{}

			updateMaxConnections(powerMaxSvc)

			assert.Equal(t, tt.expected, powerMaxSvc.MaxPowerMaxConnections)
		})
	}
}
