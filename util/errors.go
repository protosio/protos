package util

import "strings"

// ProtosError is a custom error that implements the error interface, used to convey some extra information
type ProtosError struct {
	err  string
	Type int
}

func (e *ProtosError) Error() string {
	return e.err
}

// ErrorContainsTransform checks if an error contains a specific string and if it does, add the custom error type
func ErrorContainsTransform(err error, str string, errType int) error {
	if strings.Contains(err.Error(), str) {
		return &ProtosError{err: err.Error(), Type: errType}
	}
	return err
}

// NewError creates a new error without specifying a type
func NewError(msg string) error {
	return &ProtosError{err: msg}
}

// NewTypedError creates a new error and adds a Protos specific type
func NewTypedError(msg string, etype int) error {
	return &ProtosError{err: msg, Type: etype}
}

// IsErrorType takes an error and checks if it matached the Protos error type
func IsErrorType(err error, etype int) bool {
	switch e := err.(type) {
	case *ProtosError:
		if e.Type == etype {
			return true
		}
		return false
	default:
		return false
	}
}
