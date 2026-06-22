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

#CommitDiffValue: {
	value?:   string
	is_null?: bool
}

#CommitDiffField: {
	name?:        string
	before?:      #CommitDiffValue
	after?:       #CommitDiffValue
	before_cue?:  string
	after_cue?:   string
	changed?:     bool
}

#CommitDiffRow: {
	change_type?: string
	key?:         string
	fields?:      [...#CommitDiffField]
	before_cue?:  string
	after_cue?:   string
	cue?:         string
}

#CommitDiffTable: {
	name?: string
	rows?: [...#CommitDiffRow]
	cue?:  string
}

#CommitDiffTaskContext: {
	id?:             string
	stream?:         string
	subject_type?:   string
	subject_id?:     string
	owner_peer_id?:  string
	status?:         string
	title?:          string
	message?:        string
	progress?:       int
	change_sources?: [...string]
	event_count?:    int
	summary?:        string
}

#CommitDiff: {
	base_hash?:     string
	target_hash?:   string
	tables?:        [...#CommitDiffTable]
	cue?:           string
	truncated?:     bool
	message?:       string
	unified_diff?: string
	related_tasks?: [...#CommitDiffTaskContext]
	sql?: string
}

#GetCommitDiffRequest: {
	commit_hash?: string
	base_hash?:   string
}

#GetCommitDiffResponse: diff?: #CommitDiff

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
				{name: "GetCommitDiff", request: "GetCommitDiffRequest", response: "GetCommitDiffResponse"},
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
			{kind: "message", name: "CommitDiffValue", fields: [
				{type: "string", name: "value", number: 1},
				{type: "bool", name: "is_null", number: 2},
			]},
			{kind: "message", name: "CommitDiffField", fields: [
				{type: "string", name: "name", number: 1},
				{type: "CommitDiffValue", name: "before", number: 2},
				{type: "CommitDiffValue", name: "after", number: 3},
				{type: "string", name: "before_cue", number: 4},
				{type: "string", name: "after_cue", number: 5},
				{type: "bool", name: "changed", number: 6},
			]},
			{kind: "message", name: "CommitDiffRow", fields: [
				{type: "string", name: "change_type", number: 1},
				{type: "string", name: "key", number: 2},
				{rule: "repeated", type: "CommitDiffField", name: "fields", number: 3},
				{type: "string", name: "before_cue", number: 4},
				{type: "string", name: "after_cue", number: 5},
				{type: "string", name: "cue", number: 6},
			]},
			{kind: "message", name: "CommitDiffTable", fields: [
				{type: "string", name: "name", number: 1},
				{rule: "repeated", type: "CommitDiffRow", name: "rows", number: 2},
				{type: "string", name: "cue", number: 3},
			]},
			{kind: "message", name: "CommitDiffTaskContext", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "stream", number: 2},
				{type: "string", name: "subject_type", number: 3},
				{type: "string", name: "subject_id", number: 4},
				{type: "string", name: "owner_peer_id", number: 5},
				{type: "string", name: "status", number: 6},
				{type: "string", name: "title", number: 7},
				{type: "string", name: "message", number: 8},
				{type: "int32", name: "progress", number: 9},
				{rule: "repeated", type: "string", name: "change_sources", number: 10},
				{type: "int32", name: "event_count", number: 11},
				{type: "string", name: "summary", number: 12},
			]},
			{kind: "message", name: "CommitDiff", fields: [
				{type: "string", name: "base_hash", number: 1},
				{type: "string", name: "target_hash", number: 2},
				{rule: "repeated", type: "CommitDiffTable", name: "tables", number: 3},
				{type: "string", name: "cue", number: 4},
				{type: "bool", name: "truncated", number: 5},
				{type: "string", name: "message", number: 6},
				{type: "string", name: "unified_diff", number: 7},
				{rule: "repeated", type: "CommitDiffTaskContext", name: "related_tasks", number: 8},
				{type: "string", name: "sql", number: 9},
			]},
			{kind: "message", name: "GetCommitDiffRequest", fields: [
				{type: "string", name: "commit_hash", number: 1},
				{type: "string", name: "base_hash", number: 2},
			]},
			{kind: "message", name: "GetCommitDiffResponse", fields: [
				{type: "CommitDiff", name: "diff", number: 1},
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
			CommitDiffValue?:         #CommitDiffValue
			CommitDiffField?:         #CommitDiffField
			CommitDiffRow?:           #CommitDiffRow
			CommitDiffTable?:         #CommitDiffTable
			CommitDiffTaskContext?:   #CommitDiffTaskContext
			CommitDiff?:              #CommitDiff
			GetCommitDiffRequest?:    #GetCommitDiffRequest
			GetCommitDiffResponse?:   #GetCommitDiffResponse
			GetHeadRequest?:          #GetHeadRequest
			GetHeadResponse?:         #GetHeadResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
