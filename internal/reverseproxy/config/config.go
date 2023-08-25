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
	"crypto/subtle"
	"fmt"
	"net/url"

	k8sutils "github.com/dell/csm-metrics-powermax/internal/k8sutils"
	common "github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	utils "github.com/dell/csm-metrics-powermax/internal/reverseproxy/utils"

	"github.com/spf13/viper"
)

// ProxyMode represents the mode the proxy is operating in
type ProxyMode string

// Constants for the proxy configuration
// Linked :- In linked mode, proxy simply forwards, will be obsolete
//
//	all the request to one of the configured
//	primary or backup unispheres based on a fail-over
//	mechanism which is triggered automatically, in case one
//	of the unispheres go down.
//
// StandAlone :- In stand-alone mode, proxy provides multi-array
//
//					 support with ACL to authenticate and authorize
//	              users based on credentials set via k8s secrets.
//	              Each array can have a primary and a backup unisphere
//	              which follows the same fail-over mechanism as Linked
//	              proxy
const (
	Linked     = ProxyMode("Linked")
	StandAlone = ProxyMode("StandAlone")
)

// StorageArrayConfig represents the configuration of a storage array in the config file
type StorageArrayConfig struct {
	StorageArrayID         string   `yaml:"storageArrayId"`
	PrimaryURL             string   `yaml:"primaryURL"`
	BackupURL              string   `yaml:"backupURL,omitempty"`
	ProxyCredentialSecrets []string `yaml:"proxyCredentialSecrets"`
}

// ProxyCredentialSecret is used for storing a credential for a secret
type ProxyCredentialSecret struct {
	Credentials      common.Credentials
	CredentialSecret string
}

// StorageArray represents a StorageArray (formed using StorageArrayConfig)
type StorageArray struct {
	StorageArrayIdentifier string
	PrimaryURL             url.URL
	SecondaryURL           url.URL
	ProxyCredentialSecrets map[string]ProxyCredentialSecret
}

// StorageArrayServer represents an array with its primary and backup management server
type StorageArrayServer struct {
	Array         StorageArray
	PrimaryServer ManagementServer
	BackupServer  *ManagementServer
}

// ManagementServerConfig - represents a management server configuration for the management server
type ManagementServerConfig struct {
	URL                       string        `yaml:"url"`
	ArrayCredentialSecret     string        `yaml:"arrayCredentialSecret,omitempty"`
	SkipCertificateValidation bool          `yaml:"skipCertificateValidation,omitempty"`
	CertSecret                string        `yaml:"certSecret,omitempty"`
	Limits                    common.Limits `yaml:"limits,omitempty" mapstructure:"limits"`
}

// ManagementServer - represents a Management Server (formed using ManagementServerConfig)
type ManagementServer struct {
	URL                       url.URL
	StorageArrayIdentifiers   []string
	Credentials               common.Credentials
	CredentialSecret          string
	SkipCertificateValidation bool
	CertFile                  string
	CertSecret                string
	Limits                    common.Limits
}

// LinkConfig - represents linked proxy configuration in the config file
type LinkConfig struct {
	Primary ManagementServerConfig `yaml:"primary" mapstructure:"primary"`
	Backup  ManagementServerConfig `yaml:"backup,omitempty" mapstructure:"backup"`
}

// StandAloneConfig - represents stand alone proxy configuration in the config file
type StandAloneConfig struct {
	StorageArrayConfig     []StorageArrayConfig     `yaml:"storageArrays" mapstructure:"storageArrays"`
	ManagementServerConfig []ManagementServerConfig `yaml:"managementServers" mapstructure:"managementServers"`
}

// ProxyConfigMap - represents the configuration file
type ProxyConfigMap struct {
	Mode             ProxyMode         `yaml:"mode,omitempty"`
	Port             string            `yaml:"port,omitempty"`
	LogLevel         string            `yaml:"logLevel,omitempty"`
	LogFormat        string            `yaml:"logFormat,omitempty"`
	LinkConfig       *LinkConfig       `yaml:"linkConfig,omitempty" mapstructure:"linkConfig"`
	StandAloneConfig *StandAloneConfig `yaml:"standAloneConfig,omitempty" mapstructure:"standAloneConfig"`
}

// LinkedProxyConfig - represents a configuration of the Linked Proxy (formed using LinkConfig)
type LinkedProxyConfig struct {
	Primary *ManagementServer
	Backup  *ManagementServer
}

// StandAloneProxyConfig - represents Stand Alone Proxy Config (formed using StandAloneConfig)
type StandAloneProxyConfig struct {
	managedArrays     map[string]*StorageArray
	managementServers map[url.URL]*ManagementServer
	proxyCredentials  map[string]*ProxyUser
}

func (proxy *StandAloneProxyConfig) updateProxyCredentials(creds common.Credentials, storageArrayIdentifier string) {
	if proxyUser, ok := proxy.proxyCredentials[creds.UserName]; ok {
		if subtle.ConstantTimeCompare([]byte(creds.Password), []byte(proxyUser.ProxyCredential.Password)) == 1 {
			// Credentials already exist in map
			proxyUser.StorageArrayIdentifiers = utils.AppendIfMissingStringSlice(
				proxyUser.StorageArrayIdentifiers, storageArrayIdentifier)
		}
	} else {
		proxyUser := ProxyUser{
			ProxyCredential:         creds,
			StorageArrayIdentifiers: []string{storageArrayIdentifier},
		}
		proxy.proxyCredentials[creds.UserName] = &proxyUser
	}
}

// GetManagementServers - Returns the list of management servers present in StandAloneProxyConfig
func (proxy *StandAloneProxyConfig) GetManagementServers() []ManagementServer {
	mgmtServers := make([]ManagementServer, 0)
	for _, v := range proxy.managementServers {
		mgmtServers = append(mgmtServers, *v)
	}
	return mgmtServers
}

// GetManagedArraysAndServers returns a list of arrays with their corresponding management servers
func (proxy *StandAloneProxyConfig) GetManagedArraysAndServers() map[string]StorageArrayServer {
	arrayServers := make(map[string]StorageArrayServer)
	for _, server := range proxy.managementServers {
		for _, arrayID := range server.StorageArrayIdentifiers {
			var (
				arrayServer StorageArrayServer
				ok          bool
			)
			if arrayServer, ok = arrayServers[arrayID]; !ok {
				arrayServer = StorageArrayServer{}
				arrayServer.Array = *(proxy.managedArrays[arrayID])
			}
			if server.URL == arrayServer.Array.PrimaryURL {
				arrayServer.PrimaryServer = *(server)
			} else if server.URL == arrayServer.Array.SecondaryURL {
				arrayServer.BackupServer = server
			}
			arrayServers[arrayID] = arrayServer
		}
	}
	return arrayServers
}

// GetStorageArray - Returns a list of storage array given a storage array id
func (proxy *StandAloneProxyConfig) GetStorageArray(storageArrayID string) []StorageArray {
	storageArrays := make([]StorageArray, 0)
	if storageArrayID != "" {
		if storageArray, ok := proxy.managedArrays[storageArrayID]; ok {
			storageArrays = append(storageArrays, *storageArray)
		}
	} else {
		for _, v := range proxy.managedArrays {
			storageArrays = append(storageArrays, *v)
		}
	}
	return storageArrays
}

// ProxyConfig - represents the configuration of Proxy (formed using ProxyConfigMap)
type ProxyConfig struct {
	Mode                  ProxyMode
	Port                  string
	LinkProxyConfig       *LinkedProxyConfig
	StandAloneProxyConfig *StandAloneProxyConfig
}

// ProxyUser - used for storing a proxy user and list of associated storage array identifiers
type ProxyUser struct {
	StorageArrayIdentifiers []string
	ProxyCredential         common.Credentials
}

// ParseConfig - Parses a given proxy config map
func (proxyConfig *ProxyConfig) ParseConfig(proxyConfigMap ProxyConfigMap, k8sUtils k8sutils.UtilsInterface) error {
	proxyMode := proxyConfigMap.Mode
	proxyConfig.Mode = proxyMode
	proxyConfig.Port = proxyConfigMap.Port
	fmt.Printf("ConfigMap: %v\n", proxyConfigMap)

	if proxyMode == StandAlone {
		config := proxyConfigMap.StandAloneConfig
		if config == nil {
			return fmt.Errorf("proxy mode is specified as StandAlone but unable to parse config")
		}
		var proxy StandAloneProxyConfig
		proxy.managedArrays = make(map[string]*StorageArray)
		proxy.managementServers = make(map[url.URL]*ManagementServer)
		proxy.proxyCredentials = make(map[string]*ProxyUser)
		storageArrayIdentifiers := make(map[url.URL][]string)
		ipAddresses := make([]string, 0)
		for _, mgmtServer := range config.ManagementServerConfig {
			ipAddresses = append(ipAddresses, mgmtServer.URL)
		}
		for _, array := range config.StorageArrayConfig {
			if array.PrimaryURL == "" {
				return fmt.Errorf("primary URL not configured for array: %s", array.StorageArrayID)
			}
			if !utils.IsStringInSlice(ipAddresses, array.PrimaryURL) {
				return fmt.Errorf("primary URL: %s for array: %s not present among management URL addresses",
					array.PrimaryURL, array)
			}
			if array.BackupURL != "" {
				if !utils.IsStringInSlice(ipAddresses, array.BackupURL) {
					return fmt.Errorf("backup URL: %s for array: %s is not in the list of management URL addresses. Ignoring it",
						array.BackupURL, array)
				}
			}
			primaryURL, err := url.Parse(array.PrimaryURL)
			if err != nil {
				return err
			}
			backupURL := &url.URL{}
			if array.BackupURL != "" {
				backupURL, err = url.Parse(array.BackupURL)
				if err != nil {
					return err
				}
			}
			proxy.managedArrays[array.StorageArrayID] = &StorageArray{
				StorageArrayIdentifier: array.StorageArrayID,
				PrimaryURL:             *primaryURL,
				SecondaryURL:           *backupURL,
			}
			// adding Primary and Backup URl to storageArrayIdentifier, later to be used in management server
			if _, ok := storageArrayIdentifiers[*primaryURL]; ok {
				storageArrayIdentifiers[*primaryURL] = append(storageArrayIdentifiers[*primaryURL], array.StorageArrayID)
			} else {
				storageArrayIdentifiers[*primaryURL] = []string{array.StorageArrayID}
			}
			if _, ok := storageArrayIdentifiers[*backupURL]; ok {
				storageArrayIdentifiers[*backupURL] = append(storageArrayIdentifiers[*backupURL], array.StorageArrayID)
			} else {
				storageArrayIdentifiers[*backupURL] = []string{array.StorageArrayID}
			}

			// Reading proxy credentials for the array
			if len(array.ProxyCredentialSecrets) > 0 {
				proxy.managedArrays[array.StorageArrayID].ProxyCredentialSecrets = make(map[string]ProxyCredentialSecret)
				for _, secret := range array.ProxyCredentialSecrets {
					proxyCredentials, err := k8sUtils.GetCredentialsFromSecretName(secret)
					if err != nil {
						return err
					}
					proxyCredentialSecret := &ProxyCredentialSecret{
						Credentials:      *proxyCredentials,
						CredentialSecret: secret,
					}
					proxy.managedArrays[array.StorageArrayID].ProxyCredentialSecrets[secret] = *proxyCredentialSecret
					proxy.updateProxyCredentials(*proxyCredentials, array.StorageArrayID)
				}
			}
		}
		for _, managementServer := range config.ManagementServerConfig {
			var arrayCredentials common.Credentials
			if managementServer.ArrayCredentialSecret != "" {
				credentials, err := k8sUtils.GetCredentialsFromSecretName(managementServer.ArrayCredentialSecret)
				if err != nil {
					return err
				}
				arrayCredentials = *credentials
			}
			mgmtURL, err := url.Parse(managementServer.URL)
			if err != nil {
				return err
			}
			var certFile string
			if managementServer.CertSecret != "" {
				certFile, err = k8sUtils.GetCertFileFromSecretName(managementServer.CertSecret)
				if err != nil {
					return err
				}
			}
			proxy.managementServers[*mgmtURL] = &ManagementServer{
				URL:                       *mgmtURL,
				StorageArrayIdentifiers:   storageArrayIdentifiers[*mgmtURL],
				SkipCertificateValidation: managementServer.SkipCertificateValidation,
				CertFile:                  certFile,
				CertSecret:                managementServer.CertSecret,
				Credentials:               arrayCredentials,
				CredentialSecret:          managementServer.ArrayCredentialSecret,
				Limits:                    managementServer.Limits,
			}
		}
		proxyConfig.StandAloneProxyConfig = &proxy
	} else {
		return fmt.Errorf("unknown proxy mode: %s specified", string(proxyMode))
	}
	if proxyConfig.LinkProxyConfig == nil && proxyConfig.StandAloneProxyConfig == nil {
		return fmt.Errorf("no configuration provided for the proxy")
	}
	return nil
}

// NewProxyConfig - returns a new proxy config given a proxy config map
func NewProxyConfig(configMap *ProxyConfigMap, k8sUtils k8sutils.UtilsInterface) (*ProxyConfig, error) {
	var proxyConfig ProxyConfig
	err := proxyConfig.ParseConfig(*configMap, k8sUtils)
	if err != nil {
		return nil, err
	}
	return &proxyConfig, nil
}

// ReadConfig - uses viper to read the config from the config map
func ReadConfig(configFile string) (*ProxyConfigMap, error) {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	err := v.ReadInConfig()
	if err != nil {
		return nil, err
	}
	var configMap ProxyConfigMap
	err = v.Unmarshal(&configMap)
	if err != nil {
		return nil, err
	}
	return &configMap, nil
}
