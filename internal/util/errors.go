package util

import (
	"errors"
)

// ProtosError is a custom error that implements the error interface, used to convey some extra information
type ProtosError struct {
	Msg  string
	Type int
}

func (e *ProtosError) Error() string {
	return e.Msg
}

// IsErrorType takes an error and checks if it matched the Protos error type
func IsErrorType(err error, etype int) bool {
	var protosErr *ProtosError
	return errors.As(err, &protosErr) && protosErr.Type == etype
}
