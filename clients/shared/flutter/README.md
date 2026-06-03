# Protos Flutter Shared Package

Shared Flutter UI and API package for Protos app clients. The concrete app
wrappers live in:

- `clients/macos/`
- `clients/ios/`

The clients use the shared protobuf API surface and load the Go bridge through
`dart:ffi`:

- `ProtosStart(configJSON)`
- `ProtosCall(method, requestBytes)`
- `ProtosWatchChangesBytes(requestBytes, callback)`
- `ProtosCancelWatch(watchID)`
- `ProtosStop()`
- `ProtosFree(ptr)`

`scripts/build-native.sh` builds `core/cmd/protos-ffi-bridge` as either:

- macOS: `libprotos.dylib`, embedded into the Flutter app bundle.
- iOS: `libprotos.a`, linked into the Runner app for Dart FFI lookup through
  `DynamicLibrary.process()`.

## macOS Build

```sh
clients/macos/scripts/build.sh
```

The script regenerates Dart protobuf models from
`core/apic/proto/apic.proto`, runs `flutter pub get`, builds the macOS app, and
prints the app bundle path.

For iterative development:

```sh
cd clients/macos
flutter run -d macos
```

The macOS Xcode target runs the shared `scripts/build-native.sh` before
bundling frameworks, so `flutter run` also rebuilds the embedded Go dylib.

## iOS Build

```sh
cd clients/ios
flutter build ios --debug --no-codesign
```

To install on a physical iPhone, Xcode needs a valid Apple Development signing
identity and provisioning team, and the iPhone must be unlocked, trusted by this
Mac, and have Developer Mode enabled. Once signing is configured:

```sh
flutter run -d <device-id>
```
