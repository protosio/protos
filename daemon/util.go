package daemon

import (
	"github.com/Sirupsen/logrus"
)

var log = logrus.New()

// SetLogLevel sets the log level for the application
func SetLogLevel(level logrus.Level) {
	log.Level = level
}
