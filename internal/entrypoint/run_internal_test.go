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

package entrypoint

import (
	"context"
	"fmt"
	"maps"
	"runtime"
	"sync"
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/dell/csm-metrics-powermax/internal/service/metrictypes"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type powerMaxFailureSvc struct {
	mu      sync.RWMutex
	obs     interface{}
	clients map[string][]metrictypes.PowerMaxArray
}

func (s *powerMaxFailureSvc) GetPowerMaxClients() map[string][]metrictypes.PowerMaxArray {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return maps.Clone(s.clients)
}

func (s *powerMaxFailureSvc) setClients(clients map[string][]metrictypes.PowerMaxArray) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients = maps.Clone(clients)
}
func (s *powerMaxFailureSvc) GetMetricsRecorder() metrictypes.MetricsRecorder { return nil }
func (s *powerMaxFailureSvc) GetMaxPowerMaxConnections() int                  { return 0 }
func (s *powerMaxFailureSvc) GetVolumeFinder() metrictypes.VolumeFinder       { return nil }
func (s *powerMaxFailureSvc) ExportCapacityMetrics(context.Context)           {}
func (s *powerMaxFailureSvc) ExportPerformanceMetrics(context.Context)        {}
func (s *powerMaxFailureSvc) ExportTopologyMetrics(context.Context)           {}
func (s *powerMaxFailureSvc) GetObsInstrumenter() interface{}                 { return s.obs }

func gatherPowerMaxMetric(t *testing.T, reg prometheus.Gatherer, name string) *dto.MetricFamily {
	t.Helper()
	mfs, err := reg.Gather()
	require.NoError(t, err)
	for _, mf := range mfs {
		if mf.GetName() == name {
			return mf
		}
	}
	return nil
}

func counterPowerMaxObs(mf *dto.MetricFamily, labels map[string]string) (float64, bool) {
	for _, m := range mf.GetMetric() {
		got := map[string]string{}
		for _, lp := range m.GetLabel() {
			got[lp.GetName()] = lp.GetValue()
		}
		match := true
		for k, v := range labels {
			if got[k] != v {
				match = false
				break
			}
		}
		if match {
			return m.GetCounter().GetValue(), true
		}
	}
	return 0, false
}

func TestRecordPowerMaxExportFailure_UsesCurrentClientMap(t *testing.T) {
	reg := prometheus.NewRegistry()
	inst := service.NewPMAXObsInstrumenter(reg)
	svc := &powerMaxFailureSvc{
		obs: inst,
		clients: map[string][]metrictypes.PowerMaxArray{
			"000197902573": {{}},
		},
	}

	recordPowerMaxExportFailure(svc)
	svc.setClients(map[string][]metrictypes.PowerMaxArray{
		"000197902574": {{}},
	})
	recordPowerMaxExportFailure(svc)

	mf := gatherPowerMaxMetric(t, reg, "dell_csm_obs_export_success_total")
	require.NotNil(t, mf)

	count, ok := counterPowerMaxObs(mf, map[string]string{"module": "metrics-powermax", "array_id": "000197902573", "status": "failure"})
	require.True(t, ok)
	assert.Equal(t, 1.0, count)

	count, ok = counterPowerMaxObs(mf, map[string]string{"module": "metrics-powermax", "array_id": "000197902574", "status": "failure"})
	require.True(t, ok)
	assert.Equal(t, 1.0, count)
}

func TestRecordPowerMaxExportFailure_IsSafeDuringClientUpdates(_ *testing.T) {
	reg := prometheus.NewRegistry()
	inst := service.NewPMAXObsInstrumenter(reg)
	svc := &powerMaxFailureSvc{
		obs: inst,
		clients: map[string][]metrictypes.PowerMaxArray{
			"000197902573": {{}},
		},
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			recordPowerMaxExportFailure(svc)
			runtime.Gosched()
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			svc.setClients(map[string][]metrictypes.PowerMaxArray{
				fmt.Sprintf("0001979025%03d", i%10): {{}},
			})
			runtime.Gosched()
		}
	}()
	wg.Wait()
}
