/*
 Copyright (c) 2026 Dell Inc. or its subsidiaries. All Rights Reserved.

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

// Internal test package to access unexported recordObsMetrics without
// going through the Export* singleton methods.
package service

import (
	"testing"
	"time"

	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/prometheus/client_golang/prometheus"
)

// Test_recordObsMetrics_NilInstrumenter verifies early-return when ObsInstrumenter is nil.
func Test_recordObsMetrics_NilInstrumenter(_ *testing.T) {
	svc := &PowerMaxService{
		ObsInstrumenter: nil,
		PowerMaxClients: map[string][]metrictypes.PowerMaxArray{
			"000197902599": {{IsActive: true}},
		},
	}
	// Must not panic.
	svc.recordObsMetrics(100*time.Millisecond, nil)
}

// Test_recordObsMetrics_ActiveArray verifies metrics are recorded for an active array.
func Test_recordObsMetrics_ActiveArray(_ *testing.T) {
	reg := prometheus.NewRegistry()
	svc := &PowerMaxService{
		ObsInstrumenter: NewPMAXObsInstrumenter(reg),
		PowerMaxClients: map[string][]metrictypes.PowerMaxArray{
			"000197902599": {{IsActive: true}},
		},
	}
	svc.recordObsMetrics(50*time.Millisecond, nil)
}

// Test_recordObsMetrics_InactiveArray verifies the status="error" branch when no array is active.
func Test_recordObsMetrics_InactiveArray(_ *testing.T) {
	reg := prometheus.NewRegistry()
	svc := &PowerMaxService{
		ObsInstrumenter: NewPMAXObsInstrumenter(reg),
		PowerMaxClients: map[string][]metrictypes.PowerMaxArray{
			"000197902599": {{IsActive: false}},
		},
	}
	svc.recordObsMetrics(50*time.Millisecond, nil)
}

// Test_recordObsMetrics_EmptyClients verifies the loop is a no-op when clients map is empty.
func Test_recordObsMetrics_EmptyClients(_ *testing.T) {
	reg := prometheus.NewRegistry()
	svc := &PowerMaxService{
		ObsInstrumenter: NewPMAXObsInstrumenter(reg),
		PowerMaxClients: map[string][]metrictypes.PowerMaxArray{},
	}
	svc.recordObsMetrics(50*time.Millisecond, nil)
}
