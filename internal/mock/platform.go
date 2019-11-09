// Code generated by MockGen. DO NOT EDIT.
// Source: internal/core/platform.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	types "github.com/docker/docker/api/types"
	gomock "github.com/golang/mock/gomock"
	core "github.com/protosio/protos/internal/core"
	util "github.com/protosio/protos/internal/util"
)

// MockRuntimePlatform is a mock of RuntimePlatform interface
type MockRuntimePlatform struct {
	ctrl     *gomock.Controller
	recorder *MockRuntimePlatformMockRecorder
}

// MockRuntimePlatformMockRecorder is the mock recorder for MockRuntimePlatform
type MockRuntimePlatformMockRecorder struct {
	mock *MockRuntimePlatform
}

// NewMockRuntimePlatform creates a new mock instance
func NewMockRuntimePlatform(ctrl *gomock.Controller) *MockRuntimePlatform {
	mock := &MockRuntimePlatform{ctrl: ctrl}
	mock.recorder = &MockRuntimePlatformMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRuntimePlatform) EXPECT() *MockRuntimePlatformMockRecorder {
	return m.recorder
}

// GetSandbox mocks base method
func (m *MockRuntimePlatform) GetSandbox(id string) (core.PlatformRuntimeUnit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSandbox", id)
	ret0, _ := ret[0].(core.PlatformRuntimeUnit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSandbox indicates an expected call of GetSandbox
func (mr *MockRuntimePlatformMockRecorder) GetSandbox(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSandbox", reflect.TypeOf((*MockRuntimePlatform)(nil).GetSandbox), id)
}

// GetAllSandboxes mocks base method
func (m *MockRuntimePlatform) GetAllSandboxes() (map[string]core.PlatformRuntimeUnit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllSandboxes")
	ret0, _ := ret[0].(map[string]core.PlatformRuntimeUnit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllSandboxes indicates an expected call of GetAllSandboxes
func (mr *MockRuntimePlatformMockRecorder) GetAllSandboxes() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllSandboxes", reflect.TypeOf((*MockRuntimePlatform)(nil).GetAllSandboxes))
}

// GetImage mocks base method
func (m *MockRuntimePlatform) GetImage(id string) (types.ImageInspect, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImage", id)
	ret0, _ := ret[0].(types.ImageInspect)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImage indicates an expected call of GetImage
func (mr *MockRuntimePlatformMockRecorder) GetImage(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImage", reflect.TypeOf((*MockRuntimePlatform)(nil).GetImage), id)
}

// GetAllImages mocks base method
func (m *MockRuntimePlatform) GetAllImages() (map[string]types.ImageSummary, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllImages")
	ret0, _ := ret[0].(map[string]types.ImageSummary)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllImages indicates an expected call of GetAllImages
func (mr *MockRuntimePlatformMockRecorder) GetAllImages() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllImages", reflect.TypeOf((*MockRuntimePlatform)(nil).GetAllImages))
}

// GetImageDataPath mocks base method
func (m *MockRuntimePlatform) GetImageDataPath(image types.ImageInspect) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImageDataPath", image)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImageDataPath indicates an expected call of GetImageDataPath
func (mr *MockRuntimePlatformMockRecorder) GetImageDataPath(image interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImageDataPath", reflect.TypeOf((*MockRuntimePlatform)(nil).GetImageDataPath), image)
}

// PullImage mocks base method
func (m *MockRuntimePlatform) PullImage(task core.Task, id, name, version string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "PullImage", task, id, name, version)
	ret0, _ := ret[0].(error)
	return ret0
}

// PullImage indicates an expected call of PullImage
func (mr *MockRuntimePlatformMockRecorder) PullImage(task, id, name, version interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "PullImage", reflect.TypeOf((*MockRuntimePlatform)(nil).PullImage), task, id, name, version)
}

// RemoveImage mocks base method
func (m *MockRuntimePlatform) RemoveImage(id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveImage", id)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveImage indicates an expected call of RemoveImage
func (mr *MockRuntimePlatformMockRecorder) RemoveImage(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveImage", reflect.TypeOf((*MockRuntimePlatform)(nil).RemoveImage), id)
}

// GetOrCreateVolume mocks base method
func (m *MockRuntimePlatform) GetOrCreateVolume(id, path string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOrCreateVolume", id, path)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOrCreateVolume indicates an expected call of GetOrCreateVolume
func (mr *MockRuntimePlatformMockRecorder) GetOrCreateVolume(id, path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOrCreateVolume", reflect.TypeOf((*MockRuntimePlatform)(nil).GetOrCreateVolume), id, path)
}

// RemoveVolume mocks base method
func (m *MockRuntimePlatform) RemoveVolume(id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveVolume", id)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveVolume indicates an expected call of RemoveVolume
func (mr *MockRuntimePlatformMockRecorder) RemoveVolume(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveVolume", reflect.TypeOf((*MockRuntimePlatform)(nil).RemoveVolume), id)
}

// NewSandbox mocks base method
func (m *MockRuntimePlatform) NewSandbox(name, appID, imageID, volumeID, volumeMountPath string, publicPorts []util.Port, installerParams map[string]string) (core.PlatformRuntimeUnit, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NewSandbox", name, appID, imageID, volumeID, volumeMountPath, publicPorts, installerParams)
	ret0, _ := ret[0].(core.PlatformRuntimeUnit)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// NewSandbox indicates an expected call of NewSandbox
func (mr *MockRuntimePlatformMockRecorder) NewSandbox(name, appID, imageID, volumeID, volumeMountPath, publicPorts, installerParams interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NewSandbox", reflect.TypeOf((*MockRuntimePlatform)(nil).NewSandbox), name, appID, imageID, volumeID, volumeMountPath, publicPorts, installerParams)
}

// GetHWStats mocks base method
func (m *MockRuntimePlatform) GetHWStats() (core.HardwareStats, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetHWStats")
	ret0, _ := ret[0].(core.HardwareStats)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHWStats indicates an expected call of GetHWStats
func (mr *MockRuntimePlatformMockRecorder) GetHWStats() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHWStats", reflect.TypeOf((*MockRuntimePlatform)(nil).GetHWStats))
}

// MockPlatformRuntimeUnit is a mock of PlatformRuntimeUnit interface
type MockPlatformRuntimeUnit struct {
	ctrl     *gomock.Controller
	recorder *MockPlatformRuntimeUnitMockRecorder
}

// MockPlatformRuntimeUnitMockRecorder is the mock recorder for MockPlatformRuntimeUnit
type MockPlatformRuntimeUnitMockRecorder struct {
	mock *MockPlatformRuntimeUnit
}

// NewMockPlatformRuntimeUnit creates a new mock instance
func NewMockPlatformRuntimeUnit(ctrl *gomock.Controller) *MockPlatformRuntimeUnit {
	mock := &MockPlatformRuntimeUnit{ctrl: ctrl}
	mock.recorder = &MockPlatformRuntimeUnitMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockPlatformRuntimeUnit) EXPECT() *MockPlatformRuntimeUnitMockRecorder {
	return m.recorder
}

// Start mocks base method
func (m *MockPlatformRuntimeUnit) Start() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Start")
	ret0, _ := ret[0].(error)
	return ret0
}

// Start indicates an expected call of Start
func (mr *MockPlatformRuntimeUnitMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).Start))
}

// Stop mocks base method
func (m *MockPlatformRuntimeUnit) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop
func (mr *MockPlatformRuntimeUnitMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).Stop))
}

// Update mocks base method
func (m *MockPlatformRuntimeUnit) Update() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update")
	ret0, _ := ret[0].(error)
	return ret0
}

// Update indicates an expected call of Update
func (mr *MockPlatformRuntimeUnitMockRecorder) Update() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).Update))
}

// Remove mocks base method
func (m *MockPlatformRuntimeUnit) Remove() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Remove")
	ret0, _ := ret[0].(error)
	return ret0
}

// Remove indicates an expected call of Remove
func (mr *MockPlatformRuntimeUnitMockRecorder) Remove() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Remove", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).Remove))
}

// GetID mocks base method
func (m *MockPlatformRuntimeUnit) GetID() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetID")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetID indicates an expected call of GetID
func (mr *MockPlatformRuntimeUnitMockRecorder) GetID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetID", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).GetID))
}

// GetIP mocks base method
func (m *MockPlatformRuntimeUnit) GetIP() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetIP")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetIP indicates an expected call of GetIP
func (mr *MockPlatformRuntimeUnitMockRecorder) GetIP() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIP", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).GetIP))
}

// GetStatus mocks base method
func (m *MockPlatformRuntimeUnit) GetStatus() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStatus")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetStatus indicates an expected call of GetStatus
func (mr *MockPlatformRuntimeUnitMockRecorder) GetStatus() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStatus", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).GetStatus))
}

// GetExitCode mocks base method
func (m *MockPlatformRuntimeUnit) GetExitCode() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetExitCode")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetExitCode indicates an expected call of GetExitCode
func (mr *MockPlatformRuntimeUnitMockRecorder) GetExitCode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetExitCode", reflect.TypeOf((*MockPlatformRuntimeUnit)(nil).GetExitCode))
}

// MockHardwareStats is a mock of HardwareStats interface
type MockHardwareStats struct {
	ctrl     *gomock.Controller
	recorder *MockHardwareStatsMockRecorder
}

// MockHardwareStatsMockRecorder is the mock recorder for MockHardwareStats
type MockHardwareStatsMockRecorder struct {
	mock *MockHardwareStats
}

// NewMockHardwareStats creates a new mock instance
func NewMockHardwareStats(ctrl *gomock.Controller) *MockHardwareStats {
	mock := &MockHardwareStats{ctrl: ctrl}
	mock.recorder = &MockHardwareStatsMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockHardwareStats) EXPECT() *MockHardwareStatsMockRecorder {
	return m.recorder
}
