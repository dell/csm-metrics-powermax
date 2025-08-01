// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dell/csm-metrics-powermax/internal/k8s (interfaces: StorageClassGetter)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	v1 "k8s.io/api/storage/v1"
)

// MockStorageClassGetter is a mock of StorageClassGetter interface.
type MockStorageClassGetter struct {
	ctrl     *gomock.Controller
	recorder *MockStorageClassGetterMockRecorder
}

// MockStorageClassGetterMockRecorder is the mock recorder for MockStorageClassGetter.
type MockStorageClassGetterMockRecorder struct {
	mock *MockStorageClassGetter
}

// NewMockStorageClassGetter creates a new mock instance.
func NewMockStorageClassGetter(ctrl *gomock.Controller) *MockStorageClassGetter {
	mock := &MockStorageClassGetter{ctrl: ctrl}
	mock.recorder = &MockStorageClassGetterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageClassGetter) EXPECT() *MockStorageClassGetterMockRecorder {
	return m.recorder
}

// GetStorageClasses mocks base method.
func (m *MockStorageClassGetter) GetStorageClasses() (*v1.StorageClassList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageClasses")
	ret0, _ := ret[0].(*v1.StorageClassList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageClasses indicates an expected call of GetStorageClasses.
func (mr *MockStorageClassGetterMockRecorder) GetStorageClasses() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageClasses", reflect.TypeOf((*MockStorageClassGetter)(nil).GetStorageClasses))
}
