//go:build !darwin

package daemon

func ensureVMRunnerEntitled(executable string) error {
	return nil
}
