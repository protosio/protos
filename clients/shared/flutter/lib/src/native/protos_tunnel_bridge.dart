import 'dart:io' show Platform;

import 'package:flutter/services.dart';

import '../generated/apic/proto/apic.pb.dart' as pb;

class MobileTunnelStatus {
  const MobileTunnelStatus({
    required this.supported,
    required this.installed,
    required this.status,
    this.detail = '',
    this.configId = '',
  });

  const MobileTunnelStatus.unsupported()
    : supported = false,
      installed = false,
      status = 'unsupported',
      detail = '',
      configId = '';

  final bool supported;
  final bool installed;
  final String status;
  final String detail;
  final String configId;

  String get label {
    if (!supported) {
      return 'Unsupported';
    }
    if (!installed) {
      return 'Not installed';
    }
    return status.isEmpty ? 'Unknown' : status;
  }

  factory MobileTunnelStatus.fromMap(Map<String, Object?> map) {
    return MobileTunnelStatus(
      supported: map['supported'] == true,
      installed: map['installed'] == true,
      status: (map['status'] as String?) ?? '',
      detail: (map['detail'] as String?) ?? '',
      configId: (map['configId'] as String?) ?? '',
    );
  }
}

class ProtosTunnelBridge {
  ProtosTunnelBridge();

  static const _channel = MethodChannel('io.protos/tunnel');

  bool get isSupported => Platform.isIOS;

  Future<void> installOrUpdateTunnel(pb.MobileTunnelConfig config) async {
    _requireIOS();
    await _channel.invokeMethod<void>(
      'installOrUpdateTunnel',
      _configToMap(config),
    );
  }

  Future<void> startTunnel() async {
    _requireIOS();
    await _channel.invokeMethod<void>('startTunnel');
  }

  Future<void> stopTunnel() async {
    _requireIOS();
    await _channel.invokeMethod<void>('stopTunnel');
  }

  Future<MobileTunnelStatus> tunnelStatus() async {
    if (!isSupported) {
      return const MobileTunnelStatus.unsupported();
    }
    final result = await _channel.invokeMethod<Map<dynamic, dynamic>>(
      'tunnelStatus',
    );
    final map = <String, Object?>{};
    result?.forEach((key, value) => map['$key'] = value);
    return MobileTunnelStatus.fromMap(map);
  }

  Future<Map<String, Object?>> sendProviderMessage(
    Map<String, Object?> message,
  ) async {
    _requireIOS();
    final result = await _channel.invokeMethod<Map<dynamic, dynamic>>(
      'sendProviderMessage',
      message,
    );
    final map = <String, Object?>{};
    result?.forEach((key, value) => map['$key'] = value);
    return map;
  }

  void _requireIOS() {
    if (!isSupported) {
      throw UnsupportedError('The Protos mobile tunnel bridge is iOS-only.');
    }
  }

  Map<String, Object?> _configToMap(pb.MobileTunnelConfig config) {
    return {
      'configId': config.configId,
      'generatedAtUnix': config.generatedAtUnix.toInt(),
      'instanceId': config.instanceId,
      'instanceName': config.instanceName,
      'peerPublicKey': config.peerPublicKey,
      'peerEndpoint': config.peerEndpoint,
      'interfaceAddresses': config.interfaceAddresses.toList(growable: false),
      'dnsServers': config.dnsServers.toList(growable: false),
      'includedRoutes': config.includedRoutes.toList(growable: false),
      'excludedRoutes': config.excludedRoutes.toList(growable: false),
      'mtu': config.mtu,
      'allowedIps': config.allowedIps.toList(growable: false),
      'persistentKeepaliveSeconds': config.persistentKeepaliveSeconds,
      'keychainAccessGroup': config.keychainAccessGroup,
      'keychainAccount': config.keychainAccount,
      'wireguardPrivateKey': config.wireguardPrivateKey,
    };
  }
}
