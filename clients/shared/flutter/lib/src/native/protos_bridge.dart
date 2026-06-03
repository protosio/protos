import 'dart:async';
import 'dart:convert';
import 'dart:ffi';
import 'dart:io';
import 'dart:isolate';
import 'dart:typed_data';

import 'package:ffi/ffi.dart';
import 'package:path/path.dart' as p;
import 'package:protobuf/protobuf.dart';

import '../generated/apic/proto/apic.pb.dart' as pb;

final class NativeProtosResult extends Struct {
  external Pointer<Void> data;

  @Int64()
  external int len;

  external Pointer<Utf8> err;
}

final class NativeProtosWatchResult extends Struct {
  @Int64()
  external int watchId;

  external Pointer<Utf8> err;
}

typedef _ProtosStartNative = Pointer<Utf8> Function(Pointer<Utf8>);
typedef _ProtosStartDart = Pointer<Utf8> Function(Pointer<Utf8>);

typedef _ProtosStopNative = Pointer<Utf8> Function();
typedef _ProtosStopDart = Pointer<Utf8> Function();

typedef _ProtosCallNative =
    NativeProtosResult Function(Pointer<Utf8>, Pointer<Void>, Int64);
typedef _ProtosCallDart =
    NativeProtosResult Function(Pointer<Utf8>, Pointer<Void>, int);

typedef _ProtosFreeNative = Void Function(Pointer<Void>);
typedef _ProtosFreeDart = void Function(Pointer<Void>);

typedef _ProtosCancelWatchNative = Void Function(Int64);
typedef _ProtosCancelWatchDart = void Function(int);

typedef _ProtosCancelAllWatchesNative = Void Function();
typedef _ProtosCancelAllWatchesDart = void Function();

typedef _WatchBytesCallbackNative =
    Void Function(Pointer<Void>, Pointer<Void>, Int64, Pointer<Utf8>);

typedef _ProtosWatchChangesBytesNative =
    NativeProtosWatchResult Function(
      Pointer<Void>,
      Int64,
      Pointer<Void>,
      Pointer<NativeFunction<_WatchBytesCallbackNative>>,
    );
typedef _ProtosWatchChangesBytesDart =
    NativeProtosWatchResult Function(
      Pointer<Void>,
      int,
      Pointer<Void>,
      Pointer<NativeFunction<_WatchBytesCallbackNative>>,
    );

class ProtosBridgeConfig {
  const ProtosBridgeConfig({
    this.configFile = 'protos.yaml',
    this.dataDir = '',
    this.capabilities = 'default',
    this.logLevel = 'info',
  });

  final String configFile;
  final String dataDir;
  final String capabilities;
  final String logLevel;

  String toJsonString() {
    final effectiveDataDir = _effectiveDataDir();
    return jsonEncode({
      'config_file': _effectiveConfigFile(effectiveDataDir),
      'data_dir': effectiveDataDir,
      'capabilities': _effectiveCapabilities(),
      'log_level': logLevel,
    });
  }

  String _effectiveDataDir() {
    if (dataDir.trim().isNotEmpty || !Platform.isIOS) {
      return dataDir;
    }
    final home = Platform.environment['HOME']?.trim();
    if (home != null && home.isNotEmpty) {
      return p.join(home, 'Documents', 'Protos');
    }
    return p.join(Directory.systemTemp.parent.path, 'Documents', 'Protos');
  }

  String _effectiveConfigFile(String effectiveDataDir) {
    if (!Platform.isIOS || configFile != 'protos.yaml') {
      return configFile;
    }
    return p.join(effectiveDataDir, 'protos.yaml');
  }

  String _effectiveCapabilities() {
    if (!Platform.isIOS || capabilities != 'default') {
      return capabilities;
    }
    return 'default,no-api,no-network,no-app-runtime';
  }
}

class ProtosBridgeException implements Exception {
  const ProtosBridgeException(this.message);

  final String message;

  @override
  String toString() => message;
}

class NativeProtosBridge {
  NativeProtosBridge() : _bindings = _NativeBindings.load();

  final _NativeBindings _bindings;
  final _worker = _BridgeWorker();
  final _watchStates = <int, _WatchState>{};
  late final NativeCallable<_WatchBytesCallbackNative> _watchCallback =
      NativeCallable.listener(_handleWatchEvent);
  var _started = false;
  var _disposed = false;

  Future<void> start({
    ProtosBridgeConfig config = const ProtosBridgeConfig(),
  }) async {
    if (_disposed) {
      throw const ProtosBridgeException('The embedded daemon was disposed.');
    }
    if (_started) {
      return;
    }
    final json = config.toJsonString();
    await _worker.startBridge(json);
    _started = true;
  }

  Future<void> stop() async {
    if (!_started) {
      return;
    }
    for (final state in _watchStates.values.toList(growable: false)) {
      state.cancel(_bindings);
    }
    _watchStates.clear();
    await _worker.stopBridge();
    _started = false;
  }

  Future<Response> call<
    Request extends GeneratedMessage,
    Response extends GeneratedMessage
  >(String method, Request request, Response Function() createResponse) async {
    if (!_started) {
      throw const ProtosBridgeException('The embedded daemon is not running.');
    }
    final requestData = Uint8List.fromList(request.writeToBuffer());
    final responseData = await _worker.callBridge(method, requestData);
    return createResponse()..mergeFromBuffer(responseData);
  }

  Stream<pb.WatchChangesResponse> watchChanges({
    bool includeSnapshot = false,
    int heartbeatIntervalMs = 0,
  }) {
    if (!_started) {
      return Stream.error(
        const ProtosBridgeException('The embedded daemon is not running.'),
      );
    }

    final request = pb.WatchChangesRequest(
      includeSnapshot: includeSnapshot,
      heartbeatIntervalMs: heartbeatIntervalMs,
    );
    final requestData = request.writeToBuffer();
    final requestPointer = _copyBytes(requestData);
    final contextPointer = calloc<Int64>();
    final state = _WatchState(contextPointer);
    _watchStates[contextPointer.address] = state;

    late final StreamController<pb.WatchChangesResponse> controller;
    controller = StreamController<pb.WatchChangesResponse>(
      onCancel: () {
        state.cancel(_bindings);
        _watchStates.remove(contextPointer.address);
      },
    );
    state.controller = controller;

    try {
      final result = _bindings.watchChangesBytes(
        requestPointer.cast<Void>(),
        requestData.length,
        contextPointer.cast<Void>(),
        _watchCallback.nativeFunction,
      );
      _bindings.freeRequestPointer(requestPointer);
      if (result.err != nullptr) {
        final message = _bindings.takeNativeString(result.err);
        state.finishError(ProtosBridgeException(message), _bindings);
        _watchStates.remove(contextPointer.address);
      } else {
        state.watchId = result.watchId;
      }
    } catch (error, stackTrace) {
      _bindings.freeRequestPointer(requestPointer);
      state.finishError(error, _bindings, stackTrace);
      _watchStates.remove(contextPointer.address);
    }

    return controller.stream;
  }

  void dispose() {
    if (_disposed) {
      return;
    }
    _disposed = true;
    _bindings.cancelAllWatches();
    for (final state in _watchStates.values.toList(growable: false)) {
      state.cancel(_bindings);
    }
    _watchStates.clear();
    _watchCallback.close();
    unawaited(_worker.dispose());
  }

  void _handleWatchEvent(
    Pointer<Void> context,
    Pointer<Void> data,
    int len,
    Pointer<Utf8> err,
  ) {
    final state = _watchStates[context.address];
    if (state == null) {
      _bindings.releaseWatchPayload(data, err);
      return;
    }

    if (err != nullptr) {
      final message = _bindings.takeNativeString(err);
      state.finishError(ProtosBridgeException(message), _bindings);
      _watchStates.remove(context.address);
      if (data != nullptr) {
        _bindings.free(data);
      }
      return;
    }

    if (len == 0 && data == nullptr) {
      state.finish(_bindings);
      _watchStates.remove(context.address);
      return;
    }

    try {
      final bytes = Uint8List.fromList(data.cast<Uint8>().asTypedList(len));
      _bindings.free(data);
      state.controller?.add(pb.WatchChangesResponse.fromBuffer(bytes));
    } catch (error, stackTrace) {
      state.finishError(error, _bindings, stackTrace);
      _watchStates.remove(context.address);
    }
  }
}

class _BridgeWorker {
  Isolate? _isolate;
  SendPort? _sendPort;
  Future<void>? _starting;
  var _disposed = false;

  Future<void> startBridge(String configJson) async {
    await _send('start', configJson);
  }

  Future<void> stopBridge() async {
    await _send('stop', null);
  }

  Future<Uint8List> callBridge(String method, Uint8List requestData) async {
    final response = await _send('call', [method, requestData]);
    return response! as Uint8List;
  }

  Future<void> dispose() async {
    if (_disposed) {
      return;
    }
    _disposed = true;
    final sendPort = _sendPort;
    if (sendPort != null) {
      final reply = ReceivePort();
      sendPort.send(['close', null, reply.sendPort]);
      await reply.first.timeout(
        const Duration(seconds: 1),
        onTimeout: () => null,
      );
      reply.close();
    }
    _sendPort = null;
    _isolate?.kill(priority: Isolate.immediate);
    _isolate = null;
  }

  Future<Object?> _send(String operation, Object? payload) async {
    if (_disposed) {
      throw const ProtosBridgeException('The embedded daemon was disposed.');
    }
    await _ensureStarted();
    final reply = ReceivePort();
    _sendPort!.send([operation, payload, reply.sendPort]);
    try {
      final response = await reply.first as List<Object?>;
      if (response.first == true) {
        return response.length > 1 ? response[1] : null;
      }
      throw ProtosBridgeException(response[1] as String);
    } finally {
      reply.close();
    }
  }

  Future<void> _ensureStarted() {
    if (_sendPort != null) {
      return Future.value();
    }
    return _starting ??= _start();
  }

  Future<void> _start() async {
    final ready = ReceivePort();
    try {
      _isolate = await Isolate.spawn(_bridgeWorkerMain, ready.sendPort);
      _sendPort = await ready.first as SendPort;
    } finally {
      ready.close();
      _starting = null;
    }
  }
}

void _bridgeWorkerMain(SendPort readyPort) {
  final commands = ReceivePort();
  readyPort.send(commands.sendPort);

  commands.listen((Object? rawMessage) {
    final message = rawMessage as List<Object?>;
    final operation = message[0] as String;
    final payload = message[1];
    final replyPort = message[2] as SendPort;

    try {
      switch (operation) {
        case 'start':
          _startBridgeSync(payload! as String);
          replyPort.send([true, null]);
        case 'stop':
          _stopBridgeSync();
          replyPort.send([true, null]);
        case 'call':
          final parts = payload! as List<Object?>;
          final response = _callBridgeSync(
            parts[0] as String,
            parts[1] as Uint8List,
          );
          replyPort.send([true, response]);
        case 'close':
          commands.close();
          replyPort.send([true, null]);
        default:
          throw ProtosBridgeException(
            'Unknown bridge worker operation: $operation',
          );
      }
    } catch (error) {
      replyPort.send([false, error.toString()]);
    }
  });
}

class _WatchState {
  _WatchState(this.contextPointer);

  final Pointer<Int64> contextPointer;
  StreamController<pb.WatchChangesResponse>? controller;
  int watchId = 0;
  var _closed = false;

  void cancel(_NativeBindings bindings) {
    if (_closed) {
      return;
    }
    _closed = true;
    if (watchId > 0) {
      bindings.cancelWatch(watchId);
      watchId = 0;
    }
    calloc.free(contextPointer);
    unawaited(controller?.close());
  }

  void finish(_NativeBindings bindings) {
    if (_closed) {
      return;
    }
    _closed = true;
    calloc.free(contextPointer);
    unawaited(controller?.close());
  }

  void finishError(
    Object error,
    _NativeBindings bindings, [
    StackTrace? stackTrace,
  ]) {
    if (_closed) {
      return;
    }
    _closed = true;
    calloc.free(contextPointer);
    controller?.addError(error, stackTrace);
    unawaited(controller?.close());
  }
}

class _NativeBindings {
  _NativeBindings._(this.library)
    : start = library.lookupFunction<_ProtosStartNative, _ProtosStartDart>(
        'ProtosStart',
      ),
      stop = library.lookupFunction<_ProtosStopNative, _ProtosStopDart>(
        'ProtosStop',
      ),
      call = library.lookupFunction<_ProtosCallNative, _ProtosCallDart>(
        'ProtosCall',
      ),
      watchChangesBytes = library
          .lookupFunction<
            _ProtosWatchChangesBytesNative,
            _ProtosWatchChangesBytesDart
          >('ProtosWatchChangesBytes'),
      cancelWatch = library
          .lookupFunction<_ProtosCancelWatchNative, _ProtosCancelWatchDart>(
            'ProtosCancelWatch',
          ),
      cancelAllWatches = library
          .lookupFunction<
            _ProtosCancelAllWatchesNative,
            _ProtosCancelAllWatchesDart
          >('ProtosCancelAllWatches'),
      free = library.lookupFunction<_ProtosFreeNative, _ProtosFreeDart>(
        'ProtosFree',
      );

  final DynamicLibrary library;
  final _ProtosStartDart start;
  final _ProtosStopDart stop;
  final _ProtosCallDart call;
  final _ProtosWatchChangesBytesDart watchChangesBytes;
  final _ProtosCancelWatchDart cancelWatch;
  final _ProtosCancelAllWatchesDart cancelAllWatches;
  final _ProtosFreeDart free;

  static _NativeBindings load() {
    if (Platform.isIOS) {
      return _NativeBindings._(DynamicLibrary.process());
    }
    return _NativeBindings._(DynamicLibrary.open(_resolveLibraryPath()));
  }

  void freeRequestPointer(Pointer<Uint8> pointer) {
    if (pointer != nullptr) {
      calloc.free(pointer);
    }
  }

  String takeNativeString(Pointer<Utf8> pointer) {
    final value = pointer.toDartString();
    free(pointer.cast<Void>());
    return value;
  }

  Uint8List takeNativeBytes(NativeProtosResult result) {
    if (result.len == 0) {
      return Uint8List(0);
    }
    if (result.data == nullptr) {
      throw const ProtosBridgeException(
        'The embedded daemon returned an invalid response.',
      );
    }
    final bytes = Uint8List.fromList(
      result.data.cast<Uint8>().asTypedList(result.len),
    );
    free(result.data);
    return bytes;
  }

  void releaseWatchPayload(Pointer<Void> data, Pointer<Utf8> err) {
    if (data != nullptr) {
      free(data);
    }
    if (err != nullptr) {
      free(err.cast<Void>());
    }
  }
}

void _startBridgeSync(String configJson) {
  final bindings = _NativeBindings.load();
  final configPointer = configJson.toNativeUtf8();
  try {
    final error = bindings.start(configPointer);
    if (error != nullptr) {
      throw ProtosBridgeException(bindings.takeNativeString(error));
    }
  } finally {
    calloc.free(configPointer);
  }
}

void _stopBridgeSync() {
  final bindings = _NativeBindings.load();
  final error = bindings.stop();
  if (error != nullptr) {
    throw ProtosBridgeException(bindings.takeNativeString(error));
  }
}

Uint8List _callBridgeSync(String method, Uint8List requestData) {
  final bindings = _NativeBindings.load();
  final methodPointer = method.toNativeUtf8();
  final requestPointer = _copyBytes(requestData);
  try {
    final result = bindings.call(
      methodPointer,
      requestPointer.cast<Void>(),
      requestData.length,
    );
    if (result.err != nullptr) {
      throw ProtosBridgeException(bindings.takeNativeString(result.err));
    }
    return bindings.takeNativeBytes(result);
  } finally {
    calloc.free(methodPointer);
    bindings.freeRequestPointer(requestPointer);
  }
}

Pointer<Uint8> _copyBytes(List<int> bytes) {
  if (bytes.isEmpty) {
    return nullptr;
  }
  final pointer = calloc<Uint8>(bytes.length);
  pointer.asTypedList(bytes.length).setAll(0, bytes);
  return pointer;
}

String _resolveLibraryPath() {
  if (!Platform.isMacOS) {
    throw const ProtosBridgeException(
      'The embedded Protos bridge is only available on macOS.',
    );
  }

  final executableDir = p.dirname(Platform.resolvedExecutable);
  final candidates = <String>[
    p.normalize(p.join(executableDir, '..', 'Frameworks', 'libprotos.dylib')),
    p.normalize(
      p.join(
        Directory.current.path,
        'macos',
        'Runner',
        'Frameworks',
        'libprotos.dylib',
      ),
    ),
    p.normalize(
      p.join(Directory.current.path, 'Runner', 'Frameworks', 'libprotos.dylib'),
    ),
  ];

  for (final candidate in candidates) {
    if (File(candidate).existsSync()) {
      return candidate;
    }
  }

  throw ProtosBridgeException(
    'Could not find libprotos.dylib. Looked in: ${candidates.join(', ')}',
  );
}
