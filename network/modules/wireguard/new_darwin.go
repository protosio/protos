//go:build darwin

package wireguard

import "fmt"

func New() (*Module, error) {
	dnsManager, err := NewDNSManager()
	if err != nil {
		return nil, fmt.Errorf("wireguard module: %w", err)
	}

	return &Module{
		dnsManager: dnsManager,
		platformState: platformState{
			routes: map[string]struct{}{},
			peers:  map[string]struct{}{},
		},
	}, nil
}
