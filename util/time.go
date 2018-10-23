package util

import (
	"fmt"
	"time"
)

// ProtosTime is a custom time that formats to a shorter form in all JSON messages
type ProtosTime time.Time

// MarshalJSON is a customer JSON marshallers for the ProtosTime
func (pt ProtosTime) MarshalJSON() ([]byte, error) {
	t := time.Time(pt)
	stamp := fmt.Sprintf("\"%s\"", t.Format(time.StampMilli))
	return []byte(stamp), nil
}
