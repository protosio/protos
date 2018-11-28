package task

import (
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/protosio/protos/config"
	"github.com/protosio/protos/util"
	"github.com/rs/xid"
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

// Task is the interface that all task types need to adhere too
type Task interface {
	Run()
	Name() string
	SetBase(*Base)
	// fulfilled by base
	GetID() string
	Wait() error
	SetPercentage(int)
	GetPercentage() int
	SetState(string)
	Save()
}

// Progress tracks the percentage and message of a task
type Progress struct {
	Percentage int    `json:"percentage"`
	State      string `json:"state"`
}

// Base represents an (a)synchronous piece of work that Protos acts upon
type Base struct {
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     string           `json:"status"`
	Progress   Progress         `json:"progress"`
	StartedAt  *util.ProtosTime `json:"started-at,omitempty"`
	FinishedAt *util.ProtosTime `json:"finished-at,omitempty"`
	Children   []string         `json:"-"`
	Apps       []string         `json:"apps"`
	err        error

	// Communication channels
	quitChan chan bool
	finish   chan error
}

// GetID returns the id of the task
func (b *Base) GetID() string {
	return b.ID
}

// SetPercentage sets the progress percentage of the task base
func (b *Base) SetPercentage(percent int) {
	b.Progress.Percentage = percent
}

// GetPercentage gets the progress percentage of the task base
func (b *Base) GetPercentage() int {
	return b.Progress.Percentage
}

// SetState sets the progress state of the task base
func (b *Base) SetState(msg string) {
	b.Progress.State = msg
}

// Save sends a copy of the running task to the task scheduler
func (b *Base) Save() {
	saveTaskQueue <- *b
}

// Wait waits for the task to finish and returns an error if there was one. Used to mimic a blocking call
func (b *Base) Wait() error {
	err := <-b.finish
	return err
}

// Finish updates the task and signals the scheduler and the possible parent of the task
func (b *Base) Finish(err error) {
	b.Progress.Percentage = 100
	ts := util.ProtosTime(time.Now())
	b.FinishedAt = &ts
	if err != nil {
		log.WithField("proc", b.ID).Error("Failed to finish task: ", err.Error())
		b.Progress.State = err.Error()
		b.Status = FAILED
		b.err = err
	} else {
		log.WithField("proc", b.ID).Infof("Task %s finished successfully", b.ID)
		b.Status = FINISHED
	}
	b.Save()
	b.finish <- err
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

//
// General methods
//

// New creates a new task and returns it
func New(tsk Task) Task {
	ts := util.ProtosTime(time.Now())
	base := &Base{
		ID:        xid.New().String(),
		Name:      tsk.Name(),
		Status:    REQUESTED,
		Progress:  Progress{Percentage: 0},
		StartedAt: &ts,

		quitChan: make(chan bool, 1),
		finish:   make(chan error, 1),
	}
	base.Save()
	tsk.SetBase(base)
	return tsk
}

// GetAll returns all the available tasks
func GetAll() linkedhashmap.Map {
	resp := make(chan linkedhashmap.Map)
	readAllQueue <- resp
	return <-resp
}

// GetLast returns last 36 available tasks
func GetLast() linkedhashmap.Map {
	resp := make(chan linkedhashmap.Map)
	readAllQueue <- resp
	return getLastNTasks(36, <-resp)
}

// Get returns a task based on its id
func Get(id string) (Base, error) {
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
