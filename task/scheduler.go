package task

import (
	"fmt"
	"reflect"
	"time"

	"github.com/rs/xid"
)

// all ongoing tasks
var tasks = make(map[string]Task)

type readTaskResp struct {
	tsk Task
	err error
}

type readTaskReq struct {
	id   string
	resp chan readTaskResp
}

type requestTask struct {
	t    Type
	resp chan Task
}

// queue is a buffered channel to which we submit all the task requests
var schedulerQueue = make(chan requestTask, 100)

// readTaskQueue receives read requests for specific tasks, based on task id
var readTaskQueue = make(chan readTaskReq)

// updateTaskQueue receives updated information for a task
var updateTaskQueue = make(chan Task, 1000)

// readAllQueue receives read requests for the whole task list
var readAllQueue = make(chan chan map[string]Task)

func createTask(taskType Type) Task {
	ts := time.Now().Truncate(time.Millisecond)
	tsk := Task{
		ID:        xid.New().String(),
		Status:    REQUESTED,
		Progress:  &Progress{Percentage: 0},
		StartedAt: &ts,
		taskType:  taskType,

		quitChan: make(chan bool),
	}
	log.WithField("proc", "scheduler").Debugf("Created new task %s with id %s", reflect.TypeOf(taskType), tsk.ID)
	return tsk
}

// Scheduler takes care of scheduling long running background tasks
func Scheduler(nrWorkers int) {
	log.WithField("proc", "taskscheduler").Info("Starting the task scheduler")
	for {
		select {
		case taskReq := <-schedulerQueue:
			tsk := createTask(taskReq.t)
			tasks[tsk.ID] = tsk
			taskReq.resp <- tsk
			log.WithField("proc", "scheduler").Infof("Running new task %s with id %s", reflect.TypeOf(taskReq.t), tsk.ID)
			go tsk.Run()
		case tsk := <-updateTaskQueue:
			log.WithField("proc", "scheduler").Debugf("Updating task %s", tsk.ID)
			tasks[tsk.ID] = tsk
		case readReq := <-readTaskQueue:
			if tsk, found := tasks[readReq.id]; found {
				readReq.resp <- readTaskResp{tsk: tsk}
			} else {
				readReq.resp <- readTaskResp{err: fmt.Errorf("Could not find task %s", readReq.id)}
			}
		case readAllResp := <-readAllQueue:
			readAllResp <- tasks
		}
	}

}
