import 'dart:io';

import 'package:flutter/material.dart';
import 'package:protos_flutter/protos_flutter.dart';

void main() {
  final dataDir = _env('PROTOS_FLUTTER_DATA_DIR', _defaultDataDir());
  runApp(
    ProtosFlutterApp(
      bridgeConfig: ProtosBridgeConfig(
        configFile: _env('PROTOS_FLUTTER_CONFIG_FILE', '$dataDir/protos.yaml'),
        dataDir: dataDir,
        capabilities: _env('PROTOS_FLUTTER_CAPABILITIES', 'default'),
        logLevel: _env('PROTOS_FLUTTER_LOG_LEVEL', 'info'),
      ),
      onDaemonStartFailed: (error) {
        stderr.writeln('Protos macOS startup failed: $error');
        exit(1);
      },
    ),
  );
}

String _env(String key, String fallback) {
  final value = Platform.environment[key]?.trim();
  if (value == null || value.isEmpty) {
    return fallback;
  }
  return value;
}

String _defaultDataDir() {
  final home = Platform.environment['HOME']?.trim();
  if (home == null || home.isEmpty) {
    return '.protos';
  }
  return '$home/.protos';
}
