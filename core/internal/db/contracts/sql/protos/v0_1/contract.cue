package protos

#Machine: {
	id:              string
	name:            string
	kind:            string
	desired_status?: string
	replication_priority: int
}

#CloudMachineMetadata: {
	id:                    string
	cloud_id:              string
	provider_resource_id?: string
	public_ip:             string
	location:              string
	architecture:          string
	public_key:            string
	lifecycle_owner_peer_id: string
}

#CloudProvider: {
	id:   string
	name: string
	type: string
	auth: _
}

#App: {
	id:             string
	name:           string
	installer_ref:  string
	instance_id:    string
	desired_status: string
	persistence:    bool
	public_key:     string
}

#Organisation: {
	id:         string
	name:       string
	created_at: string
}

#User: {
	id:          string
	username:    string
	name?:       string
	is_disabled: bool
}

#UserDeviceMetadata: {
	id:           string
	public_key:   string
	user_id:      string
	name:         string
	replication_priority: int
}

#Peer: {
	id:         string
	public_key: string
}

#Task: {
	id:            string
	task_stream:   string
	subject_type:  string
	subject_id:    string
	status:        string
	title:         string
	message:       string
	progress:      int
	payload:       _
	result?:       _
	error_message?: string
	attempts:      int
	max_attempts:  int
	created_at:    string
	updated_at:    string
	started_at?:   string
	finished_at?:  string
}

#TaskEvent: {
	id:         string
	task_id:    string
	status:     string
	message:    string
	progress:   int
	details?:   _
	created_at: string
}

// TaskOperationFact is immutable, append-only recovery authority for a task
// operation. Its identity and payload are deterministic application data: it
// deliberately has no observer timestamp, executor, attempt, or mutable
// progress fields, so two peers recording the same fact produce identical
// content.
#TaskOperationFact: {
	id:             string
	task_id:        string
	fact_kind:      string
	operation_key:  string
	intent_digest:  string
	author_peer_id: string
	subject_type:   string
	subject_id:     string
	payload:        _
}

#ExitRoute: {
	id:             string
	device_id:      string
	instance_id:    string
	desired_status: string
	dns_server:     string
	cidrs:          string
}

contract: {
	surface: "sql"
	migration: {
		id:                  "protos-db-v0.1"
		lineage_id:          "protos.db"
		from_version:        "0.0"
		to_version:          "0.1"
		compatibility:       "breaking"
		backward_compatible: false
		forward_compatible:  false
		required_write_columns: []
		forbidden_write_columns: []
		initial_transition: true
	}
	schema: {
		tables: [
			{
				name: "machines"
				go: name: "MACHINE"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "kind", type: "VARCHAR(255)", not_null: true, go: {name: "KIND", sq_type: "StringField", ddl: "notnull index"}},
					{name: "desired_status", type: "VARCHAR(255)", go: {name: "DESIRED_STATUS", sq_type: "StringField", ddl: "index"}},
					{name: "replication_priority", type: "INT", not_null: true, go: {name: "REPLICATION_PRIORITY", sq_type: "NumberField", ddl: "notnull"}},
				]
				indexes: [
					{name: "machines_name_idx", columns: ["name"]},
					{name: "machines_kind_idx", columns: ["kind"]},
					{name: "machines_desired_status_idx", columns: ["desired_status"]},
				]
			},
			{
				name: "cloud_machines_metadata"
				go: name: "CLOUD_MACHINE_METADATA"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "cloud_id", type: "VARCHAR(255)", not_null: true, go: {name: "CLOUD_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "provider_resource_id", type: "VARCHAR(255)", go: {name: "PROVIDER_RESOURCE_ID", sq_type: "StringField", ddl: "index"}},
					{name: "public_ip", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_IP", sq_type: "StringField", ddl: "notnull"}},
					{name: "location", type: "VARCHAR(255)", not_null: true, go: {name: "LOCATION", sq_type: "StringField", ddl: "notnull"}},
					{name: "architecture", type: "VARCHAR(255)", not_null: true, go: {name: "ARCHITECTURE", sq_type: "StringField", ddl: "notnull"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
					{name: "lifecycle_owner_peer_id", type: "VARCHAR(255)", not_null: true, go: {name: "LIFECYCLE_OWNER_PEER_ID", sq_type: "StringField", ddl: "notnull"}},
				]
				indexes: [
					{name: "cloud_machines_metadata_cloud_id_idx", columns: ["cloud_id"]},
					{name: "cloud_machines_metadata_provider_resource_id_idx", columns: ["provider_resource_id"]},
					{name: "cloud_machines_metadata_public_key_idx", columns: ["public_key"]},
				]
			},
			{
				name: "cloud_providers"
				go: name: "CLOUD_PROVIDER"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "type", type: "VARCHAR(255)", not_null: true, go: {name: "TYPE", sq_type: "StringField", ddl: "notnull index"}},
					{name: "auth", type: "JSON", not_null: true, go: {name: "AUTH", sq_type: "JSONField", ddl: "notnull"}},
				]
				indexes: [
					{name: "cloud_providers_name_idx", columns: ["name"]},
					{name: "cloud_providers_type_idx", columns: ["type"]},
				]
			},
			{
				name: "apps"
				go: name: "APP"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "installer_ref", type: "VARCHAR(255)", not_null: true, go: {name: "INSTALLER_REF", sq_type: "StringField", ddl: "notnull"}},
					{name: "instance_id", type: "VARCHAR(255)", not_null: true, go: {name: "INSTANCE_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "desired_status", type: "VARCHAR(255)", not_null: true, go: {name: "DESIRED_STATUS", sq_type: "StringField", ddl: "notnull"}},
					{name: "persistence", type: "TINYINT(1)", not_null: true, go: {name: "PERSISTENCE", sq_type: "BooleanField", ddl: "notnull"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
				]
				indexes: [
					{name: "apps_name_idx", columns: ["name"]},
					{name: "apps_instance_id_idx", columns: ["instance_id"]},
					{name: "apps_public_key_idx", columns: ["public_key"]},
				]
			},
			{
				name: "organisations"
				go: name: "ORGANISATION"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "created_at", type: "VARCHAR(64)", not_null: true, go: {name: "CREATED_AT", sq_type: "StringField", ddl: "notnull"}},
				]
				indexes: [{name: "organisations_name_idx", columns: ["name"]}]
			},
			{
				name: "users"
				go: name: "USER"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "username", type: "VARCHAR(255)", not_null: true, go: {name: "USERNAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "name", type: "VARCHAR(255)", go: {name: "NAME", sq_type: "StringField"}},
					{name: "is_disabled", type: "TINYINT(1)", not_null: true, go: {name: "IS_DISABLED", sq_type: "BooleanField", ddl: "notnull"}},
				]
				indexes: [{name: "users_username_idx", columns: ["username"]}]
			},
			{
				name: "user_devices_metadata"
				go: name: "USER_DEVICE_METADATA"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
					{name: "user_id", type: "BINARY(16)", not_null: true, go: {name: "USER_ID", sq_type: "BinaryField", ddl: "notnull index"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull"}},
					{name: "replication_priority", type: "INT", not_null: true, go: {name: "REPLICATION_PRIORITY", sq_type: "NumberField", ddl: "notnull"}},
				]
				indexes: [
					{name: "user_devices_metadata_public_key_idx", columns: ["public_key"]},
					{name: "user_devices_metadata_user_id_idx", columns: ["user_id"]},
				]
			},
			{
				name: "peers"
				go: name: "PEER"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
				]
				indexes: [{name: "peers_public_key_idx", columns: ["public_key"]}]
			},
			{
				name: "tasks"
				go: name: "TASK"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "task_stream", type: "VARCHAR(255)", not_null: true, go: {name: "TASK_STREAM", sq_type: "StringField", ddl: "notnull index"}},
					{name: "subject_type", type: "VARCHAR(255)", not_null: true, go: {name: "SUBJECT_TYPE", sq_type: "StringField", ddl: "notnull index"}},
					{name: "subject_id", type: "VARCHAR(255)", not_null: true, go: {name: "SUBJECT_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "owner_peer_id", type: "VARCHAR(255)", not_null: true, go: {name: "OWNER_PEER_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "status", type: "VARCHAR(64)", not_null: true, go: {name: "STATUS", sq_type: "StringField", ddl: "notnull index"}},
					{name: "title", type: "VARCHAR(255)", not_null: true, go: {name: "TITLE", sq_type: "StringField", ddl: "notnull"}},
					{name: "message", type: "TEXT", not_null: true, go: {name: "MESSAGE", sq_type: "StringField", ddl: "notnull"}},
					{name: "progress", type: "INT", not_null: true, go: {name: "PROGRESS", sq_type: "NumberField", ddl: "notnull"}},
					{name: "payload", type: "JSON", not_null: true, go: {name: "PAYLOAD", sq_type: "JSONField", ddl: "notnull"}},
					{name: "result", type: "JSON", go: {name: "RESULT", sq_type: "JSONField"}},
					{name: "error_message", type: "TEXT", go: {name: "ERROR_MESSAGE", sq_type: "StringField"}},
					{name: "attempts", type: "INT", not_null: true, go: {name: "ATTEMPTS", sq_type: "NumberField", ddl: "notnull"}},
					{name: "max_attempts", type: "INT", not_null: true, go: {name: "MAX_ATTEMPTS", sq_type: "NumberField", ddl: "notnull"}},
					{name: "created_at", type: "VARCHAR(64)", not_null: true, go: {name: "CREATED_AT", sq_type: "StringField", ddl: "notnull"}},
					{name: "updated_at", type: "VARCHAR(64)", not_null: true, go: {name: "UPDATED_AT", sq_type: "StringField", ddl: "notnull"}},
					{name: "started_at", type: "VARCHAR(64)", go: {name: "STARTED_AT", sq_type: "StringField"}},
					{name: "finished_at", type: "VARCHAR(64)", go: {name: "FINISHED_AT", sq_type: "StringField"}},
				]
				indexes: [
					{name: "tasks_task_stream_idx", columns: ["task_stream"]},
					{name: "tasks_subject_type_idx", columns: ["subject_type"]},
					{name: "tasks_subject_id_idx", columns: ["subject_id"]},
					{name: "tasks_owner_peer_id_idx", columns: ["owner_peer_id"]},
					{name: "tasks_status_idx", columns: ["status"]},
				]
			},
			{
				name: "task_events"
				go: name: "TASK_EVENT"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "task_id", type: "BINARY(16)", not_null: true, go: {name: "TASK_ID", sq_type: "BinaryField", ddl: "notnull index"}},
					{name: "status", type: "VARCHAR(64)", not_null: true, go: {name: "STATUS", sq_type: "StringField", ddl: "notnull index"}},
					{name: "message", type: "TEXT", not_null: true, go: {name: "MESSAGE", sq_type: "StringField", ddl: "notnull"}},
					{name: "progress", type: "INT", not_null: true, go: {name: "PROGRESS", sq_type: "NumberField", ddl: "notnull"}},
					{name: "details", type: "JSON", go: {name: "DETAILS", sq_type: "JSONField"}},
					{name: "created_at", type: "VARCHAR(64)", not_null: true, go: {name: "CREATED_AT", sq_type: "StringField", ddl: "notnull"}},
				]
				indexes: [
					{name: "task_events_task_id_idx", columns: ["task_id"]},
					{name: "task_events_status_idx", columns: ["status"]},
				]
			},
			{
				name: "task_operation_facts"
				go: name: "TASK_OPERATION_FACT"
				columns: [
					{name: "id", type: "CHAR(64)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "task_id", type: "BINARY(16)", not_null: true, go: {name: "TASK_ID", sq_type: "BinaryField", ddl: "notnull index"}},
					{name: "fact_kind", type: "VARCHAR(128)", not_null: true, go: {name: "FACT_KIND", sq_type: "StringField", ddl: "notnull index"}},
					{name: "operation_key", type: "VARCHAR(255)", not_null: true, go: {name: "OPERATION_KEY", sq_type: "StringField", ddl: "notnull"}},
					{name: "intent_digest", type: "CHAR(64)", not_null: true, go: {name: "INTENT_DIGEST", sq_type: "StringField", ddl: "notnull"}},
					{name: "author_peer_id", type: "VARCHAR(255)", not_null: true, go: {name: "AUTHOR_PEER_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "subject_type", type: "VARCHAR(128)", not_null: true, go: {name: "SUBJECT_TYPE", sq_type: "StringField", ddl: "notnull index"}},
					{name: "subject_id", type: "VARCHAR(255)", not_null: true, go: {name: "SUBJECT_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "payload", type: "JSON", not_null: true, go: {name: "PAYLOAD", sq_type: "JSONField", ddl: "notnull"}},
				]
				indexes: [
					{name: "task_operation_facts_task_id_idx", columns: ["task_id"]},
					{name: "task_operation_facts_kind_idx", columns: ["fact_kind"]},
					{name: "task_operation_facts_author_idx", columns: ["author_peer_id"]},
					{name: "task_operation_facts_subject_type_idx", columns: ["subject_type"]},
					{name: "task_operation_facts_subject_id_idx", columns: ["subject_id"]},
				]
			},
			{
				name: "exit_routes"
				go: name: "EXIT_ROUTE"
				columns: [
					{name: "id", type: "BINARY(16)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "BinaryField", ddl: "notnull primarykey"}},
					{name: "device_id", type: "BINARY(16)", not_null: true, go: {name: "DEVICE_ID", sq_type: "BinaryField", ddl: "notnull index"}},
					{name: "instance_id", type: "BINARY(16)", not_null: true, go: {name: "INSTANCE_ID", sq_type: "BinaryField", ddl: "notnull index"}},
					{name: "desired_status", type: "VARCHAR(255)", not_null: true, go: {name: "DESIRED_STATUS", sq_type: "StringField", ddl: "notnull"}},
					{name: "dns_server", type: "VARCHAR(255)", not_null: true, go: {name: "DNS_SERVER", sq_type: "StringField", ddl: "notnull"}},
					{name: "cidrs", type: "TEXT", not_null: true, go: {name: "CIDRS", sq_type: "StringField", ddl: "notnull"}},
				]
				indexes: [
					{name: "exit_routes_device_id_idx", columns: ["device_id"]},
					{name: "exit_routes_instance_id_idx", columns: ["instance_id"]},
				]
			},
		]
	}
	go: package: "protosv01"
}

migration: contract.migration
schema:    contract.schema
go:        contract.go
