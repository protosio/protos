//go:build darwin

package wireguard

import (
	"net/netip"
	"os"
	"testing"

	wgtun "golang.zx2c4.com/wireguard/tun"
)

type fakeTun struct {
	readCh       chan []byte
	readOffsetCh chan int
	writeCh      chan []byte
	events       chan wgtun.Event
}

func newFakeTun() *fakeTun {
	return &fakeTun{
		readCh:       make(chan []byte),
		readOffsetCh: make(chan int, 4),
		writeCh:      make(chan []byte, 4),
		events:       make(chan wgtun.Event),
	}
}

func (tun *fakeTun) File() *os.File { return nil }

func (tun *fakeTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	packet, ok := <-tun.readCh
	if !ok {
		return 0, os.ErrClosed
	}
	tun.readOffsetCh <- offset
	sizes[0] = copy(bufs[0][offset:], packet)
	return 1, nil
}

func (tun *fakeTun) Write(bufs [][]byte, offset int) (int, error) {
	for _, buf := range bufs {
		packet := make([]byte, len(buf)-offset)
		copy(packet, buf[offset:])
		tun.writeCh <- packet
	}
	return len(bufs), nil
}

func (tun *fakeTun) MTU() (int, error)     { return 1500, nil }
func (tun *fakeTun) Name() (string, error) { return "fake0", nil }
func (tun *fakeTun) Events() <-chan wgtun.Event {
	return tun.events
}
func (tun *fakeTun) Close() error {
	close(tun.readCh)
	return nil
}
func (tun *fakeTun) BatchSize() int { return 1 }

func TestHairpinTunFeedsPeerPacketsBackToReadPath(t *testing.T) {
	inner := newFakeTun()
	tun := newHairpinTun(inner)
	defer tun.Close()

	tun.setLocalAddress("200::1")
	tun.setHairpinRoutes(map[string]struct{}{
		"202::1/128": {},
	})

	packet := ipv6Packet(t, "203::1", "202::1")
	written, err := tun.Write([][]byte{packet}, 0)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 1 {
		t.Fatalf("written = %d, want 1", written)
	}
	select {
	case got := <-inner.writeCh:
		t.Fatalf("packet was written to inner TUN instead of hairpinned: %x", got)
	default:
	}

	buf := make([]byte, 128)
	sizes := make([]int, 1)
	read, err := tun.Read([][]byte{buf}, sizes, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read != 1 {
		t.Fatalf("read = %d, want 1", read)
	}
	if string(buf[:sizes[0]]) != string(packet) {
		t.Fatalf("read packet mismatch")
	}
	stats := tun.stats()
	if stats.HairpinnedPackets != 1 {
		t.Fatalf("hairpinned packets = %d, want 1", stats.HairpinnedPackets)
	}
	if stats.DroppedPackets != 0 {
		t.Fatalf("dropped packets = %d, want 0", stats.DroppedPackets)
	}
}

func TestHairpinTunPassesLocalPacketsToInnerTun(t *testing.T) {
	inner := newFakeTun()
	tun := newHairpinTun(inner)
	defer tun.Close()

	tun.setLocalAddress("200::1")
	tun.setHairpinRoutes(map[string]struct{}{
		"202::1/128": {},
	})

	packet := ipv6Packet(t, "203::1", "200::1")
	written, err := tun.Write([][]byte{packet}, 0)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if written != 1 {
		t.Fatalf("written = %d, want 1", written)
	}
	select {
	case got := <-inner.writeCh:
		if string(got) != string(packet) {
			t.Fatalf("inner packet mismatch")
		}
	default:
		t.Fatal("packet was not written to inner TUN")
	}
}

func TestHairpinTunReadsInnerTunWithDarwinPacketHeadroom(t *testing.T) {
	inner := newFakeTun()
	tun := newHairpinTun(inner)
	defer tun.Close()

	packet := ipv6Packet(t, "203::1", "200::1")
	inner.readCh <- packet

	buf := make([]byte, 128)
	sizes := make([]int, 1)
	read, err := tun.Read([][]byte{buf}, sizes, 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if read != 1 {
		t.Fatalf("read = %d, want 1", read)
	}
	if string(buf[:sizes[0]]) != string(packet) {
		t.Fatalf("read packet mismatch")
	}
	select {
	case offset := <-inner.readOffsetCh:
		if offset < 4 {
			t.Fatalf("inner TUN read offset = %d, want at least 4", offset)
		}
	default:
		t.Fatal("inner TUN read offset was not recorded")
	}
}

func ipv6Packet(t *testing.T, src string, dst string) []byte {
	t.Helper()
	srcAddr := netip.MustParseAddr(src).As16()
	dstAddr := netip.MustParseAddr(dst).As16()
	packet := make([]byte, 48)
	packet[0] = 0x60
	packet[6] = 58
	packet[7] = 64
	copy(packet[8:24], srcAddr[:])
	copy(packet[24:40], dstAddr[:])
	return packet
}
