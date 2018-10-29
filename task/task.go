package task

import (
	"time"

	"github.com/protosio/protos/util"
)

var log = util.GetLogger("task")

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
	Status     string           `json:"status"`
	Progress   Progress         `json:"progress"`
	StartedAt  *util.ProtosTime `json:"started-at,omitempty"`
	FinishedAt *util.ProtosTime `json:"finished-at,omitempty"`
	taskType   Type

	// Communication channels
	quitChan chan bool
}

// Run starts the task
func (t *Task) Run() {
	log.WithField("proc", t.ID).Infof("Started task %s", t.ID)
	err := t.taskType.Run(t)
	if err != nil {
		log.WithField("proc", t.ID).Error("Failed to finish task: ", err.Error())
		t.Progress.State = err.Error()
		t.Status = FAILED
	} else {
		log.WithField("proc", t.ID).Infof("Task %s finished successfully", t.ID)
		t.Status = FINISHED
	}
	t.Progress.Percentage = 100
	ts := util.ProtosTime(time.Now())
	t.FinishedAt = &ts
	t.Update()
}

// Update sends a copy of the running task to the task scheduler
func (t *Task) Update() {
	updateTaskQueue <- *t
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
func GetAll() map[string]Task {
	resp := make(chan map[string]Task)
	readAllQueue <- resp
	return <-resp
}

// Get returns a task based on its id
func Get(id string) (Task, error) {
	rt := readTaskReq{id: id, resp: make(chan readTaskResp)}
	readTaskQueue <- rt
	resp := <-rt.resp
	return resp.tsk, resp.err
}
