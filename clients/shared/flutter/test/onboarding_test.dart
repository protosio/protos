import 'dart:async';

import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:protobuf/protobuf.dart';
import 'package:protos_flutter/src/app_model.dart';
import 'package:protos_flutter/src/generated/apic/proto/apic.pb.dart' as pb;
import 'package:protos_flutter/src/native/protos_bridge.dart';
import 'package:protos_flutter/src/ui/protos_app.dart';

void main() {
  testWidgets('create onboarding sends organisation name', (tester) async {
    final harness = await _pumpOnboarding(tester);

    await tester.enterText(find.widgetWithText(TextField, 'Username'), 'alex');
    await tester.enterText(find.widgetWithText(TextField, 'Name'), 'Alex');
    await tester.pump();
    final create = find.widgetWithText(FilledButton, 'Create');
    await tester.ensureVisible(create);
    expect(tester.widget<FilledButton>(create).onPressed, isNotNull);
    await tester.tap(create);
    await tester.pumpAndSettle();

    expect(harness.bridge.initRequest?.username, 'alex');
    expect(harness.bridge.initRequest?.name, 'Alex');
    expect(harness.bridge.initRequest?.organisation, 'home');
    expect(harness.model.needsInitialization, isFalse);
    expect(harness.model.userInfo?.organisationName, 'home');
  });

  testWidgets('join onboarding scans nearby organisations', (tester) async {
    final harness = await _pumpOnboarding(
      tester,
      nearbyOrganisations: [
        pb.NearbyOrganisation(
          organisationId: 'org-1',
          organisationName: 'Home Lab',
          deviceName: 'MacBook',
          peerId: 'peer-1',
          inviteId: 'invite-1',
          channel: 'mdns',
          joinMode: 'new_user',
        ),
      ],
    );

    await tester.tap(find.text('Join').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('New user'));
    await tester.pumpAndSettle();

    expect(harness.bridge.calls, contains('ListNearbyOrganisations'));
    expect(find.text('Home Lab'), findsOneWidget);
    expect(find.text('MacBook'), findsOneWidget);
    await tester.enterText(
      find.widgetWithText(TextField, 'New username'),
      'alex',
    );
    await tester.enterText(find.widgetWithText(TextField, 'Name'), 'Alex');
    await tester.enterText(
      find.widgetWithText(TextField, 'Invite code'),
      '12345678',
    );
    await tester.tap(find.text('Home Lab'));
    await tester.pump();
    await tester.tap(find.widgetWithText(FilledButton, 'Join'));
    await tester.pumpAndSettle();

    expect(harness.bridge.joinRequest?.organisationId, 'org-1');
    expect(harness.bridge.joinRequest?.peerId, 'peer-1');
    expect(harness.bridge.joinRequest?.inviteId, 'invite-1');
    expect(harness.bridge.joinRequest?.channel, 'mdns');
    expect(harness.bridge.joinRequest?.verificationCode, '12345678');
    expect(harness.bridge.joinRequest?.joinMode, 'new_user');
  });

  testWidgets('join onboarding can add a device for an existing user', (
    tester,
  ) async {
    final harness = await _pumpOnboarding(
      tester,
      nearbyOrganisations: [
        pb.NearbyOrganisation(
          organisationId: 'org-1',
          organisationName: 'Home Lab',
          deviceName: 'MacBook',
          peerId: 'peer-1',
          inviteId: 'invite-1',
          channel: 'mdns',
          joinMode: 'new_device',
        ),
      ],
    );

    await tester.tap(find.text('Join').first);
    await tester.pumpAndSettle();

    expect(find.text('Home Lab'), findsOneWidget);
    expect(find.widgetWithText(TextField, 'Name'), findsNothing);
    expect(find.widgetWithText(TextField, 'Existing username'), findsNothing);
    await tester.enterText(
      find.widgetWithText(TextField, 'Invite code'),
      '12345678',
    );
    await tester.tap(find.text('Home Lab'));
    await tester.pump();
    await tester.tap(find.widgetWithText(FilledButton, 'Join'));
    await tester.pumpAndSettle();

    expect(harness.bridge.joinRequest?.organisationId, 'org-1');
    expect(harness.bridge.joinRequest?.username, '');
    expect(harness.bridge.joinRequest?.name, '');
    expect(harness.bridge.joinRequest?.joinMode, 'new_device');
  });

  testWidgets('overview starts a device invite', (tester) async {
    final harness = await _pumpOverview(tester);

    final invite = find.widgetWithText(OutlinedButton, 'Invite device');
    await tester.ensureVisible(invite);
    await tester.tap(invite);
    await tester.pumpAndSettle();

    expect(harness.bridge.inviteRequest?.organisationId, 'org-1');
    expect(harness.bridge.inviteRequest?.channel, 'mdns');
    expect(harness.bridge.inviteRequest?.joinMode, 'new_device');
    expect(harness.model.deviceInvite?.inviteId, 'invite-1');
    expect(find.text('New device invite code 12345678'), findsOneWidget);
  });

  testWidgets('overview starts a user invite', (tester) async {
    final harness = await _pumpOverview(tester);

    final invite = find.widgetWithText(OutlinedButton, 'Invite user');
    await tester.ensureVisible(invite);
    await tester.tap(invite);
    await tester.pumpAndSettle();

    expect(harness.bridge.inviteRequest?.organisationId, 'org-1');
    expect(harness.bridge.inviteRequest?.joinMode, 'new_user');
    expect(find.text('New user invite code 12345678'), findsOneWidget);
  });
}

Future<_Harness> _pumpOnboarding(
  WidgetTester tester, {
  List<pb.NearbyOrganisation> nearbyOrganisations = const [],
}) async {
  tester.view.physicalSize = const Size(1000, 1000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  final bridge = _FakeNativeProtosBridge(
    nearbyOrganisations: nearbyOrganisations,
  );
  final model = AppModel(bridge: bridge)
    ..daemonState = DaemonState.running
    ..needsInitialization = true;
  addTearDown(model.dispose);

  await tester.pumpWidget(
    AppScope(
      notifier: model,
      child: MaterialApp(
        theme: ThemeData(useMaterial3: true),
        home: const Scaffold(body: InitializationView()),
      ),
    ),
  );
  await tester.pump();
  return _Harness(model: model, bridge: bridge);
}

Future<_Harness> _pumpOverview(WidgetTester tester) async {
  tester.view.physicalSize = const Size(1000, 1000);
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  final bridge = _FakeNativeProtosBridge();
  final model = AppModel(bridge: bridge)
    ..daemonState = DaemonState.running
    ..needsInitialization = false
    ..userInfo = pb.GetUserInfoResponse(
      username: 'alex',
      name: 'Alex',
      isAdmin: true,
      organisationId: 'org-1',
      organisationName: 'home',
    );
  addTearDown(model.dispose);

  await tester.pumpWidget(
    AppScope(
      notifier: model,
      child: MaterialApp(
        theme: ThemeData(useMaterial3: true),
        home: const Scaffold(body: OverviewView()),
      ),
    ),
  );
  await tester.pump();
  return _Harness(model: model, bridge: bridge);
}

class _Harness {
  const _Harness({required this.model, required this.bridge});

  final AppModel model;
  final _FakeNativeProtosBridge bridge;
}

class _FakeNativeProtosBridge implements NativeProtosBridge {
  _FakeNativeProtosBridge({this.nearbyOrganisations = const []});

  final List<pb.NearbyOrganisation> nearbyOrganisations;
  final calls = <String>[];
  pb.InitRequest? initRequest;
  pb.StartDeviceInviteRequest? inviteRequest;
  pb.JoinOrganisationRequest? joinRequest;
  var started = false;

  @override
  Future<void> start({
    ProtosBridgeConfig config = const ProtosBridgeConfig(),
  }) async {
    started = true;
  }

  @override
  Future<void> stop() async {
    started = false;
  }

  @override
  Future<Response> call<
    Request extends GeneratedMessage,
    Response extends GeneratedMessage
  >(String method, Request request, Response Function() createResponse) async {
    calls.add(method);
    switch (method) {
      case 'Init':
        initRequest = request as pb.InitRequest;
        return createResponse();
      case 'GetUserInfo':
        return _response(
          createResponse,
          pb.GetUserInfoResponse(
            username: initRequest?.username ?? 'alex',
            name: initRequest?.name ?? 'Alex',
            isAdmin: true,
            organisationId: 'org-1',
            organisationName: initRequest?.organisation ?? 'home',
          ),
        );
      case 'ListOrganisations':
        return _response(
          createResponse,
          pb.ListOrganisationsResponse(
            organisations: [
              pb.Organisation(
                id: 'org-1',
                name: initRequest?.organisation ?? 'home',
                createdAt: '2026-06-04T00:00:00Z',
              ),
            ],
          ),
        );
      case 'ListNearbyOrganisations':
        return _response(
          createResponse,
          pb.ListNearbyOrganisationsResponse(
            organisations: nearbyOrganisations,
          ),
        );
      case 'StartDeviceInvite':
        inviteRequest = request as pb.StartDeviceInviteRequest;
        return _response(
          createResponse,
          pb.StartDeviceInviteResponse(
            inviteId: 'invite-1',
            expiresAtUnix: Int64(1790000000),
            advertiseName: 'home',
            advertiseService: '_protos._tcp',
            channel: 'mdns',
            verificationCode: '12345678',
            joinMode: inviteRequest?.joinMode ?? 'new_device',
          ),
        );
      case 'JoinOrganisation':
        joinRequest = request as pb.JoinOrganisationRequest;
        return createResponse();
      case 'GetUserDevices':
        return _response(createResponse, pb.GetUserDevicesResponse());
      case 'GetLocalSSHKey':
        return _response(createResponse, pb.GetLocalSSHKeyResponse());
      case 'GetApps':
        return _response(createResponse, pb.GetAppsResponse());
      case 'GetSupportedProvisioners':
        return _response(createResponse, pb.GetSupportedProvisionersResponse());
      case 'GetProvisioners':
        return _response(createResponse, pb.GetProvisionersResponse());
      case 'GetInstances':
        return _response(createResponse, pb.GetInstancesResponse());
      case 'GetTasks':
        return _response(createResponse, pb.GetTasksResponse());
      case 'GetNetworkState':
        return _response(createResponse, pb.GetNetworkStateResponse());
      case 'GetExitRoutes':
        return _response(createResponse, pb.GetExitRoutesResponse());
      case 'GetRuntimeState':
        return _response(createResponse, pb.GetRuntimeStateResponse());
      case 'GetLocalCommits':
        return _response(createResponse, pb.GetLocalCommitsResponse());
      case 'GetSystemStatus':
        return _response(createResponse, pb.GetSystemStatusResponse());
      default:
        return createResponse();
    }
  }

  Response _response<Response extends GeneratedMessage>(
    Response Function() createResponse,
    GeneratedMessage source,
  ) {
    return createResponse()..mergeFromBuffer(source.writeToBuffer());
  }

  @override
  Stream<pb.WatchChangesResponse> watchChanges({
    bool includeSnapshot = false,
    int heartbeatIntervalMs = 0,
  }) {
    return const Stream.empty();
  }

  @override
  Stream<pb.WatchTaskResponse> watchTask({
    required String id,
    bool includeSnapshot = true,
    bool includeEvents = false,
    int heartbeatIntervalMs = 0,
  }) {
    return const Stream.empty();
  }

  @override
  void dispose() {}
}
