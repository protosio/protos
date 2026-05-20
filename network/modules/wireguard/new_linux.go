//go:build linux

package wireguard

import (
	"fmt"

	wglink "github.com/protosio/protos/internal/wireguard"
)

func New() (*Module, error) {
	linkManager, err := wglink.NewManager()
	if err != nil {
		return nil, fmt.Errorf("wireguard module: %w", err)
	}

	dnsManager, err := NewDNSManager()
	if err != nil {
		_ = linkManager.Close()
		return nil, fmt.Errorf("wireguard module: %w", err)
	}

	return &Module{
		dnsManager: dnsManager,
		platformState: platformState{
			linkManager: linkManager,
		},
	}, nil
}
