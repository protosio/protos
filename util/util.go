package util

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
)

var Log = logrus.New()

// SetLogLevel sets the log level for the application
func SetLogLevel(level logrus.Level) {
	Log.Level = level
}

// HTTPLogger is a logger used in the http middleware
func HTTPLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		Log.Debugf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}
