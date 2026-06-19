package db

import (
	"errors"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/protosio/protos/internal/util"
)

type Notifier interface {
	Notify()
}

type ChangeNotifier interface {
	Notifier
	NotifyChange(tableNames []string)
}

type ChangeEvent struct {
	TableNames []string
}

type tableChangeCallback struct {
	tableName string
	notifier  Notifier
}

var notifyLog = util.GetLogger("db")

var asyncNotify = struct {
	mu     sync.Mutex
	states map[uintptr]*asyncNotifyState
}{
	states: map[uintptr]*asyncNotifyState{},
}

type asyncNotifyState struct {
	running       bool
	pendingAll    bool
	pendingTables map[string]struct{}
}

func (db *DB) RegisterNotifier(model any, notifier Notifier) error {
	tableName := getTableName(model)
	if tableName == "" {
		return errors.New("model does not have a table name tag")
	}
	db.RegisterTableChangeCallback(tableName, notifier)
	return nil
}

func getTableName(model any) string {
	typ := reflect.TypeOf(model)
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	field, ok := typ.FieldByName("TableStruct")
	if !ok {
		return ""
	}

	tag := field.Tag.Get("sq")
	return tag
}

func notifyChangeAsync(notifier Notifier, tableNames []string) {
	id, ok := notifierIdentity(notifier)
	if !ok {
		go notifySafely(notifier, tableNames)
		return
	}

	asyncNotify.mu.Lock()
	state := asyncNotify.states[id]
	if state == nil {
		state = &asyncNotifyState{}
		asyncNotify.states[id] = state
	}
	state.add(tableNames)
	if state.running {
		asyncNotify.mu.Unlock()
		return
	}
	state.running = true
	nextTables := state.drain()
	asyncNotify.mu.Unlock()

	go func() {
		for {
			notifySafely(notifier, nextTables)

			asyncNotify.mu.Lock()
			if !state.hasPending() {
				state.running = false
				asyncNotify.mu.Unlock()
				return
			}
			nextTables = state.drain()
			asyncNotify.mu.Unlock()
		}
	}()
}

func (s *asyncNotifyState) add(tableNames []string) {
	if len(tableNames) == 0 {
		s.pendingAll = true
		s.pendingTables = nil
		return
	}
	if s.pendingAll {
		return
	}
	if s.pendingTables == nil {
		s.pendingTables = map[string]struct{}{}
	}
	for _, tableName := range tableNames {
		if tableName != "" {
			s.pendingTables[tableName] = struct{}{}
		}
	}
}

func (s *asyncNotifyState) hasPending() bool {
	return s.pendingAll || len(s.pendingTables) > 0
}

func (s *asyncNotifyState) drain() []string {
	if s.pendingAll {
		s.pendingAll = false
		s.pendingTables = nil
		return nil
	}
	if len(s.pendingTables) == 0 {
		return nil
	}
	tableNames := make([]string, 0, len(s.pendingTables))
	for tableName := range s.pendingTables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)
	s.pendingTables = nil
	return tableNames
}

func notifySafely(notifier Notifier, tableNames []string) {
	defer func() {
		if r := recover(); r != nil {
			notifyLog.Errorf("database notifier panic: %v", r)
		}
	}()
	if changeNotifier, ok := notifier.(ChangeNotifier); ok {
		changeNotifier.NotifyChange(tableNames)
		return
	}
	notifier.Notify()
}

func StartPeriodicNotifier(notifier Notifier, interval time.Duration) func() error {
	if notifier == nil {
		return func() error { return nil }
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				notifier.Notify()
			case <-done:
				return
			}
		}
	}()

	var once sync.Once
	return func() error {
		once.Do(func() {
			close(done)
		})
		return nil
	}
}

func notifierIdentity(notifier Notifier) (uintptr, bool) {
	value := reflect.ValueOf(notifier)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return 0, false
	}
	return value.Pointer(), true
}
