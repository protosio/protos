//go:build !darwin

package localvm

import "fmt"

func Run(manifestPath string) error {
	return fmt.Errorf("host agent local VM lifecycle is only available on macOS")
}
