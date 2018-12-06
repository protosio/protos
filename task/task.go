package task

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/icholy/killable"
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

// ErrKilledByUser is returned when a task is canceled/killed by the user
var ErrKilledByUser = errors.New("Task cancelled by user")

// CustomTask is the interface that is implemented by custom tasks in various packages
type CustomTask interface {
	Run() error
	Name() string
	SetBase(Task)
}

// Task is the interface that all task types need to adhere too
type Task interface {
	GetID() string
	Wait() error
	SetPercentage(int)
	GetPercentage() int
	SetState(string)
	SetStatus(string)
	AddApp(string)
	Copy() Base
	SetKillable()
	Dying() <-chan struct{}
	Save()
}

// Progress tracks the percentage and message of a task
type Progress struct {
	Percentage int    `json:"percentage"`
	State      string `json:"state"`
}

// Base represents an (a)synchronous piece of work that Protos acts upon
type Base struct {
	access *sync.Mutex
	custom CustomTask

	// public members
	ID         string           `json:"id"`
	Name       string           `json:"name"`
	Status     string           `json:"status"`
	Progress   Progress         `json:"progress"`
	StartedAt  *util.ProtosTime `json:"started-at,omitempty"`
	FinishedAt *util.ProtosTime `json:"finished-at,omitempty"`
	Children   []string         `json:"-"`
	Apps       []string         `json:"apps"`
	Killable   bool             `json:"killable"`
	err        error

	// Communication channels
	killable killable.Killable
	finish   chan error
}

// GetID returns the id of the task
func (b *Base) GetID() string {
	b.access.Lock()
	defer b.access.Unlock()
	return b.ID
}

// SetPercentage sets the progress percentage of the task base
func (b *Base) SetPercentage(percent int) {
	b.access.Lock()
	b.Progress.Percentage = percent
	b.access.Unlock()
}

// GetPercentage gets the progress percentage of the task base
func (b *Base) GetPercentage() int {
	b.access.Lock()
	defer b.access.Unlock()
	return b.Progress.Percentage
}

// SetState sets the progress state of the task base
func (b *Base) SetState(msg string) {
	b.access.Lock()
	b.Progress.State = msg
	b.access.Unlock()
}

// SetStatus sets the progress state of the task base
func (b *Base) SetStatus(msg string) {
	b.access.Lock()
	b.Status = msg
	b.access.Unlock()
}

// AddApp adds an app id to the task
func (b *Base) AddApp(id string) {
	b.access.Lock()
	b.Apps = append(b.Apps, id)
	b.access.Unlock()
}

// SetKillable makes a task killable
func (b *Base) SetKillable() {
	b.access.Lock()
	b.Killable = true
	b.killable = killable.New()
	b.access.Unlock()
}

// Kill stops a killable task
func (b *Base) Kill() error {
	b.access.Lock()
	defer b.access.Unlock()
	if b.Killable == false && b.Status != FINISHED && b.Status != FAILED {
		return fmt.Errorf("Task %s(%s) is not cancellable or is not running anymore", b.ID, b.Name)
	}
	b.killable.Kill(ErrKilledByUser)
	return nil
}

// Copy returns a copy of the task base
func (b *Base) Copy() Base {
	b.access.Lock()
	baseCopy := *b
	b.access.Unlock()
	return baseCopy
}

// Dying returns a channel that can be used to listen for the kill command
func (b *Base) Dying() <-chan struct{} {
	return b.killable.Dying()
}

// Save sends a copy of the running task to the task scheduler
func (b *Base) Save() {
	saveTask(b)
}

// Wait waits for the task to finish and returns an error if there was one. Used to mimic a blocking call
func (b *Base) Wait() error {
	err := <-b.finish
	return err
}

// Run starts the task
func (b *Base) Run() {
	b.SetStatus(INPROGRESS)
	b.Save()

	// run custom task
	err := b.custom.Run()
	// update final result and save task
	b.access.Lock()
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
	b.access.Unlock()
	b.Save()
	// return error on finish channel
	b.finish <- err
}
