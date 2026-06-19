//go:build darwin

package localvm

import (
	"fmt"
	"net"
	"sync"
	"syscall"

	"github.com/protosio/protos/internal/util"
	"github.com/tmc/apple/foundation"
	"github.com/tmc/apple/objectivec"
	vz "github.com/tmc/apple/virtualization"
	"github.com/tmc/apple/vmnet"
)

var vmnetLog = util.GetLogger("localvm")

const vmnetFrameBufferSize = 2048

// configureSharedVMNetNetwork attaches the VM to a private, isolation-off shared
// vmnet network via a socketpair + VZFileHandleNetworkDeviceAttachment, and
// starts a 1:1 packet pump between that socket and the vmnet interface. Because
// vmnet shared mode is a single host-wide switch, every VM started this way
// shares one L2 segment and can reach the others directly - giving WireGuard a
// real single-hop path and letting libp2p connect directly instead of relaying.
// Requires root (the host-agent already runs as root). Returns a cleanup that
// stops the pump and releases the interface.
func configureSharedVMNetNetwork(config vz.VZVirtualMachineConfiguration) (func(), error) {
	sharedNet, err := startSharedVMNet()
	if err != nil {
		return nil, fmt.Errorf("start shared vmnet: %w", err)
	}
	fail := func(e error) (func(), error) {
		_ = sharedNet.stop()
		return nil, e
	}

	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return fail(fmt.Errorf("create vmnet socketpair: %w", err))
	}
	vmFD, pumpFD := fds[0], fds[1]
	for _, fd := range fds {
		_ = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_SNDBUF, 4<<20)
		_ = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4<<20)
	}
	closeFDs := func() {
		_ = syscall.Close(vmFD)
		_ = syscall.Close(pumpFD)
	}

	// VZ takes ownership of vmFD (closes it on dealloc).
	fileHandle := foundation.NewFileHandleWithFileDescriptorCloseOnDealloc(vmFD, true)
	attachment := vz.NewFileHandleNetworkDeviceAttachmentWithFileHandle(fileHandle)
	if attachment.ID == 0 {
		closeFDs()
		return fail(fmt.Errorf("create file handle network attachment"))
	}
	attachment.SetMaximumTransmissionUnit(int(sharedNet.mtu))
	attachment.Retain()

	networkConfig := vz.NewVZVirtioNetworkDeviceConfiguration()
	if networkConfig.ID == 0 {
		closeFDs()
		return fail(fmt.Errorf("create network device"))
	}
	networkConfig.SetAttachment(&attachment.VZNetworkDeviceAttachment)

	// Use the MAC vmnet assigned to this interface so vmnet and the guest agree
	// (avoids any source-MAC forwarding ambiguity on the shared switch). The
	// guest IP is static and keyed by interface name, not MAC, so this is safe.
	hwAddr, err := net.ParseMAC(sharedNet.mac)
	if err != nil {
		closeFDs()
		return fail(fmt.Errorf("parse vmnet MAC %q: %w", sharedNet.mac, err))
	}
	vzMAC := vz.NewMACAddressWithString(hwAddr.String())
	if vzMAC.ID == 0 {
		closeFDs()
		return fail(fmt.Errorf("create VZ MAC address"))
	}
	networkConfig.SetMACAddress(&vzMAC)
	config.SetNetworkDevices([]vz.VZNetworkDeviceConfiguration{networkConfig.VZNetworkDeviceConfiguration})

	pump := newVMNetPump(sharedNet, pumpFD)
	pump.start()
	vmnetLog.Infof("attached VM to shared vmnet: mac=%s mtu=%d subnet_mask=%s gateway=%s", sharedNet.mac, sharedNet.mtu, sharedNet.subnet, sharedNet.startAddr)

	cleanup := func() {
		pump.stop()
		_ = syscall.Close(pumpFD)
		_ = sharedNet.stop()
	}
	return cleanup, nil
}

// vmnetPump moves ethernet frames 1:1 between a vmnet interface and the VM's
// socketpair end. vmnet->VM is event-callback driven (PACKETS_AVAILABLE); VM->
// vmnet is a blocking read loop. Each AF_UNIX SOCK_DGRAM datagram is exactly one
// ethernet frame, matching what VZFileHandleNetworkDeviceAttachment expects.
type vmnetPump struct {
	net      *sharedVMNet
	fd       int
	readMu   sync.Mutex
	stopCh   chan struct{}
	stopOnce sync.Once
}

func newVMNetPump(n *sharedVMNet, fd int) *vmnetPump {
	return &vmnetPump{net: n, fd: fd, stopCh: make(chan struct{})}
}

func (p *vmnetPump) start() {
	vmnet.Vmnet_interface_set_event_callback(p.net.iface, vmnet.VMNET_INTERFACE_PACKETS_AVAILABLE, p.net.queue, func(_ vmnet.Interface_event_t, _ objectivec.Object) {
		p.drainToVM()
	})
	go p.pumpToVMNet()
}

// drainToVM reads all currently-available vmnet packets and writes each as one
// datagram to the VM. Serialized so concurrent callback invocations don't race
// on vmnet_read.
func (p *vmnetPump) drainToVM() {
	p.readMu.Lock()
	defer p.readMu.Unlock()
	buf := make([]byte, vmnetFrameBufferSize)
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}
		n, ok := p.net.readPacket(buf)
		if !ok || n <= 0 {
			return
		}
		if _, err := syscall.Write(p.fd, buf[:n]); err != nil {
			return
		}
	}
}

func (p *vmnetPump) pumpToVMNet() {
	buf := make([]byte, vmnetFrameBufferSize)
	for {
		select {
		case <-p.stopCh:
			return
		default:
		}
		n, err := syscall.Read(p.fd, buf)
		if err != nil {
			if err == syscall.EINTR {
				continue
			}
			return
		}
		if n <= 0 {
			continue
		}
		p.net.writePacket(buf[:n])
	}
}

func (p *vmnetPump) stop() {
	p.stopOnce.Do(func() { close(p.stopCh) })
}
