package protos

#Machine: {
	id:              string
	name:            string
	kind:            string
	desired_status?: string
}

#CloudMachineMetadata: {
	id:                    string
	cloud_id:              string
	provider_resource_id?: string
	public_ip:             string
	location:              string
	architecture:          string
	public_key:            string
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

#User: {
	username:    string
	name?:       string
	is_disabled: bool
}

#UserDeviceMetadata: {
	id:         string
	public_key: string
	user_id:    string
	name:       string
}

#Peer: {
	id: string
}

contract: {
	surface: "sql"
	migration: {
		id:                  "protos-db-v0.0"
		lineage_id:          "protos.db"
		from_version:        ""
		to_version:          "0.0"
		compatibility:       "full"
		backward_compatible: true
		forward_compatible:  true
		required_write_columns: []
		forbidden_write_columns: []
	}
	schema: {
		tables: [
			{
				name: "machines"
				go: name: "MACHINE"
				columns: [
					{name: "id", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "kind", type: "VARCHAR(255)", not_null: true, go: {name: "KIND", sq_type: "StringField", ddl: "notnull"}},
					{name: "desired_status", type: "VARCHAR(255)", go: {name: "DESIRED_STATUS", sq_type: "StringField"}},
				]
				indexes: [{name: "machines_name_idx", columns: ["name"]}]
			},
			{
				name: "cloud_machines_metadata"
				go: name: "CLOUD_MACHINE_METADATA"
				columns: [
					{name: "id", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "cloud_id", type: "VARCHAR(255)", not_null: true, go: {name: "CLOUD_ID", sq_type: "StringField", ddl: "notnull"}},
					{name: "provider_resource_id", type: "VARCHAR(255)", go: {name: "PROVIDER_RESOURCE_ID", sq_type: "StringField"}},
					{name: "public_ip", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_IP", sq_type: "StringField", ddl: "notnull"}},
					{name: "location", type: "VARCHAR(255)", not_null: true, go: {name: "LOCATION", sq_type: "StringField", ddl: "notnull"}},
					{name: "architecture", type: "VARCHAR(255)", not_null: true, go: {name: "ARCHITECTURE", sq_type: "StringField", ddl: "notnull"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
				]
				indexes: [{name: "cloud_machines_metadata_public_key_idx", columns: ["public_key"]}]
			},
			{
				name: "cloud_providers"
				go: name: "CLOUD_PROVIDER"
				columns: [
					{name: "id", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "type", type: "VARCHAR(255)", not_null: true, go: {name: "TYPE", sq_type: "StringField", ddl: "notnull"}},
					{name: "auth", type: "JSON", not_null: true, go: {name: "AUTH", sq_type: "JSONField", ddl: "notnull"}},
				]
				indexes: [{name: "cloud_providers_name_idx", columns: ["name"]}]
			},
			{
				name: "apps"
				go: name: "APP"
				columns: [
					{name: "id", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull index"}},
					{name: "installer_ref", type: "VARCHAR(255)", not_null: true, go: {name: "INSTALLER_REF", sq_type: "StringField", ddl: "notnull"}},
					{name: "instance_id", type: "VARCHAR(255)", not_null: true, go: {name: "INSTANCE_ID", sq_type: "StringField", ddl: "notnull"}},
					{name: "desired_status", type: "VARCHAR(255)", not_null: true, go: {name: "DESIRED_STATUS", sq_type: "StringField", ddl: "notnull"}},
					{name: "persistence", type: "TINYINT(1)", not_null: true, go: {name: "PERSISTENCE", sq_type: "BooleanField", ddl: "notnull"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
				]
				indexes: [
					{name: "apps_name_idx", columns: ["name"]},
					{name: "apps_public_key_idx", columns: ["public_key"]},
				]
			},
			{
				name: "users"
				go: name: "USER"
				columns: [
					{name: "username", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "USERNAME", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "name", type: "VARCHAR(255)", go: {name: "NAME", sq_type: "StringField"}},
					{name: "is_disabled", type: "TINYINT(1)", not_null: true, go: {name: "IS_DISABLED", sq_type: "BooleanField", ddl: "notnull"}},
				]
				indexes: []
			},
			{
				name: "user_devices_metadata"
				go: name: "USER_DEVICE_METADATA"
				columns: [
					{name: "id", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
					{name: "public_key", type: "VARCHAR(255)", not_null: true, go: {name: "PUBLIC_KEY", sq_type: "StringField", ddl: "notnull index"}},
					{name: "user_id", type: "VARCHAR(255)", not_null: true, go: {name: "USER_ID", sq_type: "StringField", ddl: "notnull index"}},
					{name: "name", type: "VARCHAR(255)", not_null: true, go: {name: "NAME", sq_type: "StringField", ddl: "notnull"}},
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
					{name: "id", type: "VARCHAR(255)", primary_key: true, not_null: true, go: {name: "ID", sq_type: "StringField", ddl: "notnull primarykey"}},
				]
				indexes: []
			},
		]
	}
	go: package: "protosv00"
}

lineage: {
	name: "protos.db"
	schemas: [{
		version: [0, 0]
		schema: {
			machines?:                 [...#Machine]
			cloud_machines_metadata?: [...#CloudMachineMetadata]
			cloud_providers?:         [...#CloudProvider]
			apps?:                    [...#App]
			users?:                   [...#User]
			user_devices_metadata?:   [...#UserDeviceMetadata]
			peers?:                   [...#Peer]
		}
	}]
	lenses: []
}

migration: contract.migration
schema:    contract.schema
go:        contract.go
