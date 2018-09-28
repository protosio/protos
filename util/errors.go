package util

// ProtosError is a custom error that implements the error interface, used to convey some extra information
type ProtosError struct {
	err  string
	Type int
}

func (e *ProtosError) Error() string {
	return e.err
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
