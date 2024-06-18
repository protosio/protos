package wireguard

import (
	"errors"
	"fmt"
	"log"
	"net"
	"syscall"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type AddrScope int

type Address struct {
	net.IPNet
	Peer  *net.IPNet
	Scope AddrScope
}

type Route struct {
	Dest net.IPNet
	Src  net.IP
}

type Link interface {
	Interface() net.Interface
	Name() string
	Index() int

	IsUp() bool
	SetUp(bool) error
	Addrs() ([]Address, error)
	DelAddr(a Address) error
	AddAddr(a Address) error

	ConfigureWG(wgtypes.Config) error
	WGConfig() (*wgtypes.Device, error)

	DialUDP(local, remote net.UDPAddr) (*net.UDPConn, error)
	ListenUDP(local net.UDPAddr) (*net.UDPConn, error)

	GetRoutes() ([]Route, error)
	AddRoute(Route) error
	DelRoute(Route) error
}

type Manager interface {
	Links() ([]Link, error)
	CreateLink(name string) (Link, error)
	DelLink(name string) error
	GetLink(name string) (Link, error)

	Close() error
}

const (
	RouteProto = 157
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
