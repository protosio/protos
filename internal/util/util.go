package util

import (
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// log is the global logger
var log = logrus.New()

// SetLogLevel sets the log level for the application
func SetLogLevel(level logrus.Level) {
	log.Formatter = &logrus.TextFormatter{FullTimestamp: true, QuoteEmptyFields: true}
	log.Level = level
}

// GetLogger returns the main logger
func GetLogger(context string) *logrus.Entry {
	return log.WithField("context", context)
}

// WaitForPort is a utility method that waits until a specific port is open on a specific host
func WaitForPort(host string, port string, maxTries int) error {
	tries := 0
	for {
		timeout := time.Second
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)
		if err == nil && conn != nil {
			conn.Close()
			return nil
		}
		time.Sleep(3 * time.Second)
		tries++
		if tries == maxTries {
			return fmt.Errorf("failed to connect to '%s:%s' after %d tries", host, port, maxTries)
		}
	}
}
