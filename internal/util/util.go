package util

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// log is the global logger
var log = logrus.New()

// Omit is field used for ommiting fields in public structs, when marshalling them to JSON or other serialization formats
type Omit *struct{}

// SetLogLevel sets the log level for the application
func SetLogLevel(level logrus.Level) {
	log.Formatter = &logrus.TextFormatter{FullTimestamp: true, QuoteEmptyFields: true}
	log.Level = level
}

// GetLogger returns the main logger
func GetLogger(context string) *logrus.Entry {
	return log.WithField("context", context)
}

func assertAvailablePRNG() {
	// Assert that a cryptographically secure PRNG is available.
	// Panic otherwise.
	buf := make([]byte, 1)

	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		log.Panicf("crypto/rand is unavailable: Read() failed with %#v", err.Error())
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

//HTTPBadResponse checks a HTTP response code and returns an error if its not ok
func HTTPBadResponse(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		return fmt.Errorf("Error (HTTP %d) while performing HTTP request: \"%s\"", resp.StatusCode, string(bodyBytes))
	}
	return nil
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

// WaitForHTTP is a utility method that waits until a specific URL returns a succesful response
func WaitForHTTP(url string, maxTries int) error {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	tries := 0
	for {
		resp, err := client.Get(url)
		if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		time.Sleep(3 * time.Second)
		tries++
		if tries == maxTries {
			return fmt.Errorf("failed to do HTTP req to '%s' after %d tries", url, maxTries)
		}
	}
}

func DelimReader(r io.Reader, delim byte) <-chan []byte {
	ch := make(chan []byte)

	go func() {
		buf := bufio.NewReader(r)

		for {
			bytes, err := buf.ReadBytes('\n')
			if len(bytes) != 0 {
				ch <- bytes
			}

			if err != nil {
				break
			}
		}

		close(ch)
	}()

	return ch
}
