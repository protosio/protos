package daemon

import (
	"github.com/Sirupsen/logrus"
)

var log = logrus.New()

func SetLogLevel(level logrus.Level) {
	log.Level = level
}
