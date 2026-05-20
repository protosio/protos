//go:build !cgo

package main

func runClientApp() error {
	if err := startClient(); err != nil {
		stopServers()
		return err
	}
	waitForInterrupt()
	return nil
}
