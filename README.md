# Protos Backend

This repository contains the shared Protos core and the clients that talk to it.

## Layout

- `core/`: Go runtime, daemon, API contracts, provisioners, and shared packages.
- `cloud-provisioning/`: LinuxKit image contracts and build tasks for local and cloud targets.
- `clients/shared/flutter/`: shared Flutter UI/API package used by the mobile and desktop Flutter clients.
- `clients/ios/`: iOS Flutter app wrapper.
- `clients/macos/`: macOS Flutter app wrapper.
- `clients/cli/`: Go command-line client.

The old Swift macOS client has been removed. Native app clients now use the
Flutter UI with the Go FFI bridge in `core/cmd/protos-ffi-bridge`.

## Common Commands

```sh
go test ./clients/cli
task -t core/Taskfile.yaml test
task -t core/Taskfile.yaml verify
task -t clients/macos/Taskfile.yml build
task -t clients/macos/Taskfile.yml run
task -t clients/ios/Taskfile.yml build:no-tunnel
task -t clients/ios/Taskfile.yml run:no-tunnel DEVICE=<device-id>
task -t cloud-provisioning/Taskfile.yml image-contracts
```

Use `task -t core/Taskfile.yaml test` instead of raw `go test ./core/...`.
The core test path needs the same pure-Go Dolt/Swarmion tags used by the
Taskfile; running Go directly can select native ICU/zstd dependencies that are
not part of the normal backend test environment.

## Design Notes

See [Durable operations on Swarmion](core/SWARMION_DURABLE_OPERATIONS.md) for
the receipt, recovery, SQL-adapter, and dependency-pinning contracts used by
restart-sensitive backend workflows.

- The declarative core should describe the desired state.
- The imperative layer should reconcile that state and expose read-only status
  to clients where useful.
- Destructive operations should require confirmation in client workflows.
- Clients should have a clear way to report whether observed runtime state
  matches declared state.
- Stopping `protosd` should not automatically stop networking or local VMs when
  the host agent remains active.
