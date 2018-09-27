package task

import (
	"time"

	"github.com/rs/xid"
)

const (
	// REQUESTED - task state created
	REQUESTED = "REQUESTED"
)

// Task represents an (a)synchronous piece of work that Protos acts upon
type Task struct {
	ID         string    `json:"id"`
	State      string    `json:"state"`
	Resolution string    `json:"resolution"`
	Progress   int       `json:"progress"`
	StartedAt  time.Time `json:"started-at"`
	FinishedAt time.Time `json:"finished-at"`
}

var tasks = make(map[string]*Task)

// New creates a new task and returns it
func New() (Task, error) {
	task := Task{
		ID:        xid.New().String(),
		State:     REQUESTED,
		Progress:  10,
		StartedAt: time.Now(),
	}
	tasks[task.ID] = &task
	return task, nil
}

// GetAll returns all the available tasks
func GetAll() map[string]*Task {
	return tasks
}
