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

package service_test

import (
	"testing"

	"github.com/dell/csm-metrics-powermax/internal/service"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// gatherPMAXObsMetric gathers metrics from the registry and returns the named MetricFamily.
func gatherPMAXObsMetric(t *testing.T, reg *prometheus.Registry, name string) *dto.MetricFamily {
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

// counterPMAXObs returns the value of the first counter in the MetricFamily matching the given labels.
func counterPMAXObs(mf *dto.MetricFamily, labels map[string]string) (float64, bool) {
	for _, m := range mf.GetMetric() {
		if labelsMatch(m.GetLabel(), labels) {
			return m.GetCounter().GetValue(), true
		}
	}
	return 0, false
}

// gaugePMAXObs returns the value of the first gauge in the MetricFamily matching the given labels.
func gaugePMAXObs(mf *dto.MetricFamily, labels map[string]string) (float64, bool) {
	for _, m := range mf.GetMetric() {
		if labelsMatch(m.GetLabel(), labels) {
			return m.GetGauge().GetValue(), true
		}
	}
	return 0, false
}

// histogramPMAXObsCount returns the sample count of the first histogram matching the given labels.
func histogramPMAXObsCount(mf *dto.MetricFamily, labels map[string]string) (uint64, bool) {
	for _, m := range mf.GetMetric() {
		if labelsMatch(m.GetLabel(), labels) {
			return m.GetHistogram().GetSampleCount(), true
		}
	}
	return 0, false
}

// labelsMatch checks that all expected key/value pairs are present in the label set.
func labelsMatch(got []*dto.LabelPair, want map[string]string) bool {
	index := make(map[string]string, len(got))
	for _, lp := range got {
		index[lp.GetName()] = lp.GetValue()
	}
	for k, v := range want {
		if index[k] != v {
			return false
		}
	}
	return true
}

// U-OBS-PMAX-01: RecordsAllSelfMetrics registers and records all four self-metrics.
func TestPMAXObsInstrumenter_RecordsAllSelfMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	inst := service.NewPMAXObsInstrumenter(reg)

	inst.RecordCollectionRate("000197902573", 2.5)
	inst.RecordExportSuccess("000197902573", "success")
	inst.SetArrayConnectivity("000197902573", true)
	inst.RecordProcessingLatency("000197902573", 0.25)

	labels := map[string]string{"module": "metrics-powermax", "array_id": "000197902573"}

	mfRate := gatherPMAXObsMetric(t, reg, "dell_csm_obs_collection_rate")
	require.NotNil(t, mfRate, "dell_csm_obs_collection_rate must be registered")
	v, ok := gaugePMAXObs(mfRate, labels)
	assert.True(t, ok, "collection_rate metric must have expected labels")
	assert.Equal(t, 2.5, v, "collection rate must match")

	mfExport := gatherPMAXObsMetric(t, reg, "dell_csm_obs_export_success_total")
	require.NotNil(t, mfExport, "dell_csm_obs_export_success_total must be registered")
	exportLabels := map[string]string{"module": "metrics-powermax", "array_id": "000197902573", "status": "success"}
	cv, ok := counterPMAXObs(mfExport, exportLabels)
	assert.True(t, ok, "export_success counter must have expected labels")
	assert.Equal(t, 1.0, cv, "export success counter must be 1 after one call")

	mfConn := gatherPMAXObsMetric(t, reg, "dell_csm_obs_array_connectivity")
	require.NotNil(t, mfConn, "dell_csm_obs_array_connectivity must be registered")
	connVal, ok := gaugePMAXObs(mfConn, labels)
	assert.True(t, ok, "connectivity gauge must have expected labels")
	assert.Equal(t, 1.0, connVal, "connected array must report 1")

	mfLatency := gatherPMAXObsMetric(t, reg, "dell_csm_obs_processing_latency_seconds")
	require.NotNil(t, mfLatency, "dell_csm_obs_processing_latency_seconds must be registered")
	latencyCount, ok := histogramPMAXObsCount(mfLatency, labels)
	assert.True(t, ok, "latency histogram must have expected labels")
	assert.Equal(t, uint64(1), latencyCount, "latency histogram must have one sample")
}

// U-OBS-PMAX-02: SetArrayConnectivity reports 0 for disconnected arrays.
func TestPMAXObsInstrumenter_DisconnectedArray(t *testing.T) {
	reg := prometheus.NewRegistry()
	inst := service.NewPMAXObsInstrumenter(reg)

	inst.SetArrayConnectivity("000197902574", false)

	labels := map[string]string{"module": "metrics-powermax", "array_id": "000197902574"}
	mfConn := gatherPMAXObsMetric(t, reg, "dell_csm_obs_array_connectivity")
	require.NotNil(t, mfConn)
	v, ok := gaugePMAXObs(mfConn, labels)
	assert.True(t, ok)
	assert.Equal(t, 0.0, v, "disconnected array must report 0")
}

// U-OBS-PMAX-03: nil receiver does not panic on any method.
func TestPMAXObsInstrumenter_NilReceiverDoesNotPanic(t *testing.T) {
	var inst *service.PMAXObsInstrumenter

	assert.NotPanics(t, func() { inst.RecordCollectionRate("000197902573", 1.0) })
	assert.NotPanics(t, func() { inst.RecordExportSuccess("000197902573", "success") })
	assert.NotPanics(t, func() { inst.SetArrayConnectivity("000197902573", true) })
	assert.NotPanics(t, func() { inst.RecordProcessingLatency("000197902573", 0.1) })
}
