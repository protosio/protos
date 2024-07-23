package db

import (
	"errors"
	"reflect"

	"github.com/nustiueudinastea/doltswarm"
)

type Notifier doltswarm.Notifier

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
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	field, ok := typ.FieldByName("TableStruct")
	if !ok {
		return ""
	}

	tag := field.Tag.Get("sq")
	return tag
}
