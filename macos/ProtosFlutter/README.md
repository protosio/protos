# Protos Flutter

Flutter macOS desktop client for the embedded Protos daemon.

The app uses the same protobuf API surface as `macos/ProtosMac` and loads the Go bridge through `dart:ffi`:

- `ProtosStart(configJSON)`
- `ProtosCall(method, requestBytes)`
- `ProtosWatchChangesBytes(requestBytes, callback)`
- `ProtosCancelWatch(watchID)`
- `ProtosStop()`
- `ProtosFree(ptr)`

`scripts/build-native.sh` builds `cmd/protos-macos-bridge` as `libprotos.dylib` and the Xcode build embeds it into `ProtosFlutter.app/Contents/Frameworks`.

## Build

```sh
macos/ProtosFlutter/scripts/build.sh
```

The script regenerates Dart protobuf models from `apic/proto/apic.proto`, runs `flutter pub get`, builds the macOS app, and prints the app bundle path.

For iterative development:

```sh
cd macos/ProtosFlutter
flutter run -d macos
```

The macOS Xcode target runs `scripts/build-native.sh` before bundling frameworks, so `flutter run` also rebuilds the embedded Go dylib.

