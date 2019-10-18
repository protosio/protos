// Code generated by MockGen. DO NOT EDIT.
// Source: internal/installer/installer.go

// Package installer is a generated GoMock package.
package installer

import (
	gomock "github.com/golang/mock/gomock"
	http "net/http"
	core "protos/internal/core"
	reflect "reflect"
)

// MockinstallerParent is a mock of installerParent interface
type MockinstallerParent struct {
	ctrl     *gomock.Controller
	recorder *MockinstallerParentMockRecorder
}

// MockinstallerParentMockRecorder is the mock recorder for MockinstallerParent
type MockinstallerParentMockRecorder struct {
	mock *MockinstallerParent
}

// NewMockinstallerParent creates a new mock instance
func NewMockinstallerParent(ctrl *gomock.Controller) *MockinstallerParent {
	mock := &MockinstallerParent{ctrl: ctrl}
	mock.recorder = &MockinstallerParentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockinstallerParent) EXPECT() *MockinstallerParentMockRecorder {
	return m.recorder
}

// getPlatform mocks base method
func (m *MockinstallerParent) getPlatform() core.RuntimePlatform {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getPlatform")
	ret0, _ := ret[0].(core.RuntimePlatform)
	return ret0
}

// getPlatform indicates an expected call of getPlatform
func (mr *MockinstallerParentMockRecorder) getPlatform() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getPlatform", reflect.TypeOf((*MockinstallerParent)(nil).getPlatform))
}

// getTaskManager mocks base method
func (m *MockinstallerParent) getTaskManager() core.TaskManager {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getTaskManager")
	ret0, _ := ret[0].(core.TaskManager)
	return ret0
}

// getTaskManager indicates an expected call of getTaskManager
func (mr *MockinstallerParentMockRecorder) getTaskManager() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getTaskManager", reflect.TypeOf((*MockinstallerParent)(nil).getTaskManager))
}

// MockhttpClient is a mock of httpClient interface
type MockhttpClient struct {
	ctrl     *gomock.Controller
	recorder *MockhttpClientMockRecorder
}

// MockhttpClientMockRecorder is the mock recorder for MockhttpClient
type MockhttpClientMockRecorder struct {
	mock *MockhttpClient
}

// NewMockhttpClient creates a new mock instance
func NewMockhttpClient(ctrl *gomock.Controller) *MockhttpClient {
	mock := &MockhttpClient{ctrl: ctrl}
	mock.recorder = &MockhttpClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockhttpClient) EXPECT() *MockhttpClientMockRecorder {
	return m.recorder
}

// Get mocks base method
func (m *MockhttpClient) Get(url string) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", url)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockhttpClientMockRecorder) Get(url interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockhttpClient)(nil).Get), url)
}
