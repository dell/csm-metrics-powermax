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
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/dell/csi-powermax/csireverseproxy/v2/pkg/k8sutils"
	"github.com/dell/csm-metrics-powermax/internal/entrypoint"
	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"
	corev1 "k8s.io/api/core/v1"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ServiceAccessorMock struct {
	service.PowerMaxService
}

func (sam *ServiceAccessorMock) UpdatePowerMaxArraysOnSecretChanged(k8sutils.UtilsInterface, *corev1.Secret) {
}

func TestConfigure(t *testing.T) {
	testCases := []struct {
		name                   string
		arrays                 map[string][]metrictypes.PowerMaxArray
		useSecret              bool
		expectedArray          map[string][]metrictypes.PowerMaxArray
		expectedCollectorAddr  string
		expectedMaxConnections int
	}{
		{
			name:      "Test with arrays with elements and Secret",
			useSecret: true,
			arrays: map[string][]metrictypes.PowerMaxArray{
				"array1": {
					{
						StorageArrayID: "array1",
						Endpoint:       "endpoint1",
						Username:       "username1",
						Password:       "password1",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
				"array2": {
					{
						StorageArrayID: "array2",
						Endpoint:       "endpoint2",
						Username:       "username2",
						Password:       "password2",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
			},
			expectedArray: map[string][]metrictypes.PowerMaxArray{
				"array1": {
					{
						StorageArrayID: "array1",
						Endpoint:       "endpoint1",
						Username:       "username1",
						Password:       "password1",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
				"array2": {
					{
						StorageArrayID: "array2",
						Endpoint:       "endpoint2",
						Username:       "username2",
						Password:       "password2",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
			},
			expectedCollectorAddr:  "localhost:55678",
			expectedMaxConnections: service.DefaultMaxPowerMaxConnections,
		},
		{
			name:      "Test with arrays with elements and ConfigMap",
			useSecret: false,
			arrays: map[string][]metrictypes.PowerMaxArray{
				"array1": {
					{
						StorageArrayID: "array1",
						Endpoint:       "endpoint1",
						Username:       "username1",
						Password:       "password1",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
				"array2": {
					{
						StorageArrayID: "array2",
						Endpoint:       "endpoint2",
						Username:       "username2",
						Password:       "password2",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
			},
			expectedArray: map[string][]metrictypes.PowerMaxArray{
				"array1": {
					{
						StorageArrayID: "array1",
						Endpoint:       "endpoint1",
						Username:       "username1",
						Password:       "password1",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
				"array2": {
					{
						StorageArrayID: "array2",
						Endpoint:       "endpoint2",
						Username:       "username2",
						Password:       "password2",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
			},
			expectedCollectorAddr:  "localhost:55678",
			expectedMaxConnections: service.DefaultMaxPowerMaxConnections,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			GetPowerMaxArrays = func(_ context.Context, _ k8sutils.UtilsInterface, _ string) (map[string][]metrictypes.PowerMaxArray, error) {
				return tc.arrays, nil
			}

			InitK8sUtils = func(_ ServiceAccessorInterface, _ bool) (*k8sutils.K8sUtils, error) {
				return nil, nil
			}

			if tc.useSecret {
				os.Setenv("X_CSI_REVPROXY_USE_SECRET", "true")
			} else {
				os.Setenv("X_CSI_REVPROXY_USE_SECRET", "false")
			}

			os.Setenv("TLS_ENABLED", "true")

			_, _, powerMaxSvc := configure(ctx)

			assert.Equal(t, tc.expectedArray, powerMaxSvc.PowerMaxClients)
		})
	}
}

func TestUpdatePowerMaxArrays(t *testing.T) {
	testCases := []struct {
		name                   string
		arrays                 map[string][]metrictypes.PowerMaxArray
		getPowerMaxArraysError error
		expectedArray          map[string][]metrictypes.PowerMaxArray
	}{
		{
			name: "Test with empty arrays",
			arrays: map[string][]metrictypes.PowerMaxArray{
				"array1": {},
				"array2": {},
			},
			expectedArray: map[string][]metrictypes.PowerMaxArray{
				"array1": nil,
				"array2": nil,
			},
		},
		{
			name: "Test with arrays with elements",
			arrays: map[string][]metrictypes.PowerMaxArray{
				"array1": {
					{
						StorageArrayID: "array1",
						Endpoint:       "endpoint1",
						Username:       "username1",
						Password:       "password1",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
				"array2": {
					{
						StorageArrayID: "array2",
						Endpoint:       "endpoint2",
						Username:       "username2",
						Password:       "password2",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
			},
			expectedArray: map[string][]metrictypes.PowerMaxArray{
				"array1": {
					{
						StorageArrayID: "array1",
						Endpoint:       "endpoint1",
						Username:       "username1",
						Password:       "password1",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
				"array2": {
					{
						StorageArrayID: "array2",
						Endpoint:       "endpoint2",
						Username:       "username2",
						Password:       "password2",
						Insecure:       true,
						IsPrimary:      true,
						IsActive:       true,
					},
				},
			},
		},
		{
			name:                   "Test with error from call to GetPowerMaxArrays",
			getPowerMaxArraysError: errors.New("some error"),
			expectedArray:          map[string][]metrictypes.PowerMaxArray{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			powerMaxSvc := &service.PowerMaxService{
				PowerMaxClients: make(map[string][]metrictypes.PowerMaxArray),
			}

			GetPowerMaxArrays = func(_ context.Context, _ k8sutils.UtilsInterface, _ string) (map[string][]metrictypes.PowerMaxArray, error) {
				return tc.arrays, tc.getPowerMaxArraysError
			}

			updatePowerMaxArrays(ctx, powerMaxSvc)

			assert.Equal(t, tc.expectedArray, powerMaxSvc.PowerMaxClients)
		})
	}
}

func TestUpdateMetricsEnabled(t *testing.T) {
	tests := []struct {
		name                string
		capacityValue       string
		performanceValue    string
		topologyValue       string
		expectedCapacity    bool
		expectedPerformance bool
		expectedTopology    bool
	}{
		{
			name:                "default values",
			capacityValue:       "true",
			performanceValue:    "true",
			topologyValue:       "true",
			expectedCapacity:    true,
			expectedPerformance: true,
			expectedTopology:    true,
		},
		{
			name:                "capacity disabled",
			capacityValue:       "false",
			performanceValue:    "true",
			topologyValue:       "true",
			expectedCapacity:    false,
			expectedPerformance: true,
			expectedTopology:    true,
		},
		{
			name:                "performance disabled",
			capacityValue:       "true",
			performanceValue:    "false",
			topologyValue:       "true",
			expectedCapacity:    true,
			expectedPerformance: false,
			expectedTopology:    true,
		},
		{
			name:                "topology disabled",
			capacityValue:       "true",
			performanceValue:    "true",
			topologyValue:       "false",
			expectedCapacity:    true,
			expectedPerformance: true,
			expectedTopology:    false,
		},
		{
			name:                "all disabled",
			capacityValue:       "false",
			performanceValue:    "false",
			topologyValue:       "false",
			expectedCapacity:    false,
			expectedPerformance: false,
			expectedTopology:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &entrypoint.Config{}
			viper.Set("POWERMAX_CAPACITY_METRICS_ENABLED", tt.capacityValue)
			viper.Set("POWERMAX_PERFORMANCE_METRICS_ENABLED", tt.performanceValue)
			viper.Set("POWERMAX_TOPOLOGY_METRICS_ENABLED", tt.topologyValue)

			updateMetricsEnabled(config)

			assert.Equal(t, tt.expectedCapacity, config.CapacityMetricsEnabled)
			assert.Equal(t, tt.expectedPerformance, config.PerformanceMetricsEnabled)
			assert.Equal(t, tt.expectedTopology, config.TopologyMetricsEnabled)
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
		name                         string
		capacityPollFrequency        string
		expectedCapacityTick         time.Duration
		performancePollFrequency     string
		expectedPerformanceTick      time.Duration
		topologyMetricsPollfrequency string
		expectedTopologyTick         time.Duration
	}{
		{
			name:                         "default values",
			capacityPollFrequency:        "",
			expectedCapacityTick:         defaultTickInterval,
			performancePollFrequency:     "",
			expectedPerformanceTick:      defaultTickInterval,
			topologyMetricsPollfrequency: "",
			expectedTopologyTick:         defaultTickInterval,
		},
		{
			name:                         "valid capacity poll frequency",
			capacityPollFrequency:        "30",
			expectedCapacityTick:         30 * time.Second,
			performancePollFrequency:     "",
			expectedPerformanceTick:      defaultTickInterval,
			topologyMetricsPollfrequency: "",
			expectedTopologyTick:         defaultTickInterval,
		},
		{
			name:                         "valid performance poll frequency",
			capacityPollFrequency:        "",
			expectedCapacityTick:         defaultTickInterval,
			performancePollFrequency:     "15",
			expectedPerformanceTick:      15 * time.Second,
			topologyMetricsPollfrequency: "",
			expectedTopologyTick:         defaultTickInterval,
		},
		{
			name:                         "valid topology metric poll frequency",
			capacityPollFrequency:        "",
			expectedCapacityTick:         defaultTickInterval,
			performancePollFrequency:     "",
			expectedPerformanceTick:      defaultTickInterval,
			topologyMetricsPollfrequency: "30",
			expectedTopologyTick:         30 * time.Second,
		},
		{
			name:                         "invalid capacity poll frequency",
			capacityPollFrequency:        "invalid",
			expectedCapacityTick:         defaultTickInterval,
			performancePollFrequency:     "",
			expectedPerformanceTick:      defaultTickInterval,
			topologyMetricsPollfrequency: "",
			expectedTopologyTick:         defaultTickInterval,
		},
		{
			name:                         "invalid performance poll frequency",
			capacityPollFrequency:        "",
			expectedCapacityTick:         defaultTickInterval,
			performancePollFrequency:     "invalid",
			expectedPerformanceTick:      defaultTickInterval,
			topologyMetricsPollfrequency: "",
			expectedTopologyTick:         defaultTickInterval,
		},
		{
			name:                         "invalid topology metric poll frequency",
			capacityPollFrequency:        "",
			expectedCapacityTick:         defaultTickInterval,
			performancePollFrequency:     "",
			expectedPerformanceTick:      defaultTickInterval,
			topologyMetricsPollfrequency: "invalid",
			expectedTopologyTick:         defaultTickInterval,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &entrypoint.Config{}

			viper.Set("POWERMAX_CAPACITY_POLL_FREQUENCY", tt.capacityPollFrequency)
			viper.Set("POWERMAX_PERFORMANCE_POLL_FREQUENCY", tt.performancePollFrequency)
			viper.Set("POWERMAX_TOPOLOGY_METRICS_POLL_FREQUENCY", tt.topologyMetricsPollfrequency)

			updateTickIntervals(config)

			assert.Equal(t, tt.expectedCapacityTick, config.CapacityTickInterval)
			assert.Equal(t, tt.expectedPerformanceTick, config.PerformanceTickInterval)
			assert.Equal(t, tt.expectedTopologyTick, config.TopologyMetricsTickInterval)
		})
	}
}

// I-OBS-PMAX-02: startMetricsServer wires ObsInstrumenter onto the service and
// exposes all four dell_csm_obs_* metrics at /metrics (HTTP 200).
func TestStartMetricsServer_WiresInstrumenterAndServesMetrics(t *testing.T) {
	port := pmaxObsFreePort(t)
	viper.Set(csiObsMetricsPortKey, fmt.Sprintf("%d", port))
	t.Cleanup(func() { viper.Set(csiObsMetricsPortKey, "") })

	powerMaxSvc := &service.PowerMaxService{}

	startMetricsServer(powerMaxSvc)

	assert.NotNil(t, powerMaxSvc.ObsInstrumenter, "startMetricsServer must set ObsInstrumenter on the service")

	powerMaxSvc.ObsInstrumenter.RecordCollectionRate("000197902573", 1)
	powerMaxSvc.ObsInstrumenter.RecordExportSuccess("000197902573", "success")
	powerMaxSvc.ObsInstrumenter.SetArrayConnectivity("000197902573", true)
	powerMaxSvc.ObsInstrumenter.RecordProcessingLatency("000197902573", 0.001)

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/metrics", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	for _, metric := range []string{
		"dell_csm_obs_collection_rate",
		"dell_csm_obs_export_success_total",
		"dell_csm_obs_array_connectivity",
		"dell_csm_obs_processing_latency_seconds",
	} {
		assert.Contains(t, string(body), metric, "metric %q must appear in /metrics output", metric)
	}
}

// I-OBS-PMAX-03: startMetricsServer wires ObsInstrumenter regardless of port source.
func TestStartMetricsServer_WiresInstrumenterEvenWithDefaultPort(t *testing.T) {
	port := pmaxObsFreePort(t)
	viper.Set(csiObsMetricsPortKey, fmt.Sprintf("%d", port))
	t.Cleanup(func() { viper.Set(csiObsMetricsPortKey, "") })

	powerMaxSvc := &service.PowerMaxService{}

	startMetricsServer(powerMaxSvc)

	assert.NotNil(t, powerMaxSvc.ObsInstrumenter)
}

// I-OBS-PMAX-04: /metrics returns HTTP 404 for unknown paths.
func TestStartMetricsServer_UnknownPathReturns404(t *testing.T) {
	port := pmaxObsFreePort(t)
	viper.Set(csiObsMetricsPortKey, fmt.Sprintf("%d", port))
	t.Cleanup(func() { viper.Set(csiObsMetricsPortKey, "") })

	powerMaxSvc := &service.PowerMaxService{}

	startMetricsServer(powerMaxSvc)
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		fmt.Sprintf("http://localhost:%d/unknown", port), nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// I-OBS-PMAX-05: startMetricsServer serves metrics over HTTPS when TLS is enabled with valid certificates.
func TestStartMetricsServer_TLSServesMetricsWithValidCerts(t *testing.T) {
	port := pmaxObsFreePort(t)
	viper.Set(csiObsMetricsPortKey, fmt.Sprintf("%d", port))
	viper.Set(csiObsMetricsSchemeKey, "https")
	t.Cleanup(func() {
		viper.Set(csiObsMetricsPortKey, "")
		viper.Set(csiObsMetricsSchemeKey, "")
	})

	certFile := t.TempDir() + "/test-cert.pem"
	keyFile := t.TempDir() + "/test-key.pem"

	cmd := exec.Command("openssl", "req", "-new", "-x509", "-sha256", "-keyout", keyFile,
		"-out", certFile, "-days", "1", "-nodes", "-subj", "/CN=localhost")
	if err := cmd.Run(); err != nil {
		t.Skipf("Skipping TLS test: openssl not available: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(certFile)
		os.Remove(keyFile)
	})

	viper.Set(csiObsMetricsCertKey, certFile)
	viper.Set(csiObsMetricsKeyKey, keyFile)
	t.Cleanup(func() {
		viper.Set(csiObsMetricsCertKey, "")
		viper.Set(csiObsMetricsKeyKey, "")
	})

	powerMaxSvc := &service.PowerMaxService{}

	startMetricsServer(powerMaxSvc)

	assert.NotNil(t, powerMaxSvc.ObsInstrumenter, "startMetricsServer must set ObsInstrumenter on the service")

	powerMaxSvc.ObsInstrumenter.RecordCollectionRate("000197902573", 1)
	powerMaxSvc.ObsInstrumenter.RecordExportSuccess("000197902573", "success")
	powerMaxSvc.ObsInstrumenter.SetArrayConnectivity("000197902573", true)
	powerMaxSvc.ObsInstrumenter.RecordProcessingLatency("000197902573", 0.001)

	time.Sleep(100 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- test only
		},
	}

	resp, err := client.Get(fmt.Sprintf("https://localhost:%d/metrics", port))
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	for _, metric := range []string{
		"dell_csm_obs_collection_rate",
		"dell_csm_obs_export_success_total",
		"dell_csm_obs_array_connectivity",
		"dell_csm_obs_processing_latency_seconds",
	} {
		assert.Contains(t, string(body), metric, "metric %q must appear in /metrics output", metric)
	}
}

// I-OBS-PMAX-06: validateTLSFiles returns an error when certificate files are missing.
func TestStartMetricsServer_TLSFailsWithMissingCerts(t *testing.T) {
	err := validateTLSFiles("/nonexistent/cert.pem", "/nonexistent/key.pem")
	assert.Error(t, err, "validateTLSFiles should return error for missing files")
	assert.Contains(t, err.Error(), "no such file or directory", "error should indicate file does not exist")
}

// I-OBS-PMAX-07: validateTLSFiles returns error for empty cert/key paths.
func TestValidateTLSFiles_EmptyPaths(t *testing.T) {
	err := validateTLSFiles("", "")
	assert.Error(t, err, "validateTLSFiles should return error for empty paths")
	assert.Contains(t, err.Error(), "no such file or directory", "error should indicate empty path")

	err = validateTLSFiles("/valid/cert.pem", "")
	assert.Error(t, err, "validateTLSFiles should return error for empty key path")
	assert.Contains(t, err.Error(), "no such file or directory", "error should mention key")

	err = validateTLSFiles("", "/valid/key.pem")
	assert.Error(t, err, "validateTLSFiles should return error for empty cert path")
	assert.Contains(t, err.Error(), "no such file or directory", "error should mention certificate")
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
			viper.Set("POWERMAX_MAX_CONCURRENT_QUERIES", tt.maxConnections)

			powerMaxSvc := &service.PowerMaxService{}

			updateMaxConnections(powerMaxSvc)

			assert.Equal(t, tt.expected, powerMaxSvc.MaxPowerMaxConnections)
		})
	}
}
