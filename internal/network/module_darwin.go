//go:build darwin

package network

import (
	"fmt"
	"os"
	"strings"

	networkmodule "github.com/protosio/protos/internal/network/module"
	ipcmodule "github.com/protosio/protos/network/modules/ipc"
	networkextension "github.com/protosio/protos/network/modules/networkextension"
	wireguardmodule "github.com/protosio/protos/network/modules/wireguard"
)

const NetworkModuleEnv = "PROTOS_NETWORK_MODULE"

func newModule() (networkmodule.Module, error) {
	switch moduleName := strings.ToLower(os.Getenv(NetworkModuleEnv)); moduleName {
	case "", "ipc", "daemon", "networkd":
		return ipcmodule.New()
	case "wireguard", "wireguard-go", "tun":
		return wireguardmodule.New()
	case "networkextension", "network-extension", "ne":
		return networkextension.New()
	default:
		return nil, fmt.Errorf("unknown network module %q", moduleName)
	}
}
