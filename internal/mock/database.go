// Code generated by MockGen. DO NOT EDIT.
// Source: internal/core/db.go

// Package mock is a generated GoMock package.
package mock

import (
	gomock "github.com/golang/mock/gomock"
	reflect "reflect"
)

// MockDB is a mock of DB interface
type MockDB struct {
	ctrl     *gomock.Controller
	recorder *MockDBMockRecorder
}

// MockDBMockRecorder is the mock recorder for MockDB
type MockDBMockRecorder struct {
	mock *MockDB
}

// NewMockDB creates a new mock instance
func NewMockDB(ctrl *gomock.Controller) *MockDB {
	mock := &MockDB{ctrl: ctrl}
	mock.recorder = &MockDBMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDB) EXPECT() *MockDBMockRecorder {
	return m.recorder
}

// SaveStruct mocks base method
func (m *MockDB) SaveStruct(dataset string, data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveStruct", dataset, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveStruct indicates an expected call of SaveStruct
func (mr *MockDBMockRecorder) SaveStruct(dataset, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveStruct", reflect.TypeOf((*MockDB)(nil).SaveStruct), dataset, data)
}

// GetStruct mocks base method
func (m *MockDB) GetStruct(dataset string, to interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStruct", dataset, to)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetStruct indicates an expected call of GetStruct
func (mr *MockDBMockRecorder) GetStruct(dataset, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStruct", reflect.TypeOf((*MockDB)(nil).GetStruct), dataset, to)
}

// GetSet mocks base method
func (m *MockDB) GetSet(dataset string, to interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSet", dataset, to)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetSet indicates an expected call of GetSet
func (mr *MockDBMockRecorder) GetSet(dataset, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSet", reflect.TypeOf((*MockDB)(nil).GetSet), dataset, to)
}

// InsertInSet mocks base method
func (m *MockDB) InsertInSet(dataset string, data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertInSet", dataset, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// InsertInSet indicates an expected call of InsertInSet
func (mr *MockDBMockRecorder) InsertInSet(dataset, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertInSet", reflect.TypeOf((*MockDB)(nil).InsertInSet), dataset, data)
}

// RemoveFromSet mocks base method
func (m *MockDB) RemoveFromSet(dataset string, data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveFromSet", dataset, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveFromSet indicates an expected call of RemoveFromSet
func (mr *MockDBMockRecorder) RemoveFromSet(dataset, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveFromSet", reflect.TypeOf((*MockDB)(nil).RemoveFromSet), dataset, data)
}

// GetMap mocks base method
func (m *MockDB) GetMap(dataset string, to interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMap", dataset, to)
	ret0, _ := ret[0].(error)
	return ret0
}

// GetMap indicates an expected call of GetMap
func (mr *MockDBMockRecorder) GetMap(dataset, to interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMap", reflect.TypeOf((*MockDB)(nil).GetMap), dataset, to)
}

// InsertInMap mocks base method
func (m *MockDB) InsertInMap(dataset, id string, data interface{}) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "InsertInMap", dataset, id, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// InsertInMap indicates an expected call of InsertInMap
func (mr *MockDBMockRecorder) InsertInMap(dataset, id, data interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "InsertInMap", reflect.TypeOf((*MockDB)(nil).InsertInMap), dataset, id, data)
}

// RemoveFromMap mocks base method
func (m *MockDB) RemoveFromMap(dataset, id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveFromMap", dataset, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveFromMap indicates an expected call of RemoveFromMap
func (mr *MockDBMockRecorder) RemoveFromMap(dataset, id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveFromMap", reflect.TypeOf((*MockDB)(nil).RemoveFromMap), dataset, id)
}

// Close mocks base method
func (m *MockDB) Close() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Close")
	ret0, _ := ret[0].(error)
	return ret0
}

// Close indicates an expected call of Close
func (mr *MockDBMockRecorder) Close() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockDB)(nil).Close))
}
