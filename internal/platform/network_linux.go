package platform

import (
	"fmt"
	"net"
	"syscall"

	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	"github.com/vishvananda/netlink"
)

func initBridge(name string) (*netlink.Bridge, error) {

	brInterface := &netlink.Bridge{
		LinkAttrs: netlink.LinkAttrs{
			Name:   name,
			TxQLen: -1,
		},
	}

	err := netlink.LinkAdd(brInterface)
	if err != nil && err != syscall.EEXIST {
		return nil, fmt.Errorf("Failed to create bridge interface '%q': %v", name, err)
	}

	l, err := netlink.LinkByName(name)
	if err != nil {
		return nil, fmt.Errorf("Could not find newly created bridge interface '%q': %v", name, err)
	}
	brInterface, ok := l.(*netlink.Bridge)
	if !ok {
		return nil, fmt.Errorf("Interface '%q' found but is not a bridge", name)
	}

	_, _ = sysctl.Sysctl(fmt.Sprintf("net/ipv6/conf/%s/accept_ra", name), "0")

	if err := netlink.LinkSetUp(brInterface); err != nil {
		return nil, err
	}

	return brInterface, nil
}

func configureInterface(name string, ip net.IP) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("Failed to find interface %q: %v", name, err)
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("Failed to bring interface %q UP: %v", name, err)
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{Mask: net.CIDRMask(24, 32), IP: ip}, Label: ""}
	if err = netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("Failed to configure IP address '%s' on interface %v", ip.String(), err)
	}

	return nil
}
