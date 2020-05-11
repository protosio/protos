package env

import (
	"github.com/protosio/protos/internal/core"
	"github.com/sirupsen/logrus"
)

// Env is a struct that containts program dependencies that get injected in other modules
type Env struct {
	DB  core.DBCLI
	Log *logrus.Logger
}

// New creates and returns an instance of Env
func New(db core.DBCLI, log *logrus.Logger) *Env {

	if db == nil || log == nil {
		panic("env: db || log should not be nil")
	}
	return &Env{DB: db, Log: log}
}
