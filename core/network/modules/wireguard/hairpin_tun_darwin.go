//go:build darwin

package wireguard

import (
	"errors"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"

	wgtun "golang.zx2c4.com/wireguard/tun"
)

const nativeTunPacketOffset = 16

type hairpinTun struct {
	inner wgtun.Device

	packets chan []byte
	done    chan struct{}

	closeOnce sync.Once
	doneOnce  sync.Once

	mu        sync.RWMutex
	localAddr netip.Addr
	routes    map[netip.Addr]struct{}

	hairpinnedPackets atomic.Uint64
	droppedPackets    atomic.Uint64
}

type hairpinTunStats struct {
	HairpinnedPackets uint64
	DroppedPackets    uint64
	HairpinRoutes     int
}

func newHairpinTun(inner wgtun.Device) *hairpinTun {
	tun := &hairpinTun{
		inner:   inner,
		packets: make(chan []byte, 1024),
		done:    make(chan struct{}),
		routes:  map[netip.Addr]struct{}{},
	}
	go tun.readInner()
	return tun
}

func (tun *hairpinTun) File() *os.File {
	return tun.inner.File()
}

func (tun *hairpinTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	var packet []byte
	select {
	case packet = <-tun.packets:
	case <-tun.done:
		return 0, errors.New("hairpin tun is closed")
	}

	count := 0
	for {
		if count >= len(bufs) || count >= len(sizes) {
			return count, nil
		}
		sizes[count] = copy(bufs[count][offset:], packet)
		count++

		select {
		case packet = <-tun.packets:
		default:
			return count, nil
		}
	}
}

func (tun *hairpinTun) Write(bufs [][]byte, offset int) (int, error) {
	passThrough := make([][]byte, 0, len(bufs))
	hairpinned := 0

	for _, buf := range bufs {
		if offset > len(buf) {
			continue
		}
		packet := buf[offset:]
		if dst, ok := tun.hairpinDestination(packet); ok {
			if tun.enqueue(packet) {
				tun.recordHairpin(dst)
			} else {
				tun.recordDrop("hairpin", dst)
			}
			hairpinned++
			continue
		}
		passThrough = append(passThrough, buf)
	}

	if len(passThrough) == 0 {
		return hairpinned, nil
	}
	written, err := tun.inner.Write(passThrough, offset)
	return hairpinned + written, err
}

func (tun *hairpinTun) MTU() (int, error) {
	return tun.inner.MTU()
}

func (tun *hairpinTun) Name() (string, error) {
	return tun.inner.Name()
}

func (tun *hairpinTun) Events() <-chan wgtun.Event {
	return tun.inner.Events()
}

func (tun *hairpinTun) Close() error {
	var err error
	tun.closeOnce.Do(func() {
		tun.signalDone()
		err = tun.inner.Close()
	})
	return err
}

func (tun *hairpinTun) BatchSize() int {
	return tun.inner.BatchSize()
}

func (tun *hairpinTun) setLocalAddress(address string) {
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return
	}
	tun.mu.Lock()
	tun.localAddr = addr
	tun.mu.Unlock()
}

func (tun *hairpinTun) setHairpinRoutes(routes map[string]struct{}) {
	parsed := make(map[netip.Addr]struct{}, len(routes))
	for route := range routes {
		prefix, err := netip.ParsePrefix(route)
		if err == nil {
			parsed[prefix.Addr()] = struct{}{}
			continue
		}
		addr, err := netip.ParseAddr(route)
		if err == nil {
			parsed[addr] = struct{}{}
		}
	}

	tun.mu.Lock()
	tun.routes = parsed
	tun.mu.Unlock()
	log.Debugf("Configured %d WireGuard hairpin routes", len(parsed))
}

func (tun *hairpinTun) readInner() {
	batchSize := tun.inner.BatchSize()
	if batchSize <= 0 {
		batchSize = 1
	}
	mtu, err := tun.inner.MTU()
	if err != nil || mtu <= 0 {
		mtu = 1500
	}

	bufs := make([][]byte, batchSize)
	sizes := make([]int, batchSize)
	for i := range bufs {
		bufs[i] = make([]byte, nativeTunPacketOffset+mtu+128)
	}

	for {
		n, err := tun.inner.Read(bufs, sizes, nativeTunPacketOffset)
		if err != nil {
			tun.signalDone()
			return
		}
		for i := 0; i < n; i++ {
			packet := make([]byte, sizes[i])
			copy(packet, bufs[i][nativeTunPacketOffset:nativeTunPacketOffset+sizes[i]])
			if !tun.enqueue(packet) {
				tun.recordDrop("inner-tun", netip.Addr{})
			}
		}
	}
}

func (tun *hairpinTun) signalDone() {
	tun.doneOnce.Do(func() {
		close(tun.done)
	})
}

func (tun *hairpinTun) enqueue(packet []byte) bool {
	copyPacket := make([]byte, len(packet))
	copy(copyPacket, packet)

	select {
	case tun.packets <- copyPacket:
		return true
	case <-tun.done:
		return false
	default:
		return false
	}
}

func (tun *hairpinTun) hairpinDestination(packet []byte) (netip.Addr, bool) {
	dst, ok := ipv6Destination(packet)
	if !ok {
		return netip.Addr{}, false
	}

	tun.mu.RLock()
	defer tun.mu.RUnlock()
	if dst == tun.localAddr {
		return netip.Addr{}, false
	}
	_, found := tun.routes[dst]
	if !found {
		return netip.Addr{}, false
	}
	return dst, true
}

func (tun *hairpinTun) recordHairpin(dst netip.Addr) {
	total := tun.hairpinnedPackets.Add(1)
	if total == 1 || total%1024 == 0 {
		log.Debugf("Hairpinned %d WireGuard IPv6 packets through userspace; latest dst=%s", total, dst)
	}
}

func (tun *hairpinTun) recordDrop(source string, dst netip.Addr) {
	total := tun.droppedPackets.Add(1)
	if dst.IsValid() {
		log.Warnf("Dropped WireGuard packet from %s for dst=%s because the userspace queue is full or closed; total_dropped=%d", source, dst, total)
		return
	}
	log.Warnf("Dropped WireGuard packet from %s because the userspace queue is full or closed; total_dropped=%d", source, total)
}

func (tun *hairpinTun) stats() hairpinTunStats {
	tun.mu.RLock()
	routes := len(tun.routes)
	tun.mu.RUnlock()

	return hairpinTunStats{
		HairpinnedPackets: tun.hairpinnedPackets.Load(),
		DroppedPackets:    tun.droppedPackets.Load(),
		HairpinRoutes:     routes,
	}
}

func ipv6Destination(packet []byte) (netip.Addr, bool) {
	if len(packet) < 40 || packet[0]>>4 != 6 {
		return netip.Addr{}, false
	}
	var dst [16]byte
	copy(dst[:], packet[24:40])
	return netip.AddrFrom16(dst), true
}
