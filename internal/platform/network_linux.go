package platform

import (
	"fmt"
	"syscall"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/vishvananda/netlink"
)

func initBridge(name string) error {

	brInterface := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name,
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(brInterface)
	if err != nil && err != syscall.EEXIST {
		return fmt.Errorf("Failed to create bridge interface '%q': %v", name, err)
	}

	l, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("Could not find newly created bridge interface '%q': %v", name, err)
	}
	brInterface, ok := l.(*netlink.Bridge)
	if !ok {
		return fmt.Errorf("Interface '%q' found but is not a bridge", name)
	}

	_, _ = sysctl.Sysctl(fmt.Sprintf("net/ipv6/conf/%s/accept_ra", name), "0")

	if err := netlink.LinkSetUp(brInterface); err != nil {
		return err
	}

	return nil
}
