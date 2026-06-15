package apicv1

#InitRequest: {
	username?:     string
	name?:         string
	organisation?: string
}
#InitResponse: {}

#UserDevice: {
	id?:                   string
	name?:                 string
	public_key?:           string
	public_key_wireguard?: string
}
#GetUserDevicesRequest: {}
#GetUserDevicesResponse: devices?: [...#UserDevice]
#GetUserInfoRequest: {}
#GetUserInfoResponse: {
	username?:      string
	name?:          string
	is_admin?:      bool
	organisation_id?:   string
	organisation_name?: string
}
#Organisation: {
	id?:         string
	name?:       string
	created_at?: string
}
#ListOrganisationsRequest: {}
#ListOrganisationsResponse: organisations?: [...#Organisation]
#StartDeviceInviteRequest: {
	organisation_id?: string
	channel?:         string
	join_mode?:       string
	username?:        string
}
#StartDeviceInviteResponse: {
	invite_id?:        string
	expires_at_unix?:  int64
	advertise_name?:   string
	advertise_service?: string
	channel?:          string
	verification_code?: string
	join_mode?:         string
}
#NearbyOrganisation: {
	organisation_id?:   string
	organisation_name?: string
	device_name?:       string
	peer_id?:           string
	invite_id?:         string
	channel?:           string
	join_mode?:         string
}
#ListNearbyOrganisationsRequest: channel?: string
#ListNearbyOrganisationsResponse: organisations?: [...#NearbyOrganisation]
#JoinOrganisationRequest: {
	organisation_id?:   string
	peer_id?:           string
	invite_id?:         string
	username?:          string
	name?:              string
	channel?:           string
	verification_code?: string
	join_mode?:         string
}
#JoinOrganisationResponse: {}
#GetLocalSSHKeyRequest: {}
#GetLocalSSHKeyResponse: {
	public?:  string
	private?: string
}

#App: {
	id?:            string
	name?:          string
	version?:       string
	status?:        string
	instance_name?: string
	ip?:            string
	installer?:     string
	persistence?:   bool
}
#GetAppsRequest: {}
#GetAppsResponse: apps?: [...#App]
#CreateAppRequest: {
	name?:         string
	installer_id?: string
	instance_id?:  string
	persistence?:  bool
}
#CreateAppResponse: id?: string
#StartAppRequest: name?: string
#StartAppResponse: {}
#StopAppRequest: name?: string
#StopAppResponse: {}
#RemoveAppRequest: name?: string
#RemoveAppResponse: {}
#GetAppLogsRequest: name?: string
#GetAppLogsResponse: logs?: bytes

#Installer: {
	id?:                 string
	name?:               string
	version?:            string
	description?:        string
	requires_resources?: [...string]
	provides_resources?: [...string]
	capabilities?:       [...string]
}
#GetInstallersRequest: {}
#GetInstallersResponse: installers?: [...#Installer]
#GetInstallerRequest: id?: string
#GetInstallerResponse: installer?: #Installer

#CloudMachineSpec: {
	cores?:                  int
	memory?:                 int
	default_storage?:        int
	bandwidth?:              int
	included_data_transfer?: int
	baremetal?:              bool
	price_monthly?:          number
}
#CloudType: {
	name?:                  string
	authentication_fields?: [...string]
}
#CloudProvider: {
	name?:               string
	type?:               #CloudType
	supported_locations?: [...string]
	supported_machines?:  [string]: #CloudMachineSpec
}
#ProvisionerMachineSpec: {
	cores?:                  int
	memory?:                 int
	default_storage?:        int
	bandwidth?:              int
	included_data_transfer?: int
	baremetal?:              bool
	price_monthly?:          number
}
#ProvisionerType: {
	name?:                  string
	authentication_fields?: [...string]
}
#Provisioner: {
	name?:               string
	type?:               #ProvisionerType
	supported_locations?: [...string]
	supported_machines?:  [string]: #ProvisionerMachineSpec
}
#GetSupportedCloudProvidersRequest: {}
#GetSupportedCloudProvidersResponse: cloud_types?: [...#CloudType]
#GetCloudProvidersRequest: {}
#GetCloudProvidersResponse: cloud_providers?: [...#CloudProvider]
#GetCloudProviderRequest: name?: string
#GetCloudProviderResponse: cloud_provider?: #CloudProvider
#AddCloudProviderRequest: {
	name?:        string
	type?:        string
	credentials?: [string]: string
}
#AddCloudProviderResponse: {}
#RemoveCloudProviderRequest: name?: string
#RemoveCloudProviderResponse: {}
#GetSupportedProvisionersRequest: {}
#GetSupportedProvisionersResponse: provisioner_types?: [...#ProvisionerType]
#GetProvisionersRequest: {}
#GetProvisionersResponse: provisioners?: [...#Provisioner]
#GetProvisionerRequest: name?: string
#GetProvisionerResponse: provisioner?: #Provisioner
#AddProvisionerRequest: {
	name?:        string
	type?:        string
	credentials?: [string]: string
}
#AddProvisionerResponse: {}
#RemoveProvisionerRequest: name?: string
#RemoveProvisionerResponse: {}

#CloudInstance: {
	name?:                 string
	public_ip?:            string
	internal_ip?:          string
	cloud_name?:           string
	cloud_type?:           string
	vm_id?:                string
	location?:             string
	public_key?:           string
	public_key_wireguard?: string
	protos_version?:       string
	status?:               string
	architecture?:         string
	peers?:                [string]: string
}
#GetInstancesRequest: {}
#GetInstancesResponse: instances?: [...#CloudInstance]
#GetInstanceRequest: name?: string
#GetInstanceResponse: instance?: #CloudInstance
#InstanceDeployFieldOption: {
	value?:       string
	label?:       string
	description?: string
}
#InstanceDeployField: {
	name?:     string
	label?:    string
	kind?:     string
	required?: bool
	visible?:  bool
	value?:    string
	helper?:   string
	options?:  [...#InstanceDeployFieldOption]
}
#GetInstanceDeployOptionsRequest: {
	provisioner?: string
	location?:    string
}
#GetInstanceDeployOptionsResponse: fields?: [...#InstanceDeployField]
#DeployInstanceRequest: {
	name?:            string
	cloud_name?:      string
	cloud_location?:  string
	machine_type?:    string
	protos_version?:  string
	dev_img?:         string
}
#DeployInstanceResponse: instance?: #CloudInstance
#RemoveInstanceRequest: {
	name?:       string
	local_only?: bool
}
#RemoveInstanceResponse: {}
#StartInstanceRequest: name?: string
#StartInstanceResponse: {}
#StopInstanceRequest: name?: string
#StopInstanceResponse: {}
#GetInstanceKeyRequest: name?: string
#GetInstanceKeyResponse: key?: string
#GetInstanceLogsRequest: name?: string
#GetInstanceLogsResponse: logs?: string
#InitInstanceRequest: {
	name?: string
	ip?:   string
}
#InitInstanceResponse: {}
#UpdateInstanceRequest: {
	id?: string
	ip?: string
}
#UpdateInstanceResponse: {}

#GetNetworkStateRequest: instance?: string
#GetNetworkStateResponse: state?: #NetworkState
#SetNetworkEnabledRequest: enabled?: bool
#SetNetworkEnabledResponse: status?: #NetworkRuntimeStatus
#NetworkRuntimeStatus: {
	supported?:       bool
	desired_enabled?: bool
	enabled?:         bool
	state?:           string
	message?:         string
	network_state?:   #NetworkState
}
#NetworkState: {
	module?:          string
	up?:              bool
	interface_name?:  string
	interfaces?:      [...#NetworkInterface]
	addresses?:       [...#NetworkAddress]
	routes?:          [...#NetworkRoute]
	wireguard_peers?: [...#WireGuardPeer]
	firewall_tables?: [...#FirewallTable]
	dns?:             [...#DNSState]
	messages?:        [...string]
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
	public_key?:       string
	endpoint?:         string
	allowed_ips?:      [...string]
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
	rules?:    [...#FirewallRule]
}
#FirewallRule: {
	expressions?: [...string]
	packets?:     uint
	bytes?:       uint
}
#DNSState: {
	scope?:   string
	domain?:  string
	servers?: [...string]
	port?:    int
	active?:  bool
	source?:  string
}

#ExitRoute: {
	id?:            string
	device_id?:     string
	instance_id?:   string
	instance_name?: string
	public_ip?:     string
	location?:      string
	status?:        string
	dns_server?:    string
	cidrs?:         [...string]
}
#GetExitRoutesRequest: instance?: string
#GetExitRoutesResponse: routes?: [...#ExitRoute]
#GetMobileTunnelConfigRequest: {
	instance?:   string
	device_id?:  string
	dns_server?: string
	cidrs?:      [...string]
}
#MobileTunnelConfig: {
	config_id?:                    string
	generated_at_unix?:            int64
	instance_id?:                  string
	instance_name?:                string
	peer_public_key?:              string
	peer_endpoint?:                string
	interface_addresses?:          [...string]
	dns_servers?:                  [...string]
	included_routes?:              [...string]
	excluded_routes?:              [...string]
	mtu?:                          int
	allowed_ips?:                  [...string]
	persistent_keepalive_seconds?: int
	keychain_access_group?:        string
	keychain_account?:             string
	wireguard_private_key?:        string
}
#GetMobileTunnelConfigResponse: config?: #MobileTunnelConfig
#GetRuntimeStateRequest: instance?: string
#GetRuntimeStateResponse: state?: #RuntimeState
#WatchChangesRequest: {
	include_snapshot?:       bool
	heartbeat_interval_ms?: uint32
}
#WatchChangesResponse: {
	sequence?:        uint64
	table_names?:     [...string]
	runtime_changed?: bool
	reason?:          string
}
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
	tasks?:     [...#Task]
	truncated?: bool
}
#GetTaskRequest: {
	id?:             string
	include_events?: bool
}
#GetTaskResponse: {
	task?:   #Task
	events?: [...#TaskEvent]
}
#SetExitRouteRequest: {
	instance?:   string
	device_id?:  string
	dns_server?: string
	cidrs?:      [...string]
}
#SetExitRouteResponse: route?: #ExitRoute
#ClearExitRouteRequest: {
	device_id?: string
}
#ClearExitRouteResponse: {}

#RuntimeState: {
	peer_id?:                         string
	manifest_digest?:                 string
	checkpoint_root_hash?:            string
	tentative_root_hash?:             string
	protocol_checkpoint_root_hash?:   string
	durable_main_root_hash?:          string
	state_providers?:                 [...string]
	connected_peers?:                 [...string]
	fatal_state?:                     string
	runtime_refresh_pending?:         bool
	runtime_refresh_last_error?:      string
	runtime_checkpoint_pending?:      bool
	runtime_checkpoint_last_error?:   string
	runtime_materialization_policy?:  string
	peer_statuses?:                  [...#RuntimePeerStatus]
	compatibility?:                  [...#RuntimeCompatibility]
	content_sync_trace?:              [...string]
	protocol_checkpoint_digest?:      string
}
#RuntimePeerStatus: {
	peer_id?:          string
	connected?:        bool
	dialable?:         bool
	state_provider?:   bool
	compatible?:       bool
	incompatible?:     bool
	ignored?:          bool
	relay_only?:       bool
	addresses?:        [...string]
	last_dial_errors?: [string]: string
	reason?:           string
	replication_priority?:     int
	replication_device_class?: string
}
#RuntimeCompatibility: {
	peer_id?:       string
	local_digest?:  string
	remote_digest?: string
	compatible?:    bool
	blocking?:      bool
	reason?:        string
}

#CloudImage: {
	provider?:     string
	url?:          string
	digest?:       string
	release_date?: int
}
#CloudSpecificImage: {
	id?:       string
	name?:     string
	location?: string
}
#Release: {
	cloud_images?: [string]: #CloudImage
	version?:      string
	description?:  string
	release_date?: int
}
#GetProtosdReleasesRequest: {}
#GetProtosdReleasesResponse: releases?: [...#Release]
#GetCloudImagesRequest: name?: string
#GetCloudImagesResponse: cloud_images?: [string]: #CloudSpecificImage
#GetProvisionerImagesRequest: name?: string
#GetProvisionerImagesResponse: images?: [string]: #CloudSpecificImage
#UploadCloudImageRequest: {
	image_path?:     string
	image_name?:     string
	cloud_name?:     string
	cloud_location?: string
	timeout?:        int
}
#UploadCloudImageResponse: {}
#UploadProvisionerImageRequest: {
	image_path?: string
	image_name?: string
	provisioner_name?: string
	location?:   string
	timeout?:    int
}
#UploadProvisionerImageResponse: {}
#RemoveCloudImageRequest: {
	image_name?:     string
	cloud_name?:     string
	cloud_location?: string
}
#RemoveCloudImageResponse: {}
#RemoveProvisionerImageRequest: {
	image_name?:      string
	provisioner_name?: string
	location?:        string
}
#RemoveProvisionerImageResponse: {}
#ImageContentDescriptor: {
	media_type?:  string
	digest?:      string
	size_bytes?:  uint
	platform?:    string
	annotations?: [string]: string
}
#GetInstanceImageRequest: {
	instance?:  string
	image_ref?: string
}
#GetInstanceImageResponse: {
	found?:         bool
	image_ref?:     string
	target_digest?: string
	platform?:      string
	labels?:        [string]: string
	has_content?:   bool
	target?:        #ImageContentDescriptor
	descriptors?:   [...#ImageContentDescriptor]
}
#UploadInstanceImageArchiveChunkRequest: {
	instance?:  string
	upload_id?: string
	image_ref?: string
	offset?:    uint
	data?:      bytes
	eof?:       bool
}
#UploadInstanceImageArchiveChunkResponse: {
	received_bytes?: uint
	loaded?:         bool
	image_ref?:      string
	target_digest?:  string
	platform?:       string
}

#Commit: {
	hash?:      string
	committer?: string
	message?:   string
	states?:    [...string]
	date_unix?: int
	parent_hashes?: [...string]
	refs?:          [...string]
}
#CommitGraphRelation: {
	parent_hash?: string
	parent_row?:  int
	from_lane?:   int
	to_lane?:     int
	visible?:     bool
}
#CommitGraphItem: {
	commit?:       #Commit
	row?:          int
	lane?:         int
	active_lanes?: [...int]
	relations?:    [...#CommitGraphRelation]
}
#CommitGraph: {
	items?:      [...#CommitGraphItem]
	lane_count?: int
}
#GetLocalCommitsRequest: {}
#GetLocalCommitsResponse: {
	commits?: [...#Commit]
	graph?:   #CommitGraph
}
#GetRemoteCommitsRequest: remote?: string
#GetRemoteCommitsResponse: {
	commits?: [...#Commit]
	graph?:   #CommitGraph
}
#SqlCell: {
	value?:   string
	is_null?: bool
}
#SqlRow: cells?: [...#SqlCell]
#ExecuteSqlRequest: {
	sql?:      string
	max_rows?: int
}
#ExecuteSqlResponse: {
	columns?:       [...string]
	rows?:          [...#SqlRow]
	rows_affected?: int
	truncated?:     bool
	message?:       string
}

#CoreEndpoint: {
	kind?:    string
	address?: string
	active?:  bool
	message?: string
}
#HostAgentConnectionStatus: {
	connected?: bool
	socket?:    string
	message?:   string
}
#SystemStatus: {
	core_status?:  string
	work_dir?:     string
	capabilities?: string
	p2p_port?:     int
	endpoints?:    [...#CoreEndpoint]
	host_agent?:   #HostAgentConnectionStatus
	network_enabled?: bool
	host_agent_supported?: bool
	network?: #NetworkRuntimeStatus
}
#GetSystemStatusRequest: {}
#GetSystemStatusResponse: status?: #SystemStatus
#StartHostAgentRequest: {}
#StartHostAgentResponse: status?: #HostAgentConnectionStatus
#StopHostAgentRequest: {}
#StopHostAgentResponse: status?: #HostAgentConnectionStatus

contract: {
	surface: "client-api-grpc"
	migration: {
		id:                  "protos-client-api-v0.0"
		lineage_id:          "protos.client_api"
		from_version:        ""
		to_version:          "0.0"
		compatibility:       "full"
		backward_compatible: true
		forward_compatible:  true
	}
	proto: {
		syntax:     "proto3"
		package:    "apic"
		go_package: "github.com/protosio/protos/internal/apic;proto"
		services: [{
			name: "ProtosClientApi"
			rpcs: [
				{name: "Init", request: "InitRequest", response: "InitResponse"},
				{name: "GetUserDevices", request: "GetUserDevicesRequest", response: "GetUserDevicesResponse"},
				{name: "GetUserInfo", request: "GetUserInfoRequest", response: "GetUserInfoResponse"},
				{name: "ListOrganisations", request: "ListOrganisationsRequest", response: "ListOrganisationsResponse"},
				{name: "StartDeviceInvite", request: "StartDeviceInviteRequest", response: "StartDeviceInviteResponse"},
				{name: "ListNearbyOrganisations", request: "ListNearbyOrganisationsRequest", response: "ListNearbyOrganisationsResponse"},
				{name: "JoinOrganisation", request: "JoinOrganisationRequest", response: "JoinOrganisationResponse"},
				{name: "GetLocalSSHKey", request: "GetLocalSSHKeyRequest", response: "GetLocalSSHKeyResponse"},
				{name: "GetApps", request: "GetAppsRequest", response: "GetAppsResponse"},
				{name: "CreateApp", request: "CreateAppRequest", response: "CreateAppResponse"},
				{name: "StartApp", request: "StartAppRequest", response: "StartAppResponse"},
				{name: "StopApp", request: "StopAppRequest", response: "StopAppResponse"},
				{name: "RemoveApp", request: "RemoveAppRequest", response: "RemoveAppResponse"},
				{name: "GetAppLogs", request: "GetAppLogsRequest", response: "GetAppLogsResponse"},
				{name: "GetSupportedCloudProviders", request: "GetSupportedCloudProvidersRequest", response: "GetSupportedCloudProvidersResponse"},
				{name: "GetCloudProviders", request: "GetCloudProvidersRequest", response: "GetCloudProvidersResponse"},
				{name: "GetCloudProvider", request: "GetCloudProviderRequest", response: "GetCloudProviderResponse"},
				{name: "AddCloudProvider", request: "AddCloudProviderRequest", response: "AddCloudProviderResponse"},
				{name: "RemoveCloudProvider", request: "RemoveCloudProviderRequest", response: "RemoveCloudProviderResponse"},
				{name: "GetSupportedProvisioners", request: "GetSupportedProvisionersRequest", response: "GetSupportedProvisionersResponse"},
				{name: "GetProvisioners", request: "GetProvisionersRequest", response: "GetProvisionersResponse"},
				{name: "GetProvisioner", request: "GetProvisionerRequest", response: "GetProvisionerResponse"},
				{name: "AddProvisioner", request: "AddProvisionerRequest", response: "AddProvisionerResponse"},
				{name: "RemoveProvisioner", request: "RemoveProvisionerRequest", response: "RemoveProvisionerResponse"},
				{name: "GetInstances", request: "GetInstancesRequest", response: "GetInstancesResponse"},
				{name: "GetInstance", request: "GetInstanceRequest", response: "GetInstanceResponse"},
				{name: "GetInstanceDeployOptions", request: "GetInstanceDeployOptionsRequest", response: "GetInstanceDeployOptionsResponse"},
				{name: "DeployInstance", request: "DeployInstanceRequest", response: "DeployInstanceResponse"},
				{name: "RemoveInstance", request: "RemoveInstanceRequest", response: "RemoveInstanceResponse"},
				{name: "StartInstance", request: "StartInstanceRequest", response: "StartInstanceResponse"},
				{name: "StopInstance", request: "StopInstanceRequest", response: "StopInstanceResponse"},
				{name: "GetInstanceKey", request: "GetInstanceKeyRequest", response: "GetInstanceKeyResponse"},
				{name: "GetInstanceLogs", request: "GetInstanceLogsRequest", response: "GetInstanceLogsResponse"},
				{name: "InitInstance", request: "InitInstanceRequest", response: "InitInstanceResponse"},
				{name: "UpdateInstance", request: "UpdateInstanceRequest", response: "UpdateInstanceResponse"},
				{name: "GetNetworkState", request: "GetNetworkStateRequest", response: "GetNetworkStateResponse"},
				{name: "SetNetworkEnabled", request: "SetNetworkEnabledRequest", response: "SetNetworkEnabledResponse"},
				{name: "GetExitRoutes", request: "GetExitRoutesRequest", response: "GetExitRoutesResponse"},
				{name: "GetMobileTunnelConfig", request: "GetMobileTunnelConfigRequest", response: "GetMobileTunnelConfigResponse"},
				{name: "GetRuntimeState", request: "GetRuntimeStateRequest", response: "GetRuntimeStateResponse"},
				{name: "WatchChanges", request: "WatchChangesRequest", response: "WatchChangesResponse", response_stream: true},
				{name: "GetTasks", request: "GetTasksRequest", response: "GetTasksResponse"},
				{name: "GetTask", request: "GetTaskRequest", response: "GetTaskResponse"},
				{name: "SetExitRoute", request: "SetExitRouteRequest", response: "SetExitRouteResponse"},
				{name: "ClearExitRoute", request: "ClearExitRouteRequest", response: "ClearExitRouteResponse"},
				{name: "GetProtosdReleases", request: "GetProtosdReleasesRequest", response: "GetProtosdReleasesResponse"},
				{name: "GetCloudImages", request: "GetCloudImagesRequest", response: "GetCloudImagesResponse"},
				{name: "UploadCloudImage", request: "UploadCloudImageRequest", response: "UploadCloudImageResponse"},
				{name: "RemoveCloudImage", request: "RemoveCloudImageRequest", response: "RemoveCloudImageResponse"},
				{name: "GetProvisionerImages", request: "GetProvisionerImagesRequest", response: "GetProvisionerImagesResponse"},
				{name: "UploadProvisionerImage", request: "UploadProvisionerImageRequest", response: "UploadProvisionerImageResponse"},
				{name: "RemoveProvisionerImage", request: "RemoveProvisionerImageRequest", response: "RemoveProvisionerImageResponse"},
				{name: "GetInstanceImage", request: "GetInstanceImageRequest", response: "GetInstanceImageResponse"},
				{name: "UploadInstanceImageArchiveChunk", request: "UploadInstanceImageArchiveChunkRequest", response: "UploadInstanceImageArchiveChunkResponse"},
				{name: "GetSystemStatus", request: "GetSystemStatusRequest", response: "GetSystemStatusResponse"},
				{name: "StartHostAgent", request: "StartHostAgentRequest", response: "StartHostAgentResponse"},
				{name: "StopHostAgent", request: "StopHostAgentRequest", response: "StopHostAgentResponse"},
				{name: "GetLocalCommits", request: "GetLocalCommitsRequest", response: "GetLocalCommitsResponse"},
				{name: "GetRemoteCommits", request: "GetRemoteCommitsRequest", response: "GetRemoteCommitsResponse"},
				{name: "ExecuteSql", request: "ExecuteSqlRequest", response: "ExecuteSqlResponse"},
			]
		}]
		declarations: [
			{kind: "message", name: "InitRequest", fields: [
				{type: "string", name: "username", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "organisation", number: 3},
			]},
			{kind: "message", name: "InitResponse", fields: []},
			{kind: "message", name: "UserDevice", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "public_key", number: 3},
				{type: "string", name: "public_key_wireguard", number: 4},
			]},
			{kind: "message", name: "GetUserDevicesRequest", fields: []},
			{kind: "message", name: "GetUserDevicesResponse", fields: [
				{rule: "repeated", type: "UserDevice", name: "devices", number: 1},
			]},
			{kind: "message", name: "GetUserInfoRequest", fields: []},
			{kind: "message", name: "GetUserInfoResponse", fields: [
				{type: "string", name: "username", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "bool", name: "is_admin", number: 3},
				{type: "string", name: "organisation_id", number: 4},
				{type: "string", name: "organisation_name", number: 5},
			]},
			{kind: "message", name: "Organisation", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "created_at", number: 3},
			]},
			{kind: "message", name: "ListOrganisationsRequest", fields: []},
			{kind: "message", name: "ListOrganisationsResponse", fields: [
				{rule: "repeated", type: "Organisation", name: "organisations", number: 1},
			]},
			{kind: "message", name: "StartDeviceInviteRequest", fields: [
				{type: "string", name: "organisation_id", number: 1},
				{type: "string", name: "channel", number: 2},
				{type: "string", name: "join_mode", number: 3},
				{type: "string", name: "username", number: 4},
			]},
			{kind: "message", name: "StartDeviceInviteResponse", fields: [
				{type: "string", name: "invite_id", number: 1},
				{type: "int64", name: "expires_at_unix", number: 2},
				{type: "string", name: "advertise_name", number: 3},
				{type: "string", name: "advertise_service", number: 4},
				{type: "string", name: "channel", number: 5},
				{type: "string", name: "verification_code", number: 6},
				{type: "string", name: "join_mode", number: 7},
			]},
			{kind: "message", name: "NearbyOrganisation", fields: [
				{type: "string", name: "organisation_id", number: 1},
				{type: "string", name: "organisation_name", number: 2},
				{type: "string", name: "device_name", number: 3},
				{type: "string", name: "peer_id", number: 4},
				{type: "string", name: "invite_id", number: 5},
				{type: "string", name: "channel", number: 6},
				{type: "string", name: "join_mode", number: 7},
			]},
			{kind: "message", name: "ListNearbyOrganisationsRequest", fields: [
				{type: "string", name: "channel", number: 1},
			]},
			{kind: "message", name: "ListNearbyOrganisationsResponse", fields: [
				{rule: "repeated", type: "NearbyOrganisation", name: "organisations", number: 1},
			]},
			{kind: "message", name: "JoinOrganisationRequest", fields: [
				{type: "string", name: "organisation_id", number: 1},
				{type: "string", name: "peer_id", number: 2},
				{type: "string", name: "invite_id", number: 3},
				{type: "string", name: "username", number: 4},
				{type: "string", name: "name", number: 5},
				{type: "string", name: "channel", number: 6},
				{type: "string", name: "verification_code", number: 7},
				{type: "string", name: "join_mode", number: 8},
			]},
			{kind: "message", name: "JoinOrganisationResponse", fields: []},
			{kind: "message", name: "GetLocalSSHKeyRequest", fields: []},
			{kind: "message", name: "GetLocalSSHKeyResponse", fields: [
				{type: "string", name: "public", number: 1},
				{type: "string", name: "private", number: 2},
			]},
			{kind: "message", name: "App", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "version", number: 3},
				{type: "string", name: "status", number: 4},
				{type: "string", name: "instance_name", number: 5},
				{type: "string", name: "ip", number: 6},
				{type: "string", name: "installer", number: 7},
				{type: "bool", name: "persistence", number: 8},
			]},
			{kind: "message", name: "GetAppsRequest", fields: []},
			{kind: "message", name: "GetAppsResponse", fields: [
				{rule: "repeated", type: "App", name: "apps", number: 1},
			]},
			{kind: "message", name: "CreateAppRequest", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "installer_id", number: 2},
				{type: "string", name: "instance_id", number: 3},
				{type: "bool", name: "persistence", number: 4},
			]},
			{kind: "message", name: "CreateAppResponse", fields: [
				{type: "string", name: "id", number: 1},
			]},
			{kind: "message", name: "StartAppRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "StartAppResponse", fields: []},
			{kind: "message", name: "StopAppRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "StopAppResponse", fields: []},
			{kind: "message", name: "RemoveAppRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "RemoveAppResponse", fields: []},
			{kind: "message", name: "GetAppLogsRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetAppLogsResponse", fields: [{type: "bytes", name: "logs", number: 1}]},
			{kind: "message", name: "Installer", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "version", number: 3},
				{type: "string", name: "description", number: 4},
				{rule: "repeated", type: "string", name: "requires_resources", number: 5},
				{rule: "repeated", type: "string", name: "provides_resources", number: 6},
				{rule: "repeated", type: "string", name: "capabilities", number: 7},
			]},
			{kind: "message", name: "GetInstallersRequest", fields: []},
			{kind: "message", name: "GetInstallersResponse", fields: [{rule: "repeated", type: "Installer", name: "installers", number: 1}]},
			{kind: "message", name: "GetInstallerRequest", fields: [{type: "string", name: "id", number: 1}]},
			{kind: "message", name: "GetInstallerResponse", fields: [{type: "Installer", name: "installer", number: 1}]},
			{kind: "message", name: "CloudMachineSpec", fields: [
				{type: "int32", name: "cores", number: 1},
				{type: "int32", name: "memory", number: 2},
				{type: "int32", name: "default_storage", number: 3},
				{type: "int32", name: "bandwidth", number: 4},
				{type: "int32", name: "included_data_transfer", number: 5},
				{type: "bool", name: "baremetal", number: 6},
				{type: "float", name: "price_monthly", number: 7},
			]},
			{kind: "message", name: "CloudType", fields: [
				{type: "string", name: "name", number: 1},
				{rule: "repeated", type: "string", name: "authentication_fields", number: 2},
			]},
			{kind: "message", name: "CloudProvider", fields: [
				{type: "string", name: "name", number: 1},
				{type: "CloudType", name: "type", number: 2},
				{rule: "repeated", type: "string", name: "supported_locations", number: 3},
				{type: "map<string, CloudMachineSpec>", name: "supported_machines", number: 4},
			]},
			{kind: "message", name: "GetSupportedCloudProvidersRequest", fields: []},
			{kind: "message", name: "GetSupportedCloudProvidersResponse", fields: [{rule: "repeated", type: "CloudType", name: "cloud_types", number: 1}]},
			{kind: "message", name: "GetCloudProvidersRequest", fields: []},
			{kind: "message", name: "GetCloudProvidersResponse", fields: [{rule: "repeated", type: "CloudProvider", name: "cloud_providers", number: 1}]},
			{kind: "message", name: "GetCloudProviderRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetCloudProviderResponse", fields: [{type: "CloudProvider", name: "cloud_provider", number: 1}]},
			{kind: "message", name: "AddCloudProviderRequest", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "type", number: 2},
				{type: "map<string, string>", name: "credentials", number: 3},
			]},
			{kind: "message", name: "AddCloudProviderResponse", fields: []},
			{kind: "message", name: "RemoveCloudProviderRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "RemoveCloudProviderResponse", fields: []},
			{kind: "message", name: "ProvisionerMachineSpec", fields: [
				{type: "int32", name: "cores", number: 1},
				{type: "int32", name: "memory", number: 2},
				{type: "int32", name: "default_storage", number: 3},
				{type: "int32", name: "bandwidth", number: 4},
				{type: "int32", name: "included_data_transfer", number: 5},
				{type: "bool", name: "baremetal", number: 6},
				{type: "float", name: "price_monthly", number: 7},
			]},
			{kind: "message", name: "ProvisionerType", fields: [
				{type: "string", name: "name", number: 1},
				{rule: "repeated", type: "string", name: "authentication_fields", number: 2},
			]},
			{kind: "message", name: "Provisioner", fields: [
				{type: "string", name: "name", number: 1},
				{type: "ProvisionerType", name: "type", number: 2},
				{rule: "repeated", type: "string", name: "supported_locations", number: 3},
				{type: "map<string, ProvisionerMachineSpec>", name: "supported_machines", number: 4},
			]},
			{kind: "message", name: "GetSupportedProvisionersRequest", fields: []},
			{kind: "message", name: "GetSupportedProvisionersResponse", fields: [{rule: "repeated", type: "ProvisionerType", name: "provisioner_types", number: 1}]},
			{kind: "message", name: "GetProvisionersRequest", fields: []},
			{kind: "message", name: "GetProvisionersResponse", fields: [{rule: "repeated", type: "Provisioner", name: "provisioners", number: 1}]},
			{kind: "message", name: "GetProvisionerRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetProvisionerResponse", fields: [{type: "Provisioner", name: "provisioner", number: 1}]},
			{kind: "message", name: "AddProvisionerRequest", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "type", number: 2},
				{type: "map<string, string>", name: "credentials", number: 3},
			]},
			{kind: "message", name: "AddProvisionerResponse", fields: []},
			{kind: "message", name: "RemoveProvisionerRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "RemoveProvisionerResponse", fields: []},
			{kind: "message", name: "CloudInstance", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "public_ip", number: 2},
				{type: "string", name: "internal_ip", number: 3},
				{type: "string", name: "cloud_name", number: 4},
				{type: "string", name: "cloud_type", number: 5},
				{type: "string", name: "vm_id", number: 6},
				{type: "string", name: "location", number: 7},
				{type: "string", name: "public_key", number: 8},
				{type: "string", name: "public_key_wireguard", number: 9},
				{type: "string", name: "protos_version", number: 10},
				{type: "string", name: "status", number: 11},
				{type: "string", name: "architecture", number: 12},
				{type: "map<string, string>", name: "peers", number: 13},
			]},
			{kind: "message", name: "GetInstancesRequest", fields: []},
			{kind: "message", name: "GetInstancesResponse", fields: [{rule: "repeated", type: "CloudInstance", name: "instances", number: 1}]},
			{kind: "message", name: "GetInstanceRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetInstanceResponse", fields: [{type: "CloudInstance", name: "instance", number: 1}]},
			{kind: "message", name: "InstanceDeployFieldOption", fields: [
				{type: "string", name: "value", number: 1},
				{type: "string", name: "label", number: 2},
				{type: "string", name: "description", number: 3},
			]},
			{kind: "message", name: "InstanceDeployField", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "label", number: 2},
				{type: "string", name: "kind", number: 3},
				{type: "bool", name: "required", number: 4},
				{type: "bool", name: "visible", number: 5},
				{type: "string", name: "value", number: 6},
				{type: "string", name: "helper", number: 7},
				{rule: "repeated", type: "InstanceDeployFieldOption", name: "options", number: 8},
			]},
			{kind: "message", name: "GetInstanceDeployOptionsRequest", fields: [
				{type: "string", name: "provisioner", number: 1},
				{type: "string", name: "location", number: 2},
			]},
			{kind: "message", name: "GetInstanceDeployOptionsResponse", fields: [
				{rule: "repeated", type: "InstanceDeployField", name: "fields", number: 1},
			]},
			{kind: "message", name: "DeployInstanceRequest", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "cloud_name", number: 2},
				{type: "string", name: "cloud_location", number: 3},
				{type: "string", name: "machine_type", number: 4},
				{type: "string", name: "protos_version", number: 5},
				{type: "string", name: "dev_img", number: 6},
			]},
			{kind: "message", name: "DeployInstanceResponse", fields: [{type: "CloudInstance", name: "instance", number: 1}]},
			{kind: "message", name: "RemoveInstanceRequest", fields: [
				{type: "string", name: "name", number: 1},
				{type: "bool", name: "local_only", number: 2},
			]},
			{kind: "message", name: "RemoveInstanceResponse", fields: []},
			{kind: "message", name: "StartInstanceRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "StartInstanceResponse", fields: []},
			{kind: "message", name: "StopInstanceRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "StopInstanceResponse", fields: []},
			{kind: "message", name: "GetInstanceKeyRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetInstanceKeyResponse", fields: [{type: "string", name: "key", number: 1}]},
			{kind: "message", name: "GetInstanceLogsRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetInstanceLogsResponse", fields: [{type: "string", name: "logs", number: 1}]},
			{kind: "message", name: "InitInstanceRequest", fields: [
				{type: "string", name: "name", number: 1},
				{type: "string", name: "ip", number: 2},
			]},
			{kind: "message", name: "InitInstanceResponse", fields: []},
			{kind: "message", name: "UpdateInstanceRequest", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "ip", number: 2},
			]},
			{kind: "message", name: "UpdateInstanceResponse", fields: []},
			{kind: "message", name: "GetNetworkStateRequest", fields: [
				{type: "string", name: "instance", number: 1},
			]},
			{kind: "message", name: "GetNetworkStateResponse", fields: [
				{type: "NetworkState", name: "state", number: 1},
			]},
			{kind: "message", name: "SetNetworkEnabledRequest", fields: [
				{type: "bool", name: "enabled", number: 1},
			]},
			{kind: "message", name: "SetNetworkEnabledResponse", fields: [
				{type: "NetworkRuntimeStatus", name: "status", number: 1},
			]},
			{kind: "message", name: "NetworkRuntimeStatus", fields: [
				{type: "bool", name: "supported", number: 1},
				{type: "bool", name: "desired_enabled", number: 2},
				{type: "bool", name: "enabled", number: 3},
				{type: "string", name: "state", number: 4},
				{type: "string", name: "message", number: 5},
				{type: "NetworkState", name: "network_state", number: 6},
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
				{type: "string", name: "instance_name", number: 4},
				{type: "string", name: "public_ip", number: 5},
				{type: "string", name: "location", number: 6},
				{type: "string", name: "status", number: 7},
				{type: "string", name: "dns_server", number: 8},
				{rule: "repeated", type: "string", name: "cidrs", number: 9},
			]},
			{kind: "message", name: "GetExitRoutesRequest", fields: [
				{type: "string", name: "instance", number: 1},
			]},
			{kind: "message", name: "GetExitRoutesResponse", fields: [
				{rule: "repeated", type: "ExitRoute", name: "routes", number: 1},
			]},
			{kind: "message", name: "GetMobileTunnelConfigRequest", fields: [
				{type: "string", name: "instance", number: 1},
				{type: "string", name: "device_id", number: 2},
				{type: "string", name: "dns_server", number: 3},
				{rule: "repeated", type: "string", name: "cidrs", number: 4},
			]},
			{kind: "message", name: "MobileTunnelConfig", fields: [
				{type: "string", name: "config_id", number: 1},
				{type: "int64", name: "generated_at_unix", number: 2},
				{type: "string", name: "instance_id", number: 3},
				{type: "string", name: "instance_name", number: 4},
				{type: "string", name: "peer_public_key", number: 5},
				{type: "string", name: "peer_endpoint", number: 6},
				{rule: "repeated", type: "string", name: "interface_addresses", number: 7},
				{rule: "repeated", type: "string", name: "dns_servers", number: 8},
				{rule: "repeated", type: "string", name: "included_routes", number: 9},
				{rule: "repeated", type: "string", name: "excluded_routes", number: 10},
				{type: "int32", name: "mtu", number: 11},
				{rule: "repeated", type: "string", name: "allowed_ips", number: 12},
				{type: "int32", name: "persistent_keepalive_seconds", number: 13},
				{type: "string", name: "keychain_access_group", number: 14},
				{type: "string", name: "keychain_account", number: 15},
				{type: "string", name: "wireguard_private_key", number: 16},
			]},
			{kind: "message", name: "GetMobileTunnelConfigResponse", fields: [
				{type: "MobileTunnelConfig", name: "config", number: 1},
			]},
			{kind: "message", name: "GetRuntimeStateRequest", fields: [
				{type: "string", name: "instance", number: 1},
			]},
			{kind: "message", name: "GetRuntimeStateResponse", fields: [
				{type: "RuntimeState", name: "state", number: 1},
			]},
			{kind: "message", name: "WatchChangesRequest", fields: [
				{type: "bool", name: "include_snapshot", number: 1},
				{type: "uint32", name: "heartbeat_interval_ms", number: 2},
			]},
			{kind: "message", name: "WatchChangesResponse", fields: [
				{type: "uint64", name: "sequence", number: 1},
				{rule: "repeated", type: "string", name: "table_names", number: 2},
				{type: "bool", name: "runtime_changed", number: 3},
				{type: "string", name: "reason", number: 4},
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
			{kind: "message", name: "SetExitRouteRequest", fields: [
				{type: "string", name: "instance", number: 1},
				{type: "string", name: "device_id", number: 2},
				{type: "string", name: "dns_server", number: 3},
				{rule: "repeated", type: "string", name: "cidrs", number: 4},
			]},
			{kind: "message", name: "SetExitRouteResponse", fields: [
				{type: "ExitRoute", name: "route", number: 1},
			]},
			{kind: "message", name: "ClearExitRouteRequest", fields: [
				{type: "string", name: "device_id", number: 1},
			]},
			{kind: "message", name: "ClearExitRouteResponse", fields: []},
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
				{type: "int32", name: "replication_priority", number: 14},
				{type: "string", name: "replication_device_class", number: 15},
			]},
			{kind: "message", name: "RuntimeCompatibility", fields: [
				{type: "string", name: "peer_id", number: 1},
				{type: "string", name: "local_digest", number: 2},
				{type: "string", name: "remote_digest", number: 3},
				{type: "bool", name: "compatible", number: 4},
				{type: "bool", name: "blocking", number: 5},
				{type: "string", name: "reason", number: 6},
			]},
			{kind: "message", name: "CloudImage", fields: [
				{type: "string", name: "provider", number: 1},
				{type: "string", name: "url", number: 2},
				{type: "string", name: "digest", number: 3},
				{type: "int64", name: "release_date", number: 4},
			]},
			{kind: "message", name: "CloudSpecificImage", fields: [
				{type: "string", name: "id", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "location", number: 3},
			]},
			{kind: "message", name: "Release", fields: [
				{type: "map<string, CloudImage>", name: "cloud_images", number: 1},
				{type: "string", name: "version", number: 2},
				{type: "string", name: "description", number: 3},
				{type: "int64", name: "release_date", number: 4},
			]},
			{kind: "message", name: "GetProtosdReleasesRequest", fields: []},
			{kind: "message", name: "GetProtosdReleasesResponse", fields: [{rule: "repeated", type: "Release", name: "releases", number: 1}]},
			{kind: "message", name: "GetCloudImagesRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetCloudImagesResponse", fields: [{type: "map<string, CloudSpecificImage>", name: "cloud_images", number: 1}]},
			{kind: "message", name: "GetProvisionerImagesRequest", fields: [{type: "string", name: "name", number: 1}]},
			{kind: "message", name: "GetProvisionerImagesResponse", fields: [{type: "map<string, CloudSpecificImage>", name: "images", number: 1}]},
			{kind: "message", name: "UploadCloudImageRequest", fields: [
				{type: "string", name: "image_path", number: 1},
				{type: "string", name: "image_name", number: 2},
				{type: "string", name: "cloud_name", number: 3},
				{type: "string", name: "cloud_location", number: 4},
				{type: "int32", name: "timeout", number: 5},
			]},
			{kind: "message", name: "UploadCloudImageResponse", fields: []},
			{kind: "message", name: "UploadProvisionerImageRequest", fields: [
				{type: "string", name: "image_path", number: 1},
				{type: "string", name: "image_name", number: 2},
				{type: "string", name: "provisioner_name", number: 3},
				{type: "string", name: "location", number: 4},
				{type: "int32", name: "timeout", number: 5},
			]},
			{kind: "message", name: "UploadProvisionerImageResponse", fields: []},
			{kind: "message", name: "RemoveCloudImageRequest", fields: [
				{type: "string", name: "image_name", number: 2},
				{type: "string", name: "cloud_name", number: 3},
				{type: "string", name: "cloud_location", number: 4},
			]},
			{kind: "message", name: "RemoveCloudImageResponse", fields: []},
			{kind: "message", name: "RemoveProvisionerImageRequest", fields: [
				{type: "string", name: "image_name", number: 1},
				{type: "string", name: "provisioner_name", number: 2},
				{type: "string", name: "location", number: 3},
			]},
			{kind: "message", name: "RemoveProvisionerImageResponse", fields: []},
			{kind: "message", name: "ImageContentDescriptor", fields: [
				{type: "string", name: "media_type", number: 1},
				{type: "string", name: "digest", number: 2},
				{type: "uint64", name: "size_bytes", number: 3},
				{type: "string", name: "platform", number: 4},
				{type: "map<string, string>", name: "annotations", number: 5},
			]},
			{kind: "message", name: "GetInstanceImageRequest", fields: [
				{type: "string", name: "instance", number: 1},
				{type: "string", name: "image_ref", number: 2},
			]},
			{kind: "message", name: "GetInstanceImageResponse", fields: [
				{type: "bool", name: "found", number: 1},
				{type: "string", name: "image_ref", number: 2},
				{type: "string", name: "target_digest", number: 3},
				{type: "string", name: "platform", number: 4},
				{type: "map<string, string>", name: "labels", number: 5},
				{type: "bool", name: "has_content", number: 6},
				{type: "ImageContentDescriptor", name: "target", number: 7},
				{rule: "repeated", type: "ImageContentDescriptor", name: "descriptors", number: 8},
			]},
			{kind: "message", name: "UploadInstanceImageArchiveChunkRequest", fields: [
				{type: "string", name: "instance", number: 1},
				{type: "string", name: "upload_id", number: 2},
				{type: "string", name: "image_ref", number: 3},
				{type: "uint64", name: "offset", number: 4},
				{type: "bytes", name: "data", number: 5},
				{type: "bool", name: "eof", number: 6},
			]},
			{kind: "message", name: "UploadInstanceImageArchiveChunkResponse", fields: [
				{type: "uint64", name: "received_bytes", number: 1},
				{type: "bool", name: "loaded", number: 2},
				{type: "string", name: "image_ref", number: 3},
				{type: "string", name: "target_digest", number: 4},
				{type: "string", name: "platform", number: 5},
			]},
			{kind: "message", name: "CoreEndpoint", fields: [
				{type: "string", name: "kind", number: 1},
				{type: "string", name: "address", number: 2},
				{type: "bool", name: "active", number: 3},
				{type: "string", name: "message", number: 4},
			]},
			{kind: "message", name: "HostAgentConnectionStatus", fields: [
				{type: "bool", name: "connected", number: 1},
				{type: "string", name: "socket", number: 2},
				{type: "string", name: "message", number: 3},
			]},
			{kind: "message", name: "SystemStatus", fields: [
				{type: "string", name: "core_status", number: 1},
				{type: "string", name: "work_dir", number: 2},
				{type: "string", name: "capabilities", number: 3},
				{type: "int32", name: "p2p_port", number: 4},
				{rule: "repeated", type: "CoreEndpoint", name: "endpoints", number: 5},
				{type: "HostAgentConnectionStatus", name: "host_agent", number: 6},
				{type: "bool", name: "network_enabled", number: 7},
				{type: "bool", name: "host_agent_supported", number: 8},
				{type: "NetworkRuntimeStatus", name: "network", number: 9},
			]},
			{kind: "message", name: "GetSystemStatusRequest", fields: []},
			{kind: "message", name: "GetSystemStatusResponse", fields: [
				{type: "SystemStatus", name: "status", number: 1},
			]},
			{kind: "message", name: "StartHostAgentRequest", fields: []},
			{kind: "message", name: "StartHostAgentResponse", fields: [
				{type: "HostAgentConnectionStatus", name: "status", number: 1},
			]},
			{kind: "message", name: "StopHostAgentRequest", fields: []},
			{kind: "message", name: "StopHostAgentResponse", fields: [
				{type: "HostAgentConnectionStatus", name: "status", number: 1},
			]},
				{kind: "message", name: "Commit", fields: [
					{type: "string", name: "hash", number: 1},
					{type: "string", name: "committer", number: 2},
					{type: "string", name: "message", number: 3},
					{rule: "repeated", type: "string", name: "states", number: 4},
					{type: "int64", name: "date_unix", number: 5},
					{rule: "repeated", type: "string", name: "parent_hashes", number: 6},
					{rule: "repeated", type: "string", name: "refs", number: 7},
				]},
				{kind: "message", name: "CommitGraphRelation", fields: [
					{type: "string", name: "parent_hash", number: 1},
					{type: "int32", name: "parent_row", number: 2},
					{type: "int32", name: "from_lane", number: 3},
					{type: "int32", name: "to_lane", number: 4},
					{type: "bool", name: "visible", number: 5},
				]},
				{kind: "message", name: "CommitGraphItem", fields: [
					{type: "Commit", name: "commit", number: 1},
					{type: "int32", name: "row", number: 2},
					{type: "int32", name: "lane", number: 3},
					{rule: "repeated", type: "int32", name: "active_lanes", number: 4},
					{rule: "repeated", type: "CommitGraphRelation", name: "relations", number: 5},
				]},
				{kind: "message", name: "CommitGraph", fields: [
					{rule: "repeated", type: "CommitGraphItem", name: "items", number: 1},
					{type: "int32", name: "lane_count", number: 2},
				]},
				{kind: "message", name: "GetLocalCommitsRequest", fields: []},
				{kind: "message", name: "GetLocalCommitsResponse", fields: [
					{rule: "repeated", type: "Commit", name: "commits", number: 1},
					{type: "CommitGraph", name: "graph", number: 2},
				]},
				{kind: "message", name: "GetRemoteCommitsRequest", fields: [{type: "string", name: "remote", number: 1}]},
				{kind: "message", name: "GetRemoteCommitsResponse", fields: [
					{rule: "repeated", type: "Commit", name: "commits", number: 1},
					{type: "CommitGraph", name: "graph", number: 2},
				]},
			{kind: "message", name: "SqlCell", fields: [
				{type: "string", name: "value", number: 1},
				{type: "bool", name: "is_null", number: 2},
			]},
			{kind: "message", name: "SqlRow", fields: [
				{rule: "repeated", type: "SqlCell", name: "cells", number: 1},
			]},
			{kind: "message", name: "ExecuteSqlRequest", fields: [
				{type: "string", name: "sql", number: 1},
				{type: "int32", name: "max_rows", number: 2},
			]},
			{kind: "message", name: "ExecuteSqlResponse", fields: [
				{rule: "repeated", type: "string", name: "columns", number: 1},
				{rule: "repeated", type: "SqlRow", name: "rows", number: 2},
				{type: "int64", name: "rows_affected", number: 3},
				{type: "bool", name: "truncated", number: 4},
				{type: "string", name: "message", number: 5},
			]},
		]
	}
}

lineage: {
	name: "protos.client_api"
	schemas: [{
		version: [0, 0]
		schema: {
			InitRequest?:                         #InitRequest
			InitResponse?:                        #InitResponse
			UserDevice?:                          #UserDevice
			GetUserDevicesRequest?:               #GetUserDevicesRequest
			GetUserDevicesResponse?:              #GetUserDevicesResponse
			GetUserInfoRequest?:                  #GetUserInfoRequest
			GetUserInfoResponse?:                 #GetUserInfoResponse
			Organisation?:                            #Organisation
			ListOrganisationsRequest?:                #ListOrganisationsRequest
			ListOrganisationsResponse?:               #ListOrganisationsResponse
			StartDeviceInviteRequest?:            #StartDeviceInviteRequest
			StartDeviceInviteResponse?:           #StartDeviceInviteResponse
			NearbyOrganisation?:                      #NearbyOrganisation
			ListNearbyOrganisationsRequest?:          #ListNearbyOrganisationsRequest
			ListNearbyOrganisationsResponse?:         #ListNearbyOrganisationsResponse
			JoinOrganisationRequest?:                 #JoinOrganisationRequest
			JoinOrganisationResponse?:                #JoinOrganisationResponse
			GetLocalSSHKeyRequest?:               #GetLocalSSHKeyRequest
			GetLocalSSHKeyResponse?:              #GetLocalSSHKeyResponse
			App?:                                 #App
			GetAppsRequest?:                      #GetAppsRequest
			GetAppsResponse?:                     #GetAppsResponse
			CreateAppRequest?:                    #CreateAppRequest
			CreateAppResponse?:                   #CreateAppResponse
			StartAppRequest?:                     #StartAppRequest
			StartAppResponse?:                    #StartAppResponse
			StopAppRequest?:                      #StopAppRequest
			StopAppResponse?:                     #StopAppResponse
			RemoveAppRequest?:                    #RemoveAppRequest
			RemoveAppResponse?:                   #RemoveAppResponse
			GetAppLogsRequest?:                   #GetAppLogsRequest
			GetAppLogsResponse?:                  #GetAppLogsResponse
			Installer?:                           #Installer
			GetInstallersRequest?:                #GetInstallersRequest
			GetInstallersResponse?:               #GetInstallersResponse
			GetInstallerRequest?:                 #GetInstallerRequest
			GetInstallerResponse?:                #GetInstallerResponse
			CloudMachineSpec?:                    #CloudMachineSpec
			CloudType?:                           #CloudType
			CloudProvider?:                       #CloudProvider
			GetSupportedCloudProvidersRequest?:   #GetSupportedCloudProvidersRequest
			GetSupportedCloudProvidersResponse?:  #GetSupportedCloudProvidersResponse
			GetCloudProvidersRequest?:            #GetCloudProvidersRequest
			GetCloudProvidersResponse?:           #GetCloudProvidersResponse
			GetCloudProviderRequest?:             #GetCloudProviderRequest
			GetCloudProviderResponse?:            #GetCloudProviderResponse
			AddCloudProviderRequest?:             #AddCloudProviderRequest
			AddCloudProviderResponse?:            #AddCloudProviderResponse
			RemoveCloudProviderRequest?:          #RemoveCloudProviderRequest
			RemoveCloudProviderResponse?:         #RemoveCloudProviderResponse
			ProvisionerMachineSpec?:              #ProvisionerMachineSpec
			ProvisionerType?:                     #ProvisionerType
			Provisioner?:                         #Provisioner
			GetSupportedProvisionersRequest?:      #GetSupportedProvisionersRequest
			GetSupportedProvisionersResponse?:     #GetSupportedProvisionersResponse
			GetProvisionersRequest?:               #GetProvisionersRequest
			GetProvisionersResponse?:              #GetProvisionersResponse
			GetProvisionerRequest?:                #GetProvisionerRequest
			GetProvisionerResponse?:               #GetProvisionerResponse
			AddProvisionerRequest?:                #AddProvisionerRequest
			AddProvisionerResponse?:               #AddProvisionerResponse
			RemoveProvisionerRequest?:             #RemoveProvisionerRequest
			RemoveProvisionerResponse?:            #RemoveProvisionerResponse
			CloudInstance?:                       #CloudInstance
			GetInstancesRequest?:                 #GetInstancesRequest
			GetInstancesResponse?:                #GetInstancesResponse
			GetInstanceRequest?:                  #GetInstanceRequest
			GetInstanceResponse?:                 #GetInstanceResponse
			InstanceDeployFieldOption?:           #InstanceDeployFieldOption
			InstanceDeployField?:                 #InstanceDeployField
			GetInstanceDeployOptionsRequest?:     #GetInstanceDeployOptionsRequest
			GetInstanceDeployOptionsResponse?:    #GetInstanceDeployOptionsResponse
			DeployInstanceRequest?:               #DeployInstanceRequest
			DeployInstanceResponse?:              #DeployInstanceResponse
			RemoveInstanceRequest?:               #RemoveInstanceRequest
			RemoveInstanceResponse?:              #RemoveInstanceResponse
			StartInstanceRequest?:                #StartInstanceRequest
			StartInstanceResponse?:               #StartInstanceResponse
			StopInstanceRequest?:                 #StopInstanceRequest
			StopInstanceResponse?:                #StopInstanceResponse
			GetInstanceKeyRequest?:               #GetInstanceKeyRequest
			GetInstanceKeyResponse?:              #GetInstanceKeyResponse
			GetInstanceLogsRequest?:              #GetInstanceLogsRequest
			GetInstanceLogsResponse?:             #GetInstanceLogsResponse
			InitInstanceRequest?:                 #InitInstanceRequest
			InitInstanceResponse?:                #InitInstanceResponse
			UpdateInstanceRequest?:               #UpdateInstanceRequest
			UpdateInstanceResponse?:              #UpdateInstanceResponse
			GetNetworkStateRequest?:              #GetNetworkStateRequest
			GetNetworkStateResponse?:             #GetNetworkStateResponse
			SetNetworkEnabledRequest?:            #SetNetworkEnabledRequest
			SetNetworkEnabledResponse?:           #SetNetworkEnabledResponse
			NetworkRuntimeStatus?:                #NetworkRuntimeStatus
			NetworkState?:                        #NetworkState
			NetworkInterface?:                    #NetworkInterface
			NetworkAddress?:                      #NetworkAddress
			NetworkRoute?:                        #NetworkRoute
			WireGuardPeer?:                       #WireGuardPeer
			FirewallTable?:                       #FirewallTable
			FirewallChain?:                       #FirewallChain
			FirewallRule?:                        #FirewallRule
			DNSState?:                            #DNSState
			ExitRoute?:                          #ExitRoute
			GetExitRoutesRequest?:                #GetExitRoutesRequest
			GetExitRoutesResponse?:               #GetExitRoutesResponse
			GetMobileTunnelConfigRequest?:        #GetMobileTunnelConfigRequest
			MobileTunnelConfig?:                  #MobileTunnelConfig
			GetMobileTunnelConfigResponse?:       #GetMobileTunnelConfigResponse
			GetRuntimeStateRequest?:              #GetRuntimeStateRequest
			GetRuntimeStateResponse?:             #GetRuntimeStateResponse
			WatchChangesRequest?:                 #WatchChangesRequest
			WatchChangesResponse?:                #WatchChangesResponse
			Task?:                                #Task
			TaskEvent?:                           #TaskEvent
			GetTasksRequest?:                     #GetTasksRequest
			GetTasksResponse?:                    #GetTasksResponse
			GetTaskRequest?:                      #GetTaskRequest
			GetTaskResponse?:                     #GetTaskResponse
			RuntimeState?:                        #RuntimeState
			RuntimePeerStatus?:                   #RuntimePeerStatus
			RuntimeCompatibility?:                #RuntimeCompatibility
			SetExitRouteRequest?:                 #SetExitRouteRequest
			SetExitRouteResponse?:                #SetExitRouteResponse
			ClearExitRouteRequest?:               #ClearExitRouteRequest
			ClearExitRouteResponse?:              #ClearExitRouteResponse
			CloudImage?:                          #CloudImage
			CloudSpecificImage?:                  #CloudSpecificImage
			Release?:                             #Release
			GetProtosdReleasesRequest?:           #GetProtosdReleasesRequest
			GetProtosdReleasesResponse?:          #GetProtosdReleasesResponse
			GetCloudImagesRequest?:               #GetCloudImagesRequest
			GetCloudImagesResponse?:              #GetCloudImagesResponse
			GetProvisionerImagesRequest?:          #GetProvisionerImagesRequest
			GetProvisionerImagesResponse?:         #GetProvisionerImagesResponse
			UploadCloudImageRequest?:             #UploadCloudImageRequest
			UploadCloudImageResponse?:            #UploadCloudImageResponse
			UploadProvisionerImageRequest?:        #UploadProvisionerImageRequest
			UploadProvisionerImageResponse?:       #UploadProvisionerImageResponse
			RemoveCloudImageRequest?:             #RemoveCloudImageRequest
			RemoveCloudImageResponse?:            #RemoveCloudImageResponse
			RemoveProvisionerImageRequest?:        #RemoveProvisionerImageRequest
			RemoveProvisionerImageResponse?:       #RemoveProvisionerImageResponse
			CoreEndpoint?:                         #CoreEndpoint
			HostAgentConnectionStatus?:            #HostAgentConnectionStatus
			SystemStatus?:                         #SystemStatus
			GetSystemStatusRequest?:               #GetSystemStatusRequest
			GetSystemStatusResponse?:              #GetSystemStatusResponse
			StartHostAgentRequest?:                #StartHostAgentRequest
			StartHostAgentResponse?:               #StartHostAgentResponse
			StopHostAgentRequest?:                 #StopHostAgentRequest
			StopHostAgentResponse?:                #StopHostAgentResponse
				Commit?:                              #Commit
				CommitGraphRelation?:                 #CommitGraphRelation
				CommitGraphItem?:                     #CommitGraphItem
				CommitGraph?:                         #CommitGraph
				GetLocalCommitsRequest?:              #GetLocalCommitsRequest
			GetLocalCommitsResponse?:             #GetLocalCommitsResponse
			GetRemoteCommitsRequest?:             #GetRemoteCommitsRequest
			GetRemoteCommitsResponse?:            #GetRemoteCommitsResponse
			SqlCell?:                             #SqlCell
			SqlRow?:                              #SqlRow
			ExecuteSqlRequest?:                   #ExecuteSqlRequest
			ExecuteSqlResponse?:                  #ExecuteSqlResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
