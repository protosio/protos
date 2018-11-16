package task

import (
	"encoding/gob"
	"fmt"
	"reflect"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/pkg/errors"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
	"github.com/rs/xid"
)

// all ongoing tasks
var tasks *linkedhashmap.Map

type readTaskResp struct {
	tsk Task
	err error
}

type readTaskReq struct {
	id   string
	resp chan readTaskResp
}

type readTasksReq struct {
	ids  []string
	resp chan linkedhashmap.Map
}

type requestTask struct {
	t    Type
	resp chan Task
}

// queue is a buffered channel to which we submit all the task requests
var schedulerQueue = make(chan requestTask, 100)

// readTaskQueue receives read requests for specific tasks, based on task id
var readTaskQueue = make(chan readTaskReq)

// readTasksQueue receives read requests for multiple tasks, based on task id
var readTasksQueue = make(chan readTasksReq)

// updateTaskQueue receives updated information for a task
var updateTaskQueue = make(chan Task, 1000)

// readAllQueue receives read requests for the whole task list
var readAllQueue = make(chan chan linkedhashmap.Map)

func initDB() *linkedhashmap.Map {
	log.WithField("proc", "taskscheduler").Debug("Retrieving tasks from DB")
	gob.Register(&Task{})
	gob.Register(&util.ProtosTime{})

	dbtasks := []Task{}
	err := database.All(&dbtasks)
	if err != nil {
		log.Fatal("Could not retrieve tasks from database: ", err)
	}

	ltasks := linkedhashmap.New()
	for _, task := range dbtasks {
		ltasks.Put(task.ID, task)
	}
	return ltasks
}

func saveTask(task Task) {
	log.WithField("proc", "taskscheduler").Debugf("Saving task %s to database", task.ID)
	err := database.Save(&task)
	if err != nil {
		log.Panic(errors.Wrapf(err, "Could not save task %s to database", task.ID))
	}
}

func createTask(taskType Type) Task {
	ts := util.ProtosTime(time.Now())
	tsk := Task{
		ID:        xid.New().String(),
		Name:      taskType.Name(),
		Status:    REQUESTED,
		Progress:  Progress{Percentage: 0},
		StartedAt: &ts,
		taskType:  taskType,

		quitChan: make(chan bool, 1),
		finish:   make(chan error, 1),
	}
	saveTask(tsk)
	log.WithField("proc", "taskscheduler").Debugf("Created new task %s with id %s", reflect.TypeOf(taskType), tsk.ID)
	return tsk
}

// Scheduler takes care of scheduling long running background tasks
func Scheduler(quit chan bool) {
	log.WithField("proc", "taskscheduler").Info("Starting the task scheduler")
	tasks = initDB()
	for {
		select {
		case taskReq := <-schedulerQueue:
			tsk := createTask(taskReq.t)
			tasks.Put(tsk.ID, tsk)
			taskReq.resp <- tsk
			log.WithField("proc", "taskscheduler").Infof("Running new task %s with id %s", reflect.TypeOf(taskReq.t), tsk.ID)
			go tsk.Run()
		case tsk := <-updateTaskQueue:
			log.WithField("proc", "taskscheduler").Debugf("Updating task %s", tsk.ID)
			tasks.Put(tsk.ID, tsk)
			gconfig.WSPublish <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeTask, PayloadValue: tsk}
			saveTask(tsk)
		case readReq := <-readTaskQueue:
			if tsk, found := tasks.Get(readReq.id); found {
				readReq.resp <- readTaskResp{tsk: tsk.(Task)}
			} else {
				readReq.resp <- readTaskResp{err: fmt.Errorf("Could not find task %s", readReq.id)}
			}
		case readReq := <-readTasksQueue:
			filter := func(k interface{}, v interface{}) bool {
				if found, _ := util.StringInSlice(k.(string), readReq.ids); found {
					return true
				}
				return false
			}
			requestedTasks := tasks.Select(filter)
			readReq.resp <- *requestedTasks
		case readAllResp := <-readAllQueue:
			readAllResp <- *tasks
		case <-quit:
			log.Info("Shutting down task scheduler")
			return
		}
	}

}
