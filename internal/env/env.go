package env

import (
	dbcli "github.com/protosio/protos/internal/dbcli"
	"github.com/sirupsen/logrus"
)

// Env is a struct that containts program dependencies that get injected in other modules
type Env struct {
	DB  dbcli.DB
	Log *logrus.Logger
}

// New creates and returns an instance of Env
func New(db dbcli.DB, log *logrus.Logger) *Env {

	if db == nil || log == nil {
		panic("env: db || log should not be nil")
	}
	return &Env{DB: db, Log: log}
}
