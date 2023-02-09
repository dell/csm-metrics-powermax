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

// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/dell/csm-metrics-powermax/internal/service/types (interfaces: PowerMaxClient)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	pmax "github.com/dell/gopowermax/v2"
	v100 "github.com/dell/gopowermax/v2/types/v100"
	gomock "github.com/golang/mock/gomock"
)

// MockPowerMaxClient is a mock of PowerMaxClient interface.
type MockPowerMaxClient struct {
	ctrl     *gomock.Controller
	recorder *MockPowerMaxClientMockRecorder
}

// MockPowerMaxClientMockRecorder is the mock recorder for MockPowerMaxClient.
type MockPowerMaxClientMockRecorder struct {
	mock *MockPowerMaxClient
}

// NewMockPowerMaxClient creates a new mock instance.
func NewMockPowerMaxClient(ctrl *gomock.Controller) *MockPowerMaxClient {
	mock := &MockPowerMaxClient{ctrl: ctrl}
	mock.recorder = &MockPowerMaxClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPowerMaxClient) EXPECT() *MockPowerMaxClientMockRecorder {
	return m.recorder
}

// Authenticate mocks base method.
func (m *MockPowerMaxClient) Authenticate(arg0 context.Context, arg1 *pmax.ConfigConnect) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Authenticate", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Authenticate indicates an expected call of Authenticate.
func (mr *MockPowerMaxClientMockRecorder) Authenticate(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Authenticate", reflect.TypeOf((*MockPowerMaxClient)(nil).Authenticate), arg0, arg1)
}

// GetArrayPerfKeys mocks base method.
func (m *MockPowerMaxClient) GetArrayPerfKeys(arg0 context.Context) (*v100.ArrayKeysResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetArrayPerfKeys", arg0)
	ret0, _ := ret[0].(*v100.ArrayKeysResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetArrayPerfKeys indicates an expected call of GetArrayPerfKeys.
func (mr *MockPowerMaxClientMockRecorder) GetArrayPerfKeys(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetArrayPerfKeys", reflect.TypeOf((*MockPowerMaxClient)(nil).GetArrayPerfKeys), arg0)
}

// GetStorageGroup mocks base method.
func (m *MockPowerMaxClient) GetStorageGroup(arg0 context.Context, arg1, arg2 string) (*v100.StorageGroup, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageGroup", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v100.StorageGroup)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageGroup indicates an expected call of GetStorageGroup.
func (mr *MockPowerMaxClientMockRecorder) GetStorageGroup(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageGroup", reflect.TypeOf((*MockPowerMaxClient)(nil).GetStorageGroup), arg0, arg1, arg2)
}

// GetStorageGroupMetrics mocks base method.
func (m *MockPowerMaxClient) GetStorageGroupMetrics(arg0 context.Context, arg1, arg2 string, arg3 []string, arg4, arg5 int64) (*v100.StorageGroupMetricsIterator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageGroupMetrics", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(*v100.StorageGroupMetricsIterator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageGroupMetrics indicates an expected call of GetStorageGroupMetrics.
func (mr *MockPowerMaxClientMockRecorder) GetStorageGroupMetrics(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageGroupMetrics", reflect.TypeOf((*MockPowerMaxClient)(nil).GetStorageGroupMetrics), arg0, arg1, arg2, arg3, arg4, arg5)
}

// GetStorageGroupPerfKeys mocks base method.
func (m *MockPowerMaxClient) GetStorageGroupPerfKeys(arg0 context.Context, arg1 string) (*v100.StorageGroupKeysResult, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStorageGroupPerfKeys", arg0, arg1)
	ret0, _ := ret[0].(*v100.StorageGroupKeysResult)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetStorageGroupPerfKeys indicates an expected call of GetStorageGroupPerfKeys.
func (mr *MockPowerMaxClientMockRecorder) GetStorageGroupPerfKeys(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStorageGroupPerfKeys", reflect.TypeOf((*MockPowerMaxClient)(nil).GetStorageGroupPerfKeys), arg0, arg1)
}

// GetVolumeByID mocks base method.
func (m *MockPowerMaxClient) GetVolumeByID(arg0 context.Context, arg1, arg2 string) (*v100.Volume, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolumeByID", arg0, arg1, arg2)
	ret0, _ := ret[0].(*v100.Volume)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolumeByID indicates an expected call of GetVolumeByID.
func (mr *MockPowerMaxClientMockRecorder) GetVolumeByID(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolumeByID", reflect.TypeOf((*MockPowerMaxClient)(nil).GetVolumeByID), arg0, arg1, arg2)
}

// GetVolumesMetrics mocks base method.
func (m *MockPowerMaxClient) GetVolumesMetrics(arg0 context.Context, arg1, arg2 string, arg3 []string, arg4, arg5 int64) (*v100.VolumeMetricsIterator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVolumesMetrics", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(*v100.VolumeMetricsIterator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetVolumesMetrics indicates an expected call of GetVolumesMetrics.
func (mr *MockPowerMaxClientMockRecorder) GetVolumesMetrics(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVolumesMetrics", reflect.TypeOf((*MockPowerMaxClient)(nil).GetVolumesMetrics), arg0, arg1, arg2, arg3, arg4, arg5)
}
