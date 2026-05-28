//go:build darwin

package wireguard

import (
	"fmt"
	"net/netip"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	darwinIfNameSize = 16

	darwinSIOCAIFADDRIn6 = 0x8080691a
	darwinSIOCDIFADDRIn6 = 0x81206919

	darwinIn6IfreqSize       = 288
	darwinIn6AliasreqSize    = 128
	darwinIn6AddrlifetimeMax = 0xffffffff
	darwinPFInet6            = 30
	darwinIPV6CTLForwarding  = 1
)

type darwinIfreq struct {
	Name [darwinIfNameSize]byte
	Data [16]byte
}

type darwinIfAliasreq struct {
	Name      [darwinIfNameSize]byte
	Addr      unix.RawSockaddrInet4
	Broadaddr unix.RawSockaddrInet4
	Mask      unix.RawSockaddrInet4
}

type darwinIn6Ifreq struct {
	Name [darwinIfNameSize]byte
	Addr unix.RawSockaddrInet6
	Pad  [244]byte
}

type darwinIn6Aliasreq struct {
	Name       [darwinIfNameSize]byte
	Addr       unix.RawSockaddrInet6
	Dstaddr    unix.RawSockaddrInet6
	Prefixmask unix.RawSockaddrInet6
	Flags      int32
	Lifetime   darwinIn6Addrlifetime
}

type darwinIn6Addrlifetime struct {
	Expire    int64
	Preferred int64
	Vltime    uint32
	Pltime    uint32
}

func setInterfaceUp(iface string) error {
	return withIOCTLSocket(unix.AF_INET, func(fd int) error {
		var req darwinIfreq
		if err := copyDarwinIfName(req.Name[:], iface); err != nil {
			return err
		}
		if err := ioctlDarwin(fd, unix.SIOCGIFFLAGS, unsafe.Pointer(&req)); err != nil {
			return fmt.Errorf("get %s flags: %w", iface, err)
		}
		flags := (*int16)(unsafe.Pointer(&req.Data[0]))
		*flags |= int16(unix.IFF_UP)
		if err := ioctlDarwin(fd, unix.SIOCSIFFLAGS, unsafe.Pointer(&req)); err != nil {
			return fmt.Errorf("set %s up: %w", iface, err)
		}
		return nil
	})
}

func addInterfaceIPv6Address(iface string, address string) error {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return err
	}
	if !addr.Is6() || addr.Is4In6() {
		return fmt.Errorf("invalid IPv6 interface address %q", address)
	}

	return withIOCTLSocket(unix.AF_INET6, func(fd int) error {
		var req darwinIn6Aliasreq
		if err := copyDarwinIfName(req.Name[:], iface); err != nil {
			return err
		}
		req.Addr = rawSockaddrInet6(addr.As16())
		req.Prefixmask = rawSockaddrInet6(prefixMaskBytes(128, 128))
		req.Lifetime.Vltime = darwinIn6AddrlifetimeMax
		req.Lifetime.Pltime = darwinIn6AddrlifetimeMax
		if err := ioctlDarwin(fd, darwinSIOCAIFADDRIn6, unsafe.Pointer(&req)); err != nil {
			return fmt.Errorf("add IPv6 address %s to %s: %w", address, iface, err)
		}
		return nil
	})
}

func deleteInterfaceIPv6Address(iface string, address string) error {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return err
	}
	if !addr.Is6() || addr.Is4In6() {
		return fmt.Errorf("invalid IPv6 interface address %q", address)
	}

	return withIOCTLSocket(unix.AF_INET6, func(fd int) error {
		var req darwinIn6Ifreq
		if err := copyDarwinIfName(req.Name[:], iface); err != nil {
			return err
		}
		req.Addr = rawSockaddrInet6(addr.As16())
		if err := ioctlDarwin(fd, darwinSIOCDIFADDRIn6, unsafe.Pointer(&req)); err != nil {
			return fmt.Errorf("delete IPv6 address %s from %s: %w", address, iface, err)
		}
		return nil
	})
}

func addInterfaceIPv4Address(iface string, address netip.Addr) error {
	if !address.IsValid() || !address.Is4() {
		return fmt.Errorf("invalid IPv4 interface address %q", address.String())
	}

	return withIOCTLSocket(unix.AF_INET, func(fd int) error {
		var req darwinIfAliasreq
		if err := copyDarwinIfName(req.Name[:], iface); err != nil {
			return err
		}
		octets := address.As4()
		req.Addr = rawSockaddrInet4(octets)
		req.Broadaddr = rawSockaddrInet4(octets)
		req.Mask = rawSockaddrInet4([4]byte{255, 255, 255, 255})
		if err := ioctlDarwin(fd, unix.SIOCAIFADDR, unsafe.Pointer(&req)); err != nil {
			return fmt.Errorf("add IPv4 address %s to %s: %w", address.String(), iface, err)
		}
		return nil
	})
}

func deleteInterfaceIPv4Address(iface string, address string) error {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return err
	}
	if !addr.Is4() {
		return fmt.Errorf("invalid IPv4 interface address %q", address)
	}

	return withIOCTLSocket(unix.AF_INET, func(fd int) error {
		var req darwinIfreq
		if err := copyDarwinIfName(req.Name[:], iface); err != nil {
			return err
		}
		octets := addr.As4()
		req.setIPv4Address(octets)
		if err := ioctlDarwin(fd, unix.SIOCDIFADDR, unsafe.Pointer(&req)); err != nil {
			return fmt.Errorf("delete IPv4 address %s from %s: %w", address, iface, err)
		}
		return nil
	})
}

func sysctlUint32(name string) (uint32, error) {
	value, err := unix.SysctlUint32(name)
	if err != nil {
		return 0, fmt.Errorf("read sysctl %s: %w", name, err)
	}
	return value, nil
}

func setSysctlUint32(name string, value uint32) error {
	var mib []int32
	switch name {
	case ipv6ForwardingSysctl:
		mib = []int32{unix.CTL_NET, darwinPFInet6, unix.IPPROTO_IPV6, darwinIPV6CTLForwarding}
	default:
		return fmt.Errorf("unsupported sysctl %s", name)
	}
	if err := setDarwinSysctlUint32(mib, value); err != nil {
		return fmt.Errorf("set sysctl %s: %w", name, err)
	}
	return nil
}

func (req *darwinIfreq) setIPv4Address(addr [4]byte) {
	raw := rawSockaddrInet4(addr)
	*(*unix.RawSockaddrInet4)(unsafe.Pointer(&req.Data[0])) = raw
}

func rawSockaddrInet4(addr [4]byte) unix.RawSockaddrInet4 {
	return unix.RawSockaddrInet4{
		Len:    unix.SizeofSockaddrInet4,
		Family: unix.AF_INET,
		Addr:   addr,
	}
}

func rawSockaddrInet6(addr [16]byte) unix.RawSockaddrInet6 {
	return unix.RawSockaddrInet6{
		Len:    unix.SizeofSockaddrInet6,
		Family: unix.AF_INET6,
		Addr:   addr,
	}
}

func copyDarwinIfName(dst []byte, iface string) error {
	if len(iface) >= len(dst) {
		return fmt.Errorf("interface name %q is too long", iface)
	}
	clear(dst)
	copy(dst, iface)
	return nil
}

func withIOCTLSocket(family int, fn func(fd int) error) error {
	fd, err := unix.Socket(family, unix.SOCK_DGRAM, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	return fn(fd)
}

func ioctlDarwin(fd int, req uint, data unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), uintptr(req), uintptr(data))
	if errno != 0 {
		return errno
	}
	return nil
}

func setDarwinSysctlUint32(mib []int32, value uint32) error {
	if len(mib) == 0 {
		return fmt.Errorf("empty sysctl mib")
	}
	_, _, errno := unix.Syscall6(
		unix.SYS___SYSCTL,
		uintptr(unsafe.Pointer(&mib[0])),
		uintptr(len(mib)),
		0,
		0,
		uintptr(unsafe.Pointer(&value)),
		unsafe.Sizeof(value),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
