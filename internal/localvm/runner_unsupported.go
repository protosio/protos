//go:build !darwin

package localvm

import "fmt"

func Run(manifestPath string) error {
	return fmt.Errorf("local macOS VM host agent is only available on macOS")
}
