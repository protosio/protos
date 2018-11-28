package task

import (
	"encoding/gob"
	"fmt"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/pkg/errors"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
)

// all ongoing tasks
var tasks *linkedhashmap.Map

// queue is a work queue to which we submit all the tasks to be worked upon
var workQueue = make(chan Task, 100)

type readTaskResp struct {
	tsk Base
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

// readTaskQueue receives read requests for specific tasks, based on task id
var readTaskQueue = make(chan readTaskReq)

// readTasksQueue receives read requests for multiple tasks, based on task id
var readTasksQueue = make(chan readTasksReq)

// saveTaskQueue receives updated information for a task
var saveTaskQueue = make(chan Base, 1000)

// readAllQueue receives read requests for the whole task list
var readAllQueue = make(chan chan linkedhashmap.Map)

func initDB() *linkedhashmap.Map {
	log.WithField("proc", "taskManager").Debug("Retrieving tasks from DB")
	gob.Register(&Base{})
	gob.Register(&util.ProtosTime{})

	dbtasks := []Base{}
	err := database.All(&dbtasks)
	if err != nil {
		log.Fatal("Could not retrieve tasks from database: ", err)
	}

	ltasks := linkedhashmap.New()
	for _, task := range dbtasks {
		ltasks.Put(task.GetID(), task)
	}
	return ltasks
}

func saveTask(btsk Base) {
	log.WithField("proc", "taskManager").Debugf("Saving task %s to database", btsk.ID)
	err := database.Save(&btsk)
	if err != nil {
		log.Panic(errors.Wrapf(err, "Could not save task %s to database", btsk.ID))
	}
}

// Manager takes care of scheduling long running background tasks
func Manager(quit chan bool) {
	log.WithField("proc", "taskmanager").Info("Starting the task manager")
	tasks = initDB()
	for {
		select {
		case tsk := <-workQueue:
			log.WithField("proc", "taskscheduler").Infof("Running new task %s with id %s", tsk.Name(), tsk.GetID())
			go tsk.Run()
		case btsk := <-saveTaskQueue:
			log.WithField("proc", "taskmanager").Debugf("Updating and saving task %s", btsk.ID)
			tasks.Put(btsk.ID, btsk)
			gconfig.WSPublish <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeTask, PayloadValue: btsk}
			saveTask(btsk)
		case readReq := <-readTaskQueue:
			if btsk, found := tasks.Get(readReq.id); found {
				readReq.resp <- readTaskResp{tsk: btsk.(Base)}
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
			tasksCopy := *tasks
			readAllResp <- tasksCopy
		case <-quit:
			log.Info("Shutting down task manager")
			return
		}
	}

}

// Schedule takes a task and starts it in a separate goroutine
func Schedule(tsk Task) {
	workQueue <- tsk
}
