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
