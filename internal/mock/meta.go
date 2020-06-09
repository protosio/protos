// Code generated by MockGen. DO NOT EDIT.
// Source: internal/core/meta.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	gomock "github.com/golang/mock/gomock"
	core "github.com/protosio/protos/internal/core"
	util "github.com/protosio/protos/internal/util"
	wgtypes "golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	net "net"
	reflect "reflect"
)

// MockMeta is a mock of Meta interface
type MockMeta struct {
	ctrl     *gomock.Controller
	recorder *MockMetaMockRecorder
}

// MockMetaMockRecorder is the mock recorder for MockMeta
type MockMetaMockRecorder struct {
	mock *MockMeta
}

// NewMockMeta creates a new mock instance
func NewMockMeta(ctrl *gomock.Controller) *MockMeta {
	mock := &MockMeta{ctrl: ctrl}
	mock.recorder = &MockMetaMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockMeta) EXPECT() *MockMetaMockRecorder {
	return m.recorder
}

// GetPublicIP mocks base method
func (m *MockMeta) GetPublicIP() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPublicIP")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetPublicIP indicates an expected call of GetPublicIP
func (mr *MockMetaMockRecorder) GetPublicIP() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPublicIP", reflect.TypeOf((*MockMeta)(nil).GetPublicIP))
}

// SetDomain mocks base method
func (m *MockMeta) SetDomain(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetDomain", arg0)
}

// SetDomain indicates an expected call of SetDomain
func (mr *MockMetaMockRecorder) SetDomain(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetDomain", reflect.TypeOf((*MockMeta)(nil).SetDomain), arg0)
}

// GetDomain mocks base method
func (m *MockMeta) GetDomain() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomain")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetDomain indicates an expected call of GetDomain
func (mr *MockMetaMockRecorder) GetDomain() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomain", reflect.TypeOf((*MockMeta)(nil).GetDomain))
}

// SetNetwork mocks base method
func (m *MockMeta) SetNetwork(arg0 net.IPNet) net.IP {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetNetwork", arg0)
	ret0, _ := ret[0].(net.IP)
	return ret0
}

// SetNetwork indicates an expected call of SetNetwork
func (mr *MockMetaMockRecorder) SetNetwork(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetNetwork", reflect.TypeOf((*MockMeta)(nil).SetNetwork), arg0)
}

// GetNetwork mocks base method
func (m *MockMeta) GetNetwork() net.IPNet {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetNetwork")
	ret0, _ := ret[0].(net.IPNet)
	return ret0
}

// GetNetwork indicates an expected call of GetNetwork
func (mr *MockMetaMockRecorder) GetNetwork() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetNetwork", reflect.TypeOf((*MockMeta)(nil).GetNetwork))
}

// GetInternalIP mocks base method
func (m *MockMeta) GetInternalIP() net.IP {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInternalIP")
	ret0, _ := ret[0].(net.IP)
	return ret0
}

// GetInternalIP indicates an expected call of GetInternalIP
func (mr *MockMetaMockRecorder) GetInternalIP() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInternalIP", reflect.TypeOf((*MockMeta)(nil).GetInternalIP))
}

// GetKey mocks base method
func (m *MockMeta) GetKey() wgtypes.Key {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetKey")
	ret0, _ := ret[0].(wgtypes.Key)
	return ret0
}

// GetKey indicates an expected call of GetKey
func (mr *MockMetaMockRecorder) GetKey() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetKey", reflect.TypeOf((*MockMeta)(nil).GetKey))
}

// GetTLSCertificate mocks base method
func (m *MockMeta) GetTLSCertificate() core.Resource {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTLSCertificate")
	ret0, _ := ret[0].(core.Resource)
	return ret0
}

// GetTLSCertificate indicates an expected call of GetTLSCertificate
func (mr *MockMetaMockRecorder) GetTLSCertificate() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTLSCertificate", reflect.TypeOf((*MockMeta)(nil).GetTLSCertificate))
}

// SetAdminUser mocks base method
func (m *MockMeta) SetAdminUser(arg0 string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetAdminUser", arg0)
}

// SetAdminUser indicates an expected call of SetAdminUser
func (mr *MockMetaMockRecorder) SetAdminUser(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetAdminUser", reflect.TypeOf((*MockMeta)(nil).SetAdminUser), arg0)
}

// CreateProtosResources mocks base method
func (m *MockMeta) CreateProtosResources() (map[string]core.Resource, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateProtosResources")
	ret0, _ := ret[0].(map[string]core.Resource)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateProtosResources indicates an expected call of CreateProtosResources
func (mr *MockMetaMockRecorder) CreateProtosResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateProtosResources", reflect.TypeOf((*MockMeta)(nil).CreateProtosResources))
}

// GetProtosResources mocks base method
func (m *MockMeta) GetProtosResources() map[string]core.Resource {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetProtosResources")
	ret0, _ := ret[0].(map[string]core.Resource)
	return ret0
}

// GetProtosResources indicates an expected call of GetProtosResources
func (mr *MockMetaMockRecorder) GetProtosResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetProtosResources", reflect.TypeOf((*MockMeta)(nil).GetProtosResources))
}

// CleanProtosResources mocks base method
func (m *MockMeta) CleanProtosResources() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CleanProtosResources")
	ret0, _ := ret[0].(error)
	return ret0
}

// CleanProtosResources indicates an expected call of CleanProtosResources
func (mr *MockMetaMockRecorder) CleanProtosResources() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CleanProtosResources", reflect.TypeOf((*MockMeta)(nil).CleanProtosResources))
}

// GetDashboardDomain mocks base method
func (m *MockMeta) GetDashboardDomain() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDashboardDomain")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetDashboardDomain indicates an expected call of GetDashboardDomain
func (mr *MockMetaMockRecorder) GetDashboardDomain() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDashboardDomain", reflect.TypeOf((*MockMeta)(nil).GetDashboardDomain))
}

// GetService mocks base method
func (m *MockMeta) GetService() util.Service {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetService")
	ret0, _ := ret[0].(util.Service)
	return ret0
}

// GetService indicates an expected call of GetService
func (mr *MockMetaMockRecorder) GetService() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetService", reflect.TypeOf((*MockMeta)(nil).GetService))
}

// GetAdminUser mocks base method
func (m *MockMeta) GetAdminUser() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAdminUser")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetAdminUser indicates an expected call of GetAdminUser
func (mr *MockMetaMockRecorder) GetAdminUser() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAdminUser", reflect.TypeOf((*MockMeta)(nil).GetAdminUser))
}

// GetVersion mocks base method
func (m *MockMeta) GetVersion() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetVersion")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetVersion indicates an expected call of GetVersion
func (mr *MockMetaMockRecorder) GetVersion() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetVersion", reflect.TypeOf((*MockMeta)(nil).GetVersion))
}

// InitMode mocks base method
func (m *MockMeta) InitMode() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InitMode")
	ret0, _ := ret[0].(bool)
	return ret0
}

// InitMode indicates an expected call of InitMode
func (mr *MockMetaMockRecorder) InitMode() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InitMode", reflect.TypeOf((*MockMeta)(nil).InitMode))
}

// WaitForInit mocks base method
func (m *MockMeta) WaitForInit(ctx context.Context) (net.IP, net.IPNet, string, string) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WaitForInit", ctx)
	ret0, _ := ret[0].(net.IP)
	ret1, _ := ret[1].(net.IPNet)
	ret2, _ := ret[2].(string)
	ret3, _ := ret[3].(string)
	return ret0, ret1, ret2, ret3
}

// WaitForInit indicates an expected call of WaitForInit
func (mr *MockMetaMockRecorder) WaitForInit(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WaitForInit", reflect.TypeOf((*MockMeta)(nil).WaitForInit), ctx)
}
