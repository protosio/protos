import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/foundation.dart'
    show TargetPlatform, defaultTargetPlatform;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../app_model.dart';
import '../generated/apic/proto/apic.pb.dart' as pb;
import '../join_modes.dart';
import '../native/protos_bridge.dart';
import '../text_helpers.dart';

class ProtosFlutterApp extends StatefulWidget {
  const ProtosFlutterApp({
    this.bridgeConfig = const ProtosBridgeConfig(),
    this.onDaemonStartFailed,
    super.key,
  });

  final ProtosBridgeConfig bridgeConfig;
  final void Function(Object error)? onDaemonStartFailed;

  @override
  State<ProtosFlutterApp> createState() => _ProtosFlutterAppState();
}

class _ProtosFlutterAppState extends State<ProtosFlutterApp> {
  late final AppModel model = AppModel(
    bridgeConfig: widget.bridgeConfig,
    onDaemonStartFailed: widget.onDaemonStartFailed,
  );

  @override
  void dispose() {
    model.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final lightScheme = ColorScheme.fromSeed(
      seedColor: const Color(0xff0f766e),
    );
    final darkScheme = ColorScheme.fromSeed(
      seedColor: const Color(0xff14b8a6),
      brightness: Brightness.dark,
    );

    return AppScope(
      notifier: model,
      child: MaterialApp(
        debugShowCheckedModeBanner: false,
        title: 'Protos',
        theme: ThemeData(
          colorScheme: lightScheme,
          useMaterial3: true,
          visualDensity: VisualDensity.compact,
          inputDecorationTheme: _inputTheme(lightScheme),
        ),
        darkTheme: ThemeData(
          colorScheme: darkScheme,
          useMaterial3: true,
          visualDensity: VisualDensity.compact,
          inputDecorationTheme: _inputTheme(darkScheme),
        ),
        home: const ProtosShell(),
      ),
    );
  }
}

InputDecorationTheme _inputTheme(ColorScheme scheme) {
  return InputDecorationTheme(
    border: const OutlineInputBorder(),
    isDense: true,
    filled: true,
    fillColor: scheme.surfaceContainerLowest,
  );
}

class AppScope extends InheritedNotifier<AppModel> {
  const AppScope({required super.notifier, required super.child, super.key});

  static AppModel of(BuildContext context) {
    final scope = context.dependOnInheritedWidgetOfExactType<AppScope>();
    assert(scope != null, 'No AppScope found in context');
    return scope!.notifier!;
  }
}

class ProtosShell extends StatefulWidget {
  const ProtosShell({super.key});

  @override
  State<ProtosShell> createState() => _ProtosShellState();
}

class _ProtosShellState extends State<ProtosShell> {
  var _started = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    if (_started) {
      return;
    }
    _started = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      unawaited(AppScope.of(context).startIfNeeded());
    });
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final scheme = Theme.of(context).colorScheme;
    final compact = MediaQuery.sizeOf(context).width < 720;

    if (model.needsInitialization) {
      return Scaffold(
        backgroundColor: scheme.surface,
        body: Column(
          children: [
            const HeaderBar(),
            Divider(height: 1, color: scheme.outlineVariant),
            const Expanded(child: InitializationView()),
            if (model.message != null) MessageBar(message: model.message!),
          ],
        ),
      );
    }

    if (compact) {
      return Scaffold(
        backgroundColor: scheme.surface,
        drawer: const Drawer(
          child: Sidebar(width: double.infinity, closeOnSelect: true),
        ),
        body: Column(
          children: [
            const HeaderBar(showMenuButton: true),
            Divider(height: 1, color: scheme.outlineVariant),
            Expanded(child: SectionBody(section: model.selectedSection)),
            if (model.message != null) MessageBar(message: model.message!),
          ],
        ),
      );
    }

    return Scaffold(
      backgroundColor: scheme.surface,
      body: Row(
        children: [
          const Sidebar(),
          VerticalDivider(width: 1, color: scheme.outlineVariant),
          Expanded(
            child: Column(
              children: [
                const HeaderBar(),
                Divider(height: 1, color: scheme.outlineVariant),
                Expanded(child: SectionBody(section: model.selectedSection)),
                if (model.message != null) MessageBar(message: model.message!),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class Sidebar extends StatelessWidget {
  const Sidebar({this.width = 232, this.closeOnSelect = false, super.key});

  final double width;
  final bool closeOnSelect;

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final scheme = Theme.of(context).colorScheme;

    return Container(
      width: width,
      color: scheme.surfaceContainer,
      child: SafeArea(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(18, 18, 18, 14),
              child: Row(
                children: [
                  Icon(Icons.hexagon_outlined, color: scheme.primary),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      'Protos',
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                  ),
                  Tooltip(
                    message: 'Refresh',
                    child: SizedBox.square(
                      dimension: 32,
                      child: IconButton(
                        padding: EdgeInsets.zero,
                        visualDensity: VisualDensity.compact,
                        onPressed: model.isBusy
                            ? null
                            : () => unawaited(
                                model.run(
                                  () => model.refreshAll(notify: false),
                                ),
                              ),
                        icon: const Icon(Icons.refresh, size: 18),
                      ),
                    ),
                  ),
                ],
              ),
            ),
            Expanded(
              child: ListView(
                primary: false,
                padding: const EdgeInsets.symmetric(horizontal: 10),
                children: [
                  for (final section in SidebarSection.values)
                    if (section != SidebarSection.network ||
                        model.supportsNetwork)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 2),
                        child: _SidebarButton(
                          section: section,
                          selected: model.selectedSection == section,
                          closeOnSelect: closeOnSelect,
                        ),
                      ),
                ],
              ),
            ),
            const SidebarConnectionStatus(),
          ],
        ),
      ),
    );
  }
}

class SidebarConnectionStatus extends StatelessWidget {
  const SidebarConnectionStatus({super.key});

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final scheme = Theme.of(context).colorScheme;
    final (color, label) = switch (model.daemonState.status) {
      DaemonStatus.running => (Colors.green, 'Connected'),
      DaemonStatus.starting => (scheme.error, 'Connecting'),
      DaemonStatus.failed => (scheme.error, 'Disconnected'),
    };

    return Padding(
      padding: const EdgeInsets.fromLTRB(18, 8, 18, 18),
      child: Row(
        children: [
          Container(
            width: 10,
            height: 10,
            decoration: BoxDecoration(color: color, shape: BoxShape.circle),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              label,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
              style: Theme.of(context).textTheme.labelLarge?.copyWith(
                color: scheme.onSurfaceVariant,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _SidebarButton extends StatelessWidget {
  const _SidebarButton({
    required this.section,
    required this.selected,
    required this.closeOnSelect,
  });

  final SidebarSection section;
  final bool selected;
  final bool closeOnSelect;

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final scheme = Theme.of(context).colorScheme;

    return Tooltip(
      message: section.label,
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: () {
          model.selectedSection = section;
          if (closeOnSelect) {
            Scaffold.of(context).closeDrawer();
          }
        },
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 140),
          height: 42,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: selected ? scheme.primaryContainer : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Icon(
                section.icon,
                size: 20,
                color: selected ? scheme.onPrimaryContainer : scheme.onSurface,
              ),
              const SizedBox(width: 10),
              Expanded(
                child: Text(
                  section.label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    fontWeight: selected ? FontWeight.w700 : FontWeight.w500,
                    color: selected
                        ? scheme.onPrimaryContainer
                        : scheme.onSurface,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class HeaderBar extends StatelessWidget {
  const HeaderBar({this.showMenuButton = false, super.key});

  final bool showMenuButton;

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final title = model.needsInitialization
        ? 'Set up Protos'
        : model.selectedSection.label;

    return SafeArea(
      bottom: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(24, 14, 18, 14),
        child: Row(
          children: [
            if (showMenuButton) ...[
              Tooltip(
                message: 'Menu',
                child: IconButton(
                  onPressed: () => Scaffold.of(context).openDrawer(),
                  icon: const Icon(Icons.menu),
                ),
              ),
              const SizedBox(width: 8),
            ],
            Expanded(
              child: Text(
                title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(
                  context,
                ).textTheme.titleLarge?.copyWith(fontWeight: FontWeight.w700),
              ),
            ),
            if (model.isBusy) ...[
              const SizedBox.square(
                dimension: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
              const SizedBox(width: 12),
            ],
          ],
        ),
      ),
    );
  }
}

class MessageBar extends StatelessWidget {
  const MessageBar({required this.message, super.key});

  final String message;

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final scheme = Theme.of(context).colorScheme;

    return Material(
      color: scheme.errorContainer.withValues(alpha: 0.62),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
        child: Row(
          children: [
            Icon(Icons.warning_amber_rounded, color: scheme.onErrorContainer),
            const SizedBox(width: 10),
            Expanded(
              child: SelectableText(
                message,
                maxLines: 4,
                style: TextStyle(color: scheme.onErrorContainer),
              ),
            ),
            Tooltip(
              message: 'Copy',
              child: IconButton(
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: message));
                  ScaffoldMessenger.of(
                    context,
                  ).showSnackBar(const SnackBar(content: Text('Copied')));
                },
                icon: const Icon(Icons.copy_rounded),
              ),
            ),
            Tooltip(
              message: 'Dismiss',
              child: IconButton(
                onPressed: model.clearMessage,
                icon: const Icon(Icons.close),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

enum _InitializationMode { create, join }

enum _JoinIntent { newDevice, newUser }

class InitializationView extends StatefulWidget {
  const InitializationView({super.key});

  @override
  State<InitializationView> createState() => _InitializationViewState();
}

class _InitializationViewState extends State<InitializationView> {
  final username = TextEditingController();
  final name = TextEditingController();
  final organisation = TextEditingController(text: 'home');
  final verificationCode = TextEditingController();
  var mode = _InitializationMode.create;
  var joinIntent = _JoinIntent.newDevice;
  var selectedNearbyId = '';

  @override
  void initState() {
    super.initState();
    username.addListener(_refreshControls);
    name.addListener(_refreshControls);
    organisation.addListener(_refreshControls);
    verificationCode.addListener(_refreshControls);
  }

  void _refreshControls() => setState(() {});

  @override
  void dispose() {
    username.dispose();
    name.dispose();
    organisation.dispose();
    verificationCode.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final scheme = Theme.of(context).colorScheme;
    final requestedJoinMode = _joinModeForIntent(joinIntent);
    final nearbyRows = mode == _InitializationMode.join
        ? model.nearbyOrganisations
              .where(
                (organisation) => protosJoinModeMatches(
                  inviteMode: organisation.joinMode,
                  requestedMode: requestedJoinMode,
                ),
              )
              .toList(growable: false)
        : const <pb.NearbyOrganisation>[];
    final selectedNearby = _selectedNearby(nearbyRows);
    final requiresUsername =
        mode == _InitializationMode.create ||
        (mode == _InitializationMode.join && joinIntent == _JoinIntent.newUser);
    final requiresName =
        mode == _InitializationMode.create ||
        (mode == _InitializationMode.join && joinIntent == _JoinIntent.newUser);
    final canCreate =
        username.text.nonEmpty != null &&
        name.text.nonEmpty != null &&
        organisation.text.nonEmpty != null &&
        model.daemonState.isRunning &&
        !model.isBusy;
    final canJoin =
        (!requiresUsername || username.text.nonEmpty != null) &&
        (!requiresName || name.text.nonEmpty != null) &&
        verificationCode.text.nonEmpty != null &&
        selectedNearby != null &&
        model.daemonState.isRunning &&
        !model.isBusy;

    return PlatformSelectionArea(
      child: Align(
        alignment: Alignment.topCenter,
        child: SingleChildScrollView(
          primary: false,
          padding: const EdgeInsets.fromLTRB(24, 42, 24, 24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 520),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Icon(
                  Icons.manage_accounts_outlined,
                  size: 42,
                  color: scheme.primary,
                ),
                const SizedBox(height: 18),
                Text(
                  'Set up Protos',
                  style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Create an organisation here or join one nearby.',
                  style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    color: scheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 18),
                SegmentedButton<_InitializationMode>(
                  segments: const [
                    ButtonSegment(
                      value: _InitializationMode.create,
                      label: Text('Create'),
                      icon: Icon(Icons.add_circle_outline),
                    ),
                    ButtonSegment(
                      value: _InitializationMode.join,
                      label: Text('Join'),
                      icon: Icon(Icons.wifi_find),
                    ),
                  ],
                  selected: {mode},
                  onSelectionChanged: model.isBusy
                      ? null
                      : (selection) {
                          setState(() {
                            mode = selection.first;
                            selectedNearbyId = '';
                          });
                          if (mode == _InitializationMode.join) {
                            unawaited(
                              model.run(() => model.scanNearbyOrganisations()),
                            );
                          }
                        },
                ),
                const SizedBox(height: 18),
                if (requiresUsername) ...[
                  TextField(
                    controller: username,
                    textInputAction: TextInputAction.next,
                    decoration: textField(
                      _usernameFieldLabel(mode, joinIntent),
                    ),
                  ),
                  const SizedBox(height: 12),
                ],
                if (requiresName) ...[
                  TextField(
                    controller: name,
                    textInputAction: mode == _InitializationMode.create
                        ? TextInputAction.next
                        : TextInputAction.done,
                    decoration: textField('Name'),
                    onSubmitted: (_) {
                      if (mode == _InitializationMode.join && canJoin) {
                        _join(model, selectedNearby, joinIntent);
                      }
                    },
                  ),
                  const SizedBox(height: 12),
                ],
                if (mode == _InitializationMode.create)
                  _CreateOrganisationControls(
                    organisation: organisation,
                    canCreate: canCreate,
                    onSubmit: () => _create(model),
                  )
                else
                  _JoinOrganisationControls(
                    rows: nearbyRows,
                    selectedId: selectedNearbyId,
                    joinIntent: joinIntent,
                    canScan: model.daemonState.isRunning && !model.isBusy,
                    canJoin: canJoin,
                    verificationCode: verificationCode,
                    onSelect: (id) => setState(() => selectedNearbyId = id),
                    onJoinIntentChanged: (intent) => setState(() {
                      joinIntent = intent;
                      selectedNearbyId = '';
                    }),
                    onScan: () => unawaited(
                      model.run(() => model.scanNearbyOrganisations()),
                    ),
                    onJoin: selectedNearby == null
                        ? null
                        : () => _join(model, selectedNearby, joinIntent),
                  ),
                const SizedBox(height: 8),
                Align(
                  alignment: Alignment.centerLeft,
                  child: TextButton.icon(
                    onPressed: model.daemonState.isRunning && !model.isBusy
                        ? () => unawaited(model.run(model.checkInitialization))
                        : null,
                    icon: const Icon(Icons.sync),
                    label: const Text('Check again'),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  pb.NearbyOrganisation? _selectedNearby(List<pb.NearbyOrganisation> rows) {
    for (final organisation in rows) {
      if (_nearbyRowId(organisation) == selectedNearbyId) {
        return organisation;
      }
    }
    return null;
  }

  void _create(AppModel model) {
    unawaited(
      model.run(
        () => model.initializeUser(
          username: username.text.trim(),
          name: name.text.trim(),
          organisation: organisation.text.trim(),
        ),
      ),
    );
  }

  void _join(
    AppModel model,
    pb.NearbyOrganisation organisation,
    _JoinIntent intent,
  ) {
    final joinMode = _joinModeForIntent(intent);
    unawaited(
      model.run(
        () => model.joinOrganisation(
          organisationId: organisation.organisationId,
          peerId: organisation.peerId,
          inviteId: organisation.inviteId,
          channel: organisation.channel.nonEmpty ?? 'mdns',
          username: intent == _JoinIntent.newUser ? username.text.trim() : '',
          name: intent == _JoinIntent.newUser ? name.text.trim() : '',
          verificationCode: verificationCode.text.trim(),
          joinMode: joinMode,
        ),
      ),
    );
  }
}

String _joinModeForIntent(_JoinIntent intent) {
  return switch (intent) {
    _JoinIntent.newDevice => protosJoinModeNewDevice,
    _JoinIntent.newUser => protosJoinModeNewUser,
  };
}

String _usernameFieldLabel(_InitializationMode mode, _JoinIntent joinIntent) {
  if (mode != _InitializationMode.join) {
    return 'Username';
  }
  return switch (joinIntent) {
    _JoinIntent.newDevice => 'Existing username',
    _JoinIntent.newUser => 'New username',
  };
}

class _CreateOrganisationControls extends StatelessWidget {
  const _CreateOrganisationControls({
    required this.organisation,
    required this.canCreate,
    required this.onSubmit,
  });

  final TextEditingController organisation;
  final bool canCreate;
  final VoidCallback onSubmit;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: organisation,
          textInputAction: TextInputAction.done,
          decoration: textField('Organisation'),
          onSubmitted: (_) {
            if (canCreate) {
              onSubmit();
            }
          },
        ),
        const SizedBox(height: 18),
        FilledButton.icon(
          onPressed: canCreate ? onSubmit : null,
          icon: const Icon(Icons.add_circle_outline),
          label: const Text('Create'),
        ),
      ],
    );
  }
}

class _JoinOrganisationControls extends StatelessWidget {
  const _JoinOrganisationControls({
    required this.rows,
    required this.selectedId,
    required this.joinIntent,
    required this.canScan,
    required this.canJoin,
    required this.verificationCode,
    required this.onSelect,
    required this.onJoinIntentChanged,
    required this.onScan,
    required this.onJoin,
  });

  final List<pb.NearbyOrganisation> rows;
  final String selectedId;
  final _JoinIntent joinIntent;
  final bool canScan;
  final bool canJoin;
  final TextEditingController verificationCode;
  final ValueChanged<String> onSelect;
  final ValueChanged<_JoinIntent> onJoinIntentChanged;
  final VoidCallback onScan;
  final VoidCallback? onJoin;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SegmentedButton<_JoinIntent>(
          segments: const [
            ButtonSegment(
              value: _JoinIntent.newDevice,
              label: Text('New device'),
              icon: Icon(Icons.devices_other_outlined),
            ),
            ButtonSegment(
              value: _JoinIntent.newUser,
              label: Text('New user'),
              icon: Icon(Icons.person_add_alt_1_outlined),
            ),
          ],
          selected: {joinIntent},
          onSelectionChanged: (selection) {
            onJoinIntentChanged(selection.first);
          },
        ),
        const SizedBox(height: 12),
        RowsPanel<pb.NearbyOrganisation>(
          rows: rows,
          emptyTitle: 'No matching invites',
          selectedId: selectedId,
          onSelect: onSelect,
          idForRow: _nearbyRowId,
          height: 170,
          columns: [
            RowColumn('Organisation', (row) => row.organisationName),
            RowColumn('Device', (row) => row.deviceName),
            RowColumn('Type', (row) => protosJoinModeLabel(row.joinMode)),
            RowColumn('Channel', (row) => row.channel.nonEmpty ?? 'mdns'),
            RowColumn('Peer', (row) => row.peerId, flex: 2),
          ],
        ),
        const SizedBox(height: 18),
        TextField(
          controller: verificationCode,
          keyboardType: TextInputType.number,
          textInputAction: TextInputAction.done,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: textField('Invite code'),
          onSubmitted: (_) {
            if (canJoin) {
              onJoin?.call();
            }
          },
        ),
        const SizedBox(height: 18),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            OutlinedButton.icon(
              onPressed: canScan ? onScan : null,
              icon: const Icon(Icons.wifi_find),
              label: const Text('Scan'),
            ),
            FilledButton.icon(
              onPressed: canJoin && onJoin != null ? onJoin : null,
              icon: const Icon(Icons.login),
              label: const Text('Join'),
            ),
          ],
        ),
      ],
    );
  }
}

String _nearbyRowId(pb.NearbyOrganisation organisation) {
  final channel = organisation.channel.nonEmpty ?? 'mdns';
  final organisationID = organisation.organisationId.nonEmpty ?? 'unknown';
  final peerID = organisation.peerId.nonEmpty ?? 'peer';
  final inviteID = organisation.inviteId.nonEmpty ?? 'invite';
  return '$channel/$organisationID/$peerID/$inviteID';
}

String _inviteActiveText(pb.StartDeviceInviteResponse invite) {
  final code = invite.verificationCode.nonEmpty;
  final mode = protosJoinModeLabel(_inviteMode(invite));
  if (code == null) {
    return '$mode invite active';
  }
  return '$mode invite code $code';
}

String _inviteMode(pb.StartDeviceInviteResponse invite) {
  final mode = normalizeProtosJoinMode(invite.joinMode);
  return mode.isEmpty ? protosJoinModeNewDevice : mode;
}

class SectionBody extends StatelessWidget {
  const SectionBody({required this.section, super.key});

  final SidebarSection section;

  @override
  Widget build(BuildContext context) {
    return switch (section) {
      SidebarSection.overview => const OverviewView(),
      SidebarSection.apps => const AppsView(),
      SidebarSection.provisioners => const ProvisionersView(),
      SidebarSection.instances => const InstancesView(),
      SidebarSection.tasks => const TasksView(),
      SidebarSection.network => const NetworkView(),
      SidebarSection.releases => const ReleasesView(),
      SidebarSection.dvc => const DvcView(),
      SidebarSection.status => const StatusView(),
    };
  }
}

class OverviewView extends StatefulWidget {
  const OverviewView({super.key});

  @override
  State<OverviewView> createState() => _OverviewViewState();
}

class _OverviewViewState extends State<OverviewView> {
  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SectionHeading('User'),
          KeyValueWrap(
            items: [
              KeyValueItem('Username', model.userInfo?.username),
              KeyValueItem('Name', model.userInfo?.name),
              KeyValueItem('Organisation', model.userInfo?.organisationName),
              KeyValueItem('Organisation ID', model.userInfo?.organisationId),
              KeyValueItem(
                'Role',
                model.userInfo == null
                    ? null
                    : (model.userInfo!.isAdmin ? 'Admin' : 'User'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          CommandBar(
            children: [
              CommandButton(
                label: 'Invite user',
                icon: Icons.person_add_alt_1_outlined,
                enabled:
                    model.userInfo?.organisationId.nonEmpty != null &&
                    !model.isBusy,
                refresh: false,
                action: (model) async {
                  await model.startDeviceInvite(
                    organisationId: model.userInfo?.organisationId ?? '',
                    joinMode: protosJoinModeNewUser,
                  );
                },
              ),
              if (model.deviceInvite != null &&
                  _inviteMode(model.deviceInvite!) == protosJoinModeNewUser)
                Text(
                  _inviteActiveText(model.deviceInvite!),
                  style: Theme.of(context).textTheme.labelLarge,
                ),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Devices'),
          CommandBar(
            children: [
              CommandButton(
                label: 'Invite device',
                icon: Icons.devices_other_outlined,
                enabled:
                    model.userInfo?.organisationId.nonEmpty != null &&
                    !model.isBusy,
                refresh: false,
                action: (model) async {
                  await model.startDeviceInvite(
                    organisationId: model.userInfo?.organisationId ?? '',
                    joinMode: protosJoinModeNewDevice,
                  );
                },
              ),
              if (model.deviceInvite != null &&
                  _inviteMode(model.deviceInvite!) == protosJoinModeNewDevice)
                Text(
                  _inviteActiveText(model.deviceInvite!),
                  style: Theme.of(context).textTheme.labelLarge,
                ),
            ],
          ),
          const SizedBox(height: 12),
          RowsPanel<pb.UserDevice>(
            rows: model.devices,
            emptyTitle: 'No devices',
            idForRow: (row) => row.id,
            columns: [
              RowColumn('Name', (row) => row.name),
              RowColumn('ID', (row) => row.id),
              RowColumn('WireGuard', (row) => row.publicKeyWireguard, flex: 2),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Local SSH Key'),
          MonoPane(text: model.sshKey?.public ?? '', minHeight: 90),
        ],
      ),
    );
  }
}

class AppsView extends StatefulWidget {
  const AppsView({super.key});

  @override
  State<AppsView> createState() => _AppsViewState();
}

class _AppsViewState extends State<AppsView> {
  final name = TextEditingController();
  final installer = TextEditingController();
  final instance = TextEditingController();
  var persistence = false;
  String selectedName = '';

  @override
  void initState() {
    super.initState();
    name.addListener(_refreshControls);
    installer.addListener(_refreshControls);
    instance.addListener(_refreshControls);
  }

  void _refreshControls() => setState(() {});

  @override
  void dispose() {
    name.dispose();
    installer.dispose();
    instance.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          RowsPanel<pb.App>(
            rows: model.apps,
            emptyTitle: 'No apps',
            selectedId: selectedName,
            onSelect: (id) => setState(() => selectedName = id),
            idForRow: (row) => row.name,
            columns: [
              RowColumn('Name', (row) => row.name),
              RowColumn('Instance', (row) => row.instanceName),
              RowColumn('Status', (row) => row.status),
              RowColumn('IP', (row) => row.ip),
              RowColumn('Installer', (row) => row.installer),
            ],
          ),
          const SectionGap(),
          FormWrap(
            children: [
              FieldBox(
                width: 180,
                child: TextField(
                  controller: name,
                  decoration: textField('Name'),
                ),
              ),
              FieldBox(
                width: 180,
                child: TextField(
                  controller: installer,
                  decoration: textField('Installer'),
                ),
              ),
              FieldBox(
                width: 180,
                child: TextField(
                  controller: instance,
                  decoration: textField('Instance'),
                ),
              ),
              ToggleBox(
                label: 'Persistent',
                value: persistence,
                onChanged: (value) => setState(() => persistence = value),
              ),
              FilledButton.icon(
                onPressed:
                    name.text.nonEmpty == null ||
                        installer.text.nonEmpty == null ||
                        instance.text.nonEmpty == null
                    ? null
                    : () => unawaited(
                        model.run(() async {
                          await model.api.createApp(
                            name: name.text,
                            installerId: installer.text,
                            instanceId: instance.text,
                            persistence: persistence,
                          );
                          await model.refreshApps(notify: false);
                        }),
                      ),
                icon: const Icon(Icons.add),
                label: const Text('Create'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          CommandBar(
            children: [
              CommandButton(
                label: 'Start',
                icon: Icons.play_arrow,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  await model.api.startApp(selectedName);
                  await model.refreshApps(notify: false);
                },
              ),
              CommandButton(
                label: 'Stop',
                icon: Icons.stop,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  await model.api.stopApp(selectedName);
                  await model.refreshApps(notify: false);
                },
              ),
              CommandButton(
                label: 'Remove',
                icon: Icons.delete_outline,
                destructive: true,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  await model.api.removeApp(selectedName);
                  await model.refreshApps(notify: false);
                },
              ),
              CommandButton(
                label: 'Logs',
                icon: Icons.description_outlined,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  model.outputText = await model.api.appLogs(selectedName);
                },
              ),
            ],
          ),
          const SizedBox(height: 12),
          OutputPane(
            text: model.outputText,
            onChanged: (value) => model.outputText = value,
          ),
        ],
      ),
    );
  }
}

class ProvisionersView extends StatefulWidget {
  const ProvisionersView({super.key});

  @override
  State<ProvisionersView> createState() => _ProvisionersViewState();
}

class _ProvisionersViewState extends State<ProvisionersView> {
  final name = TextEditingController();
  final type = TextEditingController(text: 'local_macos');
  final credentials = TextEditingController();
  String selectedName = '';

  @override
  void initState() {
    super.initState();
    name.addListener(_refreshControls);
    type.addListener(_refreshControls);
  }

  void _refreshControls() => setState(() {});

  @override
  void dispose() {
    name.dispose();
    type.dispose();
    credentials.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          ResponsivePair(
            first: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SectionHeading('Provisioners'),
                RowsPanel<pb.Provisioner>(
                  rows: model.provisioners,
                  emptyTitle: 'No provisioners',
                  selectedId: selectedName,
                  onSelect: (id) => setState(() => selectedName = id),
                  idForRow: (row) => row.name,
                  columns: [
                    RowColumn('Name', (row) => row.name),
                    RowColumn('Type', (row) => row.type.name),
                    RowColumn(
                      'Locations',
                      (row) => row.supportedLocations.join(', '),
                      flex: 2,
                    ),
                  ],
                ),
              ],
            ),
            second: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SectionHeading('Supported'),
                RowsPanel<pb.ProvisionerType>(
                  rows: model.supportedProvisioners,
                  emptyTitle: 'No types',
                  idForRow: (row) => row.name,
                  columns: [
                    RowColumn('Type', (row) => row.name),
                    RowColumn(
                      'Fields',
                      (row) => row.authenticationFields.join(', '),
                      flex: 2,
                    ),
                  ],
                ),
              ],
            ),
          ),
          const SectionGap(),
          FormWrap(
            children: [
              FieldBox(
                width: 180,
                child: TextField(
                  controller: name,
                  decoration: textField('Name'),
                ),
              ),
              FieldBox(
                width: 170,
                child: TextField(
                  controller: type,
                  decoration: textField('Type'),
                ),
              ),
              FieldBox(
                width: 330,
                child: TextField(
                  controller: credentials,
                  minLines: 1,
                  maxLines: 4,
                  decoration: textField('Credentials: KEY=value'),
                ),
              ),
              FilledButton.icon(
                onPressed:
                    name.text.nonEmpty == null || type.text.nonEmpty == null
                    ? null
                    : () => unawaited(
                        model.run(() async {
                          await model.api.addProvisioner(
                            name: name.text,
                            type: type.text,
                            credentials: credentials.text.credentialPairs,
                          );
                          await model.refreshProvisioners(notify: false);
                        }),
                      ),
                icon: const Icon(Icons.add),
                label: const Text('Add'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          CommandBar(
            children: [
              CommandButton(
                label: 'Remove',
                icon: Icons.delete_outline,
                destructive: true,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  await model.api.removeProvisioner(selectedName);
                  await model.refreshProvisioners(notify: false);
                },
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class InstancesView extends StatefulWidget {
  const InstancesView({super.key});

  @override
  State<InstancesView> createState() => _InstancesViewState();
}

class _InstancesViewState extends State<InstancesView> {
  final deployName = TextEditingController();
  String selectedName = '';
  String detailLoadingFor = '';
  pb.CloudInstance? selectedInstance;
  var deploying = false;
  String deployProvisioner = '';
  String deployLocation = '';
  String deployMachine = '';
  String deployVersion = '';
  String deployImage = '';

  @override
  void initState() {
    super.initState();
    deployName.addListener(_refreshControls);
  }

  void _refreshControls() => setState(() {});

  @override
  void dispose() {
    deployName.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);

    if (deploying) {
      return _buildDeployFlow(context, model);
    }

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(child: SectionHeading('Instances')),
              FilledButton.icon(
                onPressed: () => _startDeploy(model),
                icon: const Icon(Icons.add),
                label: const Text('New VM'),
              ),
            ],
          ),
          RowsPanel<pb.CloudInstance>(
            rows: model.instances,
            emptyTitle: 'No instances',
            selectedId: selectedName,
            onSelect: (id) => _selectInstance(model, id),
            idForRow: (row) => row.name,
            columns: [
              RowColumn('Name', (row) => row.name),
              RowColumn('Provisioner', (row) => row.cloudName),
              RowColumn('Location', (row) => row.location),
              RowColumn('Status', (row) => row.status),
              RowColumn('Public IP', (row) => row.publicIp),
              RowColumn('Internal IP', (row) => row.internalIp),
            ],
          ),
          const SizedBox(height: 12),
          CommandBar(
            children: [
              CommandButton(
                label: 'Start',
                icon: Icons.play_arrow,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  await model.api.startInstance(selectedName);
                  await model.refreshInstances(notify: false);
                  await _loadInstanceDetail(model, selectedName);
                },
              ),
              CommandButton(
                label: 'Stop',
                icon: Icons.stop,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  await model.api.stopInstance(selectedName);
                  await model.refreshInstances(notify: false);
                  await _loadInstanceDetail(model, selectedName);
                },
              ),
              CommandButton(
                label: 'Remove',
                icon: Icons.delete_outline,
                destructive: true,
                enabled: selectedName.nonEmpty != null,
                refresh: false,
                action: (model) async {
                  final removed = selectedName;
                  await model.api.removeInstance(removed, localOnly: false);
                  await model.refreshInstances(notify: false);
                  if (!mounted) {
                    return;
                  }
                  setState(() {
                    selectedName = '';
                    detailLoadingFor = '';
                    selectedInstance = null;
                  });
                },
              ),
            ],
          ),
          const SectionGap(),
          InstanceDetailPanel(
            instance: selectedInstance,
            loading:
                selectedName.isNotEmpty && detailLoadingFor == selectedName,
          ),
        ],
      ),
    );
  }

  Widget _buildDeployFlow(BuildContext context, AppModel model) {
    final fields = _deployFieldsByName(model.instanceDeployOptions);
    final provisionerField = fields['cloud_name'];

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: SectionHeading(
                  deployProvisioner.isEmpty ? 'Choose Provisioner' : 'New VM',
                ),
              ),
              TextButton.icon(
                onPressed: _cancelDeploy,
                icon: const Icon(Icons.arrow_back),
                label: const Text('Instances'),
              ),
            ],
          ),
          if (deployProvisioner.isEmpty)
            _ProvisionerChooser(
              field: provisionerField,
              onSelect: (option) => _selectProvisioner(model, option),
            )
          else
            _DeploymentForm(
              name: deployName,
              fields: fields,
              provisioner: deployProvisioner,
              location: deployLocation,
              machine: deployMachine,
              version: deployVersion,
              image: deployImage,
              onChangeProvisioner: () {
                setState(() {
                  deployProvisioner = '';
                  deployLocation = '';
                  deployMachine = '';
                  deployVersion = '';
                  deployImage = '';
                });
                unawaited(
                  model.run(
                    () => model.refreshInstanceDeployOptions(notify: false),
                  ),
                );
              },
              onLocationChanged: (value) => _setDeployLocation(model, value),
              onMachineChanged: (value) =>
                  setState(() => deployMachine = value),
              onVersionChanged: (value) =>
                  setState(() => deployVersion = value),
              onImageChanged: (value) => setState(() => deployImage = value),
              onDeploy: _canDeploy(fields)
                  ? () => unawaited(model.run(() => _deployInstance(model)))
                  : null,
            ),
        ],
      ),
    );
  }

  void _startDeploy(AppModel model) {
    deployName.clear();
    setState(() {
      deploying = true;
      deployProvisioner = '';
      deployLocation = '';
      deployMachine = '';
      deployVersion = '';
      deployImage = '';
    });
    unawaited(
      model.run(() => model.refreshInstanceDeployOptions(notify: false)),
    );
  }

  void _cancelDeploy() {
    deployName.clear();
    setState(() {
      deploying = false;
      deployProvisioner = '';
      deployLocation = '';
      deployMachine = '';
      deployVersion = '';
      deployImage = '';
    });
  }

  void _selectProvisioner(AppModel model, pb.InstanceDeployFieldOption option) {
    setState(() {
      deployProvisioner = option.value;
      deployLocation = '';
      deployMachine = '';
      deployVersion = '';
      deployImage = '';
    });
    unawaited(
      model.run(() async {
        await model.refreshInstanceDeployOptions(
          provisioner: option.value,
          notify: false,
        );
        _applyDeployDefaults(model.instanceDeployOptions);
      }),
    );
  }

  void _setDeployLocation(AppModel model, String value) {
    setState(() {
      deployLocation = value;
      deployMachine = '';
      deployImage = '';
    });
    unawaited(
      model.run(() async {
        await model.refreshInstanceDeployOptions(
          provisioner: deployProvisioner,
          location: value,
          notify: false,
        );
        _applyDeployDefaults(model.instanceDeployOptions);
      }),
    );
  }

  void _applyDeployDefaults(pb.GetInstanceDeployOptionsResponse? options) {
    if (!mounted) {
      return;
    }
    final fields = _deployFieldsByName(options);
    setState(() {
      deployLocation = _preferredFieldValue(
        fields['cloud_location'],
        deployLocation,
      );
      deployMachine = _preferredFieldValue(
        fields['machine_type'],
        deployMachine,
      );
      deployVersion = _preferredFieldValue(
        fields['protos_version'],
        deployVersion,
      );
      deployImage = _preferredFieldValue(fields['dev_img'], deployImage);
    });
  }

  bool _canDeploy(Map<String, pb.InstanceDeployField> fields) {
    final needsVersion = fields['protos_version']?.visible == true;
    final needsImage = fields['dev_img']?.visible == true;
    return deployName.text.nonEmpty != null &&
        deployProvisioner.nonEmpty != null &&
        deployLocation.nonEmpty != null &&
        deployMachine.nonEmpty != null &&
        _hasOptions(fields['cloud_location']) &&
        _hasOptions(fields['machine_type']) &&
        (!needsVersion ||
            (deployVersion.nonEmpty != null &&
                _hasOptions(fields['protos_version']))) &&
        (!needsImage ||
            (deployImage.nonEmpty != null && _hasOptions(fields['dev_img'])));
  }

  Future<void> _deployInstance(AppModel model) async {
    final instanceName = deployName.text.trim();
    await model.api.deployInstance(
      name: instanceName,
      provisioner: deployProvisioner,
      location: deployLocation,
      machineType: deployMachine,
      version: deployImage.isEmpty ? deployVersion : '',
      devImage: deployImage,
    );
    await model.refreshInstances(notify: false);
    if (!mounted) {
      return;
    }
    setState(() {
      deploying = false;
      selectedName = instanceName;
      selectedInstance = null;
      detailLoadingFor = instanceName;
    });
    await _loadInstanceDetail(model, instanceName);
  }

  void _selectInstance(AppModel model, String id) {
    setState(() {
      selectedName = id;
      selectedInstance = null;
      detailLoadingFor = id;
    });
    unawaited(model.run(() => _loadInstanceDetail(model, id)));
  }

  Future<void> _loadInstanceDetail(AppModel model, String id) async {
    final response = await model.api.instance(id);
    if (!mounted || selectedName != id) {
      return;
    }
    setState(() {
      selectedInstance = response.hasInstance() ? response.instance : null;
      detailLoadingFor = '';
    });
  }
}

class _ProvisionerChooser extends StatelessWidget {
  const _ProvisionerChooser({required this.field, required this.onSelect});

  final pb.InstanceDeployField? field;
  final ValueChanged<pb.InstanceDeployFieldOption> onSelect;

  @override
  Widget build(BuildContext context) {
    final options = field?.options.toList(growable: false) ?? const [];
    final scheme = Theme.of(context).colorScheme;
    if (options.isEmpty) {
      return Container(
        width: double.infinity,
        padding: const EdgeInsets.all(18),
        decoration: BoxDecoration(
          border: Border.all(color: scheme.outlineVariant),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          field?.helper.nonEmpty ?? 'No provisioners available',
          style: TextStyle(color: scheme.onSurfaceVariant),
        ),
      );
    }
    return Wrap(
      spacing: 12,
      runSpacing: 12,
      children: [
        for (final option in options)
          _ProvisionerChoice(option: option, onTap: () => onSelect(option)),
      ],
    );
  }
}

class _ProvisionerChoice extends StatelessWidget {
  const _ProvisionerChoice({required this.option, required this.onTap});

  final pb.InstanceDeployFieldOption option;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SizedBox(
      width: 260,
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: onTap,
        child: Container(
          height: 96,
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            border: Border.all(color: scheme.outlineVariant),
            borderRadius: BorderRadius.circular(8),
            color: scheme.surfaceContainerLowest,
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(Icons.dns_outlined, size: 20, color: scheme.primary),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      fallbackText(option.label.nonEmpty),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(fontWeight: FontWeight.w800),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              Text(
                fallbackText(option.description.nonEmpty),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(color: scheme.onSurfaceVariant),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _DeploymentForm extends StatelessWidget {
  const _DeploymentForm({
    required this.name,
    required this.fields,
    required this.provisioner,
    required this.location,
    required this.machine,
    required this.version,
    required this.image,
    required this.onChangeProvisioner,
    required this.onLocationChanged,
    required this.onMachineChanged,
    required this.onVersionChanged,
    required this.onImageChanged,
    required this.onDeploy,
  });

  final TextEditingController name;
  final Map<String, pb.InstanceDeployField> fields;
  final String provisioner;
  final String location;
  final String machine;
  final String version;
  final String image;
  final VoidCallback onChangeProvisioner;
  final ValueChanged<String> onLocationChanged;
  final ValueChanged<String> onMachineChanged;
  final ValueChanged<String> onVersionChanged;
  final ValueChanged<String> onImageChanged;
  final VoidCallback? onDeploy;

  @override
  Widget build(BuildContext context) {
    final provisionerField = fields['cloud_name'];
    final selectedProvisioner = _selectedOption(provisionerField, provisioner);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        KeyValueWrap(
          items: [
            KeyValueItem(
              'Provisioner',
              selectedProvisioner?.label ?? provisioner,
            ),
            KeyValueItem('Type', selectedProvisioner?.description),
          ],
        ),
        const SizedBox(height: 12),
        OutlinedButton.icon(
          onPressed: onChangeProvisioner,
          icon: const Icon(Icons.swap_horiz),
          label: const Text('Change provisioner'),
        ),
        const SectionGap(),
        FormWrap(
          children: [
            FieldBox(
              width: 240,
              child: TextField(controller: name, decoration: textField('Name')),
            ),
            _OptionField(
              width: 190,
              field: fields['cloud_location'],
              value: location,
              onChanged: onLocationChanged,
            ),
            _OptionField(
              width: 230,
              field: fields['machine_type'],
              value: machine,
              onChanged: onMachineChanged,
            ),
            if (fields['protos_version']?.visible == true)
              _OptionField(
                width: 210,
                field: fields['protos_version'],
                value: version,
                onChanged: onVersionChanged,
              ),
            if (fields['dev_img']?.visible == true)
              _OptionField(
                width: 240,
                field: fields['dev_img'],
                value: image,
                onChanged: onImageChanged,
              ),
            FilledButton.icon(
              onPressed: onDeploy,
              icon: const Icon(Icons.cloud_upload_outlined),
              label: const Text('Deploy'),
            ),
          ],
        ),
        const SizedBox(height: 18),
        _DeploySelectionSummary(
          location: _selectedOption(fields['cloud_location'], location),
          machine: _selectedOption(fields['machine_type'], machine),
          version: _selectedOption(fields['protos_version'], version),
          image: _selectedOption(fields['dev_img'], image),
        ),
      ],
    );
  }
}

class _OptionField extends StatelessWidget {
  const _OptionField({
    required this.width,
    required this.field,
    required this.value,
    required this.onChanged,
  });

  final double width;
  final pb.InstanceDeployField? field;
  final String value;
  final ValueChanged<String> onChanged;

  @override
  Widget build(BuildContext context) {
    final current = _dropdownValue(field, value);
    final options = field?.options.toList(growable: false) ?? const [];
    final helper = field?.helper.nonEmpty;

    return FieldBox(
      width: width,
      child: DropdownButtonFormField<String>(
        key: ValueKey('${field?.name}:$current:${options.length}'),
        initialValue: current,
        isExpanded: true,
        decoration: textField(
          field?.label ?? 'Option',
        ).copyWith(helperText: helper),
        items: [
          for (final option in options)
            DropdownMenuItem<String>(
              value: option.value,
              child: Text(
                _optionDisplay(option),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
        ],
        onChanged: options.isEmpty
            ? null
            : (value) {
                if (value != null) {
                  onChanged(value);
                }
              },
      ),
    );
  }
}

class _DeploySelectionSummary extends StatelessWidget {
  const _DeploySelectionSummary({
    required this.location,
    required this.machine,
    required this.version,
    required this.image,
  });

  final pb.InstanceDeployFieldOption? location;
  final pb.InstanceDeployFieldOption? machine;
  final pb.InstanceDeployFieldOption? version;
  final pb.InstanceDeployFieldOption? image;

  @override
  Widget build(BuildContext context) {
    return KeyValueWrap(
      items: [
        KeyValueItem('Location', location?.label),
        KeyValueItem('Size', _optionDisplay(machine)),
        KeyValueItem('Version', version?.label ?? image?.label),
      ],
    );
  }
}

class InstanceDetailPanel extends StatelessWidget {
  const InstanceDetailPanel({
    required this.instance,
    required this.loading,
    super.key,
  });

  final pb.CloudInstance? instance;
  final bool loading;

  @override
  Widget build(BuildContext context) {
    if (loading) {
      return const SizedBox(
        height: 72,
        child: Center(
          child: SizedBox.square(
            dimension: 20,
            child: CircularProgressIndicator(strokeWidth: 2),
          ),
        ),
      );
    }

    final selected = instance;
    if (selected == null) {
      return const SizedBox.shrink();
    }

    final peers = selected.peers.entries.toList(growable: false)
      ..sort((a, b) => a.key.compareTo(b.key));

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const SectionHeading('Instance Details'),
        KeyValueWrap(
          items: [
            KeyValueItem('Name', selected.name),
            KeyValueItem('Status', selected.status),
            KeyValueItem('Provisioner', selected.cloudName),
            KeyValueItem('Type', selected.cloudType),
            KeyValueItem('Location', selected.location),
            KeyValueItem('VM ID', selected.vmId),
            KeyValueItem('Public IP', selected.publicIp),
            KeyValueItem('Internal IP', selected.internalIp),
            KeyValueItem('Architecture', selected.architecture),
            KeyValueItem('WireGuard', selected.publicKeyWireguard),
          ],
        ),
        const SizedBox(height: 12),
        MonoPane(text: selected.publicKey, minHeight: 72),
        if (peers.isNotEmpty) ...[
          const SizedBox(height: 18),
          RowsPanel<MapEntry<String, String>>(
            rows: peers,
            emptyTitle: 'No peers',
            height: 180,
            idForRow: (row) => row.key,
            columns: [
              RowColumn('Peer', (row) => row.key),
              RowColumn('Status', (row) => row.value),
            ],
          ),
        ],
      ],
    );
  }
}

class TasksView extends StatelessWidget {
  const TasksView({super.key});

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(child: SectionHeading('Background Tasks')),
              OutlinedButton.icon(
                onPressed: model.isBusy
                    ? null
                    : () => unawaited(
                        model.run(() => model.refreshTasks(notify: false)),
                      ),
                icon: const Icon(Icons.refresh),
                label: const Text('Refresh'),
              ),
            ],
          ),
          RowsPanel<pb.Task>(
            rows: model.tasks,
            emptyTitle: 'No tasks',
            height: 280,
            selectedId: model.selectedTaskId,
            onSelect: (id) => unawaited(model.run(() => model.selectTask(id))),
            idForRow: (row) => row.id,
            columns: [
              RowColumn('Title', _taskTitle, flex: 2),
              RowColumn('Status', _taskStatusText),
              RowColumn('Progress', (row) => _taskProgressLabel(row.progress)),
              RowColumn(
                'Updated',
                (row) => _taskDateLabel(row.updatedAt),
                flex: 2,
              ),
              RowColumn('Message', (row) => row.message, flex: 2),
            ],
          ),
          const SectionGap(),
          TaskDetailPanel(
            task: model.selectedTask,
            events: model.selectedTaskEvents,
          ),
        ],
      ),
    );
  }
}

class TaskDetailPanel extends StatelessWidget {
  const TaskDetailPanel({required this.task, required this.events, super.key});

  final pb.Task? task;
  final List<pb.TaskEvent> events;

  @override
  Widget build(BuildContext context) {
    final selected = task;
    if (selected == null) {
      return MonoPane(
        text: 'Select a task to inspect its progress and details.',
        minHeight: 64,
      );
    }

    final messageLines = [
      ?selected.message.nonEmpty,
      if (selected.errorMessage.nonEmpty != null)
        'Error: ${selected.errorMessage}',
    ].join('\n');

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(child: SectionHeading(_taskTitle(selected))),
            TaskStatusBadge(status: selected.status),
          ],
        ),
        const SizedBox(height: 2),
        LinearProgressIndicator(value: _taskProgressValue(selected.progress)),
        const SizedBox(height: 12),
        KeyValueWrap(
          items: [
            KeyValueItem('ID', selected.id),
            KeyValueItem('Stream', selected.stream),
            KeyValueItem('Subject', _taskSubjectLabel(selected)),
            KeyValueItem('Status', _taskStatusText(selected)),
            KeyValueItem('Progress', _taskProgressLabel(selected.progress)),
            KeyValueItem('Attempts', _taskAttemptsLabel(selected)),
            KeyValueItem('Created', _taskDateLabel(selected.createdAt)),
            KeyValueItem('Updated', _taskDateLabel(selected.updatedAt)),
            KeyValueItem('Started', _taskDateLabel(selected.startedAt)),
            KeyValueItem('Finished', _taskDateLabel(selected.finishedAt)),
          ],
        ),
        if (messageLines.nonEmpty != null) ...[
          const SizedBox(height: 18),
          const SectionHeading('Message'),
          MonoPane(text: messageLines, minHeight: 72),
        ],
        const SizedBox(height: 18),
        ResponsivePair(
          first: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SectionHeading('Payload'),
              MonoPane(text: _taskJson(selected.payloadJson), minHeight: 92),
            ],
          ),
          second: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const SectionHeading('Result'),
              MonoPane(text: _taskJson(selected.resultJson), minHeight: 92),
            ],
          ),
        ),
        const SectionGap(),
        const SectionHeading('Events'),
        RowsPanel<pb.TaskEvent>(
          rows: events,
          emptyTitle: 'No events',
          height: 240,
          idForRow: (row) =>
              '${row.createdAt}|${row.status}|${row.progress}|${row.message}',
          columns: [
            RowColumn('Status', _taskEventStatusText),
            RowColumn('Progress', (row) => _taskProgressLabel(row.progress)),
            RowColumn(
              'Created',
              (row) => _taskDateLabel(row.createdAt),
              flex: 2,
            ),
            RowColumn('Message', (row) => row.message, flex: 2),
            RowColumn('Details', (row) => row.detailsJson, flex: 2),
          ],
        ),
      ],
    );
  }
}

class TaskStatusBadge extends StatelessWidget {
  const TaskStatusBadge({required this.status, super.key});

  final String status;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final color = _taskStatusColor(scheme, status);
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.14),
        borderRadius: BorderRadius.circular(999),
        border: Border.all(color: color.withValues(alpha: 0.38)),
      ),
      child: Text(
        _taskStatusValue(status),
        style: TextStyle(color: color, fontWeight: FontWeight.w800),
      ),
    );
  }
}

class NetworkView extends StatefulWidget {
  const NetworkView({super.key});

  @override
  State<NetworkView> createState() => _NetworkViewState();
}

class _NetworkViewState extends State<NetworkView> {
  final inspectInstance = TextEditingController();
  final routeInstance = TextEditingController();
  final dnsServer = TextEditingController();
  final cidrs = TextEditingController();

  @override
  void initState() {
    super.initState();
    inspectInstance.addListener(_refreshControls);
    routeInstance.addListener(_refreshControls);
    dnsServer.addListener(_refreshControls);
    cidrs.addListener(_refreshControls);
  }

  void _refreshControls() => setState(() {});

  @override
  void dispose() {
    inspectInstance.dispose();
    routeInstance.dispose();
    dnsServer.dispose();
    cidrs.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    if (model.supportsMobileTunnel && !model.supportsCoreNetwork) {
      return _buildMobileTunnelView(context, model);
    }
    final runtime = model.networkRuntimeStatus;
    if (model.supportsNetwork && !model.isNetworkEnabled) {
      return DetailScroll(
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const SectionHeading('Network'),
            KeyValueWrap(
              items: [
                KeyValueItem('Status', runtime?.state),
                KeyValueItem('Message', runtime?.message),
              ],
            ),
          ],
        ),
      );
    }

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          FormWrap(
            children: [
              FieldBox(
                width: 240,
                child: TextField(
                  controller: inspectInstance,
                  decoration: textField('Target instance'),
                ),
              ),
              FilledButton.icon(
                onPressed: () => unawaited(
                  model.run(
                    () => model.refreshNetwork(
                      instance: inspectInstance.text.nonEmpty ?? '',
                      notify: false,
                    ),
                  ),
                ),
                icon: const Icon(Icons.download_outlined),
                label: const Text('Load'),
              ),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Summary'),
          KeyValueWrap(
            items: [
              KeyValueItem('Module', model.networkState?.module),
              KeyValueItem(
                'Status',
                model.networkState == null
                    ? null
                    : (model.networkState!.up ? 'up' : 'down'),
              ),
              KeyValueItem('Interface', model.networkState?.interfaceName),
              KeyValueItem('Messages', model.networkState?.messages.join(', ')),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Exit Routes'),
          RowsPanel<pb.ExitRoute>(
            rows: model.exitRoutes,
            emptyTitle: 'No exit routes',
            idForRow: (row) =>
                row.id.nonEmpty ?? '${row.deviceId}:${row.instanceId}',
            columns: [
              RowColumn('Device', (row) => row.deviceId),
              RowColumn('Instance', (row) => row.instanceId),
              RowColumn('Name', (row) => row.instanceName),
              RowColumn('Public IP', (row) => row.publicIp),
              RowColumn('Location', (row) => row.location),
              RowColumn('CIDRs', (row) => row.cidrs.join(', '), flex: 2),
              RowColumn('DNS', (row) => row.dnsServer.nonEmpty ?? 'default'),
              RowColumn('Status', (row) => row.status),
            ],
          ),
          const SizedBox(height: 12),
          FormWrap(
            children: [
              FieldBox(
                width: 180,
                child: TextField(
                  controller: routeInstance,
                  decoration: textField('Instance'),
                ),
              ),
              FieldBox(
                width: 150,
                child: TextField(
                  controller: dnsServer,
                  decoration: textField('DNS server'),
                ),
              ),
              FieldBox(
                width: 280,
                child: TextField(
                  controller: cidrs,
                  minLines: 1,
                  maxLines: 4,
                  decoration: textField('CIDRs'),
                ),
              ),
              FilledButton.icon(
                onPressed: routeInstance.text.nonEmpty == null
                    ? null
                    : () => unawaited(
                        model.run(() async {
                          final response = await model.api.setExitRoute(
                            instance: routeInstance.text,
                            dnsServer: dnsServer.text.nonEmpty ?? '',
                            cidrs: cidrs.text.routeCidrs,
                          );
                          model.outputText = _routeOutput(response.route);
                          await model.refreshNetwork(
                            instance: inspectInstance.text.nonEmpty ?? '',
                            notify: false,
                          );
                        }),
                      ),
                icon: const Icon(Icons.alt_route),
                label: const Text('Set'),
              ),
              OutlinedButton.icon(
                onPressed: () => unawaited(
                  model.run(() async {
                    await model.api.clearExitRoute();
                    model.outputText = 'Exit route disabled';
                    await model.refreshNetwork(
                      instance: inspectInstance.text.nonEmpty ?? '',
                      notify: false,
                    );
                  }),
                ),
                icon: const Icon(Icons.cancel_outlined),
                label: const Text('Clear'),
              ),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Observed Routes'),
          RowsPanel<pb.NetworkRoute>(
            rows: model.networkState?.routes ?? const [],
            emptyTitle: 'No observed routes',
            idForRow: (row) =>
                '${row.destination}|${row.gateway}|${row.interfaceName}|${row.source}|${row.priority}',
            columns: [
              RowColumn(
                'Destination',
                (row) => row.destination.nonEmpty ?? 'default',
              ),
              RowColumn('Gateway', (row) => row.gateway),
              RowColumn('Interface', (row) => row.interfaceName),
              RowColumn('Source', (row) => row.source),
              RowColumn('Family', (row) => row.family),
              RowColumn('Kind', (row) => row.kind),
              RowColumn('Priority', (row) => row.priority),
            ],
          ),
          const SectionGap(),
          ResponsivePair(
            first: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SectionHeading('Addresses'),
                RowsPanel<pb.NetworkAddress>(
                  rows: model.networkState?.addresses ?? const [],
                  emptyTitle: 'No addresses',
                  idForRow: (row) => '${row.interfaceName}|${row.cidr}',
                  columns: [
                    RowColumn('Interface', (row) => row.interfaceName),
                    RowColumn('CIDR', (row) => row.cidr),
                    RowColumn('Scope', (row) => row.scope),
                  ],
                ),
              ],
            ),
            second: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const SectionHeading('DNS'),
                RowsPanel<pb.DNSState>(
                  rows: model.networkState?.dns ?? const [],
                  emptyTitle: 'No DNS entries',
                  idForRow: (row) =>
                      '${row.scope}|${row.domain}|${row.servers.join(',')}|${row.port}',
                  columns: [
                    RowColumn('Scope', (row) => row.scope),
                    RowColumn('Domain', (row) => row.domain),
                    RowColumn(
                      'Servers',
                      (row) => row.servers.join(', '),
                      flex: 2,
                    ),
                    RowColumn('Port', (row) => '${row.port}'),
                    RowColumn('Active', (row) => row.active ? 'yes' : 'no'),
                  ],
                ),
              ],
            ),
          ),
          const SectionGap(),
          const SectionHeading('WireGuard Peers'),
          RowsPanel<pb.WireGuardPeer>(
            rows: model.networkState?.wireguardPeers ?? const [],
            emptyTitle: 'No peers',
            idForRow: (row) => row.publicKey,
            columns: [
              RowColumn(
                'Public Key',
                (row) => shortHash(row.publicKey, length: 18) ?? '',
              ),
              RowColumn('Endpoint', (row) => row.endpoint),
              RowColumn(
                'Allowed IPs',
                (row) => row.allowedIps.join(', '),
                flex: 2,
              ),
              RowColumn('Handshake', (row) => row.latestHandshake),
              RowColumn('RX', (row) => '${row.rxBytes}'),
              RowColumn('TX', (row) => '${row.txBytes}'),
            ],
          ),
          const SizedBox(height: 12),
          OutputPane(
            text: model.outputText,
            onChanged: (value) => model.outputText = value,
            minHeight: 100,
          ),
        ],
      ),
    );
  }

  Widget _buildMobileTunnelView(BuildContext context, AppModel model) {
    final status = model.mobileTunnelStatus;
    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SectionHeading('Mobile Tunnel'),
          KeyValueWrap(
            items: [
              KeyValueItem('Profile', status.installed ? 'Installed' : null),
              KeyValueItem('Status', status.label),
              KeyValueItem('Config', shortHash(status.configId, length: 18)),
              KeyValueItem('Message', status.detail),
            ],
          ),
          const SizedBox(height: 12),
          FormWrap(
            children: [
              FieldBox(
                width: 180,
                child: TextField(
                  controller: routeInstance,
                  decoration: textField('Instance'),
                ),
              ),
              FieldBox(
                width: 150,
                child: TextField(
                  controller: dnsServer,
                  decoration: textField('DNS server'),
                ),
              ),
              FieldBox(
                width: 280,
                child: TextField(
                  controller: cidrs,
                  minLines: 1,
                  maxLines: 4,
                  decoration: textField('CIDRs'),
                ),
              ),
              FilledButton.icon(
                onPressed: routeInstance.text.nonEmpty == null
                    ? null
                    : () => unawaited(
                        model.run(
                          () => model.installOrUpdateMobileTunnel(
                            instance: routeInstance.text,
                            dnsServer: dnsServer.text.nonEmpty ?? '',
                            cidrs: cidrs.text.routeCidrs,
                          ),
                        ),
                      ),
                icon: const Icon(Icons.tune),
                label: const Text('Install'),
              ),
              FilledButton.icon(
                onPressed: status.installed
                    ? () => unawaited(model.run(model.startMobileTunnel))
                    : null,
                icon: const Icon(Icons.play_arrow),
                label: const Text('Start'),
              ),
              OutlinedButton.icon(
                onPressed: status.installed
                    ? () => unawaited(model.run(model.stopMobileTunnel))
                    : null,
                icon: const Icon(Icons.stop),
                label: const Text('Stop'),
              ),
              OutlinedButton.icon(
                onPressed: () => unawaited(
                  model.run(
                    () => model.refreshMobileTunnelStatus(notify: false),
                  ),
                ),
                icon: const Icon(Icons.refresh),
                label: const Text('Refresh'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          OutputPane(
            text: model.outputText,
            onChanged: (value) => model.outputText = value,
            minHeight: 100,
          ),
        ],
      ),
    );
  }
}

class ReleasesView extends StatefulWidget {
  const ReleasesView({super.key});

  @override
  State<ReleasesView> createState() => _ReleasesViewState();
}

class _ReleasesViewState extends State<ReleasesView> {
  final provisioner = TextEditingController();
  final imagePath = TextEditingController();
  final imageName = TextEditingController();
  final location = TextEditingController(text: 'local');
  final timeout = TextEditingController(text: '600');
  String selectedImage = '';

  @override
  void initState() {
    super.initState();
    for (final controller in [
      provisioner,
      imagePath,
      imageName,
      location,
      timeout,
    ]) {
      controller.addListener(_refreshControls);
    }
  }

  void _refreshControls() => setState(() {});

  @override
  void dispose() {
    provisioner.dispose();
    imagePath.dispose();
    imageName.dispose();
    location.dispose();
    timeout.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              const Expanded(child: SectionHeading('Available Releases')),
              FilledButton.icon(
                onPressed: () => unawaited(
                  model.run(() async {
                    model.releases = (await model.api.releases()).releases;
                  }),
                ),
                icon: const Icon(Icons.download_outlined),
                label: const Text('Load'),
              ),
            ],
          ),
          RowsPanel<pb.Release>(
            rows: model.releases,
            emptyTitle: 'No releases',
            idForRow: (row) => row.version,
            columns: [
              RowColumn('Version', (row) => row.version),
              RowColumn('Description', (row) => row.description, flex: 2),
              RowColumn('Images', (row) => '${row.cloudImages.length}'),
            ],
          ),
          const SectionGap(),
          FormWrap(
            children: [
              FieldBox(
                width: 220,
                child: TextField(
                  controller: provisioner,
                  decoration: textField('Provisioner'),
                ),
              ),
              FilledButton.icon(
                onPressed: provisioner.text.nonEmpty == null
                    ? null
                    : () => unawaited(
                        model.run(() async {
                          final response = await model.api.provisionerImages(
                            provisioner.text,
                          );
                          model.setProvisionerImages(
                            response.images,
                            notify: false,
                          );
                        }),
                      ),
                icon: const Icon(Icons.photo_library_outlined),
                label: const Text('Load Images'),
              ),
            ],
          ),
          const SizedBox(height: 12),
          RowsPanel<MapEntry<String, pb.CloudSpecificImage>>(
            rows: model.sortedProvisionerImages,
            emptyTitle: 'No provisioner images',
            selectedId: selectedImage,
            onSelect: (id) => setState(() => selectedImage = id),
            idForRow: (row) => row.key,
            columns: [
              RowColumn('Key', (row) => row.key),
              RowColumn('Name', (row) => row.value.name),
              RowColumn('Location', (row) => row.value.location),
              RowColumn('ID', (row) => row.value.id),
            ],
          ),
          const SizedBox(height: 12),
          FormWrap(
            children: [
              FieldBox(
                width: 240,
                child: TextField(
                  controller: imagePath,
                  decoration: textField('Image path'),
                ),
              ),
              FieldBox(
                width: 190,
                child: TextField(
                  controller: imageName,
                  decoration: textField('Image name'),
                ),
              ),
              FieldBox(
                width: 130,
                child: TextField(
                  controller: location,
                  decoration: textField('Location'),
                ),
              ),
              FieldBox(
                width: 110,
                child: TextField(
                  controller: timeout,
                  keyboardType: TextInputType.number,
                  decoration: textField('Timeout'),
                ),
              ),
              FilledButton.icon(
                onPressed:
                    provisioner.text.nonEmpty == null ||
                        imagePath.text.nonEmpty == null ||
                        imageName.text.nonEmpty == null
                    ? null
                    : () => unawaited(
                        model.run(() async {
                          await model.api.uploadProvisionerImage(
                            imagePath: imagePath.text,
                            imageName: imageName.text,
                            provisioner: provisioner.text,
                            location: location.text,
                            timeout: int.tryParse(timeout.text) ?? 600,
                          );
                          final response = await model.api.provisionerImages(
                            provisioner.text,
                          );
                          model.setProvisionerImages(
                            response.images,
                            notify: false,
                          );
                        }),
                      ),
                icon: const Icon(Icons.upload_outlined),
                label: const Text('Upload'),
              ),
              CommandButton(
                label: 'Remove',
                icon: Icons.delete_outline,
                destructive: true,
                enabled: selectedImage.nonEmpty != null,
                action: (model) async {
                  await model.api.removeProvisionerImage(
                    imageName: selectedImage,
                    provisioner: provisioner.text,
                    location: location.text,
                  );
                  final response = await model.api.provisionerImages(
                    provisioner.text,
                  );
                  model.setProvisionerImages(response.images, notify: false);
                },
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class DvcView extends StatefulWidget {
  const DvcView({super.key});

  @override
  State<DvcView> createState() => _DvcViewState();
}

class _DvcViewState extends State<DvcView> {
  final sqlController = TextEditingController(text: 'SHOW TABLES;');
  String selectedCommitHash = '';
  pb.ExecuteSqlResponse? sqlResult;
  bool sqlRunning = false;
  bool sqlShellOpen = false;

  @override
  void dispose() {
    sqlController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final selectedCommit = _selectedCommit(model.localCommits);

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SectionHeading('Peers'),
          KeyValueWrap(
            items: [
              KeyValueItem('Peer ID', model.runtimeState?.peerId),
              KeyValueItem(
                'Finalized root',
                shortHash(model.runtimeState?.checkpointRootHash),
              ),
              KeyValueItem(
                'Tentative root',
                shortHash(model.runtimeState?.tentativeRootHash),
              ),
              KeyValueItem(
                'Connected peers',
                model.runtimeState == null
                    ? null
                    : '${model.runtimeState!.connectedPeers.length}',
              ),
              KeyValueItem(
                'State providers',
                model.runtimeState?.stateProviders.join(', '),
              ),
              KeyValueItem(
                'Materialization',
                model.runtimeState?.runtimeMaterializationPolicy,
              ),
            ],
          ),
          const SizedBox(height: 12),
          RowsPanel<pb.RuntimePeerStatus>(
            rows: model.runtimePeers,
            emptyTitle: 'No peers',
            idForRow: (row) => row.peerId,
            columns: [
              RowColumn(
                'Peer',
                (row) => _peerLabel(row, model.runtimeState?.peerId),
              ),
              RowColumn('Connection', _peerConnection),
              RowColumn(
                'Roles',
                (row) => _peerRoles(row, model.runtimeState?.peerId),
                flex: 2,
              ),
              RowColumn('Compatibility', _peerCompatibility),
              RowColumn('Reason', _peerReason, flex: 2),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Commits'),
          CommitGraphPanel(
            graph: model.localCommitGraph,
            selectedId: selectedCommitHash,
            onSelect: (id) => setState(() => selectedCommitHash = id),
          ),
          const SizedBox(height: 12),
          CommitDetailPanel(commit: selectedCommit),
          const SectionGap(),
          if (!sqlShellOpen)
            OutlinedButton.icon(
              onPressed: () => setState(() => sqlShellOpen = true),
              icon: const Icon(Icons.terminal_outlined),
              label: const Text('Open SQL shell'),
            )
          else ...[
            const SectionHeading('SQL'),
            SqlConsole(
              controller: sqlController,
              result: sqlResult,
              running: sqlRunning,
              onRun: model.isBusy || sqlRunning
                  ? null
                  : () => unawaited(model.run(() => _executeSql(model))),
              onCopy: sqlResult == null ? null : () => _copySqlResult(context),
            ),
          ],
        ],
      ),
    );
  }

  Future<void> _executeSql(AppModel model) async {
    setState(() => sqlRunning = true);
    try {
      final response = await model.api.executeSql(
        sql: sqlController.text,
        maxRows: 200,
      );
      if (!mounted) {
        return;
      }
      setState(() => sqlResult = response);
      await model.refreshDvc(notify: false);
    } finally {
      if (mounted) {
        setState(() => sqlRunning = false);
      }
    }
  }

  void _copySqlResult(BuildContext context) {
    final text = _sqlResultTsv(sqlResult);
    Clipboard.setData(ClipboardData(text: text));
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('Copied')));
  }

  pb.Commit? _selectedCommit(List<pb.Commit> commits) {
    for (final commit in commits) {
      if (commit.hash == selectedCommitHash) {
        return commit;
      }
    }
    return null;
  }
}

class SqlConsole extends StatelessWidget {
  const SqlConsole({
    required this.controller,
    required this.result,
    required this.running,
    required this.onRun,
    required this.onCopy,
    super.key,
  });

  final TextEditingController controller;
  final pb.ExecuteSqlResponse? result;
  final bool running;
  final VoidCallback? onRun;
  final VoidCallback? onCopy;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: controller,
          minLines: 4,
          maxLines: 8,
          style: const TextStyle(fontFamily: 'monospace'),
          decoration: textField('SQL'),
        ),
        const SizedBox(height: 10),
        CommandBar(
          children: [
            FilledButton.icon(
              onPressed: onRun,
              icon: running
                  ? const SizedBox.square(
                      dimension: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.play_arrow_rounded),
              label: const Text('Run'),
            ),
            OutlinedButton.icon(
              onPressed: onCopy,
              icon: const Icon(Icons.copy_rounded),
              label: const Text('Copy'),
            ),
          ],
        ),
        const SizedBox(height: 12),
        SqlResultPanel(result: result),
      ],
    );
  }
}

class SqlResultPanel extends StatefulWidget {
  const SqlResultPanel({required this.result, super.key});

  final pb.ExecuteSqlResponse? result;

  @override
  State<SqlResultPanel> createState() => _SqlResultPanelState();
}

class _SqlResultPanelState extends State<SqlResultPanel> {
  final _horizontalController = ScrollController();

  @override
  void dispose() {
    _horizontalController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final current = widget.result;
    if (current == null) {
      return const SizedBox.shrink();
    }
    final scheme = Theme.of(context).colorScheme;
    final columns = current.columns;
    final rowCount = current.rows.length;
    final message = current.message.nonEmpty;
    final status = [
      ?message,
      if (message == null && current.hasRowsAffected() && columns.isEmpty)
        '${current.rowsAffected} rows affected',
      if (current.truncated) 'truncated',
    ].whereType<String>().join(' / ');

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (status.nonEmpty != null)
          Padding(
            padding: const EdgeInsets.only(bottom: 8),
            child: Text(
              status,
              style: TextStyle(color: scheme.onSurfaceVariant),
            ),
          ),
        if (columns.isEmpty)
          MonoPane(text: status.nonEmpty ?? 'Done', minHeight: 56)
        else
          SizedBox(
            height: 280,
            child: Container(
              decoration: BoxDecoration(
                border: Border.all(color: scheme.outlineVariant),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Scrollbar(
                controller: _horizontalController,
                thumbVisibility: true,
                child: SingleChildScrollView(
                  controller: _horizontalController,
                  primary: false,
                  scrollDirection: Axis.horizontal,
                  child: SizedBox(
                    width: _sqlTableWidth(columns.length),
                    child: Column(
                      children: [
                        Container(
                          height: 36,
                          padding: const EdgeInsets.symmetric(horizontal: 10),
                          decoration: BoxDecoration(
                            color: scheme.surfaceContainerHighest,
                            borderRadius: const BorderRadius.vertical(
                              top: Radius.circular(7),
                            ),
                          ),
                          child: Row(
                            children: [
                              for (final column in columns)
                                _SqlTableCell(text: column, strong: true),
                            ],
                          ),
                        ),
                        Divider(height: 1, color: scheme.outlineVariant),
                        Expanded(
                          child: rowCount == 0
                              ? Center(
                                  child: Text(
                                    'No rows',
                                    style: TextStyle(
                                      color: scheme.onSurfaceVariant,
                                    ),
                                  ),
                                )
                              : ListView.builder(
                                  primary: false,
                                  itemCount: rowCount,
                                  itemExtent: 34,
                                  itemBuilder: (context, index) {
                                    final row = current.rows[index];
                                    return Padding(
                                      padding: const EdgeInsets.symmetric(
                                        horizontal: 10,
                                      ),
                                      child: Row(
                                        children: [
                                          for (
                                            var column = 0;
                                            column < columns.length;
                                            column++
                                          )
                                            _SqlTableCell(
                                              text: _sqlCellLabel(row, column),
                                            ),
                                        ],
                                      ),
                                    );
                                  },
                                ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
      ],
    );
  }
}

class _SqlTableCell extends StatelessWidget {
  const _SqlTableCell({required this.text, this.strong = false});

  final String text;
  final bool strong;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SizedBox(
      width: 180,
      child: Padding(
        padding: const EdgeInsets.only(right: 12),
        child: Text(
          text,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            fontFamily: 'monospace',
            color: strong ? scheme.onSurfaceVariant : null,
            fontWeight: strong ? FontWeight.w800 : FontWeight.w400,
          ),
        ),
      ),
    );
  }
}

const _commitGraphRowHeight = 50.0;
const _commitGraphHeaderHeight = 36.0;
const _commitGraphLaneGap = 20.0;
const _commitGraphLanePadding = 18.0;
const _commitGraphNodeRadius = 5.5;
const _commitGraphLaneColors = <Color>[
  Color(0xff0f766e),
  Color(0xff2563eb),
  Color(0xffdc2626),
  Color(0xffca8a04),
  Color(0xff7c3aed),
  Color(0xff0891b2),
];

class CommitGraphPanel extends StatefulWidget {
  const CommitGraphPanel({
    required this.graph,
    required this.selectedId,
    required this.onSelect,
    super.key,
  });

  final pb.CommitGraph? graph;
  final String selectedId;
  final ValueChanged<String> onSelect;

  @override
  State<CommitGraphPanel> createState() => _CommitGraphPanelState();
}

class _CommitGraphPanelState extends State<CommitGraphPanel> {
  final _controller = ScrollController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final graph = widget.graph;
    final items =
        graph?.items
            .where((item) => item.hasCommit())
            .toList(growable: false) ??
        const <pb.CommitGraphItem>[];
    final laneCount = _commitGraphLaneCount(graph, items);
    final graphWidth = _commitGraphWidth(laneCount);
    final scheme = Theme.of(context).colorScheme;

    return SizedBox(
      height: 260,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final contentWidth = math.max(
            constraints.maxWidth,
            _commitGraphContentWidth(graphWidth),
          );
          return SingleChildScrollView(
            primary: false,
            scrollDirection: Axis.horizontal,
            child: SizedBox(
              width: contentWidth,
              height: constraints.maxHeight,
              child: Column(
                children: [
                  _CommitGraphHeader(graphWidth: graphWidth),
                  Expanded(
                    child: items.isEmpty
                        ? Center(
                            child: Text(
                              'No commits',
                              style: TextStyle(
                                color: Theme.of(
                                  context,
                                ).colorScheme.onSurfaceVariant,
                              ),
                            ),
                          )
                        : Scrollbar(
                            controller: _controller,
                            thumbVisibility: items.length > 5,
                            child: SingleChildScrollView(
                              controller: _controller,
                              primary: false,
                              child: SizedBox(
                                height: items.length * _commitGraphRowHeight,
                                child: Stack(
                                  children: [
                                    Positioned.fill(
                                      child: CustomPaint(
                                        painter: _CommitGraphPainter(
                                          items: items,
                                          laneCount: laneCount,
                                          scheme: scheme,
                                        ),
                                      ),
                                    ),
                                    for (final item in items)
                                      _CommitGraphRow(
                                        item: item,
                                        graphWidth: graphWidth,
                                        selected:
                                            widget.selectedId.nonEmpty ==
                                            item.commit.hash,
                                        onSelect: widget.onSelect,
                                      ),
                                  ],
                                ),
                              ),
                            ),
                          ),
                  ),
                ],
              ),
            ),
          );
        },
      ),
    );
  }
}

class _CommitGraphHeader extends StatelessWidget {
  const _CommitGraphHeader({required this.graphWidth});

  final double graphWidth;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final style = Theme.of(context).textTheme.labelMedium?.copyWith(
      color: scheme.onSurfaceVariant,
      fontWeight: FontWeight.w800,
    );

    return Container(
      height: _commitGraphHeaderHeight,
      padding: EdgeInsets.zero,
      child: Row(
        children: [
          SizedBox(
            width: graphWidth,
            child: Text('Graph', style: style),
          ),
          _CommitGraphHeaderCell('Hash', width: 104, style: style),
        ],
      ),
    );
  }
}

class _CommitGraphHeaderCell extends StatelessWidget {
  const _CommitGraphHeaderCell(this.text, {required this.width, this.style});

  final String text;
  final double width;
  final TextStyle? style;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: width,
      child: Padding(
        padding: const EdgeInsets.only(right: 12),
        child: Text(
          text,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: style,
        ),
      ),
    );
  }
}

class _CommitGraphRow extends StatelessWidget {
  const _CommitGraphRow({
    required this.item,
    required this.graphWidth,
    required this.selected,
    required this.onSelect,
  });

  final pb.CommitGraphItem item;
  final double graphWidth;
  final bool selected;
  final ValueChanged<String> onSelect;

  @override
  Widget build(BuildContext context) {
    final commit = item.commit;
    final scheme = Theme.of(context).colorScheme;
    final hash = commit.hash.nonEmpty;

    return Positioned(
      top: item.row * _commitGraphRowHeight,
      left: 0,
      right: 0,
      height: _commitGraphRowHeight,
      child: InkWell(
        key: ValueKey(hash ?? 'commit-row-${item.row}'),
        onTap: hash == null ? null : () => onSelect(hash),
        child: Container(
          padding: EdgeInsets.zero,
          color: selected
              ? scheme.primaryContainer.withValues(alpha: 0.56)
              : Colors.transparent,
          child: Row(
            children: [
              SizedBox(width: graphWidth),
              _CommitGraphTextCell(
                shortHash(commit.hash) ?? '',
                width: 104,
                monospace: true,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _CommitGraphTextCell extends StatelessWidget {
  const _CommitGraphTextCell(
    this.text, {
    required this.width,
    this.monospace = false,
    this.color,
  });

  final String text;
  final double width;
  final bool monospace;
  final Color? color;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: width,
      child: Padding(
        padding: const EdgeInsets.only(right: 12),
        child: Text(
          fallbackText(text),
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(
            fontFamily: monospace ? 'monospace' : null,
            color: color,
          ),
        ),
      ),
    );
  }
}

class _CommitGraphPainter extends CustomPainter {
  const _CommitGraphPainter({
    required this.items,
    required this.laneCount,
    required this.scheme,
  });

  final List<pb.CommitGraphItem> items;
  final int laneCount;
  final ColorScheme scheme;

  @override
  void paint(Canvas canvas, Size size) {
    final linePaint = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2.4
      ..strokeCap = StrokeCap.round;

    for (final item in items) {
      for (final relation in item.relations) {
        final from = _pointFor(item.row, relation.fromLane);
        final to = relation.visible && relation.parentRow >= 0
            ? _pointFor(relation.parentRow, relation.toLane)
            : Offset(from.dx, math.min(size.height, from.dy + 34));
        if (to.dy <= from.dy) {
          continue;
        }

        linePaint.color = _commitLaneColor(relation.fromLane);
        if ((from.dx - to.dx).abs() < 0.5) {
          canvas.drawLine(
            Offset(from.dx, from.dy + _commitGraphNodeRadius),
            Offset(to.dx, to.dy - _commitGraphNodeRadius),
            linePaint,
          );
        } else {
          final midY = from.dy + ((to.dy - from.dy) * 0.42);
          final path = Path()
            ..moveTo(from.dx, from.dy + _commitGraphNodeRadius)
            ..cubicTo(
              from.dx,
              midY,
              to.dx,
              midY,
              to.dx,
              to.dy - _commitGraphNodeRadius,
            );
          canvas.drawPath(path, linePaint);
        }
      }
    }

    final nodeStroke = Paint()
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..color = scheme.surface;
    for (final item in items) {
      final center = _pointFor(item.row, item.lane);
      final color = _commitLaneColor(item.lane);
      final tentative = item.commit.states.contains('tentative');
      final fill = Paint()
        ..style = PaintingStyle.fill
        ..color = tentative ? scheme.surfaceContainerLowest : color;
      canvas.drawCircle(center, _commitGraphNodeRadius, fill);
      canvas.drawCircle(
        center,
        _commitGraphNodeRadius,
        nodeStroke..color = tentative ? color : scheme.surface,
      );
      if (tentative) {
        canvas.drawCircle(
          center,
          2.5,
          Paint()
            ..style = PaintingStyle.fill
            ..color = color,
        );
      }
    }
  }

  Offset _pointFor(int row, int lane) {
    final safeLane = lane < 0 ? 0 : lane;
    return Offset(
      _commitGraphLanePadding + safeLane * _commitGraphLaneGap,
      row * _commitGraphRowHeight + (_commitGraphRowHeight / 2),
    );
  }

  @override
  bool shouldRepaint(covariant _CommitGraphPainter oldDelegate) {
    return oldDelegate.items != items ||
        oldDelegate.laneCount != laneCount ||
        oldDelegate.scheme.brightness != scheme.brightness;
  }
}

class CommitDetailPanel extends StatelessWidget {
  const CommitDetailPanel({required this.commit, super.key});

  final pb.Commit? commit;

  @override
  Widget build(BuildContext context) {
    final selected = commit;
    if (selected == null) {
      return const SizedBox.shrink();
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const SectionHeading('Commit Details'),
        KeyValueWrap(
          items: [
            KeyValueItem('State', _commitStateLabel(selected)),
            KeyValueItem('Hash', selected.hash),
            KeyValueItem('Committer', selected.committer),
            KeyValueItem('Date', _commitDateLabel(selected)),
            KeyValueItem('Refs', _commitRefsLabel(selected)),
            KeyValueItem('Parents', _commitParentLabel(selected)),
          ],
        ),
        const SizedBox(height: 12),
        MonoPane(
          text: selected.message.nonEmpty ?? 'No commit message',
          minHeight: 82,
        ),
      ],
    );
  }
}

class StatusView extends StatelessWidget {
  const StatusView({super.key});

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final status = model.systemStatus;
    final hostAgent = model.supportsHostAgent && status?.hasHostAgent() == true
        ? status!.hostAgent
        : null;
    final network = model.networkRuntimeStatus;

    return DetailScroll(
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const SectionHeading('Embedded Core'),
          KeyValueWrap(
            items: [
              KeyValueItem('Status', _statusText(status?.coreStatus)),
              KeyValueItem('Work dir', status?.workDir),
              KeyValueItem('Capabilities', status?.capabilities),
              KeyValueItem(
                'P2P port',
                status == null || status.p2pPort == 0
                    ? null
                    : '${status.p2pPort}',
              ),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Network'),
          KeyValueWrap(
            items: [
              KeyValueItem(
                'Support',
                network == null
                    ? null
                    : (network.supported ? 'Available' : null),
              ),
              KeyValueItem('Status', network?.state),
              KeyValueItem('Message', network?.message),
            ],
          ),
          const SectionGap(),
          const SectionHeading('Endpoints'),
          RowsPanel<pb.CoreEndpoint>(
            rows: status?.endpoints.toList(growable: false) ?? const [],
            emptyTitle: 'No endpoints',
            height: 260,
            idForRow: (row) => '${row.kind}|${row.address}',
            columns: [
              RowColumn('Type', (row) => _endpointKind(row.kind)),
              RowColumn('Address', (row) => row.address, flex: 3),
              RowColumn('Active', (row) => row.active ? 'yes' : 'no'),
              RowColumn('Message', (row) => row.message, flex: 2),
            ],
          ),
          if (model.supportsHostAgent) ...[
            const SectionGap(),
            const SectionHeading('Host Agent'),
            KeyValueWrap(
              items: [
                KeyValueItem(
                  'Connectivity',
                  hostAgent == null
                      ? null
                      : (hostAgent.connected ? 'Connected' : 'Disconnected'),
                ),
                KeyValueItem('Socket', hostAgent?.socket),
                KeyValueItem('Message', hostAgent?.message),
              ],
            ),
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                FilledButton.icon(
                  onPressed: hostAgent?.connected == false && !model.isBusy
                      ? () => unawaited(model.run(() => model.startHostAgent()))
                      : null,
                  icon: const Icon(Icons.admin_panel_settings_outlined),
                  label: const Text('Start Host Agent'),
                ),
                OutlinedButton.icon(
                  onPressed: hostAgent?.connected == true && !model.isBusy
                      ? () => unawaited(model.run(() => model.stopHostAgent()))
                      : null,
                  icon: const Icon(Icons.stop_circle_outlined),
                  label: const Text('Stop Host Agent'),
                ),
                OutlinedButton.icon(
                  onPressed: model.isBusy
                      ? null
                      : () => unawaited(
                          model.run(() => model.refreshStatus(notify: false)),
                        ),
                  icon: const Icon(Icons.refresh),
                  label: const Text('Refresh'),
                ),
              ],
            ),
          ],
          if (model.supportsMobileTunnel) ...[
            const SectionGap(),
            const SectionHeading('Mobile Tunnel'),
            KeyValueWrap(
              items: [
                KeyValueItem(
                  'Profile',
                  model.mobileTunnelStatus.installed ? 'Installed' : null,
                ),
                KeyValueItem('Status', model.mobileTunnelStatus.label),
                KeyValueItem(
                  'Config',
                  shortHash(model.mobileTunnelStatus.configId, length: 18),
                ),
                KeyValueItem('Message', model.mobileTunnelStatus.detail),
              ],
            ),
          ],
        ],
      ),
    );
  }
}

class DetailScroll extends StatefulWidget {
  const DetailScroll({required this.child, super.key});

  final Widget child;

  @override
  State<DetailScroll> createState() => _DetailScrollState();
}

class _DetailScrollState extends State<DetailScroll> {
  final _controller = ScrollController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final compact = MediaQuery.sizeOf(context).width < 720;
    return PlatformSelectionArea(
      child: Scrollbar(
        controller: _controller,
        thumbVisibility: true,
        child: SingleChildScrollView(
          controller: _controller,
          primary: false,
          padding: EdgeInsets.all(compact ? 16 : 24),
          child: widget.child,
        ),
      ),
    );
  }
}

class PlatformSelectionArea extends StatelessWidget {
  const PlatformSelectionArea({required this.child, super.key});

  final Widget child;

  @override
  Widget build(BuildContext context) {
    return switch (defaultTargetPlatform) {
      TargetPlatform.iOS || TargetPlatform.android => child,
      _ => SelectionArea(child: child),
    };
  }
}

class SectionHeading extends StatelessWidget {
  const SectionHeading(this.text, {super.key});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Text(
        text,
        style: Theme.of(
          context,
        ).textTheme.titleMedium?.copyWith(fontWeight: FontWeight.w700),
      ),
    );
  }
}

class SectionGap extends StatelessWidget {
  const SectionGap({super.key});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 18),
      child: Divider(color: Theme.of(context).colorScheme.outlineVariant),
    );
  }
}

class FieldBox extends StatelessWidget {
  const FieldBox({required this.width, required this.child, super.key});

  final double width;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final availableWidth = MediaQuery.sizeOf(context).width - 32;
    final effectiveWidth = availableWidth < width ? availableWidth : width;
    return SizedBox(width: effectiveWidth, child: child);
  }
}

class FormWrap extends StatelessWidget {
  const FormWrap({required this.children, super.key});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 10,
      runSpacing: 10,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: children,
    );
  }
}

class CommandBar extends StatelessWidget {
  const CommandBar({required this.children, super.key});

  final List<Widget> children;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: children,
    );
  }
}

class ToggleBox extends StatelessWidget {
  const ToggleBox({
    required this.label,
    required this.value,
    required this.onChanged,
    super.key,
  });

  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      borderRadius: BorderRadius.circular(6),
      onTap: () => onChanged(!value),
      child: SizedBox(
        height: 40,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Checkbox(
              value: value,
              onChanged: (next) => onChanged(next ?? false),
            ),
            Text(label),
          ],
        ),
      ),
    );
  }
}

InputDecoration textField(String label) => InputDecoration(labelText: label);

class CommandButton extends StatelessWidget {
  const CommandButton({
    required this.label,
    required this.icon,
    required this.action,
    this.enabled = true,
    this.refresh = true,
    this.destructive = false,
    super.key,
  });

  final String label;
  final IconData icon;
  final bool enabled;
  final bool refresh;
  final bool destructive;
  final Future<void> Function(AppModel model) action;

  @override
  Widget build(BuildContext context) {
    final model = AppScope.of(context);
    final onPressed = enabled
        ? () => unawaited(
            model.run(() async {
              await action(model);
              if (refresh) {
                await model.refreshAll(notify: false);
              }
            }),
          )
        : null;

    if (destructive) {
      return OutlinedButton.icon(
        onPressed: onPressed,
        icon: Icon(icon),
        label: Text(label),
        style: OutlinedButton.styleFrom(
          foregroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }

    return OutlinedButton.icon(
      onPressed: onPressed,
      icon: Icon(icon),
      label: Text(label),
    );
  }
}

class KeyValueWrap extends StatelessWidget {
  const KeyValueWrap({required this.items, super.key});

  final List<KeyValueItem> items;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 18,
      runSpacing: 10,
      children: [for (final item in items) item],
    );
  }
}

class KeyValueItem extends StatelessWidget {
  const KeyValueItem(this.label, this.value, {super.key});

  final String label;
  final String? value;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return SizedBox(
      width: 260,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: Theme.of(context).textTheme.labelSmall?.copyWith(
              color: scheme.onSurfaceVariant,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 3),
          Text(
            fallbackText(value),
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
          ),
        ],
      ),
    );
  }
}

class RowColumn<T> {
  const RowColumn(this.label, this.value, {this.flex = 1});

  final String label;
  final String Function(T row) value;
  final int flex;
}

class RowsPanel<T> extends StatelessWidget {
  const RowsPanel({
    required this.rows,
    required this.emptyTitle,
    required this.idForRow,
    required this.columns,
    this.selectedId,
    this.onSelect,
    this.height = 230,
    super.key,
  });

  final List<T> rows;
  final String emptyTitle;
  final String Function(T row) idForRow;
  final List<RowColumn<T>> columns;
  final String? selectedId;
  final ValueChanged<String>? onSelect;
  final double height;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Container(
      height: height,
      decoration: BoxDecoration(
        border: Border.all(color: scheme.outlineVariant),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        children: [
          Container(
            height: 36,
            padding: const EdgeInsets.symmetric(horizontal: 10),
            decoration: BoxDecoration(
              color: scheme.surfaceContainerHighest,
              borderRadius: const BorderRadius.vertical(
                top: Radius.circular(7),
              ),
            ),
            child: Row(
              children: [
                for (final column in columns)
                  Expanded(
                    flex: column.flex,
                    child: Text(
                      column.label,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: Theme.of(context).textTheme.labelMedium?.copyWith(
                        color: scheme.onSurfaceVariant,
                        fontWeight: FontWeight.w800,
                      ),
                    ),
                  ),
              ],
            ),
          ),
          Divider(height: 1, color: scheme.outlineVariant),
          Expanded(
            child: rows.isEmpty
                ? Center(
                    child: Text(
                      emptyTitle,
                      style: TextStyle(color: scheme.onSurfaceVariant),
                    ),
                  )
                : ListView.builder(
                    primary: false,
                    itemCount: rows.length,
                    itemExtent: 36,
                    itemBuilder: (context, index) {
                      final row = rows[index];
                      final id = _uniqueRowId(row, index);
                      final selected = selectedId == id;
                      return InkWell(
                        key: ValueKey(id),
                        onTap: onSelect == null ? null : () => onSelect!(id),
                        child: Container(
                          padding: const EdgeInsets.symmetric(horizontal: 10),
                          color: selected
                              ? scheme.primaryContainer.withValues(alpha: 0.56)
                              : Colors.transparent,
                          child: Row(
                            children: [
                              for (final column in columns)
                                Expanded(
                                  flex: column.flex,
                                  child: Padding(
                                    padding: const EdgeInsets.only(right: 12),
                                    child: Text(
                                      fallbackText(column.value(row)),
                                      maxLines: 1,
                                      overflow: TextOverflow.ellipsis,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }

  String _uniqueRowId(T row, int index) {
    final id = idForRow(row).nonEmpty;
    return id ?? 'row-$index';
  }
}

class MonoPane extends StatelessWidget {
  const MonoPane({required this.text, this.minHeight = 120, super.key});

  final String text;
  final double minHeight;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return Container(
      constraints: BoxConstraints(minHeight: minHeight),
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerLowest,
        border: Border.all(color: scheme.outlineVariant),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(text, style: const TextStyle(fontFamily: 'monospace')),
    );
  }
}

class OutputPane extends StatefulWidget {
  const OutputPane({
    required this.text,
    required this.onChanged,
    this.minHeight = 150,
    super.key,
  });

  final String text;
  final ValueChanged<String> onChanged;
  final double minHeight;

  @override
  State<OutputPane> createState() => _OutputPaneState();
}

class _OutputPaneState extends State<OutputPane> {
  late final TextEditingController controller;

  @override
  void initState() {
    super.initState();
    controller = TextEditingController(text: widget.text);
  }

  @override
  void didUpdateWidget(covariant OutputPane oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.text != controller.text) {
      final previousOffset = controller.selection.baseOffset;
      final offset = previousOffset < 0
          ? 0
          : (previousOffset > widget.text.length
                ? widget.text.length
                : previousOffset);
      controller.value = TextEditingValue(
        text: widget.text,
        selection: TextSelection.collapsed(offset: offset),
      );
    }
  }

  @override
  void dispose() {
    controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: widget.minHeight,
      child: TextField(
        controller: controller,
        expands: true,
        maxLines: null,
        minLines: null,
        onChanged: widget.onChanged,
        style: const TextStyle(fontFamily: 'monospace'),
        decoration: const InputDecoration(
          alignLabelWithHint: true,
          labelText: 'Output',
        ),
      ),
    );
  }
}

class ResponsivePair extends StatelessWidget {
  const ResponsivePair({required this.first, required this.second, super.key});

  final Widget first;
  final Widget second;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 900) {
          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [first, const SizedBox(height: 18), second],
          );
        }
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Expanded(child: first),
            const SizedBox(width: 18),
            Expanded(child: second),
          ],
        );
      },
    );
  }
}

String _taskTitle(pb.Task task) {
  return task.title.nonEmpty ?? task.stream.nonEmpty ?? task.id;
}

String _taskStatusText(pb.Task task) {
  return _taskStatusValue(task.status);
}

String _taskEventStatusText(pb.TaskEvent event) {
  return _taskStatusValue(event.status);
}

String _taskSubjectLabel(pb.Task task) {
  final type = task.subjectType.nonEmpty;
  final id = task.subjectId.nonEmpty;
  if (type == null) {
    return id ?? '';
  }
  if (id == null) {
    return type;
  }
  return '$type $id';
}

String _taskAttemptsLabel(pb.Task task) {
  if (task.maxAttempts <= 0) {
    return '${task.attempts}';
  }
  return '${task.attempts}/${task.maxAttempts}';
}

String _taskProgressLabel(int progress) => '$progress%';

String _taskStatusValue(String status) {
  final value = status.nonEmpty;
  if (value == null) {
    return 'n/a';
  }
  return '${value[0].toUpperCase()}${value.substring(1)}';
}

Color _taskStatusColor(ColorScheme scheme, String status) {
  return switch (status.toLowerCase()) {
    'succeeded' => Colors.green.shade700,
    'running' => scheme.primary,
    'pending' => Colors.orange.shade800,
    'failed' => scheme.error,
    'cancelled' => scheme.onSurfaceVariant,
    _ => scheme.onSurfaceVariant,
  };
}

double _taskProgressValue(int progress) {
  return progress.clamp(0, 100).toDouble() / 100;
}

String _taskDateLabel(String value) {
  final trimmed = value.nonEmpty;
  if (trimmed == null) {
    return 'n/a';
  }
  final parsed = DateTime.tryParse(trimmed);
  if (parsed == null) {
    return trimmed;
  }
  final local = parsed.toLocal();
  final y = local.year.toString().padLeft(4, '0');
  final m = local.month.toString().padLeft(2, '0');
  final d = local.day.toString().padLeft(2, '0');
  final h = local.hour.toString().padLeft(2, '0');
  final min = local.minute.toString().padLeft(2, '0');
  return '$y-$m-$d $h:$min';
}

String _taskJson(String value) {
  return value.nonEmpty ?? 'n/a';
}

String? _statusText(String? status) {
  final value = status?.nonEmpty;
  if (value == null) {
    return null;
  }
  return '${value[0].toUpperCase()}${value.substring(1)}';
}

String _endpointKind(String value) {
  return switch (value) {
    'api-socket' => 'API socket',
    'embedded-api' => 'Embedded API',
    'p2p-port' => 'P2P port',
    'p2p-listen' => 'P2P listen',
    _ => value,
  };
}

Map<String, pb.InstanceDeployField> _deployFieldsByName(
  pb.GetInstanceDeployOptionsResponse? options,
) {
  return {
    for (final field in options?.fields ?? const <pb.InstanceDeployField>[])
      field.name: field,
  };
}

bool _hasOptions(pb.InstanceDeployField? field) {
  return field != null && field.options.isNotEmpty;
}

String _preferredFieldValue(pb.InstanceDeployField? field, String current) {
  if (field == null || field.options.isEmpty) {
    return '';
  }
  if (field.options.any((option) => option.value == current)) {
    return current;
  }
  if (field.value.nonEmpty != null &&
      field.options.any((option) => option.value == field.value)) {
    return field.value;
  }
  return field.options.first.value;
}

String? _dropdownValue(pb.InstanceDeployField? field, String value) {
  if (field == null || field.options.isEmpty) {
    return null;
  }
  if (field.options.any((option) => option.value == value)) {
    return value;
  }
  if (field.value.nonEmpty != null &&
      field.options.any((option) => option.value == field.value)) {
    return field.value;
  }
  return field.options.first.value;
}

pb.InstanceDeployFieldOption? _selectedOption(
  pb.InstanceDeployField? field,
  String value,
) {
  if (field == null) {
    return null;
  }
  for (final option in field.options) {
    if (option.value == value) {
      return option;
    }
  }
  return null;
}

String _optionDisplay(pb.InstanceDeployFieldOption? option) {
  if (option == null) {
    return 'n/a';
  }
  final label = option.label.nonEmpty ?? option.value;
  final description = option.description.nonEmpty;
  if (description == null) {
    return label;
  }
  return '$label - $description';
}

String _routeOutput(pb.ExitRoute route) {
  final lines = <String>[
    'Routing local traffic through ${route.instanceName} (${route.publicIp})',
  ];
  if (route.cidrs.isNotEmpty) {
    lines.add('Routed CIDRs: ${route.cidrs.join(', ')}');
  }
  if (route.dnsServer.nonEmpty != null) {
    lines.add('Forwarding external DNS through ${route.dnsServer}');
  }
  return lines.join('\n');
}

String _peerConnection(pb.RuntimePeerStatus peer) {
  if (peer.connected) {
    return 'Connected';
  }
  if (peer.dialable) {
    return 'Dialable';
  }
  return 'Not connected';
}

String _peerLabel(pb.RuntimePeerStatus peer, String? localPeerId) {
  final label = shortHash(peer.peerId) ?? '';
  if (_isSelfPeer(peer, localPeerId)) {
    return '$label (self)';
  }
  return label;
}

String _peerRoles(pb.RuntimePeerStatus peer, String? localPeerId) {
  final roles = <String>[];
  if (_isSelfPeer(peer, localPeerId)) {
    roles.add('self');
  }
  if (peer.stateProvider) {
    roles.add('provider');
  }
  if (peer.relayOnly) {
    roles.add('relay-only');
  }
  if (peer.ignored) {
    roles.add('ignored');
  }
  return roles.isEmpty ? 'peer' : roles.join(', ');
}

bool _isSelfPeer(pb.RuntimePeerStatus peer, String? localPeerId) {
  final local = localPeerId?.nonEmpty;
  return local != null && peer.peerId == local;
}

String _peerCompatibility(pb.RuntimePeerStatus peer) {
  if (peer.incompatible) {
    return 'Incompatible';
  }
  if (peer.compatible) {
    return 'Compatible';
  }
  return 'Unknown';
}

String _peerReason(pb.RuntimePeerStatus peer) {
  final reason = peer.reason.nonEmpty;
  if (reason != null) {
    return reason;
  }
  final errors =
      peer.lastDialErrors.values
          .where((value) => value.nonEmpty != null)
          .toList(growable: false)
        ..sort();
  return errors.isEmpty ? 'n/a' : errors.first;
}

String _commitStateLabel(pb.Commit commit) {
  if (commit.states.isEmpty) {
    return 'unknown';
  }
  return commit.states
      .map(
        (state) => state.isEmpty
            ? state
            : '${state[0].toUpperCase()}${state.substring(1)}',
      )
      .join(', ');
}

String? _commitRefsLabel(pb.Commit commit) {
  final refs = commit.refs.where((ref) => ref.nonEmpty != null).toList();
  return refs.isEmpty ? null : refs.join(', ');
}

String? _commitParentLabel(pb.Commit commit) {
  final parents = commit.parentHashes
      .where((hash) => hash.nonEmpty != null)
      .map((hash) => shortHash(hash) ?? hash)
      .toList();
  return parents.isEmpty ? null : parents.join(', ');
}

String? _commitDateLabel(pb.Commit commit) {
  final unixSeconds = commit.dateUnix.toInt();
  if (unixSeconds <= 0) {
    return null;
  }
  final date = DateTime.fromMillisecondsSinceEpoch(
    unixSeconds * 1000,
    isUtc: true,
  ).toLocal();
  final y = date.year.toString().padLeft(4, '0');
  final m = date.month.toString().padLeft(2, '0');
  final d = date.day.toString().padLeft(2, '0');
  final h = date.hour.toString().padLeft(2, '0');
  final min = date.minute.toString().padLeft(2, '0');
  return '$y-$m-$d $h:$min';
}

int _commitGraphLaneCount(
  pb.CommitGraph? graph,
  List<pb.CommitGraphItem> items,
) {
  var laneCount = graph?.laneCount ?? 0;
  for (final item in items) {
    laneCount = math.max(laneCount, item.lane + 1);
    for (final relation in item.relations) {
      laneCount = math.max(laneCount, relation.fromLane + 1);
      laneCount = math.max(laneCount, relation.toLane + 1);
    }
  }
  return laneCount < 1 ? 1 : laneCount;
}

double _commitGraphWidth(int laneCount) {
  final lanes = laneCount < 1 ? 1 : laneCount;
  final width =
      (_commitGraphLanePadding * 2) + ((lanes - 1) * _commitGraphLaneGap) + 16;
  return math.min(180, math.max(58, width.toDouble()));
}

double _commitGraphContentWidth(double graphWidth) {
  return graphWidth + 104;
}

Color _commitLaneColor(int lane) {
  final safeLane = lane < 0 ? 0 : lane;
  return _commitGraphLaneColors[safeLane % _commitGraphLaneColors.length];
}

double _sqlTableWidth(int columnCount) {
  return (columnCount <= 0 ? 1 : columnCount) * 180;
}

String _sqlCellLabel(pb.SqlRow row, int column) {
  if (column < 0 || column >= row.cells.length) {
    return '';
  }
  final cell = row.cells[column];
  return cell.isNull ? 'NULL' : cell.value;
}

String _sqlResultTsv(pb.ExecuteSqlResponse? result) {
  if (result == null) {
    return '';
  }
  if (result.columns.isEmpty) {
    return result.message.nonEmpty ?? '${result.rowsAffected} rows affected';
  }
  final lines = <String>[
    result.columns.map(_tsvValue).join('\t'),
    for (final row in result.rows)
      [
        for (var column = 0; column < result.columns.length; column++)
          _tsvValue(_sqlCellLabel(row, column)),
      ].join('\t'),
  ];
  if (result.truncated) {
    lines.add('# truncated');
  }
  return lines.join('\n');
}

String _tsvValue(String value) {
  final normalized = value.replaceAll('\r\n', '\n').replaceAll('\r', '\n');
  if (!normalized.contains('\t') &&
      !normalized.contains('\n') &&
      !normalized.contains('"')) {
    return normalized;
  }
  return '"${normalized.replaceAll('"', '""')}"';
}
