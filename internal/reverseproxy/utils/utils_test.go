package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	common "github.com/dell/csm-metrics-powermax/internal/reverseproxy/common"
	"github.com/stretchr/testify/assert"
)

func TestIsStringInSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		element  string
		expected bool
	}{
		{
			name:     "Empty slice",
			slice:    []string{},
			element:  "foo",
			expected: false,
		},
		{
			name:     "Slice contains element",
			slice:    []string{"foo", "bar", "baz"},
			element:  "bar",
			expected: true,
		},
		{
			name:     "Slice does not contain element",
			slice:    []string{"foo", "bar", "baz"},
			element:  "qux",
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := IsStringInSlice(test.slice, test.element)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestAppendIfMissingStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		element  string
		expected []string
	}{
		{
			name:     "Empty slice",
			slice:    []string{},
			element:  "foo",
			expected: []string{"foo"},
		},
		{
			name:     "Slice already contains element",
			slice:    []string{"foo", "bar", "baz"},
			element:  "bar",
			expected: []string{"foo", "bar", "baz"},
		},
		{
			name:     "Slice does not contain element",
			slice:    []string{"foo", "bar", "baz"},
			element:  "qux",
			expected: []string{"foo", "bar", "baz", "qux"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := AppendIfMissingStringSlice(test.slice, test.element)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestRootDir(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Successful call",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := RootDir()
			assert.NotEmpty(t, result)
		})
	}
}

func TestRemoveTempFiles(t *testing.T) {
	tests := []struct {
		name            string
		files           []string
		expected        []string
		customCertDir   string
		customConfigDir string
	}{
		{
			name:     "Empty slice",
			files:    []string{},
			expected: []string{},
		},
		{
			name:     "Slice contains files",
			files:    []string{"file1.yaml", "file2.yaml", "file3.yaml"},
			expected: []string{},
		},
		{
			name:     "Slice contains certs",
			files:    []string{"cert1.pem", "cert2.pem", "cert3.pem"},
			expected: []string{},
		},
		{
			name:            "Invalid config dir",
			customConfigDir: "missing-directory",
		},
		{
			name:          "Invalid config dir",
			customCertDir: "missing-directory",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configDirName := filepath.Join(RootDir(), common.TempConfigDir)
			certDirName := filepath.Join(RootDir(), common.DefaultCertDirName)

			for _, file := range test.files {
				var filePath string
				if strings.Contains(file, ".pem") {
					filePath = filepath.Join(certDirName, file)
				} else {
					filePath = filepath.Join(configDirName, file)
				}
				_, err := os.Create(filePath)
				if err != nil {
					t.Fatalf("Failed to create file: %s", err.Error())
				}
			}

			customCertDir = test.customCertDir
			customConfigDir = test.customConfigDir

			err := RemoveTempFiles()

			// if passing a custom directory, we expect an error since the directory doesn't exist
			if test.customConfigDir != "" || test.customCertDir != "" {
				assert.NotNil(t, err)
				return
			}

			if err != nil {
				t.Fatalf("Failed to remove temporary files: %s", err.Error())
			}

			// check config directory
			files, err := os.ReadDir(configDirName)
			if err != nil {
				t.Fatalf("Failed to list files in directory: %s", err.Error())
			}

			for _, file := range files {
				assert.NotContains(t, test.expected, file.Name())
			}

			// check certs directory
			files, err = os.ReadDir(certDirName)
			if err != nil {
				t.Fatalf("Failed to list files in directory: %s", err.Error())
			}

			for _, file := range files {
				assert.NotContains(t, test.expected, file.Name())
			}
		})
	}
}
