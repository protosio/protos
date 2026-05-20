package wireguard

import (
	"fmt"
	"sync"

	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/util"
)

type Module struct {
	mu         sync.Mutex
	dnsManager *DNSManager
	domain     string

	platformState
}

var _ networkmodule.Module = (*Module)(nil)

var log = util.GetLogger("network/wireguard")

func (m *Module) Name() string {
	return "wireguard"
}

func (m *Module) Close() error {
	return m.closePlatform()
}

func validateConfig(config networkmodule.Config) error {
	if !config.IPv6Address.IsValid() {
		return fmt.Errorf("network IPv6 address is not configured")
	}
	if !config.IPv6Address.Is6() {
		return fmt.Errorf("network address must be IPv6")
	}
	if config.WireGuardPrivateKey == "" {
		return fmt.Errorf("WireGuard private key is not configured")
	}
	return nil
}
