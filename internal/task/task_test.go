package task

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/protosio/protos/internal/mock"
	"github.com/protosio/protos/internal/util"

	"github.com/golang/mock/gomock"
	"github.com/rs/xid"
)

func TestTaskManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	wschan := make(chan interface{}, 100)
	wsPublisherMock := mock.NewMockwsPublisher(ctrl)
	wsPublisherMock.EXPECT().GetWSPublishChannel().Return(wschan).Times(1)

	dbMock := mock.NewMockDB(ctrl)
	dbMock.EXPECT().Register(gomock.Any()).Return().Times(2)
	dbMock.EXPECT().All(gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			tasks := to.(*[]Base)
			*tasks = append(*tasks, Base{ID: "0001", Status: INPROGRESS}, Base{ID: "0002", Status: FINISHED}, Base{ID: "0003", Status: REQUESTED})
		})
	// Save() gets called only once because only the tasks with INPROGRESS status get changed and saved
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)

	tm := CreateManager(dbMock, wsPublisherMock)

	// test if correct message is published when CreateManager is created
	msg1 := (<-wschan).(util.WSMessage)
	if msg1.MsgType != util.WSMsgTypeUpdate || msg1.PayloadType != util.WSPayloadTypeTask {
		t.Errorf("WS msg1 has the wrong fields %v", msg1)
	}
	tsk1 := msg1.PayloadValue.(Base)
	if tsk1.ID != "0001" {
		t.Errorf("tsk1 should return task with id 0001, instead of %s", tsk1.ID)
	}

	// test if GetAll returns the right number of elements
	alltasks := tm.GetAll()
	if alltasks.Size() != 3 {
		t.Error("tm.GetAll should return 3 elements, but it returned", alltasks.Size())
	}

	// test if GetLast returns the right number of elements
	lasttasks := tm.GetLast()
	if lasttasks.Size() != 3 {
		t.Error("tm.GetLast should return 3 elements, but it returned", alltasks.Size())
	}

	// test if Get returns the correct results
	tsk, err := tm.Get("0001")
	if err != nil {
		t.Errorf("Get(0001) should NOT return an error: %s", err.Error())
	}
	if tsk.GetID() != "0001" {
		t.Errorf("tm.Get(0001) should return task with id 0001, instead of %s", tsk.GetID())
	}
	_, err = tm.Get("0007")
	log.Info(err)
	if err == nil {
		t.Errorf("Get(0007) should return an error")
	}

	// test if GetIDs returns the correct results
	selectedTasks := tm.GetIDs([]string{"0002", "0002", "asd", "0001", "0003", "0008"})
	if selectedTasks.Size() != 3 {
		t.Error("tm.GetIDs should return 3 elements, based on the correct IDs provided, but it returned", selectedTasks.Size())
	}

	// test if saveTask makes all the right calls
	wsPublisherMock.EXPECT().GetWSPublishChannel().Return(wschan).Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)
	tm.saveTask(&Base{ID: "0004", Status: INPROGRESS, access: &sync.Mutex{}})

	msg2 := (<-wschan).(util.WSMessage)
	if msg2.MsgType != util.WSMsgTypeUpdate || msg2.PayloadType != util.WSPayloadTypeTask {
		t.Errorf("WS msg2 has the wrong fields %v", msg2)
	}
	tsk2 := msg2.PayloadValue.(Base)
	if tsk2.ID != "0004" {
		t.Errorf("tsk2 should return task with id 0004, instead of %s", tsk2.ID)
	}

	// testing the creation of a new task
	// Note: the wait() needs to be called on the newly created task or else the test fails because
	// it continues to run while the task runs in a separate routine
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(3)
	wsPublisherMock.EXPECT().GetWSPublishChannel().Return(wschan).Times(3)
	customTask := mock.NewMockCustomTask(ctrl)
	customTask.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
	tsk3 := tm.New("customTaskName", customTask)

	err = tsk3.Wait()
	if err != nil {
		t.Errorf("tsk3.Wait() should not return an error. Err: %s", err.Error())
	}

	if tsk3.GetPercentage() != 100 {
		t.Errorf("tsk3.GetPercentage() should return percentage 100, instead of %d", tsk3.GetPercentage())
	}

	tsk3base := tsk3.(*Base)
	if tsk3base.Name != "customTaskName" || tsk3base.Status != FINISHED || tsk3base.parent != tm || tsk3base.custom != customTask {
		t.Errorf("tsk3 has one or more incorrect fields: %+v", tsk3base)
	}

	// test db failure during saveTask
	dbMock.EXPECT().Save(gomock.Any()).Return(errors.New("test db error")).Times(1)
	func() {
		defer func() {
			r := recover()
			if r == nil {
				t.Errorf("A DB error in saveTask should lead to a panic")
			}
		}()
		tm.saveTask(&Base{ID: "0005", Status: INPROGRESS, access: &sync.Mutex{}})
	}()

}

func TestTask(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	dbMock := mock.NewMockDB(ctrl)
	dbMock.EXPECT().Register(gomock.Any()).Return().Times(2)
	dbMock.EXPECT().All(gomock.Any()).Return(nil).Times(1).
		Do(func(to interface{}) {
			tasks := to.(*[]Base)
			*tasks = append(*tasks, Base{ID: "0001", Status: INPROGRESS}, Base{ID: "0002", Status: FINISHED}, Base{ID: "0003", Status: REQUESTED})
		})
	// Save() gets called only once because only the tasks with INPROGRESS status get changed and saved
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)

	wsPublisherMock := mock.NewMockwsPublisher(ctrl)
	wschan := make(chan interface{}, 100)
	wsPublisherMock.EXPECT().GetWSPublishChannel().Return(wschan).Times(1)

	tm := CreateManager(dbMock, wsPublisherMock)
	customTask := mock.NewMockCustomTask(ctrl)
	ts := util.ProtosTime(time.Now())

	task := &Base{
		access: &sync.Mutex{},
		custom: customTask,
		parent: tm,

		ID:        xid.New().String(),
		Name:      "taskName",
		Status:    REQUESTED,
		Progress:  Progress{Percentage: 0},
		StartedAt: &ts,

		finish: make(chan error, 1),
	}

	if task.GetID() != task.ID {
		t.Errorf("task.GetID() should return %s instead of %s", task.ID, task.GetID())
	}

	task.SetPercentage(45)
	if task.Progress.Percentage != 45 {
		t.Errorf("task.SetPercentage(45) should set the progress to %d instead of %d", 45, task.Progress.Percentage)
	}

	if task.GetPercentage() != task.Progress.Percentage {
		t.Errorf("task.GetPercentage() should return %d instead of %d", task.Progress.Percentage, task.GetPercentage())
	}

	state1 := "state1"
	task.SetState(state1)
	if task.Progress.State != state1 {
		t.Errorf("task.SetState(%s) should set the state to %s instead of %s", state1, state1, task.Progress.State)
	}

	status1 := "status1"
	task.SetStatus(status1)
	if task.Status != status1 {
		t.Errorf("task.SetStatus(%s) should set the status to %s instead of %s", status1, status1, task.Status)
	}

	appid := "appid"
	task.AddApp(appid)
	if found, _ := util.StringInSlice(appid, task.Apps); found != true {
		t.Errorf("task.AddApp(%s) did not add app with id '%s' to the internal list of apps", appid, appid)
	}

	// tests for Kill() related behaviour
	err := task.Kill()
	if err == nil {
		t.Errorf("task.Kill() should have returned an error")
	}

	task.SetKillable()
	if task.killable == nil {
		t.Errorf("task.SetKillable() did not correctly set the killable fields")
	}

	task.SetStatus(FINISHED)
	err = task.Kill()
	if err == nil {
		t.Errorf("task.Kill() should have returned an error")
	}

	task.SetStatus(REQUESTED)
	err = task.Kill()
	if err != nil {
		t.Fatalf("task.Kill() returned an error: %s", err.Error())
	}
	err = task.killable.Err()
	if err == nil || err.Error() != "Task cancelled by user" {
		t.Errorf("task.Kill() did not kill the task correctly: %s", err.Error())
	}

	// tests for Dying() behaviour. Soll purpose of this test is to see it return and not block the test
	dchan := task.Dying()
	<-dchan

	// tests for Copy() behaviour
	cp := (task.Copy()).(*Base)
	if cp.ID != task.ID || cp == task {
		t.Error("task.Copy() did not produce a valid copy of the task")
	}

	// tests for Save() behaviour
	wsPublisherMock.EXPECT().GetWSPublishChannel().Return(wschan).Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)
	task.Save()

	// tests for Run() behaviour are implemented in the previous method because a task is
	// also started when it is created

}
