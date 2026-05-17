package db

import (
	"errors"
	"reflect"

	"github.com/protosio/protos/internal/util"
)

type Notifier interface {
	Notify()
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
	go func() {
		defer func() {
			if r := recover(); r != nil {
				notifyLog.Errorf("database notifier panic: %v", r)
			}
		}()
		notifier.Notify()
	}()
}

func notifierIdentity(notifier Notifier) (uintptr, bool) {
	value := reflect.ValueOf(notifier)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return 0, false
	}
	return value.Pointer(), true
}
