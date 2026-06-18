package p2pv1

#ExecSQLRequest: {
	statement?: string
	max_rows?:  int
}

#ExecSQLResponse: {
	columns?:   [...string]
	rows?:      [...#SQLRow]
	truncated?: bool
	message?:   string
}

#SQLCell: {
	value?:   string
	is_null?: bool
}

#SQLRow: {
	cells?: [...#SQLCell]
}

#Commit: {
	hash?:      string
	committer?: string
	message?:   string
	date_unix?: int
	parent_hashes?: [...string]
	refs?:          [...string]
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
				{type: "int32", name: "max_rows", number: 2},
			]},
			{kind: "message", name: "ExecSQLResponse", fields: [
				{rule: "repeated", type: "string", name: "columns", number: 1},
				{rule: "repeated", type: "SQLRow", name: "rows", number: 2},
				{type: "bool", name: "truncated", number: 3},
				{type: "string", name: "message", number: 4},
			]},
			{kind: "message", name: "SQLCell", fields: [
				{type: "string", name: "value", number: 1},
				{type: "bool", name: "is_null", number: 2},
			]},
			{kind: "message", name: "SQLRow", fields: [
				{rule: "repeated", type: "SQLCell", name: "cells", number: 1},
			]},
				{kind: "message", name: "Commit", fields: [
					{type: "string", name: "hash", number: 1},
					{type: "string", name: "committer", number: 2},
					{type: "string", name: "message", number: 3},
					{type: "int64", name: "date_unix", number: 4},
					{rule: "repeated", type: "string", name: "parent_hashes", number: 5},
					{rule: "repeated", type: "string", name: "refs", number: 6},
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
			SQLCell?:                 #SQLCell
			SQLRow?:                  #SQLRow
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
