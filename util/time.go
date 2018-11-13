package util

import (
	"fmt"
	"time"

	"github.com/vjeantet/jodaTime"
)

// ProtosTime is a custom time that formats to a shorter form in all JSON messages
type ProtosTime time.Time

// MarshalJSON is a customer JSON marshallers for the ProtosTime
func (pt *ProtosTime) MarshalJSON() ([]byte, error) {
	t := time.Time(*pt)
	stamp := fmt.Sprintf("\"%s\"", jodaTime.Format("yyyyMMdd'T'HHmmss.SSSZ", t))
	return []byte(stamp), nil
}

// GobEncode is a custom gob encoder for ProtosTime. Gob is used as a serialisation format to store stuff in the db
func (pt *ProtosTime) GobEncode() ([]byte, error) {
	t := time.Time(*pt)
	return t.MarshalBinary()
}

// GobDecode is a custom gob decoder for ProtosTime. Gob is used as a serialisation format to store stuff in the db
func (pt *ProtosTime) GobDecode(buf []byte) error {
	t := time.Time(*pt)
	err := t.UnmarshalBinary(buf)
	if err != nil {
		return err
	}
	*pt = ProtosTime(t)
	return nil
}
