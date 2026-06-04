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
go test ./core/...
clients/macos/scripts/build.sh
task -t clients/ios/Taskfile.yml build -- --no-codesign
task -t clients/ios/Taskfile.yml build:no-tunnel
task -t cloud-provisioning/Taskfile.yml image-contracts
```

## Design Notes

- The declarative core should describe the desired state.
- The imperative layer should reconcile that state and expose read-only status
  to clients where useful.
- Destructive operations should require confirmation in client workflows.
- Clients should have a clear way to report whether observed runtime state
  matches declared state.
- Stopping `protosd` should not automatically stop networking or local VMs when
  the host agent remains active.
