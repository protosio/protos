package task

import (
	"errors"
	"sync"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/protosio/protos/mock"
	"github.com/protosio/protos/util"
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
	tms := tm.(*Manager)
	wsPublisherMock.EXPECT().GetWSPublishChannel().Return(wschan).Times(1)
	dbMock.EXPECT().Save(gomock.Any()).Return(nil).Times(1)
	tms.saveTask(&Base{ID: "0004", Status: INPROGRESS, access: &sync.Mutex{}})

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
	customTask.EXPECT().Name().Return("customTaskName").Times(1)
	customTask.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	tsk3 := tm.New(customTask)

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
	dbMock.EXPECT().Save(gomock.Any()).Return(errors.New("db error")).Times(1)
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("A DB error in saveTask should lead to a panic")
		}
	}()
	tms.saveTask(&Base{ID: "0005", Status: INPROGRESS, access: &sync.Mutex{}})

}
