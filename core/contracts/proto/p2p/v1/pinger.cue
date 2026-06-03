package p2pv1

#PingRequest: {
	ping?: string
}

#PingResponse: {
	pong?: string
}

contract: {
	surface: "p2p-pinger-grpc"
	migration: {
		id:                  "protos-p2p-pinger-v0.0"
		lineage_id:          "protos.p2p.pinger"
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
			name: "Pinger"
			rpcs: [{name: "Ping", request: "PingRequest", response: "PingResponse"}]
		}]
		declarations: [
			{kind: "message", name: "PingRequest", fields: [
				{type: "string", name: "ping", number: 1},
			]},
			{kind: "message", name: "PingResponse", fields: [
				{type: "string", name: "pong", number: 1},
			]},
		]
	}
}

lineage: {
	name: "protos.p2p.pinger"
	schemas: [{
		version: [0, 0]
		schema: {
			PingRequest?:  #PingRequest
			PingResponse?: #PingResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
