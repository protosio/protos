package task

import (
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/emirpasic/gods/maps/linkedhashmap"
	"github.com/pkg/errors"
	"github.com/protosio/protos/database"
	"github.com/protosio/protos/util"
	"github.com/rs/xid"
)

// tasks is a thread safe tasks map
type taskContainer struct {
	access *sync.Mutex
	all    *linkedhashmap.Map
}

// put saves an task into the task map
func (tm taskContainer) put(id string, task *Base) {
	tm.access.Lock()
	tm.all.Put(id, task)
	tm.access.Unlock()
}

// get retrieves a task from the task map
func (tm taskContainer) get(id string) (*Base, error) {
	tm.access.Lock()
	task, found := tm.all.Get(id)
	tm.access.Unlock()
	if found {
		return task.(*Base), nil
	}
	return nil, fmt.Errorf("Could not find task %s", id)
}

// remove removes a task from the task map
func (tm taskContainer) remove(id string) error {
	tm.access.Lock()
	defer tm.access.Unlock()
	_, found := tm.all.Get(id)
	if found == false {
		return fmt.Errorf("Could not find task %s", id)
	}
	tm.all.Remove(id)
	return nil
}

// copy returns a copy of the task map
func (tm taskContainer) copy() *linkedhashmap.Map {
	tsks := linkedhashmap.New()
	tm.access.Lock()
	it := tm.all.Iterator()
	for it.Next() {
		key, value := it.Key(), it.Value()
		tsk := value.(*Base)
		tsk.access.Lock()
		ltsk := *tsk
		tsk.access.Unlock()
		tsks.Put(key, &ltsk)
	}
	tm.access.Unlock()
	return tsks
}

// all tasks
var tasks taskContainer

func initDB() {
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
		task.access = &sync.Mutex{}
		ltasks.Put(task.ID, task)
	}
	tasks = taskContainer{access: &sync.Mutex{}, all: ltasks}
}

func saveTask(btsk *Base) {
	log.WithField("proc", "taskManager").Debugf("Saving task %s to database", btsk.ID)
	btsk.access.Lock()
	ltask := *btsk
	btsk.access.Unlock()
	gconfig.WSPublish <- util.WSMessage{MsgType: util.WSMsgTypeUpdate, PayloadType: util.WSPayloadTypeTask, PayloadValue: ltask}
	err := database.Save(&ltask)
	if err != nil {
		log.Panic(errors.Wrapf(err, "Could not save task %s to database", ltask.ID))
	}
}

func getLastNTasks(n int, tsks *linkedhashmap.Map) linkedhashmap.Map {
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
// Public methods
//

// Init initializes the app package by loading all the applications from the database
func Init() {
	log.Info("Initializing application manager")
	initDB()
}

// New creates a new task and returns it
func New(tsk Task) Task {
	ts := util.ProtosTime(time.Now())
	base := &Base{
		access: &sync.Mutex{},

		ID:        xid.New().String(),
		Name:      tsk.Name(),
		Status:    REQUESTED,
		Progress:  Progress{Percentage: 0},
		StartedAt: &ts,

		finish: make(chan error, 1),
	}
	base.Save()
	tsk.SetBase(base)
	tasks.put(base.ID, base)
	go tsk.Run()
	return tsk
}

// GetAll returns all the available tasks
func GetAll() *linkedhashmap.Map {
	return tasks.copy()
}

// GetLast returns last 36 available tasks
func GetLast() linkedhashmap.Map {
	tasksCopy := tasks.copy()
	return getLastNTasks(36, tasksCopy)
}

// Get returns a task based on its id
func Get(id string) (*Base, error) {
	return tasks.get(id)
}

// GetIDs returns all tasks for the provided ids
func GetIDs(ids []string) linkedhashmap.Map {
	tasksCopy := tasks.copy()
	filter := func(k interface{}, v interface{}) bool {
		if found, _ := util.StringInSlice(k.(string), ids); found {
			return true
		}
		return false
	}
	selectedTasks := tasksCopy.Select(filter)
	return getLastNTasks(10, selectedTasks)
}
