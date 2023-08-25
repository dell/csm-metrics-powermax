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

package common_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/common"
	"github.com/dell/csm-metrics-powermax/internal/k8sutils"
	"github.com/dell/csm-metrics-powermax/internal/reverseproxy/k8smock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/gorilla/mux"
)

func Test_Run(t *testing.T) {
	mockUtils := k8smock.Init()
	mockUtils.CreateNewCredentialSecret("powermax-creds")

	tests := map[string]func(t *testing.T) (filePath string, k8sUtils k8sutils.UtilsInterface, expectError bool){
		"success with default params": func(*testing.T) (string, k8sutils.UtilsInterface, bool) {
			return "testdata/sample-config-default.yaml", mockUtils, false
		},
		"nil k8sUtils": func(*testing.T) (string, k8sutils.UtilsInterface, bool) {
			return "testdata/sample-config-default.yaml", nil, true
		},
		"file format": func(*testing.T) (string, k8sutils.UtilsInterface, bool) {
			return "testdata/invalid-format.yaml", mockUtils, true
		},
		"connection failed": func(*testing.T) (string, k8sutils.UtilsInterface, bool) {
			return "testdata/connection-failed.yaml", mockUtils, true
		},
	}

	handler := getHandler(getRouter())
	server := httptest.NewTLSServer(handler)
	defer server.Close()
	urls := strings.Split(strings.TrimPrefix(server.URL, "https://"), ":")
	serverIP := urls[0]
	serverPort := urls[1]

	fmt.Println(serverIP)
	fmt.Println(serverPort)

	// mockUtils := k8smock.Init()
	// mockUtils.CreateNewCredentialSecret("powermax-creds")

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger := logrus.New()
			logger.Info(test(t))
			filePath, k8sUtils, expectError := test(t)

			fileContentBytes, _ := os.ReadFile(filePath)

			newContent := strings.ReplaceAll(string(fileContentBytes), "[serverip]", serverIP)
			newContent = strings.ReplaceAll(newContent, "[serverport]", serverPort)
			os.WriteFile(filePath, []byte(newContent), 0o600)

			clusters, err := common.GetPowerMaxArrays(context.Background(), k8sUtils, filePath, logger)

			if expectError {
				assert.Nil(t, clusters)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, clusters)
				assert.Nil(t, err)
			}
			os.WriteFile(filePath, fileContentBytes, 0o600)
		})
	}
}

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
	urls := strings.Split(strings.TrimPrefix(server.URL, "https://"), ":")
	serverIP := urls[0]
	serverPort := urls[1]

	fmt.Println(serverIP)
	fmt.Println(serverPort)

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			logger := logrus.New()
			logger.Info(test(t))
			filePath, k8sUtils, expectError := test(t)

			fileContentBytes, _ := os.ReadFile(filePath)

			newContent := strings.ReplaceAll(string(fileContentBytes), "[serverip]", serverIP)
			newContent = strings.ReplaceAll(newContent, "[serverport]", serverPort)
			os.WriteFile(filePath, []byte(newContent), 0o600)

			clusters, err := common.GetPowerMaxArrays(context.Background(), k8sUtils, filePath, logger)

			if expectError {
				assert.Nil(t, clusters)
				assert.NotNil(t, err)
			} else {
				assert.NotNil(t, clusters)
				assert.Nil(t, err)
			}
			os.WriteFile(filePath, fileContentBytes, 0o600)
		})
	}
}

func Test_GetK8sUtils(t *testing.T) {
	assert.Nil(t, common.GetK8sUtils())
}

func Test_InitK8sUtils(t *testing.T) {
	assert.Nil(t, common.InitK8sUtils(logrus.New(), nil))
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
	router.HandleFunc("/univmax/restapi/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{\"version\": \"T10.0.0.1311\"}"))
	})
	return router
}

// getRouter return an invalid REST response
func getUnauthorizedRouter() http.Handler {
	router := mux.NewRouter()
	router.HandleFunc("/univmax/restapi/version", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("<html><head><title>Error</title></head><body>Unauthorized</body></html>"))
	})
	return router
}
