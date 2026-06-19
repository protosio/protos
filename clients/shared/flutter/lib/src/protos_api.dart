import 'dart:convert';

import 'generated/apic/proto/apic.pb.dart' as pb;
import 'join_modes.dart';
import 'native/protos_bridge.dart';

class ProtosApi {
  const ProtosApi(this.bridge);

  final NativeProtosBridge bridge;

  Future<void> start({
    ProtosBridgeConfig config = const ProtosBridgeConfig(),
  }) => bridge.start(config: config);

  Future<void> stop() => bridge.stop();

  Future<void> initUser({
    required String username,
    required String name,
    required String organisation,
  }) async {
    await bridge.call(
      'Init',
      pb.InitRequest(
        username: username,
        name: name,
        organisation: organisation,
      ),
      pb.InitResponse.create,
    );
  }

  Future<pb.GetUserInfoResponse> userInfo() {
    return bridge.call(
      'GetUserInfo',
      pb.GetUserInfoRequest(),
      pb.GetUserInfoResponse.create,
    );
  }

  Future<pb.ListOrganisationsResponse> organisations() {
    return bridge.call(
      'ListOrganisations',
      pb.ListOrganisationsRequest(),
      pb.ListOrganisationsResponse.create,
    );
  }

  Future<pb.ListNearbyOrganisationsResponse> nearbyOrganisations({
    String channel = 'mdns',
  }) {
    return bridge.call(
      'ListNearbyOrganisations',
      pb.ListNearbyOrganisationsRequest(channel: channel),
      pb.ListNearbyOrganisationsResponse.create,
    );
  }

  Future<pb.StartDeviceInviteResponse> startDeviceInvite({
    required String organisationId,
    String joinMode = protosJoinModeNewDevice,
    String channel = 'mdns',
    String username = '',
  }) {
    return bridge.call(
      'StartDeviceInvite',
      pb.StartDeviceInviteRequest(
        organisationId: organisationId,
        channel: channel,
        joinMode: joinMode,
        username: username,
      ),
      pb.StartDeviceInviteResponse.create,
    );
  }

  Future<void> joinOrganisation({
    required String organisationId,
    required String peerId,
    required String username,
    required String name,
    required String verificationCode,
    required String joinMode,
    String inviteId = '',
    String channel = 'mdns',
  }) async {
    await bridge.call(
      'JoinOrganisation',
      pb.JoinOrganisationRequest(
        organisationId: organisationId,
        peerId: peerId,
        inviteId: inviteId,
        username: username,
        name: name,
        channel: channel,
        verificationCode: verificationCode,
        joinMode: joinMode,
      ),
      pb.JoinOrganisationResponse.create,
    );
  }

  Future<pb.GetUserDevicesResponse> userDevices() {
    return bridge.call(
      'GetUserDevices',
      pb.GetUserDevicesRequest(),
      pb.GetUserDevicesResponse.create,
    );
  }

  Future<pb.GetLocalSSHKeyResponse> localSshKey() {
    return bridge.call(
      'GetLocalSSHKey',
      pb.GetLocalSSHKeyRequest(),
      pb.GetLocalSSHKeyResponse.create,
    );
  }

  Future<pb.GetAppsResponse> apps() {
    return bridge.call(
      'GetApps',
      pb.GetAppsRequest(),
      pb.GetAppsResponse.create,
    );
  }

  Future<void> createApp({
    required String name,
    required String installerId,
    required String instanceId,
    required bool persistence,
  }) async {
    await bridge.call(
      'CreateApp',
      pb.CreateAppRequest(
        name: name,
        installerId: installerId,
        instanceId: instanceId,
        persistence: persistence,
      ),
      pb.CreateAppResponse.create,
    );
  }

  Future<void> startApp(String name) async {
    await bridge.call(
      'StartApp',
      pb.StartAppRequest(name: name),
      pb.StartAppResponse.create,
    );
  }

  Future<void> stopApp(String name) async {
    await bridge.call(
      'StopApp',
      pb.StopAppRequest(name: name),
      pb.StopAppResponse.create,
    );
  }

  Future<void> removeApp(String name) async {
    await bridge.call(
      'RemoveApp',
      pb.RemoveAppRequest(name: name),
      pb.RemoveAppResponse.create,
    );
  }

  Future<String> appLogs(String name) async {
    final response = await bridge.call(
      'GetAppLogs',
      pb.GetAppLogsRequest(name: name),
      pb.GetAppLogsResponse.create,
    );
    return utf8.decode(response.logs, allowMalformed: true);
  }

  Future<pb.GetSupportedProvisionersResponse> supportedProvisioners() {
    return bridge.call(
      'GetSupportedProvisioners',
      pb.GetSupportedProvisionersRequest(),
      pb.GetSupportedProvisionersResponse.create,
    );
  }

  Future<pb.GetProvisionersResponse> provisioners() {
    return bridge.call(
      'GetProvisioners',
      pb.GetProvisionersRequest(),
      pb.GetProvisionersResponse.create,
    );
  }

  Future<void> addProvisioner({
    required String name,
    required String type,
    required Map<String, String> credentials,
  }) async {
    await bridge.call(
      'AddProvisioner',
      pb.AddProvisionerRequest(
        name: name,
        type: type,
        credentials: credentials.entries,
      ),
      pb.AddProvisionerResponse.create,
    );
  }

  Future<void> removeProvisioner(String name) async {
    await bridge.call(
      'RemoveProvisioner',
      pb.RemoveProvisionerRequest(name: name),
      pb.RemoveProvisionerResponse.create,
    );
  }

  Future<pb.GetInstancesResponse> instances() {
    return bridge.call(
      'GetInstances',
      pb.GetInstancesRequest(),
      pb.GetInstancesResponse.create,
    );
  }

  Future<pb.GetInstanceResponse> instance(String name) {
    return bridge.call(
      'GetInstance',
      pb.GetInstanceRequest(name: name),
      pb.GetInstanceResponse.create,
    );
  }

  Future<pb.GetTasksResponse> tasks({
    String status = '',
    String stream = '',
    String subjectType = '',
    String subjectId = '',
    int maxResults = 200,
    String instance = '',
  }) {
    return bridge.call(
      'GetTasks',
      pb.GetTasksRequest(
        status: status,
        stream: stream,
        subjectType: subjectType,
        subjectId: subjectId,
        maxResults: maxResults,
        instance: instance,
      ),
      pb.GetTasksResponse.create,
    );
  }

  Future<pb.GetTaskResponse> task(
    String id, {
    bool includeEvents = true,
    String instance = '',
  }) {
    return bridge.call(
      'GetTask',
      pb.GetTaskRequest(
        id: id,
        includeEvents: includeEvents,
        instance: instance,
      ),
      pb.GetTaskResponse.create,
    );
  }

  Future<pb.GetInstanceDeployOptionsResponse> instanceDeployOptions({
    String provisioner = '',
    String location = '',
  }) {
    return bridge.call(
      'GetInstanceDeployOptions',
      pb.GetInstanceDeployOptionsRequest(
        provisioner: provisioner,
        location: location,
      ),
      pb.GetInstanceDeployOptionsResponse.create,
    );
  }

  Future<void> deployInstance({
    required String name,
    required String provisioner,
    required String location,
    required String machineType,
    required String version,
    required String devImage,
  }) async {
    await bridge.call(
      'DeployInstance',
      pb.DeployInstanceRequest(
        name: name,
        cloudName: provisioner,
        cloudLocation: location,
        machineType: machineType,
        protosVersion: version,
        devImg: devImage,
      ),
      pb.DeployInstanceResponse.create,
    );
  }

  Future<String> removeInstance(String name, {required bool localOnly}) async {
    final response = await bridge.call(
      'RemoveInstance',
      pb.RemoveInstanceRequest(name: name, localOnly: localOnly),
      pb.RemoveInstanceResponse.create,
    );
    return response.taskId;
  }

  Future<String> startInstance(String name) async {
    final response = await bridge.call(
      'StartInstance',
      pb.StartInstanceRequest(name: name),
      pb.StartInstanceResponse.create,
    );
    return response.taskId;
  }

  Future<String> stopInstance(String name) async {
    final response = await bridge.call(
      'StopInstance',
      pb.StopInstanceRequest(name: name),
      pb.StopInstanceResponse.create,
    );
    return response.taskId;
  }

  Future<String> instanceKey(String name) async {
    final response = await bridge.call(
      'GetInstanceKey',
      pb.GetInstanceKeyRequest(name: name),
      pb.GetInstanceKeyResponse.create,
    );
    return response.key;
  }

  Future<String> instanceLogs(String name) async {
    final response = await bridge.call(
      'GetInstanceLogs',
      pb.GetInstanceLogsRequest(name: name),
      pb.GetInstanceLogsResponse.create,
    );
    return response.logs;
  }

  Future<void> initInstance({required String name, required String ip}) async {
    await bridge.call(
      'InitInstance',
      pb.InitInstanceRequest(name: name, ip: ip),
      pb.InitInstanceResponse.create,
    );
  }

  Future<void> updateInstance({required String id, required String ip}) async {
    await bridge.call(
      'UpdateInstance',
      pb.UpdateInstanceRequest(id: id, ip: ip),
      pb.UpdateInstanceResponse.create,
    );
  }

  Future<pb.GetNetworkStateResponse> networkState({String instance = ''}) {
    return bridge.call(
      'GetNetworkState',
      pb.GetNetworkStateRequest(instance: instance),
      pb.GetNetworkStateResponse.create,
    );
  }

  Future<pb.GetExitRoutesResponse> exitRoutes({String instance = ''}) {
    return bridge.call(
      'GetExitRoutes',
      pb.GetExitRoutesRequest(instance: instance),
      pb.GetExitRoutesResponse.create,
    );
  }

  Future<pb.GetMobileTunnelConfigResponse> mobileTunnelConfig({
    String instance = '',
    String deviceId = '',
    String dnsServer = '',
    List<String> cidrs = const [],
  }) {
    return bridge.call(
      'GetMobileTunnelConfig',
      pb.GetMobileTunnelConfigRequest(
        instance: instance,
        deviceId: deviceId,
        dnsServer: dnsServer,
        cidrs: cidrs,
      ),
      pb.GetMobileTunnelConfigResponse.create,
    );
  }

  Future<pb.SetExitRouteResponse> setExitRoute({
    required String instance,
    String deviceId = '',
    String dnsServer = '',
    required List<String> cidrs,
  }) {
    return bridge.call(
      'SetExitRoute',
      pb.SetExitRouteRequest(
        instance: instance,
        deviceId: deviceId,
        dnsServer: dnsServer,
        cidrs: cidrs,
      ),
      pb.SetExitRouteResponse.create,
    );
  }

  Future<void> clearExitRoute({String deviceId = ''}) async {
    await bridge.call(
      'ClearExitRoute',
      pb.ClearExitRouteRequest(deviceId: deviceId),
      pb.ClearExitRouteResponse.create,
    );
  }

  Future<pb.GetProtosdReleasesResponse> releases() {
    return bridge.call(
      'GetProtosdReleases',
      pb.GetProtosdReleasesRequest(),
      pb.GetProtosdReleasesResponse.create,
    );
  }

  Future<pb.GetProvisionerImagesResponse> provisionerImages(
    String provisioner,
  ) {
    return bridge.call(
      'GetProvisionerImages',
      pb.GetProvisionerImagesRequest(name: provisioner),
      pb.GetProvisionerImagesResponse.create,
    );
  }

  Future<pb.UploadProvisionerImageResponse> uploadProvisionerImage({
    required String imagePath,
    required String imageName,
    required String provisioner,
    required String location,
    required int timeout,
  }) {
    return bridge.call(
      'UploadProvisionerImage',
      pb.UploadProvisionerImageRequest(
        imagePath: imagePath,
        imageName: imageName,
        provisionerName: provisioner,
        location: location,
        timeout: timeout,
      ),
      pb.UploadProvisionerImageResponse.create,
    );
  }

  Future<void> removeProvisionerImage({
    required String imageName,
    required String provisioner,
    required String location,
  }) async {
    await bridge.call(
      'RemoveProvisionerImage',
      pb.RemoveProvisionerImageRequest(
        imageName: imageName,
        provisionerName: provisioner,
        location: location,
      ),
      pb.RemoveProvisionerImageResponse.create,
    );
  }

  Future<pb.GetRuntimeStateResponse> runtimeState({
    String instance = '',
    bool allowStale = false,
  }) {
    return bridge.call(
      'GetRuntimeState',
      pb.GetRuntimeStateRequest(instance: instance, allowStale: allowStale),
      pb.GetRuntimeStateResponse.create,
    );
  }

  Future<pb.GetLocalCommitsResponse> localCommits() {
    return bridge.call(
      'GetLocalCommits',
      pb.GetLocalCommitsRequest(),
      pb.GetLocalCommitsResponse.create,
    );
  }

  Future<pb.ExecuteSqlResponse> executeSql({
    required String sql,
    int maxRows = 200,
  }) {
    return bridge.call(
      'ExecuteSql',
      pb.ExecuteSqlRequest(sql: sql, maxRows: maxRows),
      pb.ExecuteSqlResponse.create,
    );
  }

  Future<pb.GetSystemStatusResponse> systemStatus() {
    return bridge.call(
      'GetSystemStatus',
      pb.GetSystemStatusRequest(),
      pb.GetSystemStatusResponse.create,
    );
  }

  Future<pb.StartHostAgentResponse> startHostAgent() {
    return bridge.call(
      'StartHostAgent',
      pb.StartHostAgentRequest(),
      pb.StartHostAgentResponse.create,
    );
  }

  Future<pb.StopHostAgentResponse> stopHostAgent() {
    return bridge.call(
      'StopHostAgent',
      pb.StopHostAgentRequest(),
      pb.StopHostAgentResponse.create,
    );
  }

  Stream<pb.WatchChangesResponse> watchChanges({
    bool includeSnapshot = false,
    int heartbeatIntervalMs = 0,
  }) {
    return bridge.watchChanges(
      includeSnapshot: includeSnapshot,
      heartbeatIntervalMs: heartbeatIntervalMs,
    );
  }

  Stream<pb.WatchTaskResponse> watchTask({
    required String id,
    bool includeSnapshot = true,
    bool includeEvents = false,
    int heartbeatIntervalMs = 0,
  }) {
    return bridge.watchTask(
      id: id,
      includeSnapshot: includeSnapshot,
      includeEvents: includeEvents,
      heartbeatIntervalMs: heartbeatIntervalMs,
    );
  }
}
