package task

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/protosio/protos/internal/core"
	"github.com/protosio/protos/internal/util"

	"github.com/icholy/killable"
	"github.com/jinzhu/copier"
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

// ErrKilledByUser is returned when a task is canceled/killed by the user
var ErrKilledByUser = errors.New("Task cancelled by user")

// Progress tracks the percentage and message of a task
type Progress struct {
	Percentage int    `json:"percentage"`
	State      string `json:"state"`
}

// Base represents an (a)synchronous piece of work that Protos acts upon
type Base struct {
	access *sync.Mutex
	custom core.CustomTask
	parent *Manager

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
	if b.Killable == false || b.Status == FINISHED || b.Status == FAILED {
		return fmt.Errorf("Task %s(%s) is not cancellable or is not running anymore", b.ID, b.Name)
	}
	b.killable.Kill(ErrKilledByUser)
	return nil
}

// Copy returns a copy of the task base
func (b *Base) Copy() core.Task {
	var baseCopy Base
	b.access.Lock()
	err := copier.Copy(&baseCopy, b)
	b.access.Unlock()
	if err != nil {
		log.Panic(err)
	}
	return &baseCopy
}

// Dying returns a channel that can be used to listen for the kill command
func (b *Base) Dying() <-chan struct{} {
	return b.killable.Dying()
}

// Save sends a copy of the running task to the task scheduler
func (b *Base) Save() {
	b.parent.saveTask(b)
}

// Wait waits for the task to finish and returns an error if there was one. Used to mimic a blocking call
func (b *Base) Wait() error {
	err := <-b.finish
	return err
}

// Run starts the task
func (b *Base) Run() {
	log.Info("Running base task ", b.ID)
	b.SetStatus(INPROGRESS)
	b.Save()

	// run custom task
	err := b.custom.Run(b, b.ID, b)
	// update final result and save task
	b.access.Lock()
	b.Progress.Percentage = 100
	ts := util.ProtosTime(time.Now())
	b.FinishedAt = &ts
	if err != nil {
		log.WithField("proc", b.ID).Errorf("Failed to finish task '%s': %s", b.ID, err.Error())
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
