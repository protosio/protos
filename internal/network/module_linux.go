//go:build linux

package network

import (
	"fmt"
	"os"
	"strings"

	networkmodule "github.com/protosio/protos/internal/network/module"
	ipcmodule "github.com/protosio/protos/network/modules/ipc"
	wireguardmodule "github.com/protosio/protos/network/modules/wireguard"
)

const NetworkModuleEnv = "PROTOS_NETWORK_MODULE"

func newModule() (networkmodule.Module, error) {
	switch moduleName := strings.ToLower(os.Getenv(NetworkModuleEnv)); moduleName {
	case "", "wireguard", "wireguard-go", "tun":
		return wireguardmodule.New()
	case "ipc":
		return ipcmodule.New()
	default:
		return nil, fmt.Errorf("unknown network module %q", moduleName)
	}
}
