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
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
