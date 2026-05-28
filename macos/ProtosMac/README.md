# Protos macOS

SwiftUI macOS client for the embedded Protos daemon.

The app links the Go daemon through a small C ABI:

- `ProtosStart(configJSON)`
- `ProtosCall(method, requestBytes)`
- `ProtosStop()`
- `ProtosFree(ptr)`

Swift uses generated `apic/proto/apic.proto` models and sends serialized protobuf messages through `ProtosCall`. The C bridge never exposes Go service structs or daemon internals.

The embedded default capabilities are `default,no-api,no-network`: the app does not open the daemon's Unix socket and does not require the separate host agent just to launch.

## Build

```sh
macos/ProtosMac/scripts/build.sh
```

The script:

1. Builds `cmd/protos-macos-bridge` as `.build/go/libprotos.a`.
2. Generates Swift protobuf models from `apic/proto/apic.proto`.
3. Builds the SwiftUI executable.
4. Packages `.build/Protos.app`.

`SwiftProtobuf` is pinned to `1.38.0`. The script keeps that checkout under `.build/tools` so SwiftPM does not need a second network fetch.
