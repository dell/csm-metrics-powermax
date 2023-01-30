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
	"github.com/dell/csm-metrics-powermax/internal/k8sutils"
	"github.com/dell/csm-metrics-powermax/internal/service/metric"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/common"
	"github.com/dell/csm-metrics-powermax/internal/entrypoint"

	"github.com/dell/csm-metrics-powermax/internal/k8s"
	"github.com/dell/csm-metrics-powermax/internal/service"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"

	"github.com/sirupsen/logrus"

	"os"

	"go.opentelemetry.io/otel/metric/global"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

const (
	defaultTickInterval = 20 * time.Second
	defaultConfigFile   = "/etc/config/karavi-metrics-powermax.yaml"
	// defaultSecret                 = "/powermax-config/default-secret"
	defaultReverseProxyConfigFile = "/etc/reverseproxy/config.yaml"
)

var logger *logrus.Logger
var powerMaxSvc *service.PowerMaxService
var ctx context.Context

func main() {

	logger = logrus.New()

	viper.SetConfigFile(defaultConfigFile)

	err := viper.ReadInConfig()
	// if unable to read configuration file, proceed in case we use environment variables
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to read Config file: %v", err)
	}

	configFileListener := viper.New()
	configFileListener.SetConfigFile(defaultReverseProxyConfigFile)

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

	powerMaxSvc = &service.PowerMaxService{
		MetricsRecorder: &metric.MetricsRecorderWrapper{
			Meter: global.Meter("powermax"),
		},
		Logger:             logger,
		VolumeFinder:       volumeFinder,
		StorageClassFinder: storageClassFinder,
	}

	base := metric.NewBaseMetrics(powerMaxSvc)
	powerMaxSvc.ArrayCapacityMetricsInstance = &metric.ArrayCapacityMetrics{BaseMetrics: base}
	base.Collector = powerMaxSvc.ArrayCapacityMetricsInstance

	ctx = context.Background()

	common.InitK8sUtils(logger, updatePowerMaxArraysOnSecretChanged)
	updatePowerMaxConnection(ctx, powerMaxSvc, storageClassFinder, volumeFinder)
	updateCollectorAddress(config, exporter)
	updateMetricsEnabled(config)
	updateTickIntervals(config)
	updateMaxConnections(powerMaxSvc)

	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		updateLoggingSettings(logger)
		updateCollectorAddress(config, exporter)
		updatePowerMaxConnection(ctx, powerMaxSvc, storageClassFinder, volumeFinder)
		updateMetricsEnabled(config)
		updateTickIntervals(config)
		updateMaxConnections(powerMaxSvc)
	})

	configFileListener.WatchConfig()
	configFileListener.OnConfigChange(func(e fsnotify.Event) {
		updatePowerMaxConnection(ctx, powerMaxSvc, storageClassFinder, volumeFinder)
	})

	if err := entrypoint.Run(ctx, config, exporter, powerMaxSvc); err != nil {
		logger.WithError(err).Fatal("running service")
	}
}

func updatePowerMaxArraysOnSecretChanged(k8sutils.UtilsInterface, *corev1.Secret) {
	updatePowerMaxArrays(ctx, powerMaxSvc)
}

// updatePowerMaxConnection iterator all PowerMax arrays and validate connection. Inject valid pmax instances to powerMaxSvc
func updatePowerMaxConnection(ctx context.Context, powerMaxSvc *service.PowerMaxService, storageClassFinder *k8s.StorageClassFinder, volumeFinder *k8s.VolumeFinder) {
	updatePowerMaxArrays(ctx, powerMaxSvc)
	updateProvisionerNames(volumeFinder, storageClassFinder)
}

func updatePowerMaxArrays(ctx context.Context, powerMaxSvc *service.PowerMaxService) {
	arrays, err := common.GetPowerMaxArrays(ctx, common.GetK8sUtils(), defaultReverseProxyConfigFile, logger)
	if err != nil {
		logger.WithError(err).Fatal("initialize powermax arrays in controller service")
	}
	powerMaxClients := make(map[string]types.PowerMaxClient)
	storageArrayID := make([]k8s.StorageArrayID, len(arrays))

	for arrayID, powerMaxArray := range arrays {
		powerMaxClients[arrayID] = powerMaxArray.Client
		logger.WithField("arrayID", arrayID).Debug("setting powermax client")

		var arrayID = k8s.StorageArrayID{
			ID: arrayID,
		}
		storageArrayID = append(storageArrayID, arrayID)
	}
	powerMaxSvc.PowerMaxClients = powerMaxClients
}

func updateCollectorAddress(config *entrypoint.Config, exporter *otlexporters.OtlCollectorExporter) {
	collectorAddress := viper.GetString("COLLECTOR_ADDR")
	if collectorAddress == "" {
		logger.Fatal("COLLECTOR_ADDR is required")
	}
	config.CollectorAddress = collectorAddress
	exporter.CollectorAddr = collectorAddress
	logger.WithField("collector_address", collectorAddress).Debug("setting collector address")
}

func updateProvisionerNames(volumeFinder *k8s.VolumeFinder, storageClassFinder *k8s.StorageClassFinder) {
	provisionerNamesValue := viper.GetString("PROVISIONER_NAMES")
	if provisionerNamesValue == "" {
		logger.Fatal("PROVISIONER_NAMES is required")
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
	arrayCapacityTickInterval := defaultTickInterval
	arrayCapacityPollFrequencySeconds := viper.GetString("POWERMAX_ARRAY_CAPACITY_POLL_FREQUENCY")
	if arrayCapacityPollFrequencySeconds != "" {
		numSeconds, err := strconv.Atoi(arrayCapacityPollFrequencySeconds)
		if err != nil {
			logger.WithError(err).Fatal("POWERMAX_ARRAY_CAPACITY_POLL_FREQUENCY was not set to a valid number")
		}
		arrayCapacityTickInterval = time.Duration(numSeconds) * time.Second
	}
	config.ArrayCapacityTickInterval = arrayCapacityTickInterval
	logger.WithField("array_capacity_tick_interval", fmt.Sprintf("%v", arrayCapacityTickInterval)).Debug("setting array capacity tick interval")

	// quotaCapacityTickInterval := defaultTickInterval
	// quotaCapacityPollFrequencySeconds := viper.GetString("POWERMAX_QUOTA_CAPACITY_POLL_FREQUENCY")
	// if quotaCapacityPollFrequencySeconds != "" {
	// 	numSeconds, err := strconv.Atoi(quotaCapacityPollFrequencySeconds)
	// 	if err != nil {
	// 		logger.WithError(err).Fatal("POWERMAX_QUOTA_CAPACITY_POLL_FREQUENCY was not set to a valid number")
	// 	}
	// 	quotaCapacityTickInterval = time.Duration(numSeconds) * time.Second
	// }
	// config.QuotaCapacityTickInterval = quotaCapacityTickInterval
	// logger.WithField("quota_capacity_tick_interval", fmt.Sprintf("%v", quotaCapacityTickInterval)).Debug("setting quota capacity tick interval")
	//
	// clusterCapacityTickInterval := defaultTickInterval
	// clusterCapacityPollFrequencySeconds := viper.GetString("POWERMAX_CLUSTER_CAPACITY_POLL_FREQUENCY")
	// if clusterCapacityPollFrequencySeconds != "" {
	// 	numSeconds, err := strconv.Atoi(clusterCapacityPollFrequencySeconds)
	// 	if err != nil {
	// 		logger.WithError(err).Fatal("POWERMAX_CLUSTER_CAPACITY_POLL_FREQUENCY was not set to a valid number")
	// 	}
	// 	clusterCapacityTickInterval = time.Duration(numSeconds) * time.Second
	// }
	// config.ClusterCapacityTickInterval = clusterCapacityTickInterval
	// logger.WithField("cluster_capacity_tick_interval", fmt.Sprintf("%v", clusterCapacityTickInterval)).Debug("setting cluster capacity tick interval")
	//
	// clusterPerformanceTickInterval := defaultTickInterval
	// clusterPerformancePollFrequencySeconds := viper.GetString("POWERMAX_CLUSTER_PERFORMANCE_POLL_FREQUENCY")
	// if clusterPerformancePollFrequencySeconds != "" {
	// 	numSeconds, err := strconv.Atoi(clusterPerformancePollFrequencySeconds)
	// 	if err != nil {
	// 		logger.WithError(err).Fatal("POWERMAX_CLUSTER_PERFORMANCE_POLL_FREQUENCY was not set to a valid number")
	// 	}
	// 	clusterPerformanceTickInterval = time.Duration(numSeconds) * time.Second
	// }
	// config.ClusterPerformanceTickInterval = clusterPerformanceTickInterval
	// logger.WithField("cluster_performance_tick_interval", fmt.Sprintf("%v", clusterPerformanceTickInterval)).Debug("setting cluster performance tick interval")
}

func updateMaxConnections(powerMaxSvc *service.PowerMaxService) {
	maxPowerMaxConcurrentRequests := service.DefaultMaxPowerMaxConnections
	maxPowerMaxConcurrentRequestsVar := viper.GetString("POWERMAX_MAX_CONCURRENT_QUERIES")
	if maxPowerMaxConcurrentRequestsVar != "" {
		maxPowermaxConcurrentRequests, err := strconv.Atoi(maxPowerMaxConcurrentRequestsVar)
		if err != nil {
			logger.WithError(err).Fatal("POWERMAX_MAX_CONCURRENT_QUERIES was not set to a valid number")
		}
		if maxPowermaxConcurrentRequests <= 0 {
			logger.WithError(err).Fatal("POWERMAX_MAX_CONCURRENT_QUERIES value was invalid (<= 0)")
		}
	}
	powerMaxSvc.MaxPowerMaxConnections = maxPowerMaxConcurrentRequests
	logger.WithField("max_connections", maxPowerMaxConcurrentRequests).Debug("setting max powermax connections")
}
