//go:build !darwin && !linux

package wireguard

import (
	"fmt"
	"runtime"
)

type DNSManager struct{}

type platformState struct{}

func New() (*Module, error) {
	return nil, fmt.Errorf("wireguard module is not supported on %s", runtime.GOOS)
}

func (m *Module) closePlatform() error {
	return nil
}
