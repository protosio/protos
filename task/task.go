package task

import (
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/util"
)

var log = util.GetLogger("task")
var gconfig = config.Get()

const (
	// REQUESTED - task has been created
	REQUESTED = "requested"
	// INPROGRESS - task is in progress
	INPROGRESS = "inprogress"
	// FAILED - task has failed
	FAILED = "failed"
	// FINISHED - tash has been completed
	FINISHED = "finished"
)

// Type is the interface that all task types need to adhere too
type Type interface {
	Run(*Task) error
	Name() string
}

// Progress tracks the percentage and message of a task
type Progress struct {
	Percentage int    `json:"percentage"`
	State      string `json:"state"`
}

// Task represents an (a)synchronous piece of work that Protos acts upon
type Task struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     string           `json:"status"`
	Progress   Progress         `json:"progress"`
	StartedAt  *util.ProtosTime `json:"started-at,omitempty"`
	FinishedAt *util.ProtosTime `json:"finished-at,omitempty"`
	Children   []string         `json:"-"`
	Apps       []string         `json:"apps"`
	taskType   Type
	err        error

	// Communication channels
	quitChan chan bool
	finish   chan error
}

//
// Methods that run inside the goroutine and that own and update the task struct
//

func getLastNTasks(n int, tsks linkedhashmap.Map) linkedhashmap.Map {
	reversedLastTasks := linkedhashmap.New()
	lastTasks := linkedhashmap.New()
	rit := tsks.Iterator()
	i := 0
	for rit.End(); rit.Prev(); {
		if i == n {
			break
		}
		reversedLastTasks.Put(rit.Key(), rit.Value())
		i++
	}
	it := reversedLastTasks.Iterator()
	for it.End(); it.Prev(); {
		lastTasks.Put(it.Key(), it.Value())
	}
	return *lastTasks
}

// Run starts the task
func (t *Task) Run() {
	log.WithField("proc", t.ID).Infof("Started task %s", t.ID)
	err := t.taskType.Run(t)
	if err != nil {
		log.WithField("proc", t.ID).Error("Failed to finish task: ", err.Error())
		t.Progress.State = err.Error()
		t.Status = FAILED
		t.err = err
	} else {
		log.WithField("proc", t.ID).Infof("Task %s finished successfully", t.ID)
		t.Status = FINISHED
	}
	t.Finish()
}

// Finish updates the task and signals the scheduler and the possible parent of the task
func (t *Task) Finish() {
	t.Progress.Percentage = 100
	ts := util.ProtosTime(time.Now())
	t.FinishedAt = &ts
	t.finish <- t.err
	t.Update()
}

// Update sends a copy of the running task to the task scheduler
func (t *Task) Update() {
	updateTaskQueue <- *t
}

//
// Methods that act on a task copy, and they dont modify the struct
//

// Wait waits for the task to finish and returns an error if there was one. Used to mimic a blocking call
func (t Task) Wait() error {
	return <-t.finish
}

//
// General methods
//

// New creates a new task and returns it
func New(taskType Type) Task {
	rt := requestTask{t: taskType, resp: make(chan Task)}
	schedulerQueue <- rt
	return <-rt.resp
}

// GetAll returns all the available tasks
func GetAll() linkedhashmap.Map {
	resp := make(chan linkedhashmap.Map)
	readAllQueue <- resp
	return getLastNTasks(36, <-resp)
}

// Get returns a task based on its id
func Get(id string) (Task, error) {
	rt := readTaskReq{id: id, resp: make(chan readTaskResp)}
	readTaskQueue <- rt
	resp := <-rt.resp
	return resp.tsk, resp.err
}

// GetIDs returns all tasks for the provided ids
func GetIDs(ids []string) linkedhashmap.Map {
	rt := readTasksReq{ids: ids, resp: make(chan linkedhashmap.Map)}
	readTasksQueue <- rt
	return getLastNTasks(10, <-rt.resp)
}
