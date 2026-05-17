package p2pv1

#InitRequest: {
	origin_device?:            string
	origin_device_public_key?: string
	origin_swarmion_addrs?:    [...string]
	instance_name?:            string
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
		]
	}
}

lineage: {
	name: "protos.p2p.instance"
	schemas: [{
		version: [0, 0]
		schema: {
			InitRequest?:       #InitRequest
			InitResponse?:      #InitResponse
			GetPeersRequest?:   #GetPeersRequest
			GetPeersResponse?:  #GetPeersResponse
			GetLogsRequest?:    #GetLogsRequest
			GetLogsResponse?:   #GetLogsResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
