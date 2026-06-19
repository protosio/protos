package p2pv1

#InitRequest: {
	origin_device?:            string
	origin_device_public_key?: string
	origin_swarmion_addrs?: [...string]
	instance_name?: string
}

#InitResponse: {
	architecture?: string
}

#GetPeersRequest: {}

#GetPeersResponse: {
	peers?: [string]: string
}

#GetLogsRequest: {}

#GetLogsResponse: {
	logs?: string
}

#GetNetworkStateRequest: {}
#GetNetworkStateResponse: state?: #NetworkState
#GetExitRoutesRequest: {}
#GetExitRoutesResponse: routes?: [...#ExitRoute]
#GetRuntimeStateRequest: allow_stale?: bool
#GetRuntimeStateResponse: state?:      #RuntimeState
#Task: {
	id?:            string
	stream?:        string
	subject_type?:  string
	subject_id?:    string
	status?:        string
	title?:         string
	message?:       string
	progress?:      int
	payload_json?:  string
	result_json?:   string
	error_message?: string
	attempts?:      int
	max_attempts?:  int
	created_at?:    string
	updated_at?:    string
	started_at?:    string
	finished_at?:   string
}
#TaskEvent: {
	id?:           string
	task_id?:      string
	status?:       string
	message?:      string
	progress?:     int
	details_json?: string
	created_at?:   string
}
#GetTasksRequest: {
	status?:       string
	stream?:       string
	subject_type?: string
	subject_id?:   string
	max_results?:  int
}
#GetTasksResponse: {
	tasks?: [...#Task]
	truncated?: bool
}
#GetTaskRequest: {
	id?:             string
	include_events?: bool
}
#GetTaskResponse: {
	task?: #Task
	events?: [...#TaskEvent]
}
#TaskProgressUpdate: {
	task_id?:      string
	status?:       string
	message?:      string
	progress?:     int
	details_json?: string
	created_at?:   string
	durable?:      bool
}
#WatchTaskRequest: {
	id?:                    string
	include_snapshot?:      bool
	include_events?:        bool
	heartbeat_interval_ms?: uint
}
#WatchTaskResponse: {
	sequence?: uint
	task?:     #Task
	events?: [...#TaskEvent]
	update?:    #TaskProgressUpdate
	heartbeat?: bool
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
#ExitRoute: {
	id?:          string
	device_id?:   string
	instance_id?: string
	status?:      string
	dns_server?:  string
	cidrs?: [...string]
}
#RuntimeState: {
	peer_id?:                       string
	manifest_digest?:               string
	checkpoint_root_hash?:          string
	tentative_root_hash?:           string
	protocol_checkpoint_root_hash?: string
	durable_main_root_hash?:        string
	state_providers?: [...string]
	connected_peers?: [...string]
	fatal_state?:                    string
	runtime_refresh_pending?:        bool
	runtime_refresh_last_error?:     string
	runtime_checkpoint_pending?:     bool
	runtime_checkpoint_last_error?:  string
	runtime_materialization_policy?: string
	peer_statuses?: [...#RuntimePeerStatus]
	compatibility?: [...#RuntimeCompatibility]
	content_sync_trace?: [...string]
	protocol_checkpoint_digest?: string
	read_consistency?:           string
	read_error?:                 string
}
#RuntimePeerStatus: {
	peer_id?:        string
	connected?:      bool
	dialable?:       bool
	state_provider?: bool
	compatible?:     bool
	incompatible?:   bool
	ignored?:        bool
	relay_only?:     bool
	addresses?: [...string]
	last_dial_errors?: [string]: string
	reason?: string
}
#RuntimeCompatibility: {
	peer_id?:       string
	local_digest?:  string
	remote_digest?: string
	compatible?:    bool
	blocking?:      bool
	reason?:        string
}

contract: {
	surface: "p2p-instance-grpc"
	migration: {
		id:                  "protos-p2p-instance-v0.0"
		lineage_id:          "protos.p2p.instance"
		from_version:        ""
		to_version:          "0.0"
		compatibility:       "full"
		backward_compatible: true
		forward_compatible:  true
	}
	proto: {
		syntax:     "proto3"
		package:    "proto"
		go_package: "./proto"
		services: [{
			name: "Instance"
			rpcs: [
				{name: "Init", request: "InitRequest", response: "InitResponse"},
				{name: "GetPeers", request: "GetPeersRequest", response: "GetPeersResponse"},
				{name: "GetLogs", request: "GetLogsRequest", response: "GetLogsResponse"},
				{name: "GetNetworkState", request: "GetNetworkStateRequest", response: "GetNetworkStateResponse"},
				{name: "GetExitRoutes", request: "GetExitRoutesRequest", response: "GetExitRoutesResponse"},
				{name: "GetRuntimeState", request: "GetRuntimeStateRequest", response: "GetRuntimeStateResponse"},
				{name: "GetTasks", request: "GetTasksRequest", response: "GetTasksResponse"},
				{name: "GetTask", request: "GetTaskRequest", response: "GetTaskResponse"},
				{name: "WatchTask", request: "WatchTaskRequest", response: "WatchTaskResponse", response_stream: true},
			]
		}]
		declarations: [
			{kind: "message", name: "InitRequest", fields: [
				{type: "string", name: "origin_device", number: 1},
				{type: "string", name: "origin_device_public_key", number: 2},
				{rule: "repeated", type: "string", name: "origin_swarmion_addrs", number: 3},
				{type: "string", name: "instance_name", number: 4},
			]},
			{kind: "message", name: "InitResponse", fields: [
				{type: "string", name: "architecture", number: 1},
			]},
			{kind: "message", name: "GetPeersRequest", fields: []},
			{kind: "message", name: "GetPeersResponse", fields: [
				{type: "map<string, string>", name: "peers", number: 1},
			]},
			{kind: "message", name: "GetLogsRequest", fields: []},
			{kind: "message", name: "GetLogsResponse", fields: [
				{type: "string", name: "logs", number: 1},
			]},
			{kind: "message", name: "GetNetworkStateRequest", fields: []},
			{kind: "message", name: "GetNetworkStateResponse", fields: [
				{type: "NetworkState", name: "state", number: 1},
			]},
			{kind: "message", name: "GetExitRoutesRequest", fields: []},
			{kind: "message", name: "GetExitRoutesResponse", fields: [
				{rule: "repeated", type: "ExitRoute", name: "routes", number: 1},
			]},
			{kind: "message", name: "GetRuntimeStateRequest", fields: [
				{type: "bool", name: "allow_stale", number: 1},
			]},
			{kind: "message", name: "GetRuntimeStateResponse", fields: [
				{type: "RuntimeState", name: "state", number: 1},
			]},
			{kind: "message", name: "Task", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "stream", number: 2},
				{type: "string", name: "subject_type", number: 3},
				{type: "string", name: "subject_id", number: 4},
				{type: "string", name: "status", number: 5},
				{type: "string", name: "title", number: 6},
				{type: "string", name: "message", number: 7},
				{type: "int32", name: "progress", number: 8},
				{type: "string", name: "payload_json", number: 9},
				{type: "string", name: "result_json", number: 10},
				{type: "string", name: "error_message", number: 11},
				{type: "int32", name: "attempts", number: 12},
				{type: "int32", name: "max_attempts", number: 13},
				{type: "string", name: "created_at", number: 14},
				{type: "string", name: "updated_at", number: 15},
				{type: "string", name: "started_at", number: 16},
				{type: "string", name: "finished_at", number: 17},
			]},
			{kind: "message", name: "TaskEvent", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "task_id", number: 2},
				{type: "string", name: "status", number: 3},
				{type: "string", name: "message", number: 4},
				{type: "int32", name: "progress", number: 5},
				{type: "string", name: "details_json", number: 6},
				{type: "string", name: "created_at", number: 7},
			]},
			{kind: "message", name: "GetTasksRequest", fields: [
				{type: "string", name: "status", number: 1},
				{type: "string", name: "stream", number: 2},
				{type: "string", name: "subject_type", number: 3},
				{type: "string", name: "subject_id", number: 4},
				{type: "int32", name: "max_results", number: 5},
			]},
			{kind: "message", name: "GetTasksResponse", fields: [
				{rule: "repeated", type: "Task", name: "tasks", number: 1},
				{type: "bool", name: "truncated", number: 2},
			]},
			{kind: "message", name: "GetTaskRequest", fields: [
				{type: "string", name: "id", number: 1},
				{type: "bool", name: "include_events", number: 2},
			]},
			{kind: "message", name: "GetTaskResponse", fields: [
				{type: "Task", name: "task", number: 1},
				{rule: "repeated", type: "TaskEvent", name: "events", number: 2},
			]},
			{kind: "message", name: "TaskProgressUpdate", fields: [
				{type: "string", name: "task_id", number: 1},
				{type: "string", name: "status", number: 2},
				{type: "string", name: "message", number: 3},
				{type: "int32", name: "progress", number: 4},
				{type: "string", name: "details_json", number: 5},
				{type: "string", name: "created_at", number: 6},
				{type: "bool", name: "durable", number: 7},
			]},
			{kind: "message", name: "WatchTaskRequest", fields: [
				{type: "string", name: "id", number: 1},
				{type: "bool", name: "include_snapshot", number: 2},
				{type: "bool", name: "include_events", number: 3},
				{type: "uint32", name: "heartbeat_interval_ms", number: 4},
			]},
			{kind: "message", name: "WatchTaskResponse", fields: [
				{type: "uint64", name: "sequence", number: 1},
				{type: "Task", name: "task", number: 2},
				{rule: "repeated", type: "TaskEvent", name: "events", number: 3},
				{type: "TaskProgressUpdate", name: "update", number: 4},
				{type: "bool", name: "heartbeat", number: 5},
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
			{kind: "message", name: "ExitRoute", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "device_id", number: 2},
				{type: "string", name: "instance_id", number: 3},
				{type: "string", name: "status", number: 4},
				{type: "string", name: "dns_server", number: 5},
				{rule: "repeated", type: "string", name: "cidrs", number: 6},
			]},
			{kind: "message", name: "RuntimeState", fields: [
				{type: "string", name: "peer_id", number: 1},
				{type: "string", name: "manifest_digest", number: 2},
				{type: "string", name: "checkpoint_root_hash", number: 3},
				{type: "string", name: "tentative_root_hash", number: 4},
				{type: "string", name: "protocol_checkpoint_root_hash", number: 5},
				{type: "string", name: "durable_main_root_hash", number: 6},
				{rule: "repeated", type: "string", name: "state_providers", number: 10},
				{rule: "repeated", type: "string", name: "connected_peers", number: 11},
				{type: "string", name: "fatal_state", number: 12},
				{type: "bool", name: "runtime_refresh_pending", number: 13},
				{type: "string", name: "runtime_refresh_last_error", number: 14},
				{type: "bool", name: "runtime_checkpoint_pending", number: 15},
				{type: "string", name: "runtime_checkpoint_last_error", number: 16},
				{type: "string", name: "runtime_materialization_policy", number: 17},
				{rule: "repeated", type: "RuntimePeerStatus", name: "peer_statuses", number: 18},
				{rule: "repeated", type: "RuntimeCompatibility", name: "compatibility", number: 19},
				{rule: "repeated", type: "string", name: "content_sync_trace", number: 20},
				{type: "string", name: "protocol_checkpoint_digest", number: 24},
				{type: "string", name: "read_consistency", number: 25},
				{type: "string", name: "read_error", number: 26},
			]},
			{kind: "message", name: "RuntimePeerStatus", fields: [
				{type: "string", name: "peer_id", number: 1},
				{type: "bool", name: "connected", number: 2},
				{type: "bool", name: "dialable", number: 3},
				{type: "bool", name: "state_provider", number: 4},
				{type: "bool", name: "compatible", number: 7},
				{type: "bool", name: "incompatible", number: 8},
				{type: "bool", name: "ignored", number: 9},
				{type: "bool", name: "relay_only", number: 10},
				{rule: "repeated", type: "string", name: "addresses", number: 11},
				{type: "map<string, string>", name: "last_dial_errors", number: 12},
				{type: "string", name: "reason", number: 13},
			]},
			{kind: "message", name: "RuntimeCompatibility", fields: [
				{type: "string", name: "peer_id", number: 1},
				{type: "string", name: "local_digest", number: 2},
				{type: "string", name: "remote_digest", number: 3},
				{type: "bool", name: "compatible", number: 4},
				{type: "bool", name: "blocking", number: 5},
				{type: "string", name: "reason", number: 6},
			]},
		]
	}
}

lineage: {
	name: "protos.p2p.instance"
	schemas: [{
		version: [0, 0]
		schema: {
			InitRequest?:             #InitRequest
			InitResponse?:            #InitResponse
			GetPeersRequest?:         #GetPeersRequest
			GetPeersResponse?:        #GetPeersResponse
			GetLogsRequest?:          #GetLogsRequest
			GetLogsResponse?:         #GetLogsResponse
			GetNetworkStateRequest?:  #GetNetworkStateRequest
			GetNetworkStateResponse?: #GetNetworkStateResponse
			NetworkState?:            #NetworkState
			NetworkInterface?:        #NetworkInterface
			NetworkAddress?:          #NetworkAddress
			NetworkRoute?:            #NetworkRoute
			WireGuardPeer?:           #WireGuardPeer
			FirewallTable?:           #FirewallTable
			FirewallChain?:           #FirewallChain
			FirewallRule?:            #FirewallRule
			DNSState?:                #DNSState
			ExitRoute?:               #ExitRoute
			GetExitRoutesRequest?:    #GetExitRoutesRequest
			GetExitRoutesResponse?:   #GetExitRoutesResponse
			GetRuntimeStateRequest?:  #GetRuntimeStateRequest
			GetRuntimeStateResponse?: #GetRuntimeStateResponse
			Task?:                    #Task
			TaskEvent?:               #TaskEvent
			GetTasksRequest?:         #GetTasksRequest
			GetTasksResponse?:        #GetTasksResponse
			GetTaskRequest?:          #GetTaskRequest
			GetTaskResponse?:         #GetTaskResponse
			TaskProgressUpdate?:      #TaskProgressUpdate
			WatchTaskRequest?:        #WatchTaskRequest
			WatchTaskResponse?:       #WatchTaskResponse
			RuntimeState?:            #RuntimeState
			RuntimePeerStatus?:       #RuntimePeerStatus
			RuntimeCompatibility?:    #RuntimeCompatibility
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
