package core

import "github.com/emirpasic/gods/maps/linkedhashmap"

// TaskManager manages all tasks
type TaskManager interface {
	New(name string, customTask CustomTask) Task
	Get(id string) (Task, error)
	GetAll() *linkedhashmap.Map
	GetIDs(ids []string) linkedhashmap.Map
	GetLast() linkedhashmap.Map
}

// CustomTask is the interface that is implemented by custom tasks in various packages
type CustomTask interface {
	Run(parent Task, id string, progress Progress) error
}

// Progress is an interface used to communicate progress inside a task
type Progress interface {
	SetPercentage(percent int)
	SetState(stateText string)
}

// Task is the interface that all task types need to adhere too
type Task interface {
	GetID() string
	Wait() error
	SetPercentage(percentage int)
	GetPercentage() int
	SetState(stateText string)
	SetStatus(statusText string)
	AddApp(id string)
	Copy() Task
	SetKillable()
	Kill() error
	Dying() <-chan struct{}
	Save()
}
