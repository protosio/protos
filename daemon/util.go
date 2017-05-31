package daemon

import (
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
)

var log = logrus.New()

// SetLogLevel sets the log level for the application
func SetLogLevel(level logrus.Level) {
	log.Level = level
}

func httpLogger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Debugf(
			"%s\t%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}
