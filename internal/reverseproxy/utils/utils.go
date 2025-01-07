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

package utils

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	common "github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	log "github.com/sirupsen/logrus"
)

// IsStringInSlice - Returns true if a string is present in a slice
func IsStringInSlice(slice []string, str string) bool {
	for _, el := range slice {
		if el == str {
			return true
		}
	}
	return false
}

// AppendIfMissingStringSlice - appends a string to a slice if it is not present
func AppendIfMissingStringSlice(slice []string, str string) []string {
	for _, el := range slice {
		if el == str {
			return slice
		}
	}
	return append(slice, str)
}

// RootDir - returns root directory of the binary
func RootDir() string {
	_, b, _, _ := runtime.Caller(0)
	d := path.Join(path.Dir(b))
	return filepath.Dir(d)
}

var customCertDir = ""
var customConfigDir = ""

// RemoveTempFiles - Removes temporary files created during testing
func RemoveTempFiles() error {
	rootDir := RootDir()
	certDir := common.DefaultCertDirName
	if customCertDir != "" {
		certDir = customCertDir
	}
	fullCertsDir := path.Join(rootDir, certDir)

	configDir := common.TempConfigDir
	if customConfigDir != "" {
		configDir = customConfigDir
	}
	fullConfigDir := path.Join(rootDir, configDir)

	certFiles, err := os.ReadDir(fullCertsDir)
	if err != nil {
		log.Errorf("Failed to list cert files in `%s`", fullCertsDir)
		return err
	}
	configFiles, err := os.ReadDir(fullConfigDir)
	if err != nil {
		log.Errorf("Failed to list config files in `%s`", fullConfigDir)
		return err
	}
	files := append(configFiles, certFiles...)
	for _, file := range files {
		fileName := file.Name()
		var err error
		if strings.Contains(fileName, ".pem") {
			err = removeFile(fullCertsDir + "/" + fileName)
		} else if strings.Contains(fileName, ".yaml") {
			err = removeFile(fullConfigDir + "/" + fileName)
		}
		if err != nil {
			log.Errorf("Failed to remove `%s`. (%s)", fileName, err.Error())
		}
	}
	return nil
}

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func removeFile(filePath string) error {
	if checkFileExists(filePath) {
		return os.Remove(filePath)
	}
	return nil
}
