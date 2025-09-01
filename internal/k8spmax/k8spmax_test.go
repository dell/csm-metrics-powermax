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
 Copyright (c) 2022-2025 Dell Inc. or its subsidiaries. All Rights Reserved.

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

package k8spmax_test

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	revcommon "github.com/dell/csi-powermax/csireverseproxy/v2/pkg/common"
	"github.com/dell/csi-powermax/csireverseproxy/v2/pkg/k8smock"
	"github.com/dell/csi-powermax/csireverseproxy/v2/pkg/k8sutils"
	"github.com/dell/csm-metrics-powermax/internal/k8spmax"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"github.com/gorilla/mux"
)

func Test_Run_Unauthorized(t *testing.T) {
	mockUtils := k8smock.Init()
	mockUtils.CreateNewCredentialSecret("powermax-creds")

	tests := map[string]func(t *testing.T) (filePath string, k8sUtils k8sutils.UtilsInterface, expectError bool){
		"failed with unauthorized user": func(*testing.T) (string, k8sutils.UtilsInterface, bool) {
			return "testdata/sample-config-default.yaml", mockUtils, false
		},
	}

	handler := getHandler(getUnauthorizedRouter())
	server := httptest.NewTLSServer(handler)
	defer server.Close()

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger := logrus.New()
			logger.Info(test(t))
			filePath, k8sUtils, expectError := test(t)

			original := replaceEnpoints(filePath, server.URL)

			clusters, err := k8spmax.GetPowerMaxArrays(context.Background(), k8sUtils, filePath, logger)

			if expectError {
				assert.Nil(t, clusters)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, clusters)
				assert.Nil(t, err)
			}
			os.WriteFile(filePath, original, 0o600)
		})
	}
}

func Test_GetK8sUtils(t *testing.T) {
	assert.Nil(t, k8spmax.GetK8sUtils())
}

func Test_InitK8sUtils(t *testing.T) {
	os.Setenv("HOME", "")
	os.Setenv("X_CSI_KUBECONFIG_PATH", "../k8s/testdata/")
	callback := func(_ k8sutils.UtilsInterface, _ *corev1.Secret) {}
	_, err := k8spmax.InitK8sUtils(logrus.New(), callback, false)
	assert.Nil(t, err)
}

func TestGetPowerMaxArrays(t *testing.T) {
	server := createServer()
	defer server.Close()

	mockUtils := k8smock.Init()
	mockUtils.CreateNewCredentialSecret("powermax-creds")

	testCases := []struct {
		name                   string
		k8sUtils               k8sutils.UtilsInterface
		filePath               string
		logger                 *logrus.Logger
		expectedPowerMaxArrays map[string][]metrictypes.PowerMaxArray
		useSecret              bool
		expectedError          error
	}{
		{
			name:     "Success: with secret file",
			k8sUtils: &k8sutils.K8sUtils{},
			filePath: "./testdata/secret-config.yaml",
			logger:   logrus.New(),
			expectedPowerMaxArrays: map[string][]metrictypes.PowerMaxArray{
				"000000000001": {{StorageArrayID: "000000000001", Endpoint: server.URL}, {StorageArrayID: "000000000001", Endpoint: server.URL}},
				"000000000002": {{StorageArrayID: "000000000002", Endpoint: server.URL}, {StorageArrayID: "000000000002", Endpoint: server.URL}},
			},
			useSecret:     true,
			expectedError: nil,
		},
		{
			name:                   "Failed: unable to unmarshal secret file",
			k8sUtils:               &k8sutils.K8sUtils{},
			filePath:               "./testdata/invalid-format.yaml",
			logger:                 logrus.New(),
			expectedPowerMaxArrays: nil,
			useSecret:              true,
			expectedError:          errors.New("cannot unmarshal !!str `invalid...` into map[string]interface {}"),
		},
		{
			name:                   "Failed: invalid secret file",
			k8sUtils:               &k8sutils.K8sUtils{},
			filePath:               "./testdata/invalid-secret-config.yaml",
			logger:                 logrus.New(),
			expectedPowerMaxArrays: nil,
			useSecret:              true,
			expectedError:          errors.New("primary endpoint not configured"),
		},

		{
			name:     "Success: with config map",
			k8sUtils: mockUtils,
			filePath: "./testdata/sample-config-default.yaml",
			logger:   logrus.New(),
			expectedPowerMaxArrays: map[string][]metrictypes.PowerMaxArray{
				"00012345678": {{StorageArrayID: "00012345678", Endpoint: server.URL}, {StorageArrayID: "00012345678", Endpoint: server.URL}},
			},
			useSecret:     false,
			expectedError: nil,
		},
		{
			name:                   "Failed: nil k8sUtils",
			k8sUtils:               nil,
			filePath:               "./testdata/sample-config-default.yaml",
			logger:                 logrus.New(),
			expectedPowerMaxArrays: nil,
			useSecret:              false,
			expectedError:          errors.New("k8sUtils is nil"),
		},
		{
			name:                   "Failed: configMap cannot unmarshall",
			k8sUtils:               mockUtils,
			filePath:               "./testdata/invalid-format.yaml",
			logger:                 logrus.New(),
			expectedPowerMaxArrays: nil,
			useSecret:              false,
			expectedError:          errors.New("cannot unmarshal !!str `invalid...` into map[string]interface {}"),
		},
		{
			name:                   "Failed: configMap connection failed",
			k8sUtils:               mockUtils,
			filePath:               "./testdata/connection-failed.yaml",
			logger:                 logrus.New(),
			expectedPowerMaxArrays: nil,
			useSecret:              false,
			expectedError:          errors.New("not present among management URL addresses"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := replaceEnpoints(tc.filePath, server.URL)

			ctx := context.Background()
			setReverseProxyUseSecret(tc.useSecret)
			if tc.useSecret {
				setEnv(revcommon.EnvSecretFilePath, tc.filePath)
			}
			powerMaxArrays, err := k8spmax.GetPowerMaxArrays(ctx, tc.k8sUtils, tc.filePath, tc.logger)
			if err != nil {
				if !strings.Contains(err.Error(), tc.expectedError.Error()) {
					t.Errorf("Expected error: %v, but got: %v", tc.expectedError, err)
				}
			}

			if len(powerMaxArrays) != len(tc.expectedPowerMaxArrays) {
				t.Errorf("Expected powerMaxArrays: %v, but got: %v", tc.expectedPowerMaxArrays, powerMaxArrays)
			}
			os.WriteFile(tc.filePath, original, 0o600)
		})
	}
}

// getHandler returns an http.Handler that
func getHandler(router http.Handler) http.Handler {
	handler := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			log.Printf("handler called: %s %s", r.Method, r.URL)
			router.ServeHTTP(w, r)
		})

	return handler
}

// getRouter return a valid REST response
func getRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/univmax/restapi/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("{\"version\": \"T10.0.0.1311\"}"))
	})
	return router
}

// getRouter return an invalid REST response
func getUnauthorizedRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/univmax/restapi/version", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("<html><head><title>Error</title></head><body>Unauthorized</body></html>"))
	})
	return router
}

func setEnv(key, value string) error {
	err := os.Setenv(key, value)
	if err != nil {
		return err
	}
	return nil
}

func setReverseProxyUseSecret(value bool) error {
	if value {
		return setEnv(revcommon.EnvReverseProxyUseSecret, "true")
	}
	return setEnv(revcommon.EnvReverseProxyUseSecret, "false")
}

func createServer() *httptest.Server {
	handler := getHandler(getRouter())
	return httptest.NewTLSServer(handler)
}

func replaceEnpoints(configFile, endpoint string) []byte {
	fileContentBytes, _ := os.ReadFile(configFile)
	newContent := strings.ReplaceAll(string(fileContentBytes), "[REPLACE_ENDPOINT]", endpoint)
	os.WriteFile(configFile, []byte(newContent), 0o600)
	return fileContentBytes
}
