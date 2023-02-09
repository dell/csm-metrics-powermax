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

package entrypoint_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/service/types"
	mocks "github.com/dell/csm-metrics-powermax/internal/service/types/mocks"
	exportermocks "github.com/dell/csm-metrics-powermax/opentelemetry/exporters/mocks"

	"github.com/sirupsen/logrus"

	"github.com/dell/csm-metrics-powermax/internal/entrypoint"
	otlexporters "github.com/dell/csm-metrics-powermax/opentelemetry/exporters"
	"github.com/golang/mock/gomock"
)

func Test_Run(t *testing.T) {

	tests := map[string]func(t *testing.T) (expectError bool, config *entrypoint.Config, exporter otlexporters.Otlexporter, pScaleSvc types.Service, prevConfigValidationFunc func(*entrypoint.Config) error, ctrl *gomock.Controller, validatingConfig bool){
		"success": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)

			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection("karavi-metrics-powermax", "karavi").Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)

			config := &entrypoint.Config{
				LeaderElector: leaderElector,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).Return(nil)
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().ExportCapacityMetrics(gomock.Any()).AnyTimes()
			svc.EXPECT().ExportPerformanceMetrics(gomock.Any()).AnyTimes()

			return false, config, e, svc, prevConfigValidationFunc, ctrl, false
		},
		"error with invalid performance ticker interval": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			leaderElector := mocks.NewMockLeaderElector(ctrl)
			clients := make(map[string]types.PowerMaxClient)
			clients["test"] = mocks.NewMockPowerMaxClient(ctrl)
			config := &entrypoint.Config{
				CapacityMetricsEnabled:    true,
				PerformanceMetricsEnabled: true,
				LeaderElector:             leaderElector,
				CapacityTickInterval:      100 * time.Second,
				PerformanceTickInterval:   0 * time.Millisecond,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			e := exportermocks.NewMockOtlexporter(ctrl)
			svc := mocks.NewMockService(ctrl)

			return true, config, e, svc, prevConfigValidationFunc, ctrl, true
		},
		"error with invalid capacity ticker interval": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			leaderElector := mocks.NewMockLeaderElector(ctrl)
			clients := make(map[string]types.PowerMaxClient)
			clients["test"] = mocks.NewMockPowerMaxClient(ctrl)
			config := &entrypoint.Config{
				CapacityMetricsEnabled:    true,
				PerformanceMetricsEnabled: true,
				LeaderElector:             leaderElector,
				CapacityTickInterval:      0 * time.Millisecond,
				PerformanceTickInterval:   0 * time.Millisecond,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			e := exportermocks.NewMockOtlexporter(ctrl)
			svc := mocks.NewMockService(ctrl)

			return true, config, e, svc, prevConfigValidationFunc, ctrl, true
		},
		"success with capacity false enable ": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)
			clients := make(map[string]types.PowerMaxClient)
			clients["test"] = mocks.NewMockPowerMaxClient(ctrl)
			config := &entrypoint.Config{
				CapacityMetricsEnabled:    false,
				PerformanceMetricsEnabled: true,
				LeaderElector:             leaderElector,
				CapacityTickInterval:      100 * time.Millisecond,
				PerformanceTickInterval:   100 * time.Millisecond,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).Return(nil)
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().ExportCapacityMetrics(gomock.Any()).AnyTimes()
			svc.EXPECT().ExportPerformanceMetrics(gomock.Any()).AnyTimes()

			return false, config, e, svc, prevConfigValidationFunc, ctrl, true
		},
		"success with performance false enable ": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)
			clients := make(map[string]types.PowerMaxClient)
			clients["test"] = mocks.NewMockPowerMaxClient(ctrl)
			config := &entrypoint.Config{
				CapacityMetricsEnabled:    true,
				PerformanceMetricsEnabled: false,
				LeaderElector:             leaderElector,
				CapacityTickInterval:      100 * time.Millisecond,
				PerformanceTickInterval:   100 * time.Millisecond,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).Return(nil)
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().ExportCapacityMetrics(gomock.Any()).AnyTimes()
			svc.EXPECT().ExportPerformanceMetrics(gomock.Any()).AnyTimes()
			return false, config, e, svc, prevConfigValidationFunc, ctrl, true
		},
		"error nil config": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			e := exportermocks.NewMockOtlexporter(ctrl)

			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			svc := mocks.NewMockService(ctrl)

			return true, nil, e, svc, prevConfigValidationFunc, ctrl, true
		},
		"error initializing exporter": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)

			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)

			config := &entrypoint.Config{
				LeaderElector: leaderElector,
			}

			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).Return(fmt.Errorf("An error occurred while initializing the exporter"))
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)

			return true, config, e, svc, prevConfigValidationFunc, ctrl, false
		},
		"success even if leader is false": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)

			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection("karavi-metrics-powermax", "karavi").Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(false)

			config := &entrypoint.Config{
				LeaderElector: leaderElector,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).Return(nil)
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)

			return false, config, e, svc, prevConfigValidationFunc, ctrl, false
		},
		"success using TLS": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)

			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection("karavi-metrics-powermax", "karavi").Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)

			config := &entrypoint.Config{
				LeaderElector:     leaderElector,
				CollectorCertPath: "testdata/test-cert.crt",
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).Return(nil)
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().ExportCapacityMetrics(gomock.Any()).AnyTimes()
			svc.EXPECT().ExportPerformanceMetrics(gomock.Any()).AnyTimes()

			return false, config, e, svc, prevConfigValidationFunc, ctrl, false
		},
		"error reading certificate": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)

			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection("karavi-metrics-powermax", "karavi").AnyTimes().Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)

			config := &entrypoint.Config{
				LeaderElector:     leaderElector,
				CollectorCertPath: "testdata/bad-cert.crt",
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			e.EXPECT().StopExporter().Return(nil)

			svc := mocks.NewMockService(ctrl)

			return true, config, e, svc, prevConfigValidationFunc, ctrl, false
		},
		"success with LivenessProbeTick": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)

			config := &entrypoint.Config{
				CapacityMetricsEnabled:    false,
				PerformanceMetricsEnabled: false,
				LeaderElector:             leaderElector,
				CapacityTickInterval:      100 * time.Millisecond,
				PerformanceTickInterval:   100 * time.Millisecond,
				LivenessProbeTickInterval: 100 * time.Millisecond,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			e.EXPECT().StopExporter().AnyTimes().Return(nil)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().Authenticate(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client: c,
			}

			clients["000197902599"] = append(clients["000197902599"], array)
			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().GetPowerMaxClients().AnyTimes().Return(clients)
			svc.EXPECT().GetLogger().AnyTimes().Return(logrus.New())

			return false, config, e, svc, prevConfigValidationFunc, ctrl, true
		},
		"success with LivenessProbeTick unauthenticated": func(*testing.T) (bool, *entrypoint.Config, otlexporters.Otlexporter, types.Service, func(*entrypoint.Config) error, *gomock.Controller, bool) {
			ctrl := gomock.NewController(t)
			leaderElector := mocks.NewMockLeaderElector(ctrl)
			leaderElector.EXPECT().InitLeaderElection(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			leaderElector.EXPECT().IsLeader().AnyTimes().Return(true)

			config := &entrypoint.Config{
				CapacityMetricsEnabled:    false,
				PerformanceMetricsEnabled: false,
				LeaderElector:             leaderElector,
				CapacityTickInterval:      100 * time.Millisecond,
				PerformanceTickInterval:   100 * time.Millisecond,
				LivenessProbeTickInterval: 100 * time.Millisecond,
			}
			prevConfigValidationFunc := entrypoint.ConfigValidatorFunc
			entrypoint.ConfigValidatorFunc = noCheckConfig

			e := exportermocks.NewMockOtlexporter(ctrl)
			e.EXPECT().InitExporter(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
			e.EXPECT().StopExporter().AnyTimes().Return(nil)

			c := mocks.NewMockPowerMaxClient(ctrl)
			c.EXPECT().Authenticate(gomock.Any(), gomock.Any()).AnyTimes().Return(fmt.Errorf("unauthenticated"))

			clients := make(map[string][]types.PowerMaxArray)
			array := types.PowerMaxArray{
				Client: c,
			}

			clients["000197902599"] = append(clients["000197902599"], array)
			svc := mocks.NewMockService(ctrl)
			svc.EXPECT().GetPowerMaxClients().AnyTimes().Return(clients)
			svc.EXPECT().GetLogger().AnyTimes().Return(logrus.New())

			return false, config, e, svc, prevConfigValidationFunc, ctrl, true
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			expectError, config, exporter, svc, prevConfValidation, ctrl, validateConfig := test(t)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			if config != nil {
				config.Logger = logrus.New()
				if !validateConfig {
					// The configuration is not nil and the test is not attempting to validate the configuration.
					// In this case, we can use smaller intervals for testing purposes.
					config.CapacityTickInterval = 100 * time.Millisecond
					config.PerformanceTickInterval = 100 * time.Millisecond
				}
			}
			err := entrypoint.Run(ctx, config, exporter, svc)
			errorOccurred := err != nil
			if expectError != errorOccurred {
				t.Errorf("Unexpected result from test \"%v\": wanted error (%v), but got (%v)", name, expectError, errorOccurred)
			}
			entrypoint.ConfigValidatorFunc = prevConfValidation
			ctrl.Finish()
		})
	}
}

func noCheckConfig(_ *entrypoint.Config) error {
	return nil
}
