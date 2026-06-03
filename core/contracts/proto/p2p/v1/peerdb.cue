package p2pv1

#ExecSQLRequest: {
	statement?: string
	msg?:       string
}

#ExecSQLResponse: {
	commit?: string
	result?: string
	err?:    string
}

#Commit: {
	hash?:      string
	committer?: string
	message?:   string
}

#GetAllCommitsRequest: {}

#GetAllCommitsResponse: {
	commits?: [...#Commit]
}

#GetHeadRequest: {}

#GetHeadResponse: {
	commit?: string
}

contract: {
	surface: "p2p-db-grpc"
	migration: {
		id:                  "protos-p2p-db-v0.0"
		lineage_id:          "protos.p2p.db"
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
			name: "PeerDB"
			rpcs: [
				{name: "ExecSQL", request: "ExecSQLRequest", response: "ExecSQLResponse"},
				{name: "GetAllCommits", request: "GetAllCommitsRequest", response: "GetAllCommitsResponse"},
				{name: "GetHead", request: "GetHeadRequest", response: "GetHeadResponse"},
			]
		}]
		declarations: [
			{kind: "message", name: "ExecSQLRequest", fields: [
				{type: "string", name: "statement", number: 1},
				{type: "string", name: "msg", number: 2},
			]},
			{kind: "message", name: "ExecSQLResponse", fields: [
				{type: "string", name: "commit", number: 1},
				{type: "string", name: "result", number: 2},
				{type: "string", name: "err", number: 3},
			]},
			{kind: "message", name: "Commit", fields: [
				{type: "string", name: "hash", number: 1},
				{type: "string", name: "committer", number: 2},
				{type: "string", name: "message", number: 3},
			]},
			{kind: "message", name: "GetAllCommitsRequest", fields: []},
			{kind: "message", name: "GetAllCommitsResponse", fields: [
				{rule: "repeated", type: "Commit", name: "commits", number: 1},
			]},
			{kind: "message", name: "GetHeadRequest", fields: []},
			{kind: "message", name: "GetHeadResponse", fields: [
				{type: "string", name: "commit", number: 1},
			]},
		]
	}
}

lineage: {
	name: "protos.p2p.db"
	schemas: [{
		version: [0, 0]
		schema: {
			ExecSQLRequest?:          #ExecSQLRequest
			ExecSQLResponse?:         #ExecSQLResponse
			Commit?:                  #Commit
			GetAllCommitsRequest?:    #GetAllCommitsRequest
			GetAllCommitsResponse?:   #GetAllCommitsResponse
			GetHeadRequest?:          #GetHeadRequest
			GetHeadResponse?:         #GetHeadResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
