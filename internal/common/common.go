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

package common

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"

	"github.com/dell/csi-powermax/csireverseproxy/v2/pkg/config"
	"github.com/dell/csi-powermax/csireverseproxy/v2/pkg/k8sutils"
	"github.com/dell/csm-metrics-powermax/internal/service/types"
	pmax "github.com/dell/gopowermax/v2"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultEndpointPort = "8443"
	defaultInsecure     = true
	defaultU4PVersion   = "100" // defaultU4PVersion should be reset to base supported endpoint version for the CSI driver release
	// ApplicationName is the name used to register with Powermax REST APIs
	ApplicationName = "CSM Observability for Dell EMC PowerMax"
	// DefaultNameSpace karavi
	DefaultNameSpace = "karavi"
	// default directory that contains Unisphere cert (not supported for now)
	defaultUnisphereCertDir = "/certs"
)

var k8sUtils *k8sutils.K8sUtils

// GetK8sUtils return K8sUtils instance, which is supposed to be initialized by InitK8sUtils first
func GetK8sUtils() *k8sutils.K8sUtils {
	return k8sUtils
}

func getPowerMaxArrays(proxyConfig *config.ProxyConfig) map[string][]types.PowerMaxArray {
	arrayMap := make(map[string][]types.PowerMaxArray)
	for _, storageArray := range proxyConfig.GetStorageArray("") {
		var arrayList []types.PowerMaxArray
		if _, ok := arrayMap[storageArray.StorageArrayIdentifier]; ok {
			arrayList = arrayMap[storageArray.StorageArrayIdentifier]
		}

		var primaryArray types.PowerMaxArray
		primaryArray.StorageArrayID = storageArray.StorageArrayIdentifier
		primaryArray.IsPrimary = true
		primaryArray.Endpoint = storageArray.PrimaryEndpoint.String()

		managementServers := proxyConfig.GetManagementServers()
		managementServer := getManagementServer(primaryArray.Endpoint, managementServers)

		if managementServer != nil {
			primaryArray.Insecure = managementServer.SkipCertificateValidation
			primaryArray.Username = managementServer.Credentials.UserName
			primaryArray.Password = managementServer.Credentials.Password
		}

		arrayList = append(arrayList, primaryArray)

		if storageArray.SecondaryEndpoint.Host != "" {
			var backupArray types.PowerMaxArray
			backupArray.StorageArrayID = storageArray.StorageArrayIdentifier
			backupArray.IsPrimary = false
			backupArray.Endpoint = storageArray.SecondaryEndpoint.String()
			managementServer := getManagementServer(backupArray.Endpoint, managementServers)

			if managementServer != nil {
				backupArray.Insecure = managementServer.SkipCertificateValidation
				backupArray.Username = managementServer.Credentials.UserName
				backupArray.Password = managementServer.Credentials.Password
			}
			arrayList = append(arrayList, backupArray)
		}

		arrayMap[storageArray.StorageArrayIdentifier] = arrayList
	}

	return arrayMap
}

func getPowerMaxArraysFromSecret(proxyConfig *config.ProxyConfig) map[string][]types.PowerMaxArray {
	arrayMap := make(map[string][]types.PowerMaxArray)
	for _, storageArray := range proxyConfig.GetStorageArray("") {
		var arrayList []types.PowerMaxArray
		if _, ok := arrayMap[storageArray.StorageArrayIdentifier]; ok {
			arrayList = arrayMap[storageArray.StorageArrayIdentifier]
		}

		var primaryArray types.PowerMaxArray
		primaryArray.StorageArrayID = storageArray.StorageArrayIdentifier
		primaryArray.IsPrimary = true
		primaryArray.Endpoint = storageArray.PrimaryEndpoint.String()

		managementServers := proxyConfig.GetManagementServers()
		managementServer := getManagementServer(primaryArray.Endpoint, managementServers)

		if managementServer != nil {
			primaryArray.Insecure = managementServer.SkipCertificateValidation
			primaryArray.Username = managementServer.Username
			primaryArray.Password = managementServer.Password
		}

		arrayList = append(arrayList, primaryArray)

		if storageArray.SecondaryEndpoint.Host != "" {
			var backupArray types.PowerMaxArray
			backupArray.StorageArrayID = storageArray.StorageArrayIdentifier
			backupArray.IsPrimary = false
			backupArray.Endpoint = storageArray.SecondaryEndpoint.String()
			managementServer := getManagementServer(backupArray.Endpoint, managementServers)

			if managementServer != nil {
				backupArray.Insecure = managementServer.SkipCertificateValidation
				backupArray.Username = managementServer.Username
				backupArray.Password = managementServer.Password
			}
			arrayList = append(arrayList, backupArray)
		}

		arrayMap[storageArray.StorageArrayIdentifier] = arrayList
	}

	return arrayMap
}

func getManagementServer(url string, managementServers []config.ManagementServer) *config.ManagementServer {
	for _, server := range managementServers {
		if url == server.Endpoint.String() {
			return &server
		}
	}
	return nil
}

// InitK8sUtils initialize k8sutils instance with a callback method on secret change.
func InitK8sUtils(logger *logrus.Logger, callback func(k8sutils.UtilsInterface, *corev1.Secret)) *k8sutils.K8sUtils {
	if k8sUtils != nil {
		return k8sUtils
	}

	powerMaxNamespace := os.Getenv("POWERMAX_METRICS_NAMESPACE")
	if powerMaxNamespace == "" {
		powerMaxNamespace = DefaultNameSpace
	}

	var err error
	k8sUtils, err = k8sutils.Init(powerMaxNamespace, defaultUnisphereCertDir, true, time.Second*30)
	if err != nil {
		logger.WithError(err).Errorf("cannot initialize k8sUtils")
	}
	err = k8sUtils.StartInformer(callback)
	if err != nil {
		logger.WithError(err).Errorf("cannot start informer")
	}
	return k8sUtils
}

// GetPowerMaxArrays parses reverseproxy config file, initializes valid pmax and composes map of arrays for ease of access.
// Note that if you want to monitor secret change, InitK8sUtils should be invoked first.
func GetPowerMaxArrays(ctx context.Context, k8sUtils k8sutils.UtilsInterface, filePath string, logger *logrus.Logger) (map[string][]types.PowerMaxArray, error) {
	if k8sUtils == nil {
		err := errors.New("k8sUtils is nil")
		logger.WithError(err).Errorf("k8sUtils is not initialized")
		return nil, err
	}

	var (
		proxyConfig    *config.ProxyConfig
		powermaxArrays map[string][]types.PowerMaxArray
	)
	if os.Getenv("REVPROXY_USE_SECRET") == "true" {
		logger.Infof("Reading config from the Secret")
		proxyConfigMap, err := config.ReadConfigFromSecret(viper.New())
		if err != nil {
			logger.WithError(err).Errorf("fail to read ProxyConfig from %s", filePath)
			return nil, err
		}

		proxyConfig, err = config.NewProxyConfigFromSecret(proxyConfigMap, k8sUtils)
		if err != nil {
			logger.WithError(err).Errorf("cannot create new ProxyConfig")
			return nil, err
		}
		powermaxArrays = getPowerMaxArraysFromSecret(proxyConfig)
	} else {
		logger.Infof("Reading config from the ConfigMap")
		logger.Infof("Config File: %s \nConfig Directory: %s", filepath.Base(filePath), filepath.Dir(filePath))
		proxyConfigMap, err := config.ReadConfig(filepath.Base(filePath), filepath.Dir(filePath), viper.New())
		if err != nil {
			logger.WithError(err).Errorf("fail to read ProxyConfig from %s", filePath)
			return nil, err
		}

		proxyConfig, err = config.NewProxyConfig(proxyConfigMap, k8sUtils)
		if err != nil {
			logger.WithError(err).Errorf("cannot create new ProxyConfig")
			return nil, err
		}
		powermaxArrays = getPowerMaxArrays(proxyConfig)
	}

	arrayMap := make(map[string][]types.PowerMaxArray)
	for arrayID, arrayList := range powermaxArrays {
		for _, array := range arrayList {
			logger.Infof("validating PowerMax connection for %s, %s", arrayID, array.Endpoint)
			// currently bypass Unisphere cert. Will use array.Insecure after all obs modules respect array cert
			// userCerts is not used inside gopowermax, but it is supposed to be true if insecure is false
			c, err := pmax.NewClientWithArgs(
				array.Endpoint,
				ApplicationName,
				true,
				false, "")
			if err != nil {
				logger.WithError(err).Errorf("cannot connect to PowerMax array %s, %s", arrayID, array.Endpoint)
				continue
			}

			err = Authenticate(ctx, c, array)
			if err != nil {
				logger.WithError(err).Errorf("authentication failed to PowerMax array %s, %s", arrayID, array.Endpoint)
				continue
			}
			array.IsActive = true
			array.Client = c
			arrayMap[arrayID] = append(arrayMap[arrayID], array)
		}
	}
	logger.Infof("got %d valid PowerMax connections", len(arrayMap))

	return arrayMap, nil
}

// Authenticate PowerMax array if credential is correct or connection is valid
func Authenticate(ctx context.Context, client types.PowerMaxClient, array types.PowerMaxArray) error {
	return client.Authenticate(ctx, &pmax.ConfigConnect{
		Endpoint: array.Endpoint,
		Username: array.Username,
		Password: array.Password,
		Version:  defaultU4PVersion,
	})
}
