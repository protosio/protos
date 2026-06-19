//go:build !darwin

package localvm

import "fmt"

// VMNetSelftest is only implemented on darwin, where vmnet exists.
func VMNetSelftest() error {
	return fmt.Errorf("vmnet selftest is only supported on macOS")
}
