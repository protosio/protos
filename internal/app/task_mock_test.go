// Code generated by MockGen. DO NOT EDIT.
// Source: internal/app/task.go

// Package app is a generated GoMock package.
package app

import (
	gomock "github.com/golang/mock/gomock"
	core "github.com/protosio/protos/internal/core"
	reflect "reflect"
)

// MocktaskParent is a mock of taskParent interface
type MocktaskParent struct {
	ctrl     *gomock.Controller
	recorder *MocktaskParentMockRecorder
}

// MocktaskParentMockRecorder is the mock recorder for MocktaskParent
type MocktaskParentMockRecorder struct {
	mock *MocktaskParent
}

// NewMocktaskParent creates a new mock instance
func NewMocktaskParent(ctrl *gomock.Controller) *MocktaskParent {
	mock := &MocktaskParent{ctrl: ctrl}
	mock.recorder = &MocktaskParentMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MocktaskParent) EXPECT() *MocktaskParentMockRecorder {
	return m.recorder
}

// createAppForTask mocks base method
func (m *MocktaskParent) createAppForTask(installerID, installerVersion, name string, installerParams map[string]string, installerMetadata core.InstallerMetadata, taskID string) (app, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "createAppForTask", installerID, installerVersion, name, installerParams, installerMetadata, taskID)
	ret0, _ := ret[0].(app)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// createAppForTask indicates an expected call of createAppForTask
func (mr *MocktaskParentMockRecorder) createAppForTask(installerID, installerVersion, name, installerParams, installerMetadata, taskID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "createAppForTask", reflect.TypeOf((*MocktaskParent)(nil).createAppForTask), installerID, installerVersion, name, installerParams, installerMetadata, taskID)
}

// Remove mocks base method
func (m *MocktaskParent) Remove(appID string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove", appID)
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove
func (mr *MocktaskParentMockRecorder) Remove(appID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MocktaskParent)(nil).Remove), appID)
}

// getTaskManager mocks base method
func (m *MocktaskParent) getTaskManager() core.TaskManager {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getTaskManager")
	ret0, _ := ret[0].(core.TaskManager)
	return ret0
}

// getTaskManager indicates an expected call of getTaskManager
func (mr *MocktaskParentMockRecorder) getTaskManager() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getTaskManager", reflect.TypeOf((*MocktaskParent)(nil).getTaskManager))
}

// getAppStore mocks base method
func (m *MocktaskParent) getAppStore() appStore {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "getAppStore")
	ret0, _ := ret[0].(appStore)
	return ret0
}

// getAppStore indicates an expected call of getAppStore
func (mr *MocktaskParentMockRecorder) getAppStore() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "getAppStore", reflect.TypeOf((*MocktaskParent)(nil).getAppStore))
}

// Mockapp is a mock of app interface
type Mockapp struct {
	ctrl     *gomock.Controller
	recorder *MockappMockRecorder
}

// MockappMockRecorder is the mock recorder for Mockapp
type MockappMockRecorder struct {
	mock *Mockapp
}

// NewMockapp creates a new mock instance
func NewMockapp(ctrl *gomock.Controller) *Mockapp {
	mock := &Mockapp{ctrl: ctrl}
	mock.recorder = &MockappMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *Mockapp) EXPECT() *MockappMockRecorder {
	return m.recorder
}

// Start mocks base method
func (m *Mockapp) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockappMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*Mockapp)(nil).Start))
}

// Stop mocks base method
func (m *Mockapp) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (mr *MockappMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*Mockapp)(nil).Stop))
}

// AddTask mocks base method
func (m *Mockapp) AddTask(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddTask", arg0)
}

// AddTask indicates an expected call of AddTask
func (mr *MockappMockRecorder) AddTask(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddTask", reflect.TypeOf((*Mockapp)(nil).AddTask), arg0)
}

// GetID mocks base method
func (m *Mockapp) GetID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetID indicates an expected call of GetID
func (mr *MockappMockRecorder) GetID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetID", reflect.TypeOf((*Mockapp)(nil).GetID))
}

// SetStatus mocks base method
func (m *Mockapp) SetStatus(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetStatus", arg0)
}

// SetStatus indicates an expected call of SetStatus
func (mr *MockappMockRecorder) SetStatus(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStatus", reflect.TypeOf((*Mockapp)(nil).SetStatus), arg0)
}

// StartAsync mocks base method
func (m *Mockapp) StartAsync() core.Task {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartAsync")
	ret0, _ := ret[0].(core.Task)
	return ret0
}

// StartAsync indicates an expected call of StartAsync
func (mr *MockappMockRecorder) StartAsync() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartAsync", reflect.TypeOf((*Mockapp)(nil).StartAsync))
}

// createContainer mocks base method
func (m *Mockapp) createSandbox() (core.PlatformRuntimeUnit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "createContainer")
	ret0, _ := ret[0].(core.PlatformRuntimeUnit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// createContainer indicates an expected call of createContainer
func (mr *MockappMockRecorder) createSandbox() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "createContainer", reflect.TypeOf((*Mockapp)(nil).createContainer))
}
