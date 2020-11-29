// // Code generated by MockGen. DO NOT EDIT.
// // Source: internal/core/task.go

// // Package mock is a generated GoMock package.
package mock

// import (
// 	linkedhashmap "github.com/emirpasic/gods/maps/linkedhashmap"
// 	gomock "github.com/golang/mock/gomock"
// 	core "github.com/protosio/protos/internal/core"
// 	reflect "reflect"
// )

// // MockTaskManager is a mock of TaskManager interface
// type MockTaskManager struct {
// 	ctrl     *gomock.Controller
// 	recorder *MockTaskManagerMockRecorder
// }

// // MockTaskManagerMockRecorder is the mock recorder for MockTaskManager
// type MockTaskManagerMockRecorder struct {
// 	mock *MockTaskManager
// }

// // NewMockTaskManager creates a new mock instance
// func NewMockTaskManager(ctrl *gomock.Controller) *MockTaskManager {
// 	mock := &MockTaskManager{ctrl: ctrl}
// 	mock.recorder = &MockTaskManagerMockRecorder{mock}
// 	return mock
// }

// // EXPECT returns an object that allows the caller to indicate expected use
// func (m *MockTaskManager) EXPECT() *MockTaskManagerMockRecorder {
// 	return m.recorder
// }

// // New mocks base method
// func (m *MockTaskManager) New(name string, customTask core.CustomTask) core.Task {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "New", name, customTask)
// 	ret0, _ := ret[0].(core.Task)
// 	return ret0
// }

// // New indicates an expected call of New
// func (mr *MockTaskManagerMockRecorder) New(name, customTask interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "New", reflect.TypeOf((*MockTaskManager)(nil).New), name, customTask)
// }

// // Get mocks base method
// func (m *MockTaskManager) Get(id string) (core.Task, error) {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "Get", id)
// 	ret0, _ := ret[0].(core.Task)
// 	ret1, _ := ret[1].(error)
// 	return ret0, ret1
// }

// // Get indicates an expected call of Get
// func (mr *MockTaskManagerMockRecorder) Get(id interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockTaskManager)(nil).Get), id)
// }

// // GetAll mocks base method
// func (m *MockTaskManager) GetAll() *linkedhashmap.Map {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetAll")
// 	ret0, _ := ret[0].(*linkedhashmap.Map)
// 	return ret0
// }

// // GetAll indicates an expected call of GetAll
// func (mr *MockTaskManagerMockRecorder) GetAll() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAll", reflect.TypeOf((*MockTaskManager)(nil).GetAll))
// }

// // GetIDs mocks base method
// func (m *MockTaskManager) GetIDs(ids []string) linkedhashmap.Map {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetIDs", ids)
// 	ret0, _ := ret[0].(linkedhashmap.Map)
// 	return ret0
// }

// // GetIDs indicates an expected call of GetIDs
// func (mr *MockTaskManagerMockRecorder) GetIDs(ids interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetIDs", reflect.TypeOf((*MockTaskManager)(nil).GetIDs), ids)
// }

// // GetLast mocks base method
// func (m *MockTaskManager) GetLast() linkedhashmap.Map {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetLast")
// 	ret0, _ := ret[0].(linkedhashmap.Map)
// 	return ret0
// }

// // GetLast indicates an expected call of GetLast
// func (mr *MockTaskManagerMockRecorder) GetLast() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLast", reflect.TypeOf((*MockTaskManager)(nil).GetLast))
// }

// // MockCustomTask is a mock of CustomTask interface
// type MockCustomTask struct {
// 	ctrl     *gomock.Controller
// 	recorder *MockCustomTaskMockRecorder
// }

// // MockCustomTaskMockRecorder is the mock recorder for MockCustomTask
// type MockCustomTaskMockRecorder struct {
// 	mock *MockCustomTask
// }

// // NewMockCustomTask creates a new mock instance
// func NewMockCustomTask(ctrl *gomock.Controller) *MockCustomTask {
// 	mock := &MockCustomTask{ctrl: ctrl}
// 	mock.recorder = &MockCustomTaskMockRecorder{mock}
// 	return mock
// }

// // EXPECT returns an object that allows the caller to indicate expected use
// func (m *MockCustomTask) EXPECT() *MockCustomTaskMockRecorder {
// 	return m.recorder
// }

// // Run mocks base method
// func (m *MockCustomTask) Run(parent core.Task, id string, progress core.Progress) error {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "Run", parent, id, progress)
// 	ret0, _ := ret[0].(error)
// 	return ret0
// }

// // Run indicates an expected call of Run
// func (mr *MockCustomTaskMockRecorder) Run(parent, id, progress interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockCustomTask)(nil).Run), parent, id, progress)
// }

// // MockProgress is a mock of Progress interface
// type MockProgress struct {
// 	ctrl     *gomock.Controller
// 	recorder *MockProgressMockRecorder
// }

// // MockProgressMockRecorder is the mock recorder for MockProgress
// type MockProgressMockRecorder struct {
// 	mock *MockProgress
// }

// // NewMockProgress creates a new mock instance
// func NewMockProgress(ctrl *gomock.Controller) *MockProgress {
// 	mock := &MockProgress{ctrl: ctrl}
// 	mock.recorder = &MockProgressMockRecorder{mock}
// 	return mock
// }

// // EXPECT returns an object that allows the caller to indicate expected use
// func (m *MockProgress) EXPECT() *MockProgressMockRecorder {
// 	return m.recorder
// }

// // SetPercentage mocks base method
// func (m *MockProgress) SetPercentage(percent int) {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "SetPercentage", percent)
// }

// // SetPercentage indicates an expected call of SetPercentage
// func (mr *MockProgressMockRecorder) SetPercentage(percent interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPercentage", reflect.TypeOf((*MockProgress)(nil).SetPercentage), percent)
// }

// // SetState mocks base method
// func (m *MockProgress) SetState(stateText string) {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "SetState", stateText)
// }

// // SetState indicates an expected call of SetState
// func (mr *MockProgressMockRecorder) SetState(stateText interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetState", reflect.TypeOf((*MockProgress)(nil).SetState), stateText)
// }

// // MockTask is a mock of Task interface
// type MockTask struct {
// 	ctrl     *gomock.Controller
// 	recorder *MockTaskMockRecorder
// }

// // MockTaskMockRecorder is the mock recorder for MockTask
// type MockTaskMockRecorder struct {
// 	mock *MockTask
// }

// // NewMockTask creates a new mock instance
// func NewMockTask(ctrl *gomock.Controller) *MockTask {
// 	mock := &MockTask{ctrl: ctrl}
// 	mock.recorder = &MockTaskMockRecorder{mock}
// 	return mock
// }

// // EXPECT returns an object that allows the caller to indicate expected use
// func (m *MockTask) EXPECT() *MockTaskMockRecorder {
// 	return m.recorder
// }

// // GetID mocks base method
// func (m *MockTask) GetID() string {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetID")
// 	ret0, _ := ret[0].(string)
// 	return ret0
// }

// // GetID indicates an expected call of GetID
// func (mr *MockTaskMockRecorder) GetID() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetID", reflect.TypeOf((*MockTask)(nil).GetID))
// }

// // Wait mocks base method
// func (m *MockTask) Wait() error {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "Wait")
// 	ret0, _ := ret[0].(error)
// 	return ret0
// }

// // Wait indicates an expected call of Wait
// func (mr *MockTaskMockRecorder) Wait() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Wait", reflect.TypeOf((*MockTask)(nil).Wait))
// }

// // SetPercentage mocks base method
// func (m *MockTask) SetPercentage(percentage int) {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "SetPercentage", percentage)
// }

// // SetPercentage indicates an expected call of SetPercentage
// func (mr *MockTaskMockRecorder) SetPercentage(percentage interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetPercentage", reflect.TypeOf((*MockTask)(nil).SetPercentage), percentage)
// }

// // GetPercentage mocks base method
// func (m *MockTask) GetPercentage() int {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "GetPercentage")
// 	ret0, _ := ret[0].(int)
// 	return ret0
// }

// // GetPercentage indicates an expected call of GetPercentage
// func (mr *MockTaskMockRecorder) GetPercentage() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPercentage", reflect.TypeOf((*MockTask)(nil).GetPercentage))
// }

// // SetState mocks base method
// func (m *MockTask) SetState(stateText string) {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "SetState", stateText)
// }

// // SetState indicates an expected call of SetState
// func (mr *MockTaskMockRecorder) SetState(stateText interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetState", reflect.TypeOf((*MockTask)(nil).SetState), stateText)
// }

// // SetStatus mocks base method
// func (m *MockTask) SetStatus(statusText string) {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "SetStatus", statusText)
// }

// // SetStatus indicates an expected call of SetStatus
// func (mr *MockTaskMockRecorder) SetStatus(statusText interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetStatus", reflect.TypeOf((*MockTask)(nil).SetStatus), statusText)
// }

// // AddApp mocks base method
// func (m *MockTask) AddApp(id string) {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "AddApp", id)
// }

// // AddApp indicates an expected call of AddApp
// func (mr *MockTaskMockRecorder) AddApp(id interface{}) *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddApp", reflect.TypeOf((*MockTask)(nil).AddApp), id)
// }

// // Copy mocks base method
// func (m *MockTask) Copy() core.Task {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "Copy")
// 	ret0, _ := ret[0].(core.Task)
// 	return ret0
// }

// // Copy indicates an expected call of Copy
// func (mr *MockTaskMockRecorder) Copy() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Copy", reflect.TypeOf((*MockTask)(nil).Copy))
// }

// // SetKillable mocks base method
// func (m *MockTask) SetKillable() {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "SetKillable")
// }

// // SetKillable indicates an expected call of SetKillable
// func (mr *MockTaskMockRecorder) SetKillable() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetKillable", reflect.TypeOf((*MockTask)(nil).SetKillable))
// }

// // Kill mocks base method
// func (m *MockTask) Kill() error {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "Kill")
// 	ret0, _ := ret[0].(error)
// 	return ret0
// }

// // Kill indicates an expected call of Kill
// func (mr *MockTaskMockRecorder) Kill() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Kill", reflect.TypeOf((*MockTask)(nil).Kill))
// }

// // Dying mocks base method
// func (m *MockTask) Dying() <-chan struct{} {
// 	m.ctrl.T.Helper()
// 	ret := m.ctrl.Call(m, "Dying")
// 	ret0, _ := ret[0].(<-chan struct{})
// 	return ret0
// }

// // Dying indicates an expected call of Dying
// func (mr *MockTaskMockRecorder) Dying() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Dying", reflect.TypeOf((*MockTask)(nil).Dying))
// }

// // Save mocks base method
// func (m *MockTask) Save() {
// 	m.ctrl.T.Helper()
// 	m.ctrl.Call(m, "Save")
// }

// // Save indicates an expected call of Save
// func (mr *MockTaskMockRecorder) Save() *gomock.Call {
// 	mr.mock.ctrl.T.Helper()
// 	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Save", reflect.TypeOf((*MockTask)(nil).Save))
// }
