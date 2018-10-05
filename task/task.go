package task

import (
	"time"

	"github.com/protosio/protos/util"
)

var log = util.GetLogger("task")

const (
	// REQUESTED - task has been created
	REQUESTED = "REQUESTED"
	// INPROGRESS - task is in progress
	INPROGRESS = "INPROGRESS"
	// ERROR - task has failed
	ERROR = "ERROR"
	// FINISHED - tash has been completed
	FINISHED = "FINISHED"
)

// Type is the interface that all task types need to adhere too
type Type interface {
	Run() error
}

// Progress tracks the percentage and message of a task
type Progress struct {
	Percentage    int
	StatusMessage string
}

// Task represents an (a)synchronous piece of work that Protos acts upon
type Task struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	Progress   *Progress  `json:"progress"`
	StartedAt  *time.Time `json:"started-at,omitempty"`
	FinishedAt *time.Time `json:"finished-at,omitempty"`
	taskType   Type

	// Communication channels
	quitChan chan bool
	getCopy  chan Task
}
// GetCopy returns a copy of a running task
func (t *Task) GetCopy() Task {
	log.Debugf("Getting copy of task %s", t.ID)
	t.getCopy <- Task{}
	tsk := <-t.getCopy
	return tsk
}

//
// General methods
//

// New creates a new task and returns it
func New(taskType Type) (Task, error) {
	rt := requestTask{t: taskType, resp: make(chan Task)}
	schedulerQueue <- rt
	return <-rt.resp, nil
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
