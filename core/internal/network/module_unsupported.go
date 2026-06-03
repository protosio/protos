//go:build !darwin && !linux

package network

import (
	"fmt"
	"runtime"

	networkmodule "github.com/protosio/protos/internal/network/module"
)

const NetworkModuleEnv = "PROTOS_NETWORK_MODULE"

func newModule() (networkmodule.Module, error) {
	return nil, fmt.Errorf("no network module is available for %s", runtime.GOOS)
}
