package platform

import (
	"fmt"
	"net"
	"syscall"

	"github.com/containernetworking/cni/pkg/types/current"
	"github.com/containernetworking/plugins/pkg/ip"
	"github.com/containernetworking/plugins/pkg/ns"
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

func configureInterface(netNSpath string, IP net.IP, network net.IPNet) error {
	netns, err := ns.GetNS(netNSpath)
	if err != nil {
		return fmt.Errorf("Failed to open netns '%s': %v", netNSpath, err)
	}
	defer netns.Close()

	contIface := &current.Interface{}
	hostIface := &current.Interface{}
	err = netns.Do(func(hostNS ns.NetNS) error {
		// create the veth pair in the container and move host end into host netns
		name := "prts0"
		hostVeth, containerVeth, err := ip.SetupVeth(name, netBridge.MTU, hostNS)
		if err != nil {
			return err
		}
		contIface.Name = containerVeth.Name
		contIface.Mac = containerVeth.HardwareAddr.String()
		contIface.Sandbox = netns.Path()
		hostIface.Name = hostVeth.Name

		link, err := netlink.LinkByName(name)
		if err != nil {
			return fmt.Errorf("Failed to find interface %q: %v", name, err)
		}

		if err := netlink.LinkSetUp(link); err != nil {
			return fmt.Errorf("Failed to bring interface %q UP: %v", name, err)
		}

		addr := &netlink.Addr{IPNet: &net.IPNet{Mask: network.Mask, IP: IP}, Label: ""}
		if err = netlink.AddrAdd(link, addr); err != nil {
			return fmt.Errorf("Failed to configure IP address '%s' on interface %v", IP.String(), err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Failed to create veth pair: %v", err)
	}

	hostVeth, err := netlink.LinkByName(hostIface.Name)
	if err != nil {
		return fmt.Errorf("Failed to find host interface '%s': %v", hostIface.Name, err)
	}
	hostIface.Mac = hostVeth.Attrs().HardwareAddr.String()

	if err := netlink.LinkSetMaster(hostVeth, netBridge); err != nil {
		return fmt.Errorf("Failed to connect %q to bridge %v: %v", hostVeth.Attrs().Name, netBridge.Attrs().Name, err)
	}
	return nil
}

func getNetNSInterfaceIP(nsPath string, filterNetwork net.IPNet) (net.IP, error) {
	var ipAddr net.IP
	fn := func(_ ns.NetNS) error {
		interfaces, err := net.Interfaces()
		if err != nil {
			return err
		}

		for _, iface := range interfaces {
			addresses, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addresses {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil {
					continue
				}
				if filterNetwork.Contains(ip) {
					ipAddr = ip
					return nil
				}
			}
		}
		return nil
	}
	if err := ns.WithNetNSPath(nsPath, fn); err != nil {
		return ipAddr, err
	}
	return ipAddr, nil
}

// https://play.golang.org/p/m8TNTtygK0
func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func allocateIP(network net.IPNet, usedIPs map[string]bool) (net.IP, error) {
	allIPs := []net.IP{}
	for ip := network.IP.Mask(network.Mask); network.Contains(ip); incIP(ip) {
		newIP := make(net.IP, len(ip))
		copy(newIP, ip)
		allIPs = append(allIPs, newIP)
	}

	// starting from the 3 position in the slice to avoid allocating the network IP and the gateway
	for _, ip := range allIPs[2 : len(allIPs)-1] {
		if _, found := usedIPs[ip.String()]; !found {
			return ip, nil
		}
	}
	return nil, fmt.Errorf("Failed to allocate IP. No IP's left")
}
