package util

import (
	"fmt"
	"time"

	"github.com/attic-labs/noms/go/marshal"
	"github.com/attic-labs/noms/go/types"
	"github.com/attic-labs/noms/go/util/datetime"
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

// MarshalNoms encodes the resource into a noms value type
func (pt *ProtosTime) MarshalNoms(vrw types.ValueReadWriter) (val types.Value, err error) {
	dt := datetime.DateTime{time.Time(*pt)}
	return marshal.Marshal(vrw, dt)
}

// UnmarshalNoms decodes the resource value from a noms value type
func (pt *ProtosTime) UnmarshalNoms(v types.Value) error {
	var dt datetime.DateTime
	err := marshal.Unmarshal(v, &dt)
	if err != nil {
		return err
	}
	nt := ProtosTime(dt.Time)
	pt = &nt
	return nil
}
