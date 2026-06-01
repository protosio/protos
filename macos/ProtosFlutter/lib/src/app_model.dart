import 'dart:async';

import 'package:flutter/material.dart';

import 'generated/apic/proto/apic.pb.dart' as pb;
import 'native/protos_bridge.dart';
import 'protos_api.dart';
import 'text_helpers.dart';

enum SidebarSection {
  overview('Overview', Icons.speed_outlined),
  apps('Apps', Icons.inventory_2_outlined),
  provisioners('Provisioners', Icons.dns_outlined),
  instances('Instances', Icons.computer_outlined),
  tasks('Tasks', Icons.task_alt_outlined),
  network('Network', Icons.hub_outlined),
  releases('Releases', Icons.arrow_circle_up_outlined),
  dvc('P2P DVC', Icons.account_tree_outlined),
  status('Status', Icons.monitor_heart_outlined);

  const SidebarSection(this.label, this.icon);

  final String label;
  final IconData icon;
}

enum DaemonStatus { starting, running, failed }

class DaemonState {
  const DaemonState._(this.status, [this.error]);

  static const starting = DaemonState._(DaemonStatus.starting);
  static const running = DaemonState._(DaemonStatus.running);

  factory DaemonState.failed(String error) {
    return DaemonState._(DaemonStatus.failed, error);
  }

  final DaemonStatus status;
  final String? error;

  String get title {
    return switch (status) {
      DaemonStatus.starting => 'Starting',
      DaemonStatus.running => 'Running',
      DaemonStatus.failed => 'Failed',
    };
  }

  bool get isRunning => status == DaemonStatus.running;
}

class AppModel extends ChangeNotifier {
  AppModel({NativeProtosBridge? bridge})
    : bridge = bridge ?? NativeProtosBridge() {
    api = ProtosApi(this.bridge);
  }

  final NativeProtosBridge bridge;
  late final ProtosApi api;

  DaemonState daemonState = DaemonState.starting;
  SidebarSection _selectedSection = SidebarSection.overview;
  String? message;
  var isBusy = false;
  var needsInitialization = false;

  pb.GetUserInfoResponse? userInfo;
  List<pb.UserDevice> devices = [];
  pb.GetLocalSSHKeyResponse? sshKey;
  List<pb.App> apps = [];
  List<pb.ProvisionerType> supportedProvisioners = [];
  List<pb.Provisioner> provisioners = [];
  List<pb.CloudInstance> instances = [];
  List<pb.Task> tasks = [];
  String selectedTaskId = '';
  List<pb.TaskEvent> selectedTaskEvents = [];
  pb.NetworkState? networkState;
  List<pb.ExitRoute> exitRoutes = [];
  List<pb.Release> releases = [];
  Map<String, pb.CloudSpecificImage> provisionerImages = {};
  List<MapEntry<String, pb.CloudSpecificImage>> sortedProvisionerImages = [];
  pb.RuntimeState? runtimeState;
  List<pb.RuntimePeerStatus> runtimePeers = [];
  List<pb.Commit> localCommits = [];
  pb.SystemStatus? systemStatus;
  pb.GetInstanceDeployOptionsResponse? instanceDeployOptions;
  String outputText = '';

  var _hasStarted = false;
  var _busyCount = 0;
  StreamSubscription<pb.WatchChangesResponse>? _changeWatchSubscription;
  Timer? _liveRefreshTimer;
  Timer? _initializationProbeTimer;
  final _pendingLiveTables = <String>{};
  var _pendingRuntimeChange = false;
  var _pendingHeartbeat = false;

  SidebarSection get selectedSection => _selectedSection;

  set selectedSection(SidebarSection section) {
    if (needsInitialization && section != SidebarSection.overview) {
      return;
    }
    if (_selectedSection == section) {
      return;
    }
    _selectedSection = section;
    notifyListeners();
    if (daemonState.isRunning) {
      switch (section) {
        case SidebarSection.dvc:
          unawaited(run(refreshDvc));
        case SidebarSection.status:
          unawaited(run(refreshStatus));
        case SidebarSection.tasks:
          unawaited(run(refreshTasks));
        case SidebarSection.overview:
        case SidebarSection.apps:
        case SidebarSection.provisioners:
        case SidebarSection.instances:
        case SidebarSection.network:
        case SidebarSection.releases:
          break;
      }
    }
  }

  Future<void> startIfNeeded() async {
    if (_hasStarted) {
      return;
    }
    _hasStarted = true;
    await run(
      () async {
        await api.start();
        daemonState = DaemonState.running;
        try {
          userInfo = await api.userInfo();
        } catch (_) {
          _enterInitializationMode();
          return;
        }
        needsInitialization = false;
        await refreshAll();
        _startChangeWatcher();
      },
      onError: (error) {
        daemonState = DaemonState.failed(error.toString());
      },
    );
  }

  Future<void> initializeUser({
    required String username,
    required String name,
    required String organization,
  }) async {
    await api.initUser(
      username: username,
      name: name,
      organization: organization,
    );
    await _leaveInitializationMode();
  }

  Future<void> checkInitialization() async {
    userInfo = await api.userInfo();
    await _leaveInitializationMode();
  }

  Future<void> refreshAll() async {
    final user = _optional(api.userInfo());
    final deviceList = _optional(api.userDevices());
    final key = _optional(api.localSshKey());
    final appList = _optional(api.apps());
    final provisionerTypeList = _optional(api.supportedProvisioners());
    final provisionerList = _optional(api.provisioners());
    final instanceList = _optional(api.instances());
    final taskList = _optional(api.tasks(maxResults: 200));
    final network = _optional(api.networkState());
    final routeList = _optional(api.exitRoutes());
    final runtime = _optional(api.runtimeState());
    final commitList = _optional(api.localCommits());
    final status = _optional(api.systemStatus());

    userInfo = await user;
    devices = (await deviceList)?.devices.toList(growable: false) ?? [];
    sshKey = await key;
    apps = (await appList)?.apps.toList(growable: false) ?? [];
    supportedProvisioners =
        (await provisionerTypeList)?.provisionerTypes.toList(growable: false) ??
        [];
    provisioners =
        (await provisionerList)?.provisioners.toList(growable: false) ?? [];
    instances = (await instanceList)?.instances.toList(growable: false) ?? [];
    tasks = (await taskList)?.tasks.toList(growable: false) ?? [];
    _pruneSelectedTask();
    final networkResponse = await network;
    networkState = networkResponse?.hasState() == true
        ? networkResponse!.state
        : null;
    exitRoutes = (await routeList)?.routes.toList(growable: false) ?? [];
    final runtimeResponse = await runtime;
    _setRuntimeState(
      runtimeResponse?.hasState() == true ? runtimeResponse!.state : null,
    );
    localCommits = (await commitList)?.commits.toList(growable: false) ?? [];
    final statusResponse = await status;
    systemStatus = statusResponse?.hasStatus() == true
        ? statusResponse!.status
        : null;
    notifyListeners();
  }

  Future<void> refreshNetwork({String instance = ''}) async {
    final networkResponse = await api.networkState(instance: instance);
    networkState = networkResponse.hasState() ? networkResponse.state : null;
    final routeResponse = await api.exitRoutes(instance: instance);
    exitRoutes = routeResponse.routes.toList(growable: false);
    notifyListeners();
  }

  Future<void> refreshInstances() async {
    instances = (await api.instances()).instances.toList(growable: false);
    notifyListeners();
  }

  Future<void> refreshTasks() async {
    final response = await api.tasks(maxResults: 200);
    tasks = response.tasks.toList(growable: false);
    _pruneSelectedTask();
    if (selectedTaskId.nonEmpty != null) {
      await refreshSelectedTaskDetail(notify: false);
    }
    notifyListeners();
  }

  Future<void> selectTask(String id) async {
    selectedTaskId = id.trim();
    selectedTaskEvents = [];
    notifyListeners();
    if (selectedTaskId.nonEmpty == null) {
      return;
    }
    await refreshSelectedTaskDetail();
  }

  Future<void> refreshSelectedTaskDetail({bool notify = true}) async {
    final id = selectedTaskId.nonEmpty;
    if (id == null) {
      selectedTaskEvents = [];
      if (notify) {
        notifyListeners();
      }
      return;
    }
    final response = await api.task(id, includeEvents: true);
    if (response.hasTask()) {
      _upsertTask(response.task);
    }
    selectedTaskEvents = response.events.toList(growable: false);
    if (notify) {
      notifyListeners();
    }
  }

  pb.Task? get selectedTask {
    final id = selectedTaskId.nonEmpty;
    if (id == null) {
      return null;
    }
    for (final task in tasks) {
      if (task.id == id) {
        return task;
      }
    }
    return null;
  }

  Future<void> refreshInstanceDeployOptions({
    String provisioner = '',
    String location = '',
  }) async {
    instanceDeployOptions = await api.instanceDeployOptions(
      provisioner: provisioner,
      location: location,
    );
    notifyListeners();
  }

  Future<void> refreshDvc({String instance = ''}) async {
    final runtimeResponse = await api.runtimeState(instance: instance);
    _setRuntimeState(runtimeResponse.hasState() ? runtimeResponse.state : null);
    final commitResponse = await api.localCommits();
    localCommits = commitResponse.commits.toList(growable: false);
    notifyListeners();
  }

  Future<void> refreshStatus() async {
    final response = await api.systemStatus();
    systemStatus = response.hasStatus() ? response.status : null;
    notifyListeners();
  }

  void setProvisionerImages(Map<String, pb.CloudSpecificImage> images) {
    provisionerImages = images;
    sortedProvisionerImages = images.entries.toList(growable: false)
      ..sort((a, b) => a.key.compareTo(b.key));
    notifyListeners();
  }

  Future<void> run(
    Future<void> Function() operation, {
    void Function(Object error)? onError,
  }) async {
    _setBusy(true);
    message = null;
    notifyListeners();
    try {
      await operation();
    } catch (error) {
      message = error.toString();
      onError?.call(error);
    } finally {
      _setBusy(false);
    }
  }

  void clearMessage() {
    message = null;
    notifyListeners();
  }

  @override
  void dispose() {
    _changeWatchSubscription?.cancel();
    _liveRefreshTimer?.cancel();
    _initializationProbeTimer?.cancel();
    unawaited(api.stop().whenComplete(bridge.dispose));
    super.dispose();
  }

  void _setRuntimeState(pb.RuntimeState? state) {
    runtimeState = state;
    runtimePeers = _sortedRuntimePeers(state?.peerStatuses ?? const []);
  }

  void _setBusy(bool busy) {
    _busyCount += busy ? 1 : -1;
    if (_busyCount < 0) {
      _busyCount = 0;
    }
    final next = _busyCount > 0;
    if (isBusy == next) {
      return;
    }
    isBusy = next;
    notifyListeners();
  }

  void _startChangeWatcher() {
    if (needsInitialization) {
      return;
    }
    if (_changeWatchSubscription != null) {
      return;
    }
    _changeWatchSubscription = api
        .watchChanges(heartbeatIntervalMs: 3000)
        .listen(
          _scheduleLiveRefresh,
          onError: (Object error) {
            message = 'Live refresh stopped: $error';
            _changeWatchSubscription = null;
            notifyListeners();
          },
          onDone: () {
            _changeWatchSubscription = null;
            notifyListeners();
          },
        );
  }

  void _scheduleLiveRefresh(pb.WatchChangesResponse change) {
    if (needsInitialization) {
      return;
    }
    if (change.reason == 'heartbeat' &&
        selectedSection != SidebarSection.dvc &&
        selectedSection != SidebarSection.status &&
        selectedSection != SidebarSection.tasks) {
      return;
    }
    _pendingLiveTables.addAll(change.tableNames);
    _pendingRuntimeChange = _pendingRuntimeChange || change.runtimeChanged;
    _pendingHeartbeat = _pendingHeartbeat || change.reason == 'heartbeat';
    _liveRefreshTimer ??= Timer(
      const Duration(milliseconds: 250),
      () => unawaited(_performPendingLiveRefresh()),
    );
  }

  Future<void> _performPendingLiveRefresh() async {
    if (needsInitialization) {
      return;
    }
    final tables = Set<String>.from(_pendingLiveTables);
    final runtimeChanged = _pendingRuntimeChange;
    final heartbeat = _pendingHeartbeat;
    _pendingLiveTables.clear();
    _pendingRuntimeChange = false;
    _pendingHeartbeat = false;
    _liveRefreshTimer = null;

    try {
      await _refreshVisibleSection(
        tables: tables,
        runtimeChanged: runtimeChanged,
        heartbeat: heartbeat,
      );
    } catch (error) {
      message = error.toString();
      notifyListeners();
    }
  }

  Future<void> _refreshVisibleSection({
    required Set<String> tables,
    required bool runtimeChanged,
    required bool heartbeat,
  }) async {
    if (needsInitialization) {
      return;
    }
    switch (selectedSection) {
      case SidebarSection.overview:
        if (heartbeat || (runtimeChanged && tables.isEmpty)) {
          return;
        }
        final user = _optional(api.userInfo());
        final deviceList = _optional(api.userDevices());
        final key = _optional(api.localSshKey());
        userInfo = await user;
        devices = (await deviceList)?.devices.toList(growable: false) ?? [];
        sshKey = await key;
      case SidebarSection.apps:
        if (heartbeat || (runtimeChanged && tables.isEmpty)) {
          return;
        }
        apps = (await api.apps()).apps.toList(growable: false);
      case SidebarSection.provisioners:
        if (heartbeat || (runtimeChanged && tables.isEmpty)) {
          return;
        }
        final supported = _optional(api.supportedProvisioners());
        final existing = _optional(api.provisioners());
        supportedProvisioners =
            (await supported)?.provisionerTypes.toList(growable: false) ?? [];
        provisioners =
            (await existing)?.provisioners.toList(growable: false) ?? [];
      case SidebarSection.instances:
        if (heartbeat || (runtimeChanged && tables.isEmpty)) {
          return;
        }
        await refreshInstances();
      case SidebarSection.tasks:
        if (heartbeat ||
            runtimeChanged ||
            tables.contains('tasks') ||
            tables.contains('task_events')) {
          await refreshTasks();
        }
      case SidebarSection.network:
        if (heartbeat) {
          return;
        }
        await refreshNetwork();
      case SidebarSection.releases:
        return;
      case SidebarSection.dvc:
        if (heartbeat || runtimeChanged || tables.isNotEmpty) {
          await refreshDvc();
        }
      case SidebarSection.status:
        if (heartbeat || runtimeChanged || tables.isNotEmpty) {
          await refreshStatus();
        }
    }
    notifyListeners();
  }

  void _enterInitializationMode() {
    needsInitialization = true;
    _selectedSection = SidebarSection.overview;
    message = null;
    _clearLoadedData();
    _stopChangeWatcher();
    _startInitializationProbe();
    notifyListeners();
  }

  Future<void> _leaveInitializationMode() async {
    needsInitialization = false;
    _selectedSection = SidebarSection.overview;
    _initializationProbeTimer?.cancel();
    _initializationProbeTimer = null;
    await refreshAll();
    _startChangeWatcher();
    notifyListeners();
  }

  void _clearLoadedData() {
    userInfo = null;
    devices = [];
    sshKey = null;
    apps = [];
    supportedProvisioners = [];
    provisioners = [];
    instances = [];
    tasks = [];
    selectedTaskId = '';
    selectedTaskEvents = [];
    networkState = null;
    exitRoutes = [];
    releases = [];
    provisionerImages = {};
    sortedProvisionerImages = [];
    runtimeState = null;
    runtimePeers = [];
    localCommits = [];
    systemStatus = null;
    instanceDeployOptions = null;
    outputText = '';
  }

  void _stopChangeWatcher() {
    unawaited(_changeWatchSubscription?.cancel());
    _changeWatchSubscription = null;
    _liveRefreshTimer?.cancel();
    _liveRefreshTimer = null;
    _pendingLiveTables.clear();
    _pendingRuntimeChange = false;
    _pendingHeartbeat = false;
  }

  void _startInitializationProbe() {
    _initializationProbeTimer ??= Timer.periodic(
      const Duration(seconds: 3),
      (_) => unawaited(_checkExternalInitialization()),
    );
  }

  Future<void> _checkExternalInitialization() async {
    if (!needsInitialization || isBusy || !daemonState.isRunning) {
      return;
    }
    try {
      userInfo = await api.userInfo();
      await _leaveInitializationMode();
    } catch (_) {
      // Stay in the guided setup flow until initialization completes.
    }
  }

  void _pruneSelectedTask() {
    final id = selectedTaskId.nonEmpty;
    if (id == null) {
      selectedTaskEvents = [];
      return;
    }
    if (tasks.any((task) => task.id == id)) {
      return;
    }
    selectedTaskId = '';
    selectedTaskEvents = [];
  }

  void _upsertTask(pb.Task task) {
    final index = tasks.indexWhere((existing) => existing.id == task.id);
    if (index < 0) {
      tasks = [task, ...tasks];
      return;
    }
    final next = tasks.toList(growable: true);
    next[index] = task;
    tasks = next;
  }
}

Future<T?> _optional<T>(Future<T> future) async {
  try {
    return await future;
  } catch (_) {
    return null;
  }
}

List<pb.RuntimePeerStatus> _sortedRuntimePeers(
  Iterable<pb.RuntimePeerStatus> peers,
) {
  final sorted = peers.toList(growable: false);
  sorted.sort((a, b) {
    if (a.connected != b.connected) {
      return a.connected ? -1 : 1;
    }
    return a.peerId.compareTo(b.peerId);
  });
  return sorted;
}
