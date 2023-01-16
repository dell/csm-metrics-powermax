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

package config

import (
	"fmt"
	k8sutils "github.com/dell/csm-metrics-powermax/internal/k8sutils"
	common "github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	k8smock "github.com/dell/csm-metrics-powermax/internal/reverseproxy/k8smock"
	utils "github.com/dell/csm-metrics-powermax/internal/reverseproxy/utils"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
)

func readConfig() (*ProxyConfigMap, error) {
	return ReadConfig("./../" + filepath.Join(common.TestConfigDir, common.TestConfigFileName))
}

func readInvalidStandAloneConfig() (*ProxyConfigMap, error) {
	return ReadConfig("./../" + filepath.Join(common.TestConfigDir, "invalid_standalone_config.yaml"))
}

func readUnknownModeConfig() (*ProxyConfigMap, error) {
	return ReadConfig("./../" + filepath.Join(common.TestConfigDir, "unknown_mode_config.yaml"))
}

func readEmptyPrimaryConfig() (*ProxyConfigMap, error) {
	return ReadConfig("./../" + filepath.Join(common.TestConfigDir, "primary_error_config.yaml"))
}

func readPrimaryNotConfiguredConfig() (*ProxyConfigMap, error) {
	return ReadConfig("./../" + filepath.Join(common.TestConfigDir, "primary_not_configured.yaml"))
}

func TestMain(m *testing.M) {
	status := 0
	if st := m.Run(); st > status {
		status = st
	}
	err := utils.RemoveTempFiles()
	if err != nil {
		log.Fatalf("Failed to cleanup temp files. (%s)", err.Error())
		status = 1
	}
	os.Exit(status)
}

func TestReadConfig(t *testing.T) {
	proxyConfigMap, err := readConfig()
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	fmt.Printf("%v", proxyConfigMap)
}

func newProxyConfig(configMap *ProxyConfigMap, utils k8sutils.UtilsInterface) (*ProxyConfig, error) {
	return NewProxyConfig(configMap, utils)
}

func getStandAloneProxyConfig(t *testing.T) (*ProxyConfig, error) {
	k8sUtils := k8smock.Init()
	configMap, err := readConfig()
	if err != nil {
		t.Errorf("Failed to read config. (%s)", err.Error())
		return nil, err
	}
	for _, storageArray := range configMap.StandAloneConfig.StorageArrayConfig {
		for _, secretName := range storageArray.ProxyCredentialSecrets {
			_, err := k8sUtils.CreateNewCredentialSecret(secretName)
			if err != nil {
				t.Errorf("Failed to create proxy credential secret. (%s)", err.Error())
				return nil, err
			}
		}
	}
	for _, managementServer := range configMap.StandAloneConfig.ManagementServerConfig {
		_, err := k8sUtils.CreateNewCredentialSecret(managementServer.ArrayCredentialSecret)
		if err != nil {
			t.Errorf("Failed to create server credential secret. (%s)", err.Error())
			return nil, err
		}
		_, err = k8sUtils.CreateNewCertSecret(managementServer.CertSecret)
		if err != nil {
			t.Errorf("Fialed to create server cert secret. (%s)", err.Error())
			return nil, err
		}
	}
	configMap.Mode = "StandAlone"
	config, err := NewProxyConfig(configMap, k8sUtils)
	if err != nil {
		t.Errorf("Failed to create new standalone proxy config. (%s)", err.Error())
		return nil, err
	}
	return config, nil
}

func TestNewStandAloneProxyConfig(t *testing.T) {
	config, err := getStandAloneProxyConfig(t)
	if err != nil {
		return
	}
	if config.StandAloneProxyConfig == nil {
		t.Error("Config not created properly")
		return
	}
}

func TestStandAloneProxyConfig_GetStorageArray(t *testing.T) {
	config, err := getStandAloneProxyConfig(t)
	if err != nil {
		return
	}
	fmt.Printf("Storage arrays: %+v\n", config.StandAloneProxyConfig.GetStorageArray("000000000001"))
	fmt.Printf("All Storage arrays: %+v\n", config.StandAloneProxyConfig.GetStorageArray(""))
}

func TestGetManagedArraysAndServers(t *testing.T) {
	config, err := getStandAloneProxyConfig(t)
	if err != nil {
		return
	}
	if config.StandAloneProxyConfig == nil {
		t.Error("Config not created properly")
		return
	}
	fmt.Printf("Management arrays and servers: %+v\n", config.StandAloneProxyConfig.GetManagedArraysAndServers())
}

func TestGetManagementServers(t *testing.T) {
	config, err := getStandAloneProxyConfig(t)
	if err != nil {
		return
	}
	if config.StandAloneProxyConfig == nil {
		t.Error("Config not created properly")
		return
	}
	fmt.Printf("Management servers: %+v\n", config.StandAloneProxyConfig.GetManagementServers())
}

func TestGetInvalidStandAloneProxyConfig(t *testing.T) {
	k8sUtils := k8smock.Init()
	configMap, _ := readInvalidStandAloneConfig()
	config, err := NewProxyConfig(configMap, k8sUtils)

	assert.NotNil(t, err)
	assert.Nil(t, config)
}

func TestGetUnknownModeConfig(t *testing.T) {
	k8sUtils := k8smock.Init()
	configMap, _ := readUnknownModeConfig()
	config, err := NewProxyConfig(configMap, k8sUtils)

	assert.NotNil(t, err)
	assert.Nil(t, config)
}

func TestEmptyPrimaryConfig(t *testing.T) {
	k8sUtils := k8smock.Init()
	configMap, _ := readEmptyPrimaryConfig()
	config, err := NewProxyConfig(configMap, k8sUtils)

	assert.NotNil(t, err)
	assert.Nil(t, config)
}

func TestPrimaryNotConfigured(t *testing.T) {
	k8sUtils := k8smock.Init()
	configMap, _ := readPrimaryNotConfiguredConfig()
	config, err := NewProxyConfig(configMap, k8sUtils)

	assert.NotNil(t, err)
	assert.Nil(t, config)
}
