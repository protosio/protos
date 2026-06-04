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

The default iOS build includes the Packet Tunnel Network Extension. That build
requires Apple provisioning that supports Network Extensions:

```sh
task -t clients/ios/Taskfile.yml build -- --no-codesign
```

For personal Apple teams or basic app testing without VPN support, build or run
without the Packet Tunnel extension:

```sh
task -t clients/ios/Taskfile.yml build:no-tunnel
DEVICE=<device-id> task -t clients/ios/Taskfile.yml run:no-tunnel -- --no-resident
```

To install the full tunnel-capable app on a physical iPhone, Xcode needs a valid
Apple Development signing identity, a provisioning team with the Network
Extensions capability, and an unlocked/trusted iPhone with Developer Mode
enabled.
