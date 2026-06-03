//go:build darwin

package wireguard

import (
	"fmt"
	"net"
	"net/netip"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

var darwinRouteSeq atomic.Int32

func addDarwinRoute(spec routeSpec) error {
	return applyDarwinRoute(unix.RTM_ADD, spec)
}

func deleteDarwinRoute(spec routeSpec) error {
	return applyDarwinRoute(unix.RTM_DELETE, spec)
}

func applyDarwinRoute(routeType int, spec routeSpec) error {
	msg, err := darwinRouteMessage(routeType, spec)
	if err != nil {
		return err
	}
	return writeDarwinRouteMessage(msg)
}

func darwinRouteMessage(routeType int, spec routeSpec) (*route.RouteMessage, error) {
	prefix, err := netip.ParsePrefix(spec.destination)
	if err != nil {
		return nil, fmt.Errorf("invalid route destination %q: %w", spec.destination, err)
	}
	if spec.family == inetRoute && !prefix.Addr().Is4() {
		return nil, fmt.Errorf("route %s is not IPv4", spec.destination)
	}
	if spec.family == inet6Route && !prefix.Addr().Is6() {
		return nil, fmt.Errorf("route %s is not IPv6", spec.destination)
	}

	addrs := make([]route.Addr, unix.RTAX_MAX)
	addrs[unix.RTAX_DST] = routeAddrFromIP(prefix.Addr())
	addrs[unix.RTAX_NETMASK] = routeNetmaskAddr(prefix)

	flags := unix.RTF_UP | unix.RTF_STATIC
	if prefix.Bits() == prefix.Addr().BitLen() {
		flags |= unix.RTF_HOST
	}

	index := 0
	if spec.gateway != "" {
		gateway, err := netip.ParseAddr(spec.gateway)
		if err != nil {
			return nil, fmt.Errorf("invalid route gateway %q: %w", spec.gateway, err)
		}
		addrs[unix.RTAX_GATEWAY] = routeAddrFromIP(gateway)
		flags |= unix.RTF_GATEWAY
	} else if spec.iface != "" {
		iface, err := net.InterfaceByName(spec.iface)
		if err != nil {
			return nil, fmt.Errorf("find interface %s: %w", spec.iface, err)
		}
		index = iface.Index
		addrs[unix.RTAX_GATEWAY] = &route.LinkAddr{Index: iface.Index, Name: iface.Name}
	}

	return &route.RouteMessage{
		Type:  routeType,
		Flags: flags,
		Index: index,
		ID:    uintptr(os.Getpid()),
		Seq:   int(darwinRouteSeq.Add(1)),
		Addrs: addrs,
	}, nil
}

func writeDarwinRouteMessage(msg *route.RouteMessage) error {
	payload, err := msg.Marshal()
	if err != nil {
		return err
	}
	fd, err := unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	timeout := unix.NsecToTimeval(time.Second.Nanoseconds())
	_ = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &timeout)
	if _, err := unix.Write(fd, payload); err != nil {
		return err
	}

	buf := make([]byte, os.Getpagesize())
	for {
		n, err := unix.Read(fd, buf)
		if err != nil {
			return err
		}
		messages, err := route.ParseRIB(route.RIBTypeRoute, buf[:n])
		if err != nil {
			return err
		}
		for _, parsed := range messages {
			routeMsg, ok := parsed.(*route.RouteMessage)
			if !ok || routeMsg.ID != msg.ID || routeMsg.Seq != msg.Seq {
				continue
			}
			return routeMsg.Err
		}
	}
}

func defaultRouteTarget(family string) (routeTarget, error) {
	af := unix.AF_INET6
	if family == inetRoute {
		af = unix.AF_INET
	}
	rib, err := route.FetchRIB(af, route.RIBTypeRoute, 0)
	if err != nil {
		return routeTarget{}, err
	}
	messages, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return routeTarget{}, err
	}
	for _, parsed := range messages {
		routeMsg, ok := parsed.(*route.RouteMessage)
		if !ok || !isDefaultRouteMessage(routeMsg, family) {
			continue
		}
		target := routeTarget{gateway: gatewayString(routeAddrAt(routeMsg, unix.RTAX_GATEWAY))}
		if routeMsg.Index != 0 {
			if iface, err := net.InterfaceByIndex(routeMsg.Index); err == nil {
				target.iface = iface.Name
			}
		}
		if target.gateway != "" || target.iface != "" {
			return target, nil
		}
	}
	return routeTarget{}, fmt.Errorf("could not find default route gateway or interface")
}

func isDefaultRouteMessage(msg *route.RouteMessage, family string) bool {
	dst := routeAddrAt(msg, unix.RTAX_DST)
	mask := routeAddrAt(msg, unix.RTAX_NETMASK)
	if !routeAddrZero(dst, family) {
		return false
	}
	return mask == nil || routeAddrZero(mask, family)
}

func routeAddrAt(msg *route.RouteMessage, index int) route.Addr {
	if index < 0 || index >= len(msg.Addrs) {
		return nil
	}
	return msg.Addrs[index]
}

func gatewayString(addr route.Addr) string {
	switch value := addr.(type) {
	case *route.Inet4Addr:
		return netip.AddrFrom4(value.IP).String()
	case *route.Inet6Addr:
		return netip.AddrFrom16(value.IP).String()
	default:
		return ""
	}
}

func routeAddrZero(addr route.Addr, family string) bool {
	if addr == nil {
		return true
	}
	switch value := addr.(type) {
	case *route.Inet4Addr:
		return family == inetRoute && value.IP == [4]byte{}
	case *route.Inet6Addr:
		return family == inet6Route && value.IP == [16]byte{}
	default:
		return false
	}
}

func routeAddrFromIP(addr netip.Addr) route.Addr {
	if addr.Is4() {
		octets := addr.As4()
		return &route.Inet4Addr{IP: octets}
	}
	octets := addr.As16()
	return &route.Inet6Addr{IP: octets}
}

func routeNetmaskAddr(prefix netip.Prefix) route.Addr {
	mask := prefixMaskBytes(prefix.Bits(), prefix.Addr().BitLen())
	if prefix.Addr().Is4() {
		return &route.Inet4Addr{IP: [4]byte{mask[0], mask[1], mask[2], mask[3]}}
	}
	return &route.Inet6Addr{IP: mask}
}

func prefixMaskBytes(ones int, bits int) [16]byte {
	var out [16]byte
	mask := net.CIDRMask(ones, bits)
	copy(out[:], mask)
	return out
}
