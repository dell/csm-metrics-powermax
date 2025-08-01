// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dell/csm-metrics-powermax/opentelemetry/exporters (interfaces: Otlexporter)

// Package exportermocks is a generated GoMock package.
package exportermocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	otlpmetricgrpc "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
)

// MockOtlexporter is a mock of Otlexporter interface.
type MockOtlexporter struct {
	ctrl     *gomock.Controller
	recorder *MockOtlexporterMockRecorder
}

// MockOtlexporterMockRecorder is the mock recorder for MockOtlexporter.
type MockOtlexporterMockRecorder struct {
	mock *MockOtlexporter
}

// NewMockOtlexporter creates a new mock instance.
func NewMockOtlexporter(ctrl *gomock.Controller) *MockOtlexporter {
	mock := &MockOtlexporter{ctrl: ctrl}
	mock.recorder = &MockOtlexporterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOtlexporter) EXPECT() *MockOtlexporterMockRecorder {
	return m.recorder
}

// InitExporter mocks base method.
func (m *MockOtlexporter) InitExporter(arg0 ...otlpmetricgrpc.Option) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range arg0 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "InitExporter", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// InitExporter indicates an expected call of InitExporter.
func (mr *MockOtlexporterMockRecorder) InitExporter(arg0 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitExporter", reflect.TypeOf((*MockOtlexporter)(nil).InitExporter), arg0...)
}

// StopExporter mocks base method.
func (m *MockOtlexporter) StopExporter() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StopExporter")
	ret0, _ := ret[0].(error)
	return ret0
}

// StopExporter indicates an expected call of StopExporter.
func (mr *MockOtlexporterMockRecorder) StopExporter() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StopExporter", reflect.TypeOf((*MockOtlexporter)(nil).StopExporter))
}
