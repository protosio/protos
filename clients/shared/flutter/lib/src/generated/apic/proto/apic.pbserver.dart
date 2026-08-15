// This is a generated file - do not edit.
//
// Generated from apic/proto/apic.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

import 'apic.pb.dart' as $0;
import 'apic.pbjson.dart';

export 'apic.pb.dart';

abstract class ProtosClientApiServiceBase extends $pb.GeneratedService {
  $async.Future<$0.InitResponse> init(
      $pb.ServerContext ctx, $0.InitRequest request);
  $async.Future<$0.GetUserDevicesResponse> getUserDevices(
      $pb.ServerContext ctx, $0.GetUserDevicesRequest request);
  $async.Future<$0.GetUserInfoResponse> getUserInfo(
      $pb.ServerContext ctx, $0.GetUserInfoRequest request);
  $async.Future<$0.ListOrganisationsResponse> listOrganisations(
      $pb.ServerContext ctx, $0.ListOrganisationsRequest request);
  $async.Future<$0.StartDeviceInviteResponse> startDeviceInvite(
      $pb.ServerContext ctx, $0.StartDeviceInviteRequest request);
  $async.Future<$0.ListNearbyOrganisationsResponse> listNearbyOrganisations(
      $pb.ServerContext ctx, $0.ListNearbyOrganisationsRequest request);
  $async.Future<$0.JoinOrganisationResponse> joinOrganisation(
      $pb.ServerContext ctx, $0.JoinOrganisationRequest request);
  $async.Future<$0.GetLocalSSHKeyResponse> getLocalSSHKey(
      $pb.ServerContext ctx, $0.GetLocalSSHKeyRequest request);
  $async.Future<$0.GetAppsResponse> getApps(
      $pb.ServerContext ctx, $0.GetAppsRequest request);
  $async.Future<$0.CreateAppResponse> createApp(
      $pb.ServerContext ctx, $0.CreateAppRequest request);
  $async.Future<$0.StartAppResponse> startApp(
      $pb.ServerContext ctx, $0.StartAppRequest request);
  $async.Future<$0.StopAppResponse> stopApp(
      $pb.ServerContext ctx, $0.StopAppRequest request);
  $async.Future<$0.RemoveAppResponse> removeApp(
      $pb.ServerContext ctx, $0.RemoveAppRequest request);
  $async.Future<$0.GetAppLogsResponse> getAppLogs(
      $pb.ServerContext ctx, $0.GetAppLogsRequest request);
  $async.Future<$0.GetSupportedProvisionersResponse> getSupportedProvisioners(
      $pb.ServerContext ctx, $0.GetSupportedProvisionersRequest request);
  $async.Future<$0.GetProvisionersResponse> getProvisioners(
      $pb.ServerContext ctx, $0.GetProvisionersRequest request);
  $async.Future<$0.GetProvisionerResponse> getProvisioner(
      $pb.ServerContext ctx, $0.GetProvisionerRequest request);
  $async.Future<$0.AddProvisionerResponse> addProvisioner(
      $pb.ServerContext ctx, $0.AddProvisionerRequest request);
  $async.Future<$0.RemoveProvisionerResponse> removeProvisioner(
      $pb.ServerContext ctx, $0.RemoveProvisionerRequest request);
  $async.Future<$0.GetInstancesResponse> getInstances(
      $pb.ServerContext ctx, $0.GetInstancesRequest request);
  $async.Future<$0.GetInstanceResponse> getInstance(
      $pb.ServerContext ctx, $0.GetInstanceRequest request);
  $async.Future<$0.GetInstanceDeployOptionsResponse> getInstanceDeployOptions(
      $pb.ServerContext ctx, $0.GetInstanceDeployOptionsRequest request);
  $async.Future<$0.DeployInstanceResponse> deployInstance(
      $pb.ServerContext ctx, $0.DeployInstanceRequest request);
  $async.Future<$0.RemoveInstanceResponse> removeInstance(
      $pb.ServerContext ctx, $0.RemoveInstanceRequest request);
  $async.Future<$0.StartInstanceResponse> startInstance(
      $pb.ServerContext ctx, $0.StartInstanceRequest request);
  $async.Future<$0.StopInstanceResponse> stopInstance(
      $pb.ServerContext ctx, $0.StopInstanceRequest request);
  $async.Future<$0.GetInstanceKeyResponse> getInstanceKey(
      $pb.ServerContext ctx, $0.GetInstanceKeyRequest request);
  $async.Future<$0.GetInstanceLogsResponse> getInstanceLogs(
      $pb.ServerContext ctx, $0.GetInstanceLogsRequest request);
  $async.Future<$0.InitInstanceResponse> initInstance(
      $pb.ServerContext ctx, $0.InitInstanceRequest request);
  $async.Future<$0.UpdateInstanceResponse> updateInstance(
      $pb.ServerContext ctx, $0.UpdateInstanceRequest request);
  $async.Future<$0.GetNetworkStateResponse> getNetworkState(
      $pb.ServerContext ctx, $0.GetNetworkStateRequest request);
  $async.Future<$0.SetNetworkEnabledResponse> setNetworkEnabled(
      $pb.ServerContext ctx, $0.SetNetworkEnabledRequest request);
  $async.Future<$0.GetExitRoutesResponse> getExitRoutes(
      $pb.ServerContext ctx, $0.GetExitRoutesRequest request);
  $async.Future<$0.GetMobileTunnelConfigResponse> getMobileTunnelConfig(
      $pb.ServerContext ctx, $0.GetMobileTunnelConfigRequest request);
  $async.Future<$0.GetRuntimeStateResponse> getRuntimeState(
      $pb.ServerContext ctx, $0.GetRuntimeStateRequest request);
  $async.Future<$0.WatchChangesResponse> watchChanges(
      $pb.ServerContext ctx, $0.WatchChangesRequest request);
  $async.Future<$0.GetTasksResponse> getTasks(
      $pb.ServerContext ctx, $0.GetTasksRequest request);
  $async.Future<$0.GetTaskResponse> getTask(
      $pb.ServerContext ctx, $0.GetTaskRequest request);
  $async.Future<$0.WatchTaskResponse> watchTask(
      $pb.ServerContext ctx, $0.WatchTaskRequest request);
  $async.Future<$0.SetExitRouteResponse> setExitRoute(
      $pb.ServerContext ctx, $0.SetExitRouteRequest request);
  $async.Future<$0.ClearExitRouteResponse> clearExitRoute(
      $pb.ServerContext ctx, $0.ClearExitRouteRequest request);
  $async.Future<$0.GetProtosdReleasesResponse> getProtosdReleases(
      $pb.ServerContext ctx, $0.GetProtosdReleasesRequest request);
  $async.Future<$0.GetProvisionerImagesResponse> getProvisionerImages(
      $pb.ServerContext ctx, $0.GetProvisionerImagesRequest request);
  $async.Future<$0.UploadProvisionerImageResponse> uploadProvisionerImage(
      $pb.ServerContext ctx, $0.UploadProvisionerImageRequest request);
  $async.Future<$0.RemoveProvisionerImageResponse> removeProvisionerImage(
      $pb.ServerContext ctx, $0.RemoveProvisionerImageRequest request);
  $async.Future<$0.GetInstanceImageResponse> getInstanceImage(
      $pb.ServerContext ctx, $0.GetInstanceImageRequest request);
  $async.Future<$0.UploadInstanceImageArchiveResponse>
      uploadInstanceImageArchive(
          $pb.ServerContext ctx, $0.UploadInstanceImageArchiveRequest request);
  $async.Future<$0.GetSystemStatusResponse> getSystemStatus(
      $pb.ServerContext ctx, $0.GetSystemStatusRequest request);
  $async.Future<$0.StartHostAgentResponse> startHostAgent(
      $pb.ServerContext ctx, $0.StartHostAgentRequest request);
  $async.Future<$0.StopHostAgentResponse> stopHostAgent(
      $pb.ServerContext ctx, $0.StopHostAgentRequest request);
  $async.Future<$0.GetLocalCommitsResponse> getLocalCommits(
      $pb.ServerContext ctx, $0.GetLocalCommitsRequest request);
  $async.Future<$0.GetRemoteCommitsResponse> getRemoteCommits(
      $pb.ServerContext ctx, $0.GetRemoteCommitsRequest request);
  $async.Future<$0.GetCommitDiffResponse> getCommitDiff(
      $pb.ServerContext ctx, $0.GetCommitDiffRequest request);
  $async.Future<$0.ExecuteSqlResponse> executeSql(
      $pb.ServerContext ctx, $0.ExecuteSqlRequest request);

  $pb.GeneratedMessage createRequest($core.String methodName) {
    switch (methodName) {
      case 'Init':
        return $0.InitRequest();
      case 'GetUserDevices':
        return $0.GetUserDevicesRequest();
      case 'GetUserInfo':
        return $0.GetUserInfoRequest();
      case 'ListOrganisations':
        return $0.ListOrganisationsRequest();
      case 'StartDeviceInvite':
        return $0.StartDeviceInviteRequest();
      case 'ListNearbyOrganisations':
        return $0.ListNearbyOrganisationsRequest();
      case 'JoinOrganisation':
        return $0.JoinOrganisationRequest();
      case 'GetLocalSSHKey':
        return $0.GetLocalSSHKeyRequest();
      case 'GetApps':
        return $0.GetAppsRequest();
      case 'CreateApp':
        return $0.CreateAppRequest();
      case 'StartApp':
        return $0.StartAppRequest();
      case 'StopApp':
        return $0.StopAppRequest();
      case 'RemoveApp':
        return $0.RemoveAppRequest();
      case 'GetAppLogs':
        return $0.GetAppLogsRequest();
      case 'GetSupportedProvisioners':
        return $0.GetSupportedProvisionersRequest();
      case 'GetProvisioners':
        return $0.GetProvisionersRequest();
      case 'GetProvisioner':
        return $0.GetProvisionerRequest();
      case 'AddProvisioner':
        return $0.AddProvisionerRequest();
      case 'RemoveProvisioner':
        return $0.RemoveProvisionerRequest();
      case 'GetInstances':
        return $0.GetInstancesRequest();
      case 'GetInstance':
        return $0.GetInstanceRequest();
      case 'GetInstanceDeployOptions':
        return $0.GetInstanceDeployOptionsRequest();
      case 'DeployInstance':
        return $0.DeployInstanceRequest();
      case 'RemoveInstance':
        return $0.RemoveInstanceRequest();
      case 'StartInstance':
        return $0.StartInstanceRequest();
      case 'StopInstance':
        return $0.StopInstanceRequest();
      case 'GetInstanceKey':
        return $0.GetInstanceKeyRequest();
      case 'GetInstanceLogs':
        return $0.GetInstanceLogsRequest();
      case 'InitInstance':
        return $0.InitInstanceRequest();
      case 'UpdateInstance':
        return $0.UpdateInstanceRequest();
      case 'GetNetworkState':
        return $0.GetNetworkStateRequest();
      case 'SetNetworkEnabled':
        return $0.SetNetworkEnabledRequest();
      case 'GetExitRoutes':
        return $0.GetExitRoutesRequest();
      case 'GetMobileTunnelConfig':
        return $0.GetMobileTunnelConfigRequest();
      case 'GetRuntimeState':
        return $0.GetRuntimeStateRequest();
      case 'WatchChanges':
        return $0.WatchChangesRequest();
      case 'GetTasks':
        return $0.GetTasksRequest();
      case 'GetTask':
        return $0.GetTaskRequest();
      case 'WatchTask':
        return $0.WatchTaskRequest();
      case 'SetExitRoute':
        return $0.SetExitRouteRequest();
      case 'ClearExitRoute':
        return $0.ClearExitRouteRequest();
      case 'GetProtosdReleases':
        return $0.GetProtosdReleasesRequest();
      case 'GetProvisionerImages':
        return $0.GetProvisionerImagesRequest();
      case 'UploadProvisionerImage':
        return $0.UploadProvisionerImageRequest();
      case 'RemoveProvisionerImage':
        return $0.RemoveProvisionerImageRequest();
      case 'GetInstanceImage':
        return $0.GetInstanceImageRequest();
      case 'UploadInstanceImageArchive':
        return $0.UploadInstanceImageArchiveRequest();
      case 'GetSystemStatus':
        return $0.GetSystemStatusRequest();
      case 'StartHostAgent':
        return $0.StartHostAgentRequest();
      case 'StopHostAgent':
        return $0.StopHostAgentRequest();
      case 'GetLocalCommits':
        return $0.GetLocalCommitsRequest();
      case 'GetRemoteCommits':
        return $0.GetRemoteCommitsRequest();
      case 'GetCommitDiff':
        return $0.GetCommitDiffRequest();
      case 'ExecuteSql':
        return $0.ExecuteSqlRequest();
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $async.Future<$pb.GeneratedMessage> handleCall($pb.ServerContext ctx,
      $core.String methodName, $pb.GeneratedMessage request) {
    switch (methodName) {
      case 'Init':
        return init(ctx, request as $0.InitRequest);
      case 'GetUserDevices':
        return getUserDevices(ctx, request as $0.GetUserDevicesRequest);
      case 'GetUserInfo':
        return getUserInfo(ctx, request as $0.GetUserInfoRequest);
      case 'ListOrganisations':
        return listOrganisations(ctx, request as $0.ListOrganisationsRequest);
      case 'StartDeviceInvite':
        return startDeviceInvite(ctx, request as $0.StartDeviceInviteRequest);
      case 'ListNearbyOrganisations':
        return listNearbyOrganisations(
            ctx, request as $0.ListNearbyOrganisationsRequest);
      case 'JoinOrganisation':
        return joinOrganisation(ctx, request as $0.JoinOrganisationRequest);
      case 'GetLocalSSHKey':
        return getLocalSSHKey(ctx, request as $0.GetLocalSSHKeyRequest);
      case 'GetApps':
        return getApps(ctx, request as $0.GetAppsRequest);
      case 'CreateApp':
        return createApp(ctx, request as $0.CreateAppRequest);
      case 'StartApp':
        return startApp(ctx, request as $0.StartAppRequest);
      case 'StopApp':
        return stopApp(ctx, request as $0.StopAppRequest);
      case 'RemoveApp':
        return removeApp(ctx, request as $0.RemoveAppRequest);
      case 'GetAppLogs':
        return getAppLogs(ctx, request as $0.GetAppLogsRequest);
      case 'GetSupportedProvisioners':
        return getSupportedProvisioners(
            ctx, request as $0.GetSupportedProvisionersRequest);
      case 'GetProvisioners':
        return getProvisioners(ctx, request as $0.GetProvisionersRequest);
      case 'GetProvisioner':
        return getProvisioner(ctx, request as $0.GetProvisionerRequest);
      case 'AddProvisioner':
        return addProvisioner(ctx, request as $0.AddProvisionerRequest);
      case 'RemoveProvisioner':
        return removeProvisioner(ctx, request as $0.RemoveProvisionerRequest);
      case 'GetInstances':
        return getInstances(ctx, request as $0.GetInstancesRequest);
      case 'GetInstance':
        return getInstance(ctx, request as $0.GetInstanceRequest);
      case 'GetInstanceDeployOptions':
        return getInstanceDeployOptions(
            ctx, request as $0.GetInstanceDeployOptionsRequest);
      case 'DeployInstance':
        return deployInstance(ctx, request as $0.DeployInstanceRequest);
      case 'RemoveInstance':
        return removeInstance(ctx, request as $0.RemoveInstanceRequest);
      case 'StartInstance':
        return startInstance(ctx, request as $0.StartInstanceRequest);
      case 'StopInstance':
        return stopInstance(ctx, request as $0.StopInstanceRequest);
      case 'GetInstanceKey':
        return getInstanceKey(ctx, request as $0.GetInstanceKeyRequest);
      case 'GetInstanceLogs':
        return getInstanceLogs(ctx, request as $0.GetInstanceLogsRequest);
      case 'InitInstance':
        return initInstance(ctx, request as $0.InitInstanceRequest);
      case 'UpdateInstance':
        return updateInstance(ctx, request as $0.UpdateInstanceRequest);
      case 'GetNetworkState':
        return getNetworkState(ctx, request as $0.GetNetworkStateRequest);
      case 'SetNetworkEnabled':
        return setNetworkEnabled(ctx, request as $0.SetNetworkEnabledRequest);
      case 'GetExitRoutes':
        return getExitRoutes(ctx, request as $0.GetExitRoutesRequest);
      case 'GetMobileTunnelConfig':
        return getMobileTunnelConfig(
            ctx, request as $0.GetMobileTunnelConfigRequest);
      case 'GetRuntimeState':
        return getRuntimeState(ctx, request as $0.GetRuntimeStateRequest);
      case 'WatchChanges':
        return watchChanges(ctx, request as $0.WatchChangesRequest);
      case 'GetTasks':
        return getTasks(ctx, request as $0.GetTasksRequest);
      case 'GetTask':
        return getTask(ctx, request as $0.GetTaskRequest);
      case 'WatchTask':
        return watchTask(ctx, request as $0.WatchTaskRequest);
      case 'SetExitRoute':
        return setExitRoute(ctx, request as $0.SetExitRouteRequest);
      case 'ClearExitRoute':
        return clearExitRoute(ctx, request as $0.ClearExitRouteRequest);
      case 'GetProtosdReleases':
        return getProtosdReleases(ctx, request as $0.GetProtosdReleasesRequest);
      case 'GetProvisionerImages':
        return getProvisionerImages(
            ctx, request as $0.GetProvisionerImagesRequest);
      case 'UploadProvisionerImage':
        return uploadProvisionerImage(
            ctx, request as $0.UploadProvisionerImageRequest);
      case 'RemoveProvisionerImage':
        return removeProvisionerImage(
            ctx, request as $0.RemoveProvisionerImageRequest);
      case 'GetInstanceImage':
        return getInstanceImage(ctx, request as $0.GetInstanceImageRequest);
      case 'UploadInstanceImageArchive':
        return uploadInstanceImageArchive(
            ctx, request as $0.UploadInstanceImageArchiveRequest);
      case 'GetSystemStatus':
        return getSystemStatus(ctx, request as $0.GetSystemStatusRequest);
      case 'StartHostAgent':
        return startHostAgent(ctx, request as $0.StartHostAgentRequest);
      case 'StopHostAgent':
        return stopHostAgent(ctx, request as $0.StopHostAgentRequest);
      case 'GetLocalCommits':
        return getLocalCommits(ctx, request as $0.GetLocalCommitsRequest);
      case 'GetRemoteCommits':
        return getRemoteCommits(ctx, request as $0.GetRemoteCommitsRequest);
      case 'GetCommitDiff':
        return getCommitDiff(ctx, request as $0.GetCommitDiffRequest);
      case 'ExecuteSql':
        return executeSql(ctx, request as $0.ExecuteSqlRequest);
      default:
        throw $core.ArgumentError('Unknown method: $methodName');
    }
  }

  $core.Map<$core.String, $core.dynamic> get $json =>
      ProtosClientApiServiceBase$json;
  $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
      get $messageJson => ProtosClientApiServiceBase$messageJson;
}
