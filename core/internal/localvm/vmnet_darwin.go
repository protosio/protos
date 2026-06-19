//go:build darwin

package localvm

import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/tmc/apple/dispatch"
	"github.com/tmc/apple/objectivec"
	"github.com/tmc/apple/vmnet"
)

// This file hand-rolls the small amount of vmnet/xpc interop the tmc/apple
// binding does not provide (it exposes the vmnet function entrypoints but ships
// no xpc helpers and mis-types the xpc key symbols). We open a single shared
// vmnet network with isolation disabled so that all local VMs attached to it
// share one L2 segment and can reach each other directly. vmnet shared mode
// requires root, which the host-agent already has; it needs no Apple
// "networking" entitlement. The interop is purego-based, so it stays
// CGO_ENABLED=0-compatible.

const qosClassDefault dispatch.QOS = 0x15 // QOS_CLASS_DEFAULT

var (
	xpcInitOnce sync.Once
	xpcInitErr  error

	xpcDictionaryCreate    func(keys, values unsafe.Pointer, count uint64) uintptr
	xpcDictionarySetUint64 func(dict uintptr, key string, value uint64)
	xpcDictionarySetBool   func(dict uintptr, key string, value bool)
	xpcDictionaryGetUint64 func(dict uintptr, key string) uint64
	xpcDictionaryGetString func(dict uintptr, key string) uintptr
	xpcRelease             func(obj uintptr)

	// vmnet_read/vmnet_write are called directly (not via the binding) because
	// the binding's Vmpktdesc struct has the wrong field order (vm_pkt_iov is
	// declared last instead of second), which corrupts the iovec pointer and
	// segfaults. We pass a correctly-ordered struct here.
	vmnetReadFn  func(iface uintptr, packets *vmnetPacketDesc, pktcnt *int32) uint32
	vmnetWriteFn func(iface uintptr, packets *vmnetPacketDesc, pktcnt *int32) uint32

	// vmnet xpc dictionary key strings, read from the framework symbols
	// (const char * globals) at init time.
	keyOperationMode   string
	keyEnableIsolation string
	keyMTU             string
	keyMaxPacketSize   string
	keyMACAddress      string
	keySubnetMask      string
	keyStartAddress    string
	keyEndAddress      string
)

func xpcInit() error {
	xpcInitOnce.Do(func() {
		libSystem, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			xpcInitErr = fmt.Errorf("dlopen libSystem: %w", err)
			return
		}
		purego.RegisterLibFunc(&xpcDictionaryCreate, libSystem, "xpc_dictionary_create")
		purego.RegisterLibFunc(&xpcDictionarySetUint64, libSystem, "xpc_dictionary_set_uint64")
		purego.RegisterLibFunc(&xpcDictionarySetBool, libSystem, "xpc_dictionary_set_bool")
		purego.RegisterLibFunc(&xpcDictionaryGetUint64, libSystem, "xpc_dictionary_get_uint64")
		purego.RegisterLibFunc(&xpcDictionaryGetString, libSystem, "xpc_dictionary_get_string")
		purego.RegisterLibFunc(&xpcRelease, libSystem, "xpc_release")

		vmnetLib, err := purego.Dlopen("/System/Library/Frameworks/vmnet.framework/vmnet", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			xpcInitErr = fmt.Errorf("dlopen vmnet: %w", err)
			return
		}
		purego.RegisterLibFunc(&vmnetReadFn, vmnetLib, "vmnet_read")
		purego.RegisterLibFunc(&vmnetWriteFn, vmnetLib, "vmnet_write")
		keys := []struct {
			dst *string
			sym string
		}{
			{&keyOperationMode, "vmnet_operation_mode_key"},
			{&keyEnableIsolation, "vmnet_enable_isolation_key"},
			{&keyMTU, "vmnet_mtu_key"},
			{&keyMaxPacketSize, "vmnet_max_packet_size_key"},
			{&keyMACAddress, "vmnet_mac_address_key"},
			{&keySubnetMask, "vmnet_subnet_mask_key"},
			{&keyStartAddress, "vmnet_start_address_key"},
			{&keyEndAddress, "vmnet_end_address_key"},
		}
		for _, k := range keys {
			value, err := readCStringSymbol(vmnetLib, k.sym)
			if err != nil {
				xpcInitErr = err
				return
			}
			*k.dst = value
		}
	})
	return xpcInitErr
}

// readCStringSymbol resolves a `const char *name` framework symbol and returns
// the Go string it points at.
func readCStringSymbol(handle uintptr, name string) (string, error) {
	addr, err := purego.Dlsym(handle, name)
	if err != nil || addr == 0 {
		return "", fmt.Errorf("dlsym %s: %w", name, err)
	}
	cstr := *(*uintptr)(unsafe.Pointer(addr))
	if cstr == 0 {
		return "", fmt.Errorf("symbol %s points at nil", name)
	}
	return goStringFromC(cstr), nil
}

func goStringFromC(cstr uintptr) string {
	if cstr == 0 {
		return ""
	}
	var out []byte
	for i := uintptr(0); ; i++ {
		ch := *(*byte)(unsafe.Pointer(cstr + i))
		if ch == 0 {
			break
		}
		out = append(out, ch)
	}
	return string(out)
}

// sharedVMNet is a running shared-mode vmnet interface with isolation disabled.
type sharedVMNet struct {
	iface     vmnet.Interface_ref
	queue     dispatch.Queue
	mtu       uint64
	maxPacket uint64
	mac       string
	subnet    string
	startAddr string
	endAddr   string
}

type vmnetStartResult struct {
	status    vmnet.Vmnet_return_t
	mtu       uint64
	maxPacket uint64
	mac       string
	subnet    string
	startAddr string
	endAddr   string
}

// startSharedVMNet starts a shared-mode vmnet interface with isolation
// disabled. Requires root.
func startSharedVMNet() (*sharedVMNet, error) {
	if err := xpcInit(); err != nil {
		return nil, err
	}

	desc := xpcDictionaryCreate(nil, nil, 0)
	if desc == 0 {
		return nil, fmt.Errorf("xpc_dictionary_create returned nil")
	}
	defer xpcRelease(desc)
	xpcDictionarySetUint64(desc, keyOperationMode, uint64(vmnet.VMNET_SHARED_MODE))
	// Isolation off: VMs on this network can reach each other (the whole point).
	xpcDictionarySetBool(desc, keyEnableIsolation, false)

	queue := dispatch.GetGlobalQueue(qosClassDefault)

	resultCh := make(chan vmnetStartResult, 1)
	iface := vmnet.Vmnet_start_interface(unsafe.Pointer(desc), queue, func(status vmnet.Vmnet_return_t, params objectivec.Object) {
		r := vmnetStartResult{status: status}
		// Read the result dictionary inside the callback: the xpc params object
		// is only guaranteed valid for the duration of the handler.
		if status == vmnet.VMNET_SUCCESS {
			p := uintptr(params.ID)
			r.mtu = xpcDictionaryGetUint64(p, keyMTU)
			r.maxPacket = xpcDictionaryGetUint64(p, keyMaxPacketSize)
			r.mac = goStringFromC(xpcDictionaryGetString(p, keyMACAddress))
			r.subnet = goStringFromC(xpcDictionaryGetString(p, keySubnetMask))
			r.startAddr = goStringFromC(xpcDictionaryGetString(p, keyStartAddress))
			r.endAddr = goStringFromC(xpcDictionaryGetString(p, keyEndAddress))
		}
		resultCh <- r
	})
	if iface == 0 {
		return nil, fmt.Errorf("vmnet_start_interface returned a nil interface ref (root required?)")
	}

	var res vmnetStartResult
	select {
	case res = <-resultCh:
	case <-time.After(15 * time.Second):
		return nil, fmt.Errorf("vmnet_start_interface completion timed out")
	}
	if res.status != vmnet.VMNET_SUCCESS {
		return nil, fmt.Errorf("vmnet_start_interface failed: status=%d (%s)", res.status, res.status)
	}

	return &sharedVMNet{
		iface:     iface,
		queue:     queue,
		mtu:       res.mtu,
		maxPacket: res.maxPacket,
		mac:       res.mac,
		subnet:    res.subnet,
		startAddr: res.startAddr,
		endAddr:   res.endAddr,
	}, nil
}

func (n *sharedVMNet) stop() error {
	if n == nil || n.iface == 0 {
		return nil
	}
	resultCh := make(chan vmnet.Vmnet_return_t, 1)
	vmnet.Vmnet_stop_interface(n.iface, n.queue, func(status vmnet.Vmnet_return_t) {
		resultCh <- status
	})
	select {
	case status := <-resultCh:
		n.iface = 0
		if status != vmnet.VMNET_SUCCESS {
			return fmt.Errorf("vmnet_stop_interface failed: status=%d (%s)", status, status)
		}
		return nil
	case <-time.After(15 * time.Second):
		return fmt.Errorf("vmnet_stop_interface completion timed out")
	}
}

// iovec mirrors C `struct iovec { void *iov_base; size_t iov_len; }`.
type iovec struct {
	base   unsafe.Pointer
	length uintptr
}

// vmnetPacketDesc mirrors C `struct vmpktdesc` with the correct field order
// (the binding's Vmpktdesc gets this wrong). Layout:
// { size_t vm_pkt_size; struct iovec *vm_pkt_iov; uint32 vm_pkt_iovcnt; uint32 vm_flags; }.
type vmnetPacketDesc struct {
	size   uintptr
	iov    *iovec
	iovcnt uint32
	// vm_flags: kept for C struct layout; must be 0 on read/write. Unset from Go,
	// so it is a zero-initialized blank field.
	_ uint32
}

// readPacket reads one available frame from the interface into buf, returning
// the frame length. ok is false when no packet is currently available.
func (n *sharedVMNet) readPacket(buf []byte) (length int, ok bool) {
	if n == nil || n.iface == 0 || len(buf) == 0 {
		return 0, false
	}
	iv := iovec{base: unsafe.Pointer(&buf[0]), length: uintptr(len(buf))}
	pkt := vmnetPacketDesc{size: uintptr(len(buf)), iov: &iv, iovcnt: 1}
	count := int32(1)
	status := vmnetReadFn(uintptr(n.iface), &pkt, &count)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(&iv)
	if status != uint32(vmnet.VMNET_SUCCESS) || count == 0 {
		return 0, false
	}
	return int(pkt.size), true
}

// writePacket writes one frame to the interface.
func (n *sharedVMNet) writePacket(frame []byte) bool {
	if n == nil || n.iface == 0 || len(frame) == 0 {
		return false
	}
	iv := iovec{base: unsafe.Pointer(&frame[0]), length: uintptr(len(frame))}
	pkt := vmnetPacketDesc{size: uintptr(len(frame)), iov: &iv, iovcnt: 1}
	count := int32(1)
	status := vmnetWriteFn(uintptr(n.iface), &pkt, &count)
	runtime.KeepAlive(frame)
	runtime.KeepAlive(&iv)
	return status == uint32(vmnet.VMNET_SUCCESS) && count > 0
}

// VMNetSelftest opens an isolation-off shared vmnet interface, reports the
// parameters vmnet assigned (subnet/gateway/MTU/MAC), exercises one write and a
// few reads to confirm the packet-descriptor interop does not crash, and stops
// it. It is the de-risking probe for the in-process vmnet datapath and must be
// run as root (e.g. `sudo bin/protos-hostagent --vmnet-selftest`).
func VMNetSelftest() error {
	fmt.Println("vmnet selftest: starting isolation-off shared interface (requires root)…")
	n, err := startSharedVMNet()
	if err != nil {
		return fmt.Errorf("start shared vmnet: %w", err)
	}
	fmt.Printf("vmnet selftest: started OK\n")
	fmt.Printf("  iface_ref   = %#x\n", n.iface)
	fmt.Printf("  mtu         = %d\n", n.mtu)
	fmt.Printf("  max_packet  = %d\n", n.maxPacket)
	fmt.Printf("  mac_address = %s\n", n.mac)
	fmt.Printf("  subnet_mask = %s\n", n.subnet)
	fmt.Printf("  dhcp_start  = %s\n", n.startAddr)
	fmt.Printf("  dhcp_end    = %s\n", n.endAddr)

	// Exercise the packet-descriptor interop: build a broadcast ethernet frame
	// (dst=ff:ff:.., src=our vmnet MAC, ethertype 0x0806) and write it, then try
	// a few reads. We only care that read/write do not crash on the struct layout.
	srcMAC, _ := net.ParseMAC(n.mac)
	frame := make([]byte, 64)
	for i := 0; i < 6; i++ {
		frame[i] = 0xff
	}
	if len(srcMAC) == 6 {
		copy(frame[6:12], srcMAC)
	}
	frame[12] = 0x08
	frame[13] = 0x06
	if n.writePacket(frame) {
		fmt.Println("vmnet selftest: write OK (no crash)")
	} else {
		fmt.Println("vmnet selftest: write returned not-ok (acceptable; no crash)")
	}
	buf := make([]byte, vmnetFrameBufferSize)
	reads := 0
	for i := 0; i < 50; i++ {
		if length, ok := n.readPacket(buf); ok {
			reads++
			_ = length
		}
	}
	fmt.Printf("vmnet selftest: read OK (no crash), frames_read=%d\n", reads)

	if err := n.stop(); err != nil {
		return fmt.Errorf("stop shared vmnet: %w", err)
	}
	fmt.Println("vmnet selftest: stopped OK")
	return nil
}
