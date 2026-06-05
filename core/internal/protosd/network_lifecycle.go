package protosd

import (
	"context"
	"fmt"
	"net"
	"os"
	stdruntime "runtime"
	"time"

	"github.com/protosio/protos/apic"
	"github.com/protosio/protos/internal/dns"
	hostagentcontrol "github.com/protosio/protos/internal/hostagent/control"
	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	"github.com/protosio/protos/internal/network"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/user"
)

const (
	networkRuntimeStateUnsupported = "unsupported"
	networkRuntimeStateDisabled    = "disabled"
	networkRuntimeStateBlocked     = "blocked_host_agent_unavailable"
	networkRuntimeStateStarting    = "starting"
	networkRuntimeStateEnabled     = "enabled"
	networkRuntimeStateStopping    = "stopping"
	networkRuntimeStateError       = "error"
)

func (n *Node) enableNetworkAtStartup(ctx context.Context) error {
	if stdruntime.GOOS == "darwin" && !hostAgentSocketAvailable() {
		n.setNetworkRuntimeStatus(true, false, networkRuntimeStateBlocked, "Host Agent is not running.")
		return nil
	}
	err := n.enableNetwork(ctx, false)
	if err == nil {
		return nil
	}
	if stdruntime.GOOS == "darwin" {
		log.Errorf("failed to enable network at startup: %v", err)
		return nil
	}
	return err
}

func (n *Node) EnableNetwork(ctx context.Context) error {
	return n.enableNetwork(ctx, true)
}

func (n *Node) enableNetwork(ctx context.Context, startHostAgent bool) error {
	if !n.capabilities.Network {
		err := fmt.Errorf("network capability is not available")
		n.setNetworkRuntimeStatus(false, false, networkRuntimeStateUnsupported, err.Error())
		return err
	}
	n.networkMu.Lock()
	if n.networkEnabled {
		n.networkDesired = true
		n.networkState = networkRuntimeStateEnabled
		n.networkMessage = ""
		n.networkMu.Unlock()
		return nil
	}
	n.networkMu.Unlock()
	n.setNetworkRuntimeStatus(true, false, networkRuntimeStateStarting, "")

	if stdruntime.GOOS == "darwin" && !hostAgentSocketAvailable() {
		if !startHostAgent {
			err := fmt.Errorf("host agent is not running")
			n.setNetworkRuntimeStatus(true, false, networkRuntimeStateBlocked, err.Error())
			return err
		}
		if err := hostagentcontrol.Start(ctx, hostagentcontrol.StartOptions{
			SocketUID: -1,
			SocketGID: -1,
		}); err != nil {
			n.setNetworkRuntimeStatus(true, false, networkRuntimeStateBlocked, err.Error())
			return err
		}
		if err := waitForHostAgentSocket(ctx); err != nil {
			n.setNetworkRuntimeStatus(true, false, networkRuntimeStateBlocked, err.Error())
			return err
		}
	}

	manager, err := n.networkManager()
	if err != nil {
		n.setNetworkRuntimeStatus(true, false, networkRuntimeStateError, err.Error())
		return err
	}
	if err := manager.Init(n.localKey, n.cfg.InternalDomain); err != nil {
		_ = manager.Down()
		n.setNetworkRuntimeStatus(true, false, networkRuntimeStateError, err.Error())
		return err
	}

	dnsStopper := dns.StartServer(n.localKey, n.dnsPort(), n.networkExternalDNS, n.cfg.InternalDomain, n.AppManager)
	if err := n.configureLocalResolver(); err != nil {
		_ = dnsStopper()
		_ = manager.Down()
		n.setNetworkRuntimeStatus(true, false, networkRuntimeStateError, err.Error())
		return err
	}

	n.networkMu.Lock()
	if n.networkDNSStopper != nil {
		_ = n.networkDNSStopper()
	}
	n.networkDNSStopper = dnsStopper
	n.networkEnabled = true
	n.networkDesired = true
	n.networkState = networkRuntimeStateEnabled
	n.networkMessage = ""
	n.networkMu.Unlock()

	if n.dbNotifier != nil {
		n.dbNotifier.Notify()
	}
	return nil
}

func (n *Node) DisableNetwork(context.Context) error {
	if !n.capabilities.Network {
		n.setNetworkRuntimeStatus(false, false, networkRuntimeStateUnsupported, "network capability is not available")
		return nil
	}

	n.networkMu.Lock()
	if !n.networkEnabled {
		n.networkDesired = false
		n.networkState = networkRuntimeStateDisabled
		n.networkMessage = ""
		n.networkMu.Unlock()
		return nil
	}
	n.networkDesired = false
	n.networkState = networkRuntimeStateStopping
	n.networkMessage = ""
	dnsStopper := n.networkDNSStopper
	n.networkDNSStopper = nil
	manager := n.NetworkManager
	n.networkMu.Unlock()

	var firstErr error
	if dnsStopper != nil {
		if err := dnsStopper(); err != nil {
			firstErr = err
		}
	}
	if manager != nil {
		if err := manager.Down(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	if firstErr != nil {
		n.setNetworkRuntimeStatus(false, false, networkRuntimeStateError, firstErr.Error())
		return firstErr
	}
	n.setNetworkRuntimeStatus(false, false, networkRuntimeStateDisabled, "")
	return nil
}

func (n *Node) NetworkRuntimeStatus(context.Context) apic.NetworkRuntimeStatus {
	n.networkMu.Lock()
	defer n.networkMu.Unlock()

	status := apic.NetworkRuntimeStatus{
		Supported:      n.capabilities.Network,
		DesiredEnabled: n.networkDesired,
		Enabled:        n.networkEnabled,
		State:          n.networkState,
		Message:        n.networkMessage,
	}
	if !status.Supported && status.State == "" {
		status.State = networkRuntimeStateUnsupported
	}
	if status.Supported && status.State == "" {
		status.State = networkRuntimeStateDisabled
	}
	return status
}

func (n *Node) NetworkEnabled() bool {
	n.networkMu.Lock()
	defer n.networkMu.Unlock()
	return n.networkEnabled
}

func (n *Node) NetworkState(context.Context) (networkmodule.State, error) {
	n.networkMu.Lock()
	enabled := n.networkEnabled
	state := n.networkState
	message := n.networkMessage
	manager := n.NetworkManager
	n.networkMu.Unlock()

	if !n.capabilities.Network {
		return networkmodule.State{
			Module:   "none",
			Up:       false,
			Messages: []string{"network capability is not available"},
		}, nil
	}
	if !enabled {
		if message == "" {
			message = state
		}
		return networkmodule.State{
			Module:   "none",
			Up:       false,
			Messages: []string{message},
		}, nil
	}
	if manager == nil {
		return networkmodule.State{}, fmt.Errorf("network manager is not configured")
	}
	return manager.State()
}

func (n *Node) State() (networkmodule.State, error) {
	return n.NetworkState(context.Background())
}

func (n *Node) ConfigureNetworkPeers(instances []provisioners.InstanceInfo, devices []user.UserDevice, appRoutes []network.AppRoute, exitRoutes []network.ExitRoute) error {
	n.networkMu.Lock()
	enabled := n.networkEnabled
	manager := n.NetworkManager
	n.networkMu.Unlock()

	if !enabled {
		return nil
	}
	if manager == nil {
		return fmt.Errorf("network manager is not configured")
	}
	return manager.ConfigurePeers(instances, devices, appRoutes, exitRoutes)
}

func (n *Node) networkManager() (*network.Manager, error) {
	n.networkMu.Lock()
	manager := n.NetworkManager
	n.networkMu.Unlock()
	if manager != nil {
		return manager, nil
	}

	manager, err := network.NewManager()
	if err != nil {
		return nil, err
	}
	n.networkMu.Lock()
	if n.NetworkManager == nil {
		n.NetworkManager = manager
	} else {
		_ = manager.Close()
		manager = n.NetworkManager
	}
	n.networkMu.Unlock()
	return manager, nil
}

func (n *Node) setNetworkRuntimeStatus(desired bool, enabled bool, state string, message string) {
	n.networkMu.Lock()
	defer n.networkMu.Unlock()
	n.networkDesired = desired
	n.networkEnabled = enabled
	n.networkState = state
	n.networkMessage = message
}

func hostAgentSocketAvailable() bool {
	info, err := os.Stat(hostagentipc.SocketPath())
	return err == nil && info.Mode()&os.ModeSocket != 0
}

func waitForHostAgentSocket(ctx context.Context) error {
	deadline := time.Now().Add(6 * time.Second)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		conn, err := net.DialTimeout("unix", hostagentipc.SocketPath(), 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("host agent socket %s is not reachable: %w", hostagentipc.SocketPath(), err)
		}
		time.Sleep(150 * time.Millisecond)
	}
}
