package hostagentv1

#ApplyRequest: {
	desired_state?: #HostDesiredState
}

#ApplyResponse: {
	vms?:     [...#VMObservedState]
	network?: #NetworkObservedState
	message?: string
}

#StatusRequest: {
	vms?:     [...#VMRef]
	network?: bool
}

#StatusResponse: {
	vms?:     [...#VMObservedState]
	network?: #NetworkObservedState
	message?: string
}

#HostDesiredState: {
	vms?:     [...#VMDesiredState]
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
	desired_state?:         string
	config?:                #NetworkConfig
	instances?:             [...#InstancePeer]
	devices?:               [...#DevicePeer]
	namespaced_interfaces?: [...#NamespacedInterface]
	reconcile_peers?:       bool
}

#NetworkConfig: {
	ipv6_address?:          string
	wireguard_private_key?: string
	domain?:                string
}

#InstancePeer: {
	id?:         string
	name?:       string
	public_key?: string
	public_ip?:  string
	routes?:     [...string]
}

#DevicePeer: {
	name?:       string
	public_key?: string
}

#NamespacedInterface: {
	netns_path?: string
	ip?:         string
}

#NetworkObservedState: {
	module?:  string
	up?:      bool
	message?: string
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
			]},
			{kind: "message", name: "NetworkConfig", fields: [
				{type: "string", name: "ipv6_address", number: 1},
				{type: "string", name: "wireguard_private_key", number: 2},
				{type: "string", name: "domain", number: 3},
			]},
			{kind: "message", name: "InstancePeer", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "public_key", number: 3},
				{type: "string", name: "public_ip", number: 4},
				{rule: "repeated", type: "string", name: "routes", number: 5},
			]},
			{kind: "message", name: "DevicePeer", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "public_key", number: 2},
			]},
			{kind: "message", name: "NamespacedInterface", fields: [
				{type: "string", name: "netns_path", number: 1},
				{type: "string", name: "ip", number: 2},
			]},
			{kind: "message", name: "NetworkObservedState", fields: [
				{type: "string", name: "module", number: 1},
				{type: "bool", name: "up", number: 2},
				{type: "string", name: "message", number: 3},
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
			NetworkObservedState?: #NetworkObservedState
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
