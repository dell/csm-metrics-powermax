// Source: types.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	k8s "github.com/dell/csm-metrics-powermax/internal/k8s"
	types "github.com/dell/csm-metrics-powermax/internal/service/types"
	gomock "github.com/golang/mock/gomock"
	attribute "go.opentelemetry.io/otel/attribute"
	metric "go.opentelemetry.io/otel/metric"
)

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVolumeFinder) EXPECT() *MockVolumeFinderMockRecorder {
	return m.recorder
}

// GetPersistentVolumes mocks base method.
func (m *MockVolumeFinder) GetPersistentVolumes(arg0 context.Context) ([]k8s.VolumeInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPersistentVolumes", arg0)
	ret0, _ := ret[0].([]k8s.VolumeInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetPersistentVolumes indicates an expected call of GetPersistentVolumes.
func (mr *MockVolumeFinderMockRecorder) GetPersistentVolumes(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPersistentVolumes", reflect.TypeOf((*MockVolumeFinder)(nil).GetPersistentVolumes), arg0)
}

// MockMeterCreator is a mock of MeterCreator interface.
type MockMeterCreator struct {
	ctrl     *gomock.Controller
	recorder *MockMeterCreatorMockRecorder
}

// MockMeterCreatorMockRecorder is the mock recorder for MockMeterCreator.
type MockMeterCreatorMockRecorder struct {
	mock *MockMeterCreator
}

// NewMockMeterCreator creates a new mock instance.
func NewMockMeterCreator(ctrl *gomock.Controller) *MockMeterCreator {
	mock := &MockMeterCreator{ctrl: ctrl}
	mock.recorder = &MockMeterCreatorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMeterCreator) EXPECT() *MockMeterCreatorMockRecorder {
	return m.recorder
}

// MetricProvider mocks base method.
func (m *MockMeterCreator) MetricProvider() metric.Meter {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MetricProvider")
	ret0, _ := ret[0].(metric.Meter)
	return ret0
}

// MetricProvider indicates an expected call of MetricProvider.
func (mr *MockMeterCreatorMockRecorder) MetricProvider() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MetricProvider", reflect.TypeOf((*MockMeterCreator)(nil).MetricProvider))
}

// MockMetricsRecorder is a mock of MetricsRecorder interface.
type MockMetricsRecorder struct {
	ctrl     *gomock.Controller
	recorder *MockMetricsRecorderMockRecorder
}

// MockMetricsRecorderMockRecorder is the mock recorder for MockMetricsRecorder.
type MockMetricsRecorderMockRecorder struct {
	mock *MockMetricsRecorder
}

// NewMockMetricsRecorder creates a new mock instance.
func NewMockMetricsRecorder(ctrl *gomock.Controller) *MockMetricsRecorder {
	mock := &MockMetricsRecorder{ctrl: ctrl}
	mock.recorder = &MockMetricsRecorderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMetricsRecorder) EXPECT() *MockMetricsRecorderMockRecorder {
	return m.recorder
}

// RecordNumericMetrics mocks base method.
func (m *MockMetricsRecorder) RecordNumericMetrics(prefix string, labels []attribute.KeyValue, metric types.VolumeCapacityMetricsRecord) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecordNumericMetrics", prefix, labels, metric)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecordNumericMetrics indicates an expected call of RecordNumericMetrics.
func (mr *MockMetricsRecorderMockRecorder) RecordNumericMetrics(prefix, labels, metric interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecordNumericMetrics", reflect.TypeOf((*MockMetricsRecorder)(nil).RecordNumericMetrics), prefix, labels, metric)
}

// RecordStorageGroupPerfMetrics mocks base method.
func (m *MockMetricsRecorder) RecordStorageGroupPerfMetrics(prefix string, metric types.StorageGroupPerfMetricsRecord) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecordStorageGroupPerfMetrics", prefix, metric)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecordStorageGroupPerfMetrics indicates an expected call of RecordStorageGroupPerfMetrics.
func (mr *MockMetricsRecorderMockRecorder) RecordStorageGroupPerfMetrics(prefix, metric interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecordStorageGroupPerfMetrics", reflect.TypeOf((*MockMetricsRecorder)(nil).RecordStorageGroupPerfMetrics), prefix, metric)
}

// RecordVolPerfMetrics mocks base method.
func (m *MockMetricsRecorder) RecordVolPerfMetrics(prefix string, metric types.VolumePerfMetricsRecord) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecordVolPerfMetrics", prefix, metric)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecordVolPerfMetrics indicates an expected call of RecordVolPerfMetrics.
func (mr *MockMetricsRecorderMockRecorder) RecordVolPerfMetrics(prefix, metric interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecordVolPerfMetrics", reflect.TypeOf((*MockMetricsRecorder)(nil).RecordVolPerfMetrics), prefix, metric)
}
