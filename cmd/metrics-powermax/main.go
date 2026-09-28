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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dell/csi-powermax/csireverseproxy/v2/pkg/k8sutils"
	csmserver "github.com/dell/csm-metrics-common/pkg/server"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	corev1 "k8s.io/api/core/v1"

	"github.com/dell/csm-metrics-powermax/internal/entrypoint"
	"github.com/dell/csm-metrics-powermax/internal/k8spmax"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"
	"github.com/dell/csmlog"

	"go.opentelemetry.io/otel"

	"github.com/fsnotify/fsnotify"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/viper"
)

const (
	defaultTickInterval           = 5 * time.Minute
	defaultConfigFile             = "/etc/config/karavi-metrics-powermax.yaml"
	defaultReverseProxyConfigFile = "/etc/reverseproxy/config.yaml"
	defaultSecretConfigFile       = "/etc/powermax/config" // #nosec G101
	defaultObsMetricsPort         = "8443"
	defaultObsMetricsScheme       = "http"
	defaultObsMetricsCert         = "/etc/metrics-tls/tls.crt"
	defaultObsMetricsKey          = "/etc/metrics-tls/tls.key"
)

const (
	csiObsMetricsEnabledKey = "X_CSI_METRICS_ENABLED"
	csiObsMetricsPortKey    = "X_CSI_METRICS_PORT"
	csiObsMetricsSchemeKey  = "X_CSI_METRICS_SCHEME"
	csiObsMetricsCertKey    = "X_CSI_METRICS_TLS_CERT_FILE"
	csiObsMetricsKeyKey     = "X_CSI_METRICS_TLS_KEY_FILE"
)

var cPath string

func main() {
	ctx := context.Background()
	config, exporter, powerMaxSvc := configure(ctx)
	if strings.EqualFold(viper.GetString(csiObsMetricsEnabledKey), "true") {
		startMetricsServer(powerMaxSvc)
	}
	if err := entrypoint.Run(ctx, config, exporter, powerMaxSvc); err != nil {
		csmlog.WithFields(csmlog.Fields{"error": err}).Fatal("running service")
	}
}

func configure(ctx context.Context) (*entrypoint.Config, otlexporters.Otlexporter, *service.PowerMaxService) {
	viper.SetConfigFile(defaultConfigFile)

	err := viper.ReadInConfig()
	// if unable to read configuration file, proceed in case we use environment variables
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read Config file: %v", err)
	}

	configFileListener := viper.New()
	configFileListener.SetConfigType("yaml")
	if os.Getenv("X_CSI_REVPROXY_USE_SECRET") == "true" {
		csmlog.Infof("We will be using the SECRET as the config file")
		configFileListener.SetConfigFile(defaultSecretConfigFile)
		cPath = defaultSecretConfigFile
	} else {
		csmlog.Infof("We will be using the CONFIGMAP as the config file")
		configFileListener.SetConfigFile(defaultReverseProxyConfigFile)
		cPath = defaultReverseProxyConfigFile
	}

	leaderElectorGetter := &k8s.LeaderElector{
		API: &k8s.LeaderElector{},
	}

	updateLoggingSettings := func() {
		logFormat := strings.ToLower(viper.GetString("LOG_FORMAT"))
		if logFormat == "text" {
			// use text formatter when explicitly specified
			csmlog.SetFormat("text")
		} else {
			// use JSON formatter by default
			csmlog.SetFormat("json")
		}
		logLevel := viper.GetString("LOG_LEVEL")
		level, err := csmlog.ParseLevel(logLevel)
		if err != nil {
			// use INFO level by default
			level = csmlog.InfoLevel
			csmlog.WithFields(csmlog.Fields{"error": err, "log_level": logLevel}).Warn("Failed to parse user-specified log level, defaulting to INFO")
		}
		csmlog.SetLevel(level)
	}

	updateLoggingSettings()

	volumeFinder := &k8s.VolumeFinder{
		API: &k8s.API{},
	}

	storageClassFinder := &k8s.StorageClassFinder{
		API: &k8s.API{},
	}

	var collectorCertPath string
	if tls := os.Getenv("TLS_ENABLED"); tls == "true" {
		collectorCertPath = os.Getenv("COLLECTOR_CERT_PATH")
		if len(strings.TrimSpace(collectorCertPath)) < 1 {
			collectorCertPath = otlexporters.DefaultCollectorCertPath
		}
	}

	config := &entrypoint.Config{
		LeaderElector:     leaderElectorGetter,
		CollectorCertPath: collectorCertPath,
	}

	exporter := &otlexporters.OtlCollectorExporter{}

	powerMaxSvc := &service.PowerMaxService{
		MetricsRecorder: &metric.MetricsRecorderWrapper{
			Meter: otel.Meter("powermax"),
		},
		VolumeFinder:       volumeFinder,
		StorageClassFinder: storageClassFinder,
	}

	sa := &ServiceAccessor{
		powerMaxSvc: powerMaxSvc,
	}

	_, err = InitK8sUtils(sa, true)
	if err != nil {
		csmlog.WithFields(csmlog.Fields{"error": err}).Fatal("cannot initialize k8sUtils")
	}

	onChangeUpdate(ctx, config, exporter, powerMaxSvc, storageClassFinder, volumeFinder)

	viper.WatchConfig()
	viper.OnConfigChange(func(_ fsnotify.Event) {
		updateLoggingSettings()
	})

	configFileListener.WatchConfig()
	configFileListener.OnConfigChange(func(_ fsnotify.Event) {
		onChangeUpdate(ctx, config, exporter, powerMaxSvc, storageClassFinder, volumeFinder)
	})

	return config, exporter, powerMaxSvc
}

func onChangeUpdate(ctx context.Context, config *entrypoint.Config, exporter *otlexporters.OtlCollectorExporter, powerMaxSvc *service.PowerMaxService, storageClassFinder *k8s.StorageClassFinder, volumeFinder *k8s.VolumeFinder) {
	updatePowerMaxConnection(ctx, powerMaxSvc, storageClassFinder, volumeFinder)
	updateMetricsEnabled(config)
	updateCollectorAddress(config, exporter)
	updateTickIntervals(config)
	updateMaxConnections(powerMaxSvc)
}

// InitK8sUtils initializes Kubernetes utilities with the given service accessor
var InitK8sUtils = func(sa ServiceAccessorInterface, _ bool) (*k8sutils.K8sUtils, error) {
	return k8spmax.InitK8sUtils(sa.UpdatePowerMaxArraysOnSecretChanged, true)
}

// ServiceAccessor provides access to PowerMax service operations
type ServiceAccessor struct {
	powerMaxSvc *service.PowerMaxService
}

// ServiceAccessorInterface defines methods for accessing PowerMax service operations
type ServiceAccessorInterface interface {
	UpdatePowerMaxArraysOnSecretChanged(k8sutils.UtilsInterface, *corev1.Secret)
}

// UpdatePowerMaxArraysOnSecretChanged updates PowerMax arrays when secrets change
func (sa *ServiceAccessor) UpdatePowerMaxArraysOnSecretChanged(k8sutils.UtilsInterface, *corev1.Secret) {
	updatePowerMaxArrays(context.Background(), sa.powerMaxSvc)
}

// updatePowerMaxConnection iterator all PowerMax arrays and validate connection. Inject valid pmax instances to powerMaxSvc
func updatePowerMaxConnection(ctx context.Context, powerMaxSvc *service.PowerMaxService, storageClassFinder *k8s.StorageClassFinder, volumeFinder *k8s.VolumeFinder) {
	updatePowerMaxArrays(ctx, powerMaxSvc)
	updateProvisionerNames(volumeFinder, storageClassFinder)
}

func updatePowerMaxArrays(ctx context.Context, powerMaxSvc *service.PowerMaxService) {
	arrays, err := GetPowerMaxArrays(ctx, k8spmax.GetK8sUtils(), cPath)
	if err != nil {
		csmlog.WithFields(csmlog.Fields{"error": err}).Error("initialize powermax arrays in controller service")
		return
	}

	powerMaxClients := make(map[string][]metrictypes.PowerMaxArray)

	for arrayID, powerMaxArrays := range arrays {
		powerMaxClients[arrayID] = append(powerMaxClients[arrayID], powerMaxArrays...)
		csmlog.WithFields(csmlog.Fields{"arrayID": arrayID}).Debug("setting powermax client")
	}
	powerMaxSvc.SetPowerMaxClients(powerMaxClients)
}

// GetPowerMaxArrays retrieves PowerMax arrays from Kubernetes
var GetPowerMaxArrays = func(ctx context.Context, _ k8sutils.UtilsInterface, _ string) (map[string][]metrictypes.PowerMaxArray, error) {
	return k8spmax.GetPowerMaxArrays(ctx, k8spmax.GetK8sUtils(), cPath)
}

func updateCollectorAddress(config *entrypoint.Config, exporter *otlexporters.OtlCollectorExporter) {
	collectorAddress := viper.GetString("COLLECTOR_ADDR")
	if collectorAddress == "" {
		csmlog.Error("COLLECTOR_ADDR is required")
		return
	}
	config.CollectorAddress = collectorAddress
	exporter.CollectorAddr = collectorAddress
	csmlog.WithFields(csmlog.Fields{"collector_address": collectorAddress}).Debug("setting collector address")
}

func updateProvisionerNames(volumeFinder *k8s.VolumeFinder, storageClassFinder *k8s.StorageClassFinder) {
	provisionerNamesValue := viper.GetString("PROVISIONER_NAMES")
	if provisionerNamesValue == "" {
		csmlog.Error("PROVISIONER_NAMES is required")
		return
	}
	provisionerNames := strings.Split(provisionerNamesValue, ",")
	volumeFinder.DriverNames = provisionerNames

	for i := range storageClassFinder.StorageArrayID {
		storageClassFinder.StorageArrayID[i].DriverNames = provisionerNames
	}

	csmlog.WithFields(csmlog.Fields{"provisioner_names": provisionerNamesValue}).Debug("setting provisioner names")
}

func updateMetricsEnabled(config *entrypoint.Config) {
	capacityMetricsEnabled := true
	capacityMetricsEnabledValue := viper.GetString("POWERMAX_CAPACITY_METRICS_ENABLED")
	if capacityMetricsEnabledValue == "false" {
		capacityMetricsEnabled = false
	}
	config.CapacityMetricsEnabled = capacityMetricsEnabled
	csmlog.WithFields(csmlog.Fields{"capacity_metrics_enabled": capacityMetricsEnabled}).Debug("setting capacity metrics enabled")

	performanceMetricsEnabled := true
	performanceMetricsEnabledValue := viper.GetString("POWERMAX_PERFORMANCE_METRICS_ENABLED")
	if performanceMetricsEnabledValue == "false" {
		performanceMetricsEnabled = false
	}
	config.PerformanceMetricsEnabled = performanceMetricsEnabled
	csmlog.WithFields(csmlog.Fields{"performance_metrics_enabled": performanceMetricsEnabled}).Debug("setting performance metrics enabled")

	topologyMetricsEnabled := true
	topologyMetricsEnabledValue := viper.GetString("POWERMAX_TOPOLOGY_METRICS_ENABLED")
	if topologyMetricsEnabledValue == "false" {
		topologyMetricsEnabled = false
	}
	config.TopologyMetricsEnabled = topologyMetricsEnabled
	csmlog.WithFields(csmlog.Fields{"topology_metrics_enabled": topologyMetricsEnabled}).Debug("setting topology metrics enabled")
}

func updateTickIntervals(config *entrypoint.Config) {
	capacityTickInterval := defaultTickInterval
	capacityPollFrequencySeconds := viper.GetString("POWERMAX_CAPACITY_POLL_FREQUENCY")
	if capacityPollFrequencySeconds != "" {
		numSeconds, err := strconv.Atoi(capacityPollFrequencySeconds)
		if err != nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("POWERMAX_CAPACITY_POLL_FREQUENCY was not set to a valid number")
			numSeconds = int(defaultTickInterval.Seconds())
		}
		capacityTickInterval = time.Duration(numSeconds) * time.Second
	}
	config.CapacityTickInterval = capacityTickInterval
	csmlog.WithFields(csmlog.Fields{"capacity_tick_interval": fmt.Sprintf("%v", capacityTickInterval)}).Debug("setting capacity tick interval")

	performanceTickInterval := defaultTickInterval
	performancePollFrequencySeconds := viper.GetString("POWERMAX_PERFORMANCE_POLL_FREQUENCY")
	if performancePollFrequencySeconds != "" {
		numSeconds, err := strconv.Atoi(performancePollFrequencySeconds)
		if err != nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("POWERMAX_PERFORMANCE_POLL_FREQUENCY was not set to a valid number")
			numSeconds = int(defaultTickInterval.Seconds())
		}
		performanceTickInterval = time.Duration(numSeconds) * time.Second
	}
	config.PerformanceTickInterval = performanceTickInterval
	csmlog.WithFields(csmlog.Fields{"performance_tick_interval": fmt.Sprintf("%v", performanceTickInterval)}).Debug("setting performance tick interval")

	topologyMetricsTickInterval := defaultTickInterval
	topologyMetricsPollFrequencySeconds := viper.GetString("POWERMAX_TOPOLOGY_METRICS_POLL_FREQUENCY")
	if topologyMetricsPollFrequencySeconds != "" {
		numSeconds, err := strconv.Atoi(topologyMetricsPollFrequencySeconds)
		if err != nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("POWERMAX_TOPOLOGY_METRICS_POLL_FREQUENCY was not set to a valid number")
			numSeconds = int(defaultTickInterval.Seconds())
		}
		topologyMetricsTickInterval = time.Duration(numSeconds) * time.Second
	}
	config.TopologyMetricsTickInterval = topologyMetricsTickInterval
	csmlog.WithFields(csmlog.Fields{"cluster_performance_tick_interval": fmt.Sprintf("%v", topologyMetricsTickInterval)}).Debug("setting cluster performance tick interval")
}

func updateMaxConnections(powerMaxSvc *service.PowerMaxService) {
	maxPowerMaxConcurrentRequests := service.DefaultMaxPowerMaxConnections
	maxPowerMaxConcurrentRequestsVar := viper.GetString("POWERMAX_MAX_CONCURRENT_QUERIES")
	if maxPowerMaxConcurrentRequestsVar != "" {
		convertedMaxPowerMaxConcurrentRequests, err := strconv.Atoi(maxPowerMaxConcurrentRequestsVar)
		if err != nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("POWERMAX_MAX_CONCURRENT_QUERIES was not set to a valid number")
		} else if convertedMaxPowerMaxConcurrentRequests <= 0 {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("POWERMAX_MAX_CONCURRENT_QUERIES value was invalid (<= 0)")
		} else {
			maxPowerMaxConcurrentRequests = convertedMaxPowerMaxConcurrentRequests
		}
	}
	powerMaxSvc.MaxPowerMaxConnections = maxPowerMaxConcurrentRequests
	csmlog.WithFields(csmlog.Fields{"max_connections": maxPowerMaxConcurrentRequests}).Debug("setting max powermax connections")
}

// validateTLSFiles checks that the certificate and key files exist and are readable.
func validateTLSFiles(certFile, keyFile string) error {
	for _, path := range []string{certFile, keyFile} {
		f, err := os.Open(path) // #nosec G304 -- path comes from trusted configuration
		if err != nil {
			return fmt.Errorf("cannot open TLS file %q: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("cannot close TLS file %q: %w", path, err)
		}
	}
	return nil
}

// startMetricsServer creates a Prometheus registry, registers the PMAXObsInstrumenter,
// and starts the observability self-metrics HTTP(S) server in a background goroutine.
func startMetricsServer(powerMaxSvc *service.PowerMaxService) {
	reg := prometheus.NewRegistry()
	powerMaxSvc.ObsInstrumenter = service.NewPMAXObsInstrumenter(reg)

	viper.SetDefault(csiObsMetricsPortKey, defaultObsMetricsPort)
	metricsPort := viper.GetString(csiObsMetricsPortKey)

	viper.SetDefault(csiObsMetricsSchemeKey, defaultObsMetricsScheme)
	scheme := viper.GetString(csiObsMetricsSchemeKey)

	var certFile, keyFile string
	if strings.EqualFold(scheme, "https") {
		viper.SetDefault(csiObsMetricsCertKey, defaultObsMetricsCert)
		viper.SetDefault(csiObsMetricsKeyKey, defaultObsMetricsKey)
		certFile = viper.GetString(csiObsMetricsCertKey)
		keyFile = viper.GetString(csiObsMetricsKeyKey)

		if err := validateTLSFiles(certFile, keyFile); err != nil {
			csmlog.WithFields(csmlog.Fields{
				"error": err,
				"cert":  certFile,
				"key":   keyFile,
			}).Fatal("observability metrics server failed to start: invalid TLS configuration")
			return
		}
	}

	srv := csmserver.NewMetricsServer(csmserver.Config{
		Port:     fmt.Sprintf(":%s", metricsPort),
		CertFile: certFile,
		KeyFile:  keyFile,
		Registry: reg,
	})

	go func() {
		csmlog.WithFields(csmlog.Fields{
			"port":   metricsPort,
			"scheme": scheme,
		}).Info("starting observability metrics server")
		if err := srv.Start(); err != nil {
			csmlog.WithFields(csmlog.Fields{"error": err}).Error("observability metrics server closed")
		}
	}()
}
