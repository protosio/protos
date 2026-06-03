package hostagentv1

#ApplyRequest: {
	desired_state?: #HostDesiredState
}

#ApplyResponse: {
	vms?: [...#VMObservedState]
	network?: #NetworkObservedState
	message?: string
}

#StatusRequest: {
	vms?: [...#VMRef]
	network?: bool
}

#StatusResponse: {
	vms?: [...#VMObservedState]
	network?: #NetworkObservedState
	message?: string
}

#ShutdownRequest: {}

#ShutdownResponse: {
	message?: string
}

#HostDesiredState: {
	vms?: [...#VMDesiredState]
	network?: #NetworkDesiredState
}

#VMDesiredState: {
	id?:            string
	manifest_path?: string
	desired_state?: string
}

#VMRef: {
	id?:            string
	manifest_path?: string
}

#VMObservedState: {
	id?:            string
	manifest_path?: string
	status?:        string
	pid?:           int32
	public_ip?:     string
	message?:       string
}

#NetworkDesiredState: {
	desired_state?: string
	config?:        #NetworkConfig
	instances?: [...#InstancePeer]
	devices?: [...#DevicePeer]
	namespaced_interfaces?: [...#NamespacedInterface]
	reconcile_peers?: bool
	exit_routes?: [...#ExitRoute]
}

#NetworkConfig: {
	ipv6_address?:          string
	wireguard_private_key?: string
	domain?:                string
	ipv4_address?:          string
	local_peer_id?:         string
}

#InstancePeer: {
	id?:         string
	name?:       string
	public_key?: string
	public_ip?:  string
	routes?: [...string]
	ipv4_address?: string
}

#DevicePeer: {
	name?:         string
	public_key?:   string
	id?:           string
	ipv4_address?: string
}

#NamespacedInterface: {
	netns_path?: string
	ip?:         string
}

#ExitRoute: {
	id?:          string
	device_id?:   string
	instance_id?: string
	cidrs?: [...string]
}

#NetworkObservedState: {
	module?:  string
	up?:      bool
	message?: string
	state?:   #NetworkState
}

#NetworkState: {
	module?:         string
	up?:             bool
	interface_name?: string
	interfaces?: [...#NetworkInterface]
	addresses?: [...#NetworkAddress]
	routes?: [...#NetworkRoute]
	wireguard_peers?: [...#WireGuardPeer]
	firewall_tables?: [...#FirewallTable]
	dns?: [...#DNSState]
	messages?: [...string]
}

#NetworkInterface: {
	name?:        string
	type?:        string
	index?:       int
	mtu?:         int
	up?:          bool
	master?:      string
	mac_address?: string
	kind?:        string
}

#NetworkAddress: {
	interface_name?: string
	cidr?:           string
	scope?:          string
}

#NetworkRoute: {
	interface_name?: string
	destination?:    string
	gateway?:        string
	source?:         string
	family?:         string
	table?:          string
	protocol?:       string
	scope?:          string
	priority?:       string
	kind?:           string
}

#WireGuardPeer: {
	public_key?: string
	endpoint?:   string
	allowed_ips?: [...string]
	latest_handshake?: string
	rx_bytes?:         uint
	tx_bytes?:         uint
}

#FirewallTable: {
	family?: string
	name?:   string
	chains?: [...#FirewallChain]
}

#FirewallChain: {
	name?:     string
	type?:     string
	hook?:     string
	priority?: string
	rules?: [...#FirewallRule]
}

#FirewallRule: {
	expressions?: [...string]
	packets?: uint
	bytes?:   uint
}

#DNSState: {
	scope?:  string
	domain?: string
	servers?: [...string]
	port?:   int
	active?: bool
	source?: string
}

contract: {
	surface: "hostagent-grpc"
	migration: {
		id:                  "protos-hostagent-v0.0"
		lineage_id:          "protos.hostagent"
		from_version:        ""
		to_version:          "0.0"
		compatibility:       "full"
		backward_compatible: true
		forward_compatible:  true
	}
	proto: {
		syntax:     "proto3"
		package:    "hostagent"
		go_package: "github.com/protosio/protos/internal/hostagent/proto;proto"
		services: [{
			name: "HostAgent"
			rpcs: [
				{name: "Apply", request: "ApplyRequest", response: "ApplyResponse"},
				{name: "Status", request: "StatusRequest", response: "StatusResponse"},
				{name: "Shutdown", request: "ShutdownRequest", response: "ShutdownResponse"},
			]
		}]
		declarations: [
			{kind: "message", name: "ApplyRequest", fields: [
				{type: "HostDesiredState", name: "desired_state", number: 1},
			]},
			{kind: "message", name: "ApplyResponse", fields: [
				{rule: "repeated", type: "VMObservedState", name: "vms", number: 1},
				{type: "NetworkObservedState", name: "network", number: 2},
				{type: "string", name: "message", number: 3},
			]},
			{kind: "message", name: "StatusRequest", fields: [
				{rule: "repeated", type: "VMRef", name: "vms", number: 1},
				{type: "bool", name: "network", number: 2},
			]},
			{kind: "message", name: "StatusResponse", fields: [
				{rule: "repeated", type: "VMObservedState", name: "vms", number: 1},
				{type: "NetworkObservedState", name: "network", number: 2},
				{type: "string", name: "message", number: 3},
			]},
			{kind: "message", name: "ShutdownRequest", fields: []},
			{kind: "message", name: "ShutdownResponse", fields: [
				{type: "string", name: "message", number: 1},
			]},
			{kind: "message", name: "HostDesiredState", fields: [
				{rule: "repeated", type: "VMDesiredState", name: "vms", number: 1},
				{type: "NetworkDesiredState", name: "network", number: 2},
			]},
			{kind: "message", name: "VMDesiredState", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "manifest_path", number: 2},
				{type: "string", name: "desired_state", number: 3},
			]},
			{kind: "message", name: "VMRef", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "manifest_path", number: 2},
			]},
			{kind: "message", name: "VMObservedState", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "manifest_path", number: 2},
				{type: "string", name: "status", number: 3},
				{type: "int32", name: "pid", number: 4},
				{type: "string", name: "public_ip", number: 5},
				{type: "string", name: "message", number: 6},
			]},
			{kind: "message", name: "NetworkDesiredState", fields: [
				{type: "string", name: "desired_state", number: 1},
				{type: "NetworkConfig", name: "config", number: 2},
				{rule: "repeated", type: "InstancePeer", name: "instances", number: 3},
				{rule: "repeated", type: "DevicePeer", name: "devices", number: 4},
				{rule: "repeated", type: "NamespacedInterface", name: "namespaced_interfaces", number: 5},
				{type: "bool", name: "reconcile_peers", number: 6},
				{rule: "repeated", type: "ExitRoute", name: "exit_routes", number: 7},
			]},
			{kind: "message", name: "NetworkConfig", fields: [
				{type: "string", name: "ipv6_address", number: 1},
				{type: "string", name: "wireguard_private_key", number: 2},
				{type: "string", name: "domain", number: 3},
				{type: "string", name: "ipv4_address", number: 4},
				{type: "string", name: "local_peer_id", number: 5},
			]},
			{kind: "message", name: "InstancePeer", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "public_key", number: 3},
				{type: "string", name: "public_ip", number: 4},
				{rule: "repeated", type: "string", name: "routes", number: 5},
				{type: "string", name: "ipv4_address", number: 6},
			]},
			{kind: "message", name: "DevicePeer", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "public_key", number: 2},
				{type: "string", name: "id", number: 3},
				{type: "string", name: "ipv4_address", number: 4},
			]},
			{kind: "message", name: "NamespacedInterface", fields: [
				{type: "string", name: "netns_path", number: 1},
				{type: "string", name: "ip", number: 2},
			]},
			{kind: "message", name: "ExitRoute", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "device_id", number: 2},
				{type: "string", name: "instance_id", number: 3},
				{rule: "repeated", type: "string", name: "cidrs", number: 4},
			]},
			{kind: "message", name: "NetworkObservedState", fields: [
				{type: "string", name: "module", number: 1},
				{type: "bool", name: "up", number: 2},
				{type: "string", name: "message", number: 3},
				{type: "NetworkState", name: "state", number: 4},
			]},
			{kind: "message", name: "NetworkState", fields: [
				{type: "string", name: "module", number: 1},
				{type: "bool", name: "up", number: 2},
				{type: "string", name: "interface_name", number: 3},
				{rule: "repeated", type: "NetworkAddress", name: "addresses", number: 4},
				{rule: "repeated", type: "NetworkRoute", name: "routes", number: 5},
				{rule: "repeated", type: "WireGuardPeer", name: "wireguard_peers", number: 6},
				{rule: "repeated", type: "FirewallTable", name: "firewall_tables", number: 7},
				{rule: "repeated", type: "DNSState", name: "dns", number: 8},
				{rule: "repeated", type: "string", name: "messages", number: 9},
				{rule: "repeated", type: "NetworkInterface", name: "interfaces", number: 10},
			]},
			{kind: "message", name: "NetworkInterface", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "type", number: 2},
				{type: "int32", name: "index", number: 3},
				{type: "int32", name: "mtu", number: 4},
				{type: "bool", name: "up", number: 5},
				{type: "string", name: "master", number: 6},
				{type: "string", name: "mac_address", number: 7},
				{type: "string", name: "kind", number: 8},
			]},
			{kind: "message", name: "NetworkAddress", fields: [
				{type: "string", name: "interface_name", number: 1},
				{type: "string", name: "cidr", number: 2},
				{type: "string", name: "scope", number: 3},
			]},
			{kind: "message", name: "NetworkRoute", fields: [
				{type: "string", name: "interface_name", number: 1},
				{type: "string", name: "destination", number: 2},
				{type: "string", name: "gateway", number: 3},
				{type: "string", name: "source", number: 4},
				{type: "string", name: "family", number: 5},
				{type: "string", name: "table", number: 6},
				{type: "string", name: "protocol", number: 7},
				{type: "string", name: "scope", number: 8},
				{type: "string", name: "priority", number: 9},
				{type: "string", name: "kind", number: 10},
			]},
			{kind: "message", name: "WireGuardPeer", fields: [
				{type: "string", name: "public_key", number: 1},
				{type: "string", name: "endpoint", number: 2},
				{rule: "repeated", type: "string", name: "allowed_ips", number: 3},
				{type: "string", name: "latest_handshake", number: 4},
				{type: "uint64", name: "rx_bytes", number: 5},
				{type: "uint64", name: "tx_bytes", number: 6},
			]},
			{kind: "message", name: "FirewallTable", fields: [
				{type: "string", name: "family", number: 1},
				{type: "string", name: "name", number: 2},
				{rule: "repeated", type: "FirewallChain", name: "chains", number: 3},
			]},
			{kind: "message", name: "FirewallChain", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "type", number: 2},
				{type: "string", name: "hook", number: 3},
				{type: "string", name: "priority", number: 4},
				{rule: "repeated", type: "FirewallRule", name: "rules", number: 5},
			]},
			{kind: "message", name: "FirewallRule", fields: [
				{rule: "repeated", type: "string", name: "expressions", number: 1},
				{type: "uint64", name: "packets", number: 2},
				{type: "uint64", name: "bytes", number: 3},
			]},
			{kind: "message", name: "DNSState", fields: [
				{type: "string", name: "scope", number: 1},
				{type: "string", name: "domain", number: 2},
				{rule: "repeated", type: "string", name: "servers", number: 3},
				{type: "int32", name: "port", number: 4},
				{type: "bool", name: "active", number: 5},
				{type: "string", name: "source", number: 6},
			]},
		]
	}
}

lineage: {
	name: "protos.hostagent"
	schemas: [{
		version: [0, 0]
		schema: {
			ApplyRequest?:         #ApplyRequest
			ApplyResponse?:        #ApplyResponse
			StatusRequest?:        #StatusRequest
			StatusResponse?:       #StatusResponse
			HostDesiredState?:     #HostDesiredState
			VMDesiredState?:       #VMDesiredState
			VMRef?:                #VMRef
			VMObservedState?:      #VMObservedState
			NetworkDesiredState?:  #NetworkDesiredState
			NetworkConfig?:        #NetworkConfig
			InstancePeer?:         #InstancePeer
			DevicePeer?:           #DevicePeer
			NamespacedInterface?:  #NamespacedInterface
			ExitRoute?:            #ExitRoute
			NetworkObservedState?: #NetworkObservedState
			NetworkState?:         #NetworkState
			NetworkInterface?:     #NetworkInterface
			NetworkAddress?:       #NetworkAddress
			NetworkRoute?:         #NetworkRoute
			WireGuardPeer?:        #WireGuardPeer
			FirewallTable?:        #FirewallTable
			FirewallChain?:        #FirewallChain
			FirewallRule?:         #FirewallRule
			DNSState?:             #DNSState
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
