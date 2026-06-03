//go:build darwin

package wireguard

import (
	"net"
	"testing"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

func TestDarwinRouteMessageForInterfaceRoute(t *testing.T) {
	ifaces, err := net.Interfaces()
	if err != nil {
		t.Fatalf("list interfaces: %v", err)
	}
	if len(ifaces) == 0 {
		t.Fatal("no interfaces available for route message test")
	}
	iface := ifaces[0]
	msg, err := darwinRouteMessage(unix.RTM_ADD, routeSpec{
		family:      inetRoute,
		destination: "0.0.0.0/1",
		iface:       iface.Name,
	})
	if err != nil {
		t.Fatalf("route message: %v", err)
	}
	if msg.Type != unix.RTM_ADD {
		t.Fatalf("route type = %d, want RTM_ADD", msg.Type)
	}
	if msg.Flags&unix.RTF_GATEWAY != 0 {
		t.Fatal("interface route should not be marked as gateway route")
	}
	if msg.Index != iface.Index {
		t.Fatalf("route index = %d, want %d", msg.Index, iface.Index)
	}
	if _, ok := msg.Addrs[unix.RTAX_GATEWAY].(*route.LinkAddr); !ok {
		t.Fatalf("gateway addr = %T, want *route.LinkAddr", msg.Addrs[unix.RTAX_GATEWAY])
	}
	mask, ok := msg.Addrs[unix.RTAX_NETMASK].(*route.Inet4Addr)
	if !ok {
		t.Fatalf("netmask addr = %T, want *route.Inet4Addr", msg.Addrs[unix.RTAX_NETMASK])
	}
	if mask.IP != [4]byte{128, 0, 0, 0} {
		t.Fatalf("netmask = %v, want 128.0.0.0", mask.IP)
	}
}

func TestDarwinRouteMessageForGatewayHostRoute(t *testing.T) {
	msg, err := darwinRouteMessage(unix.RTM_ADD, routeSpec{
		family:      inetRoute,
		destination: "5.161.52.86/32",
		gateway:     "192.0.2.1",
	})
	if err != nil {
		t.Fatalf("route message: %v", err)
	}
	if msg.Flags&unix.RTF_GATEWAY == 0 {
		t.Fatal("gateway route should be marked as gateway route")
	}
	if msg.Flags&unix.RTF_HOST == 0 {
		t.Fatal("host route should be marked as host route")
	}
	gateway, ok := msg.Addrs[unix.RTAX_GATEWAY].(*route.Inet4Addr)
	if !ok {
		t.Fatalf("gateway addr = %T, want *route.Inet4Addr", msg.Addrs[unix.RTAX_GATEWAY])
	}
	if gateway.IP != [4]byte{192, 0, 2, 1} {
		t.Fatalf("gateway = %v, want 192.0.2.1", gateway.IP)
	}
}
