package p2pv1

#GetAppLogsRequest: {
	app_name?: string
}

#GetAppLogsResponse: {
	logs?: string
}

#GetAppStatusRequest: {
	app_name?: string
}

#GetAppStatusResponse: {
	status?: string
}

#ProbeAppHTTPRequest: {
	app_name?:        string
	url?:             string
	timeout_seconds?: int32
	max_bytes?:       int32
}

#ProbeAppHTTPResponse: {
	body?:       bytes
	bytes_read?: int32
}

contract: {
	surface: "p2p-apps-grpc"
	migration: {
		id:                  "protos-p2p-apps-v0.0"
		lineage_id:          "protos.p2p.apps"
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
			name: "Apps"
			rpcs: [
				{name: "GetAppLogs", request: "GetAppLogsRequest", response: "GetAppLogsResponse"},
				{name: "GetAppStatus", request: "GetAppStatusRequest", response: "GetAppStatusResponse"},
				{name: "ProbeAppHTTP", request: "ProbeAppHTTPRequest", response: "ProbeAppHTTPResponse"},
			]
		}]
		declarations: [
			{kind: "message", name: "GetAppLogsRequest", fields: [
				{type: "string", name: "app_name", number: 1},
			]},
			{kind: "message", name: "GetAppLogsResponse", fields: [
				{type: "string", name: "logs", number: 1},
			]},
			{kind: "message", name: "GetAppStatusRequest", fields: [
				{type: "string", name: "app_name", number: 1},
			]},
			{kind: "message", name: "GetAppStatusResponse", fields: [
				{type: "string", name: "status", number: 1},
			]},
			{kind: "message", name: "ProbeAppHTTPRequest", fields: [
				{type: "string", name: "app_name", number: 1},
				{type: "string", name: "url", number: 2},
				{type: "int32", name: "timeout_seconds", number: 3},
				{type: "int32", name: "max_bytes", number: 4},
			]},
			{kind: "message", name: "ProbeAppHTTPResponse", fields: [
				{type: "bytes", name: "body", number: 1},
				{type: "int32", name: "bytes_read", number: 2},
			]},
		]
	}
}

lineage: {
	name: "protos.p2p.apps"
	schemas: [{
		version: [0, 0]
		schema: {
			GetAppLogsRequest?:    #GetAppLogsRequest
			GetAppLogsResponse?:   #GetAppLogsResponse
			GetAppStatusRequest?:  #GetAppStatusRequest
			GetAppStatusResponse?: #GetAppStatusResponse
			ProbeAppHTTPRequest?:  #ProbeAppHTTPRequest
			ProbeAppHTTPResponse?: #ProbeAppHTTPResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
