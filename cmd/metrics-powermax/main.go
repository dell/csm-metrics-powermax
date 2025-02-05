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
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	corev1 "k8s.io/api/core/v1"

	"github.com/dell/csm-metrics-powermax/internal/common"
	"github.com/dell/csm-metrics-powermax/internal/entrypoint"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"

	"github.com/sirupsen/logrus"

	"go.opentelemetry.io/otel"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	defaultTickInterval = 20 * time.Second
	defaultConfigFile   = "/etc/config/karavi-metrics-powermax.yaml"
	// defaultSecret                 = "/powermax-config/default-secret"
	defaultReverseProxyConfigFile = "/etc/reverseproxy/config.yaml"
	defaultSecretConfigFile       = "/etc/powermax/config" // #nosec G101
)

var (
	logger *logrus.Logger
	cPath  string
)

func main() {
	ctx := context.Background()
	config, exporter, powerMaxSvc := configure(ctx)
	if err := entrypoint.Run(ctx, config, exporter, powerMaxSvc); err != nil {
		logger.WithError(err).Fatal("running service")
	}
}

func configure(ctx context.Context) (*entrypoint.Config, otlexporters.Otlexporter, *service.PowerMaxService) {
	logger = logrus.New()

	viper.SetConfigFile(defaultConfigFile)

	err := viper.ReadInConfig()
	// if unable to read configuration file, proceed in case we use environment variables
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read Config file: %v", err)
	}

	configFileListener := viper.New()
	configFileListener.SetConfigType("yaml")
	if os.Getenv("X_CSI_REVPROXY_USE_SECRET") == "true" {
		logger.Infof("We will be using the SECRET as the config file")
		configFileListener.SetConfigFile(defaultSecretConfigFile)
		cPath = defaultSecretConfigFile
	} else {
		logger.Infof("We will be using the CONFIGMAP as the config file")
		configFileListener.SetConfigFile(defaultReverseProxyConfigFile)
		cPath = defaultReverseProxyConfigFile
	}

	leaderElectorGetter := &k8s.LeaderElector{
		API: &k8s.LeaderElector{},
	}

	updateLoggingSettings := func(logger *logrus.Logger) {
		logFormat := viper.GetString("LOG_FORMAT")
		if strings.EqualFold(logFormat, "json") {
			logger.SetFormatter(&logrus.JSONFormatter{})
		} else {
			// use text formatter by default
			logger.SetFormatter(&logrus.TextFormatter{})
		}
		logLevel := viper.GetString("LOG_LEVEL")
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			// use INFO level by default
			level = logrus.InfoLevel
		}
		logger.SetLevel(level)
	}

	updateLoggingSettings(logger)

	volumeFinder := &k8s.VolumeFinder{
		API:    &k8s.API{},
		Logger: logger,
	}

	storageClassFinder := &k8s.StorageClassFinder{
		API:    &k8s.API{},
		Logger: logger,
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
		Logger:            logger,
	}

	exporter := &otlexporters.OtlCollectorExporter{}

	powerMaxSvc := &service.PowerMaxService{
		MetricsRecorder: &metric.MetricsRecorderWrapper{
			Meter: otel.Meter("powermax"),
		},
		Logger:             logger,
		VolumeFinder:       volumeFinder,
		StorageClassFinder: storageClassFinder,
	}

	sa := &ServiceAccessor{
		powerMaxSvc: powerMaxSvc,
	}

	_, err = InitK8sUtils(logger, sa, true)
	if err != nil {
		logger.WithError(err).Fatal("cannot initialize k8sUtils")
	}

	onChangeUpdate(ctx, config, exporter, powerMaxSvc, storageClassFinder, volumeFinder)

	viper.WatchConfig()
	viper.OnConfigChange(func(_ fsnotify.Event) {
		updateLoggingSettings(logger)
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

var InitK8sUtils = func(logger *logrus.Logger, sa ServiceAccessorInterface, inCluster bool) (*k8sutils.K8sUtils, error) {
	return common.InitK8sUtils(logger, sa.UpdatePowerMaxArraysOnSecretChanged, true)
}

type ServiceAccessor struct {
	powerMaxSvc *service.PowerMaxService
}

type ServiceAccessorInterface interface {
	UpdatePowerMaxArraysOnSecretChanged(k8sutils.UtilsInterface, *corev1.Secret)
}

func (sa *ServiceAccessor) UpdatePowerMaxArraysOnSecretChanged(k8sutils.UtilsInterface, *corev1.Secret) {
	updatePowerMaxArrays(context.Background(), sa.powerMaxSvc)
}

// updatePowerMaxConnection iterator all PowerMax arrays and validate connection. Inject valid pmax instances to powerMaxSvc
func updatePowerMaxConnection(ctx context.Context, powerMaxSvc *service.PowerMaxService, storageClassFinder *k8s.StorageClassFinder, volumeFinder *k8s.VolumeFinder) {
	updatePowerMaxArrays(ctx, powerMaxSvc)
	updateProvisionerNames(volumeFinder, storageClassFinder)
}

func updatePowerMaxArrays(ctx context.Context, powerMaxSvc *service.PowerMaxService) {
	arrays, err := GetPowerMaxArrays(ctx, common.GetK8sUtils(), cPath, logger)
	if err != nil {
		logger.WithError(err).Error("initialize powermax arrays in controller service")
		return
	}

	powerMaxClients := make(map[string][]types.PowerMaxArray)

	for arrayID, powerMaxArrays := range arrays {
		powerMaxClients[arrayID] = append(powerMaxClients[arrayID], powerMaxArrays...)
		logger.WithField("arrayID", arrayID).Debug("setting powermax client")
	}
	powerMaxSvc.PowerMaxClients = powerMaxClients
}

var GetPowerMaxArrays = func(ctx context.Context, k8sUtils k8sutils.UtilsInterface, filePath string, logger *logrus.Logger) (map[string][]types.PowerMaxArray, error) {
	return common.GetPowerMaxArrays(ctx, common.GetK8sUtils(), cPath, logger)
}

func updateCollectorAddress(config *entrypoint.Config, exporter *otlexporters.OtlCollectorExporter) {
	collectorAddress := viper.GetString("COLLECTOR_ADDR")
	if collectorAddress == "" {
		logger.Error("COLLECTOR_ADDR is required")
		return
	}
	config.CollectorAddress = collectorAddress
	exporter.CollectorAddr = collectorAddress
	logger.WithField("collector_address", collectorAddress).Debug("setting collector address")
}

func updateProvisionerNames(volumeFinder *k8s.VolumeFinder, storageClassFinder *k8s.StorageClassFinder) {
	provisionerNamesValue := viper.GetString("PROVISIONER_NAMES")
	if provisionerNamesValue == "" {
		logger.Error("PROVISIONER_NAMES is required")
		return
	}
	provisionerNames := strings.Split(provisionerNamesValue, ",")
	volumeFinder.DriverNames = provisionerNames

	for i := range storageClassFinder.StorageArrayID {
		storageClassFinder.StorageArrayID[i].DriverNames = provisionerNames
	}

	logger.WithField("provisioner_names", provisionerNamesValue).Debug("setting provisioner names")
}

func updateMetricsEnabled(config *entrypoint.Config) {
	capacityMetricsEnabled := true
	capacityMetricsEnabledValue := viper.GetString("POWERMAX_CAPACITY_METRICS_ENABLED")
	if capacityMetricsEnabledValue == "false" {
		capacityMetricsEnabled = false
	}
	config.CapacityMetricsEnabled = capacityMetricsEnabled
	logger.WithField("capacity_metrics_enabled", capacityMetricsEnabled).Debug("setting capacity metrics enabled")

	performanceMetricsEnabled := true
	performanceMetricsEnabledValue := viper.GetString("POWERMAX_PERFORMANCE_METRICS_ENABLED")
	if performanceMetricsEnabledValue == "false" {
		performanceMetricsEnabled = false
	}
	config.PerformanceMetricsEnabled = performanceMetricsEnabled
	logger.WithField("performance_metrics_enabled", performanceMetricsEnabled).Debug("setting performance metrics enabled")
}

func updateTickIntervals(config *entrypoint.Config) {
	capacityTickInterval := defaultTickInterval
	capacityPollFrequencySeconds := viper.GetString("POWERMAX_CAPACITY_POLL_FREQUENCY")
	if capacityPollFrequencySeconds != "" {
		numSeconds, err := strconv.Atoi(capacityPollFrequencySeconds)
		if err != nil {
			logger.WithError(err).Error("POWERMAX_CAPACITY_POLL_FREQUENCY was not set to a valid number")
			numSeconds = int(defaultTickInterval.Seconds())
		}
		capacityTickInterval = time.Duration(numSeconds) * time.Second
	}
	config.CapacityTickInterval = capacityTickInterval
	logger.WithField("capacity_tick_interval", fmt.Sprintf("%v", capacityTickInterval)).Debug("setting capacity tick interval")

	performanceTickInterval := defaultTickInterval
	performancePollFrequencySeconds := viper.GetString("POWERMAX_PERFORMANCE_POLL_FREQUENCY")
	if performancePollFrequencySeconds != "" {
		numSeconds, err := strconv.Atoi(performancePollFrequencySeconds)
		if err != nil {
			logger.WithError(err).Error("POWERMAX_PERFORMANCE_POLL_FREQUENCY was not set to a valid number")
			numSeconds = int(defaultTickInterval.Seconds())
		}
		performanceTickInterval = time.Duration(numSeconds) * time.Second
	}
	config.PerformanceTickInterval = performanceTickInterval
	logger.WithField("performance_tick_interval", fmt.Sprintf("%v", performanceTickInterval)).Debug("setting performance tick interval")
}

func updateMaxConnections(powerMaxSvc *service.PowerMaxService) {
	maxPowerMaxConcurrentRequests := service.DefaultMaxPowerMaxConnections
	maxPowerMaxConcurrentRequestsVar := viper.GetString("POWERMAX_MAX_CONCURRENT_QUERIES")
	if maxPowerMaxConcurrentRequestsVar != "" {
		convertedMaxPowerMaxConcurrentRequests, err := strconv.Atoi(maxPowerMaxConcurrentRequestsVar)
		if err != nil {
			logger.WithError(err).Error("POWERMAX_MAX_CONCURRENT_QUERIES was not set to a valid number")
		} else if convertedMaxPowerMaxConcurrentRequests <= 0 {
			logger.WithError(err).Error("POWERMAX_MAX_CONCURRENT_QUERIES value was invalid (<= 0)")
		} else {
			maxPowerMaxConcurrentRequests = convertedMaxPowerMaxConcurrentRequests
		}
	}
	powerMaxSvc.MaxPowerMaxConnections = maxPowerMaxConcurrentRequests
	logger.WithField("max_connections", maxPowerMaxConcurrentRequests).Debug("setting max powermax connections")
}
