package apicv1

#InitRequest: {
	username?:     string
	name?:         string
	organization?: string
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
	username?: string
	name?:     string
	is_admin?: bool
}
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

#Commit: {
	hash?:      string
	committer?: string
	message?:   string
}
#GetLocalCommitsRequest: {}
#GetLocalCommitsResponse: commits?: [...#Commit]
#GetRemoteCommitsRequest: remote?: string
#GetRemoteCommitsResponse: commits?: [...#Commit]

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
				{name: "DeployInstance", request: "DeployInstanceRequest", response: "DeployInstanceResponse"},
				{name: "RemoveInstance", request: "RemoveInstanceRequest", response: "RemoveInstanceResponse"},
				{name: "StartInstance", request: "StartInstanceRequest", response: "StartInstanceResponse"},
				{name: "StopInstance", request: "StopInstanceRequest", response: "StopInstanceResponse"},
				{name: "GetInstanceKey", request: "GetInstanceKeyRequest", response: "GetInstanceKeyResponse"},
				{name: "GetInstanceLogs", request: "GetInstanceLogsRequest", response: "GetInstanceLogsResponse"},
				{name: "InitInstance", request: "InitInstanceRequest", response: "InitInstanceResponse"},
				{name: "UpdateInstance", request: "UpdateInstanceRequest", response: "UpdateInstanceResponse"},
				{name: "GetProtosdReleases", request: "GetProtosdReleasesRequest", response: "GetProtosdReleasesResponse"},
				{name: "GetCloudImages", request: "GetCloudImagesRequest", response: "GetCloudImagesResponse"},
				{name: "UploadCloudImage", request: "UploadCloudImageRequest", response: "UploadCloudImageResponse"},
				{name: "RemoveCloudImage", request: "RemoveCloudImageRequest", response: "RemoveCloudImageResponse"},
				{name: "GetProvisionerImages", request: "GetProvisionerImagesRequest", response: "GetProvisionerImagesResponse"},
				{name: "UploadProvisionerImage", request: "UploadProvisionerImageRequest", response: "UploadProvisionerImageResponse"},
				{name: "RemoveProvisionerImage", request: "RemoveProvisionerImageRequest", response: "RemoveProvisionerImageResponse"},
				{name: "GetLocalCommits", request: "GetLocalCommitsRequest", response: "GetLocalCommitsResponse"},
				{name: "GetRemoteCommits", request: "GetRemoteCommitsRequest", response: "GetRemoteCommitsResponse"},
			]
		}]
		declarations: [
			{kind: "message", name: "InitRequest", fields: [
				{type: "string", name: "username", number: 1},
				{type: "string", name: "name", number: 2},
				{type: "string", name: "organization", number: 3},
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
			]},
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
			{kind: "message", name: "Commit", fields: [
				{type: "string", name: "hash", number: 1},
				{type: "string", name: "committer", number: 2},
				{type: "string", name: "message", number: 3},
			]},
			{kind: "message", name: "GetLocalCommitsRequest", fields: []},
			{kind: "message", name: "GetLocalCommitsResponse", fields: [{rule: "repeated", type: "Commit", name: "commits", number: 1}]},
			{kind: "message", name: "GetRemoteCommitsRequest", fields: [{type: "string", name: "remote", number: 1}]},
			{kind: "message", name: "GetRemoteCommitsResponse", fields: [{rule: "repeated", type: "Commit", name: "commits", number: 1}]},
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
			Commit?:                              #Commit
			GetLocalCommitsRequest?:              #GetLocalCommitsRequest
			GetLocalCommitsResponse?:             #GetLocalCommitsResponse
			GetRemoteCommitsRequest?:             #GetRemoteCommitsRequest
			GetRemoteCommitsResponse?:            #GetRemoteCommitsResponse
		}
	}]
	lenses: []
}

migration: contract.migration
proto:     contract.proto
