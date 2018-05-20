package util

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// Log is the global logger
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

func assertAvailablePRNG() {
	// Assert that a cryptographically secure PRNG is available.
	// Panic otherwise.
	buf := make([]byte, 1)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		Log.Panicf("crypto/rand is unavailable: Read() failed with %#v", err.Error())
	}
}

// GenerateRandomBytes returns securely generated random bytes.
// From https://gist.github.com/shahaya/635a644089868a51eccd6ae22b2eb800
func GenerateRandomBytes(n int) ([]byte, error) {
	assertAvailablePRNG()
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// StringInSlice checks if provided string is in provided string list
func StringInSlice(a string, list []string) (bool, int) {
	for i, b := range list {
		if b == a {
			return true, i
		}
	}
	return false, 0
}

// RemoveStringFromSlice removes a string element from a slice, usind the provided index
func RemoveStringFromSlice(s []string, i int) []string {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

// String2SHA1 converts a string to a SHA1 hash, formatted as a hex string
func String2SHA1(s string) string {
	h := sha1.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
