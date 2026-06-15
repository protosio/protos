package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	pbApic "github.com/protosio/protos/apic/proto"
	"github.com/urfave/cli/v2"
)

var inspectJSON bool

var cmdInspect *cli.Command = &cli.Command{
	Name:  "inspect",
	Usage: "Read imperative runtime state configured by Protos",
	Subcommands: []*cli.Command{
		{
			Name:      "network",
			ArgsUsage: "[instance]",
			Usage:     "Show all network state for the local daemon or a remote instance",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectNetwork(c.Args().Get(0), "all")
			},
		},
		{
			Name:      "interfaces",
			Aliases:   []string{"ifaces"},
			ArgsUsage: "[instance]",
			Usage:     "Show interfaces owned by the Protos network layer",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectNetwork(c.Args().Get(0), "interfaces")
			},
		},
		{
			Name:      "routes",
			ArgsUsage: "[instance]",
			Usage:     "Show kernel routes configured or observed by the network layer",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectNetwork(c.Args().Get(0), "routes")
			},
		},
		{
			Name:      "wg",
			ArgsUsage: "[instance]",
			Usage:     "Show WireGuard peers and allowed IPs",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectNetwork(c.Args().Get(0), "wg")
			},
		},
		{
			Name:      "dns",
			ArgsUsage: "[instance]",
			Usage:     "Show DNS state owned by Protos",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectNetwork(c.Args().Get(0), "dns")
			},
		},
		{
			Name:      "firewall",
			ArgsUsage: "[instance]",
			Usage:     "Show native firewall/NAT state owned by Protos",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectNetwork(c.Args().Get(0), "firewall")
			},
		},
		{
			Name:      "exit-routes",
			Aliases:   []string{"declarations", "desired-routes"},
			ArgsUsage: "[instance]",
			Usage:     "Show declarative exit route rows visible to a peer",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectExitRoutes(c.Args().Get(0))
			},
		},
		{
			Name:      "runtime",
			Aliases:   []string{"swarmion", "sync"},
			ArgsUsage: "[instance]",
			Usage:     "Show read-only Swarmion runtime state for a peer",
			Flags:     inspectFlags(),
			Action: func(c *cli.Context) error {
				return inspectRuntime(c.Args().Get(0))
			},
		},
	},
}

func inspectFlags() []cli.Flag {
	return []cli.Flag{
		&cli.BoolFlag{
			Name:        "json",
			Usage:       "Print raw JSON",
			Destination: &inspectJSON,
		},
	}
}

func inspectNetwork(instance string, section string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := client.GetNetworkState(ctx, &pbApic.GetNetworkStateRequest{Instance: instance})
	if err != nil {
		return fmt.Errorf("failed to retrieve network state: %w", err)
	}
	state := resp.GetState()
	if state == nil {
		return fmt.Errorf("network state response was empty")
	}
	if inspectJSON {
		encoded, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}

	printNetworkSummary(instance, state)
	switch section {
	case "all":
		printInterfaces(state.GetInterfaces())
		printAddresses(state.GetAddresses())
		printRoutes(state.GetRoutes())
		printWireGuardPeers(state.GetWireguardPeers())
		printDNSState(state.GetDns())
		printFirewallTables(state.GetFirewallTables())
	case "interfaces":
		printInterfaces(state.GetInterfaces())
	case "routes":
		printRoutes(state.GetRoutes())
	case "wg":
		printWireGuardPeers(state.GetWireguardPeers())
	case "dns":
		printDNSState(state.GetDns())
	case "firewall":
		printFirewallTables(state.GetFirewallTables())
	}
	return nil
}

func inspectExitRoutes(instance string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := client.GetExitRoutes(ctx, &pbApic.GetExitRoutesRequest{Instance: instance})
	if err != nil {
		return fmt.Errorf("failed to retrieve exit routes: %w", err)
	}
	if inspectJSON {
		encoded, err := json.MarshalIndent(resp.GetRoutes(), "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	target := "local"
	if strings.TrimSpace(instance) != "" {
		target = instance
	}
	fmt.Printf("Target: %s\n", target)
	printExitRoutes(resp.GetRoutes())
	return nil
}

func inspectRuntime(instance string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	resp, err := client.GetRuntimeState(ctx, &pbApic.GetRuntimeStateRequest{Instance: instance})
	if err != nil {
		return fmt.Errorf("failed to retrieve runtime state: %w", err)
	}
	state := resp.GetState()
	if state == nil {
		return fmt.Errorf("runtime state response was empty")
	}
	if inspectJSON {
		encoded, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	printRuntimeState(instance, state)
	return nil
}

func printNetworkSummary(instance string, state *pbApic.NetworkState) {
	target := "local"
	if strings.TrimSpace(instance) != "" {
		target = instance
	}
	status := "down"
	if state.GetUp() {
		status = "up"
	}
	fmt.Printf("Target: %s\n", target)
	fmt.Printf("Module: %s\n", emptyDefault(state.GetModule(), "n/a"))
	fmt.Printf("Status: %s\n", status)
	fmt.Printf("Interface: %s\n", emptyDefault(state.GetInterfaceName(), "n/a"))
	for _, message := range state.GetMessages() {
		if strings.TrimSpace(message) != "" {
			fmt.Printf("Message: %s\n", message)
		}
	}
}

func printInterfaces(interfaces []*pbApic.NetworkInterface) {
	fmt.Println("\nInterfaces")
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "Name", "Type", "Index", "MTU", "Up", "Master", "Kind")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "----", "----", "-----", "---", "--", "------", "----")
	for _, iface := range interfaces {
		fmt.Fprintf(w, " %s\t%s\t%d\t%d\t%t\t%s\t%s\t\n",
			iface.GetName(),
			iface.GetType(),
			iface.GetIndex(),
			iface.GetMtu(),
			iface.GetUp(),
			iface.GetMaster(),
			iface.GetKind(),
		)
	}
}

func printAddresses(addresses []*pbApic.NetworkAddress) {
	fmt.Println("\nAddresses")
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t\n", "Interface", "CIDR", "Scope")
	fmt.Fprintf(w, " %s\t%s\t%s\t\n", "---------", "----", "-----")
	for _, address := range addresses {
		fmt.Fprintf(w, " %s\t%s\t%s\t\n", address.GetInterfaceName(), address.GetCidr(), address.GetScope())
	}
}

func printRoutes(routes []*pbApic.NetworkRoute) {
	fmt.Println("\nRoutes")
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "Destination", "Gateway", "Iface", "Source", "Family", "Kind", "Priority")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "-----------", "-------", "-----", "------", "------", "----", "--------")
	for _, route := range routes {
		fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			emptyDefault(route.GetDestination(), "default"),
			route.GetGateway(),
			route.GetInterfaceName(),
			route.GetSource(),
			route.GetFamily(),
			route.GetKind(),
			route.GetPriority(),
		)
	}
}

func printWireGuardPeers(peers []*pbApic.WireGuardPeer) {
	fmt.Println("\nWireGuard Peers")
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "Public Key", "Endpoint", "Allowed IPs", "Handshake", "RX", "TX")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "----------", "--------", "-----------", "---------", "--", "--")
	for _, peer := range peers {
		fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%d\t%d\t\n",
			shortValue(peer.GetPublicKey(), 18),
			peer.GetEndpoint(),
			strings.Join(peer.GetAllowedIps(), ","),
			peer.GetLatestHandshake(),
			peer.GetRxBytes(),
			peer.GetTxBytes(),
		)
	}
}

func printDNSState(entries []*pbApic.DNSState) {
	fmt.Println("\nDNS")
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "Scope", "Domain", "Servers", "Port", "Active", "Source")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "-----", "------", "-------", "----", "------", "------")
	for _, entry := range entries {
		fmt.Fprintf(w, " %s\t%s\t%s\t%d\t%t\t%s\t\n",
			entry.GetScope(),
			entry.GetDomain(),
			strings.Join(entry.GetServers(), ","),
			entry.GetPort(),
			entry.GetActive(),
			entry.GetSource(),
		)
	}
}

func printFirewallTables(tables []*pbApic.FirewallTable) {
	fmt.Println("\nFirewall")
	if len(tables) == 0 {
		fmt.Println(" No Protos-managed firewall tables")
		return
	}
	for _, table := range tables {
		fmt.Printf(" Table: %s/%s\n", table.GetFamily(), table.GetName())
		for _, chain := range table.GetChains() {
			fmt.Printf("  Chain: %s type=%s hook=%s priority=%s\n", chain.GetName(), chain.GetType(), chain.GetHook(), chain.GetPriority())
			w := tableWriter()
			fmt.Fprintf(w, "   %s\t%s\t%s\t\n", "Packets", "Bytes", "Expressions")
			fmt.Fprintf(w, "   %s\t%s\t%s\t\n", "-------", "-----", "-----------")
			for _, rule := range chain.GetRules() {
				fmt.Fprintf(w, "   %d\t%d\t%s\t\n", rule.GetPackets(), rule.GetBytes(), strings.Join(rule.GetExpressions(), " | "))
			}
			w.Flush()
		}
	}
}

func printExitRoutes(routes []*pbApic.ExitRoute) {
	fmt.Println("\nExit Routes")
	if len(routes) == 0 {
		fmt.Println(" No declarative exit routes")
		return
	}
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "Device", "Instance", "Name", "Public IP", "Location", "CIDRs", "DNS", "Status")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n", "------", "--------", "----", "---------", "--------", "-----", "---", "------")
	for _, route := range routes {
		dnsServer := route.GetDnsServer()
		if dnsServer == "" {
			dnsServer = "default"
		}
		fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t\n",
			route.GetDeviceId(),
			route.GetInstanceId(),
			route.GetInstanceName(),
			route.GetPublicIp(),
			route.GetLocation(),
			strings.Join(route.GetCidrs(), ","),
			dnsServer,
			route.GetStatus(),
		)
	}
}

func printRuntimeState(instance string, state *pbApic.RuntimeState) {
	target := "local"
	if strings.TrimSpace(instance) != "" {
		target = instance
	}
	fmt.Printf("Target: %s\n", target)
	fmt.Printf("Peer: %s\n", emptyDefault(state.GetPeerId(), "n/a"))
	fmt.Printf("Manifest: %s\n", emptyDefault(state.GetManifestDigest(), "n/a"))
	fmt.Printf("Checkpoint Root: %s\n", emptyDefault(state.GetCheckpointRootHash(), "n/a"))
	fmt.Printf("Tentative Root: %s\n", emptyDefault(state.GetTentativeRootHash(), "n/a"))
	fmt.Printf("Protocol Checkpoint Root: %s\n", emptyDefault(state.GetProtocolCheckpointRootHash(), "n/a"))
	fmt.Printf("Protocol Checkpoint Digest: %s\n", emptyDefault(state.GetProtocolCheckpointDigest(), "n/a"))
	fmt.Printf("Durable Main Root: %s\n", emptyDefault(state.GetDurableMainRootHash(), "n/a"))
	fmt.Printf("Materialization: %s pending=%t error=%s\n",
		emptyDefault(state.GetRuntimeMaterializationPolicy(), "n/a"),
		state.GetRuntimeCheckpointPending(),
		emptyDefault(state.GetRuntimeCheckpointLastError(), "none"),
	)
	fmt.Printf("Refresh: pending=%t error=%s\n", state.GetRuntimeRefreshPending(), emptyDefault(state.GetRuntimeRefreshLastError(), "none"))
	if fatal := strings.TrimSpace(state.GetFatalState()); fatal != "" {
		fmt.Printf("Fatal: %s\n", fatal)
	}
	fmt.Printf("State Providers: %s\n", emptyDefault(strings.Join(state.GetStateProviders(), ","), "none"))
	fmt.Printf("Connected Peers: %s\n", emptyDefault(strings.Join(state.GetConnectedPeers(), ","), "none"))
	printRuntimePeerStatuses(state.GetPeerStatuses())
	printRuntimeCompatibility(state.GetCompatibility())
	printRuntimeTrace(state.GetContentSyncTrace())
}

func printRuntimePeerStatuses(peers []*pbApic.RuntimePeerStatus) {
	fmt.Println("\nRuntime Peers")
	if len(peers) == 0 {
		fmt.Println(" No runtime peers reported")
		return
	}
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "Peer", "Connected", "Dialable", "Provider", "Compatible", "Reason")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "----", "---------", "--------", "--------", "----------", "------")
	for _, peer := range peers {
		fmt.Fprintf(w, " %s\t%t\t%t\t%t\t%t\t%s\t\n",
			peer.GetPeerId(),
			peer.GetConnected(),
			peer.GetDialable(),
			peer.GetStateProvider(),
			peer.GetCompatible(),
			peer.GetReason(),
		)
	}
}

func printRuntimeCompatibility(items []*pbApic.RuntimeCompatibility) {
	fmt.Println("\nCompatibility")
	if len(items) == 0 {
		fmt.Println(" No remote compatibility entries")
		return
	}
	w := tableWriter()
	defer w.Flush()
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "Peer", "Compatible", "Blocking", "Local Digest", "Remote Digest", "Reason")
	fmt.Fprintf(w, " %s\t%s\t%s\t%s\t%s\t%s\t\n", "----", "----------", "--------", "------------", "-------------", "------")
	for _, item := range items {
		fmt.Fprintf(w, " %s\t%t\t%t\t%s\t%s\t%s\t\n",
			item.GetPeerId(),
			item.GetCompatible(),
			item.GetBlocking(),
			shortValue(item.GetLocalDigest(), 12),
			shortValue(item.GetRemoteDigest(), 12),
			item.GetReason(),
		)
	}
}

func printRuntimeTrace(trace []string) {
	fmt.Println("\nContent Sync Trace")
	if len(trace) == 0 {
		fmt.Println(" No content sync trace entries")
		return
	}
	for _, entry := range trace {
		fmt.Printf(" %s\n", entry)
	}
}

func tableWriter() *tabwriter.Writer {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 0, 2, ' ', 0)
	return w
}

func emptyDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func shortValue(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
