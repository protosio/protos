package core

import "github.com/emirpasic/gods/maps/linkedhashmap"

type TaskManager interface {
	New(CustomTask) Task
	Get(string) (Task, error)
	GetAll() *linkedhashmap.Map
	GetIDs([]string) linkedhashmap.Map
	GetLast() linkedhashmap.Map
}

// CustomTask is the interface that is implemented by custom tasks in various packages
type CustomTask interface {
	Run(string, Progress) error
	Name() string
}

type Progress interface {
	SetPercentage(int)
	SetState(string)
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
	Copy() Task
	SetKillable()
	Kill() error
	Dying() <-chan struct{}
	Save()
}
