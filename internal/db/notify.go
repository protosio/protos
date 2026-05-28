package db

import (
	"errors"
	"reflect"
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

func notifyAsync(notifier Notifier) {
	notifyChangeAsync(notifier, nil)
}

func notifyChangeAsync(notifier Notifier, tableNames []string) {
	go func() {
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
	}()
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
