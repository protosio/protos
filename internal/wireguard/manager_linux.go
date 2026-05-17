//go:build linux

package wireguard

import (
	"errors"
	"fmt"
	"log"
	"syscall"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func CreateWG(m Manager, name string, cfg wgtypes.Config, addrs []Address) (link Link, created bool, err error) {
	link, err = m.GetLink(name)
	if err != nil {
		created = true
		link, err = m.CreateLink(name)
		if err != nil {
			return nil, false, fmt.Errorf("wg create: %w", err)
		}
	}

	if err := link.ConfigureWG(cfg); err != nil {
		return nil, false, fmt.Errorf("wg create: configure: %w", err)
	}

	if err := link.SetUp(true); err != nil {
		return nil, false, fmt.Errorf("wg create: set up: %w", err)
	}

	for i, addr := range addrs {
		if err := link.AddAddr(addr); err != nil {
			if errors.Is(err, syscall.EEXIST) {
				continue
			}
			if created {
				if delerr := m.DelLink(link.Name()); delerr != nil {
					log.Println("error:", delerr)
				}
			}
			return nil, false, fmt.Errorf("wg create: set addr %v: %w", i, err)
		}
	}

	return link, created, nil
}
