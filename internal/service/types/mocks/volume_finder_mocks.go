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

// Source: github.com/dell/csm-metrics-powermax/internal/service/types (interfaces: VolumeFinder)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
)

// MockVolumeFinder is a mock of VolumeFinder interface.
type MockVolumeFinder struct {
	ctrl     *gomock.Controller
	recorder *MockVolumeFinderMockRecorder
}

// MockVolumeFinderMockRecorder is the mock recorder for MockVolumeFinder.
type MockVolumeFinderMockRecorder struct {
	mock *MockVolumeFinder
}

// NewMockVolumeFinder creates a new mock instance.
func NewMockVolumeFinder(ctrl *gomock.Controller) *MockVolumeFinder {
	mock := &MockVolumeFinder{ctrl: ctrl}
	mock.recorder = &MockVolumeFinderMockRecorder{mock}
	return mock
}

// // EXPECT returns an object that allows the caller to indicate expected use.
// func (m *MockVolumeFinder) EXPECT() *MockVolumeFinderMockRecorder {
// 	return m.recorder
// }

// // GetPersistentVolumes mocks base method.
// func (m *MockVolumeFinder) GetPersistentVolumes(arg0 context.Context) ([]k8s.VolumeInfo, error) {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetPersistentVolumes", arg0)
// 	ret0, _ := ret[0].([]k8s.VolumeInfo)
// 	ret1, _ := ret[1].(error)
// 	return ret0, ret1
// }

// // GetPersistentVolumes indicates an expected call of GetPersistentVolumes.
// func (mr *MockVolumeFinderMockRecorder) GetPersistentVolumes(arg0 interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPersistentVolumes", reflect.TypeOf((*MockVolumeFinder)(nil).GetPersistentVolumes), arg0)
// }
