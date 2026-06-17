# Protos Backend Agent Notes

## Thin Clients

Keep client code, including Flutter, CLI, iOS, macOS, and future Android
clients, as thin shells around the Go core. Product behavior, platform
orchestration, host-agent control, networking policy, persistence rules, and
cross-client workflows should live in `core/` behind Go APIs. Clients should
mostly collect user intent, render state returned by the core, and call core
APIs. Do not add client-only logic paths for behavior that must work across
multiple clients; implement one canonical core path and have each client call it.

## Build And Run Entry Points

Use Taskfiles as the canonical entry point for builds, launches, installs, and
device cleanup. Do not invoke `flutter run`, `flutter build`, `xcodebuild`, or
`xcrun devicectl` app install/launch commands directly during normal work. If a
needed workflow is missing or inconsistent, add or fix the Taskfile task first,
then run that task.

## Test And Debug Discipline

When an integration or e2e test fails, repeat the failing test after gathering
evidence, investigate the root cause through product APIs such as APIC and
grpcurl, and fix the underlying issue rather than accepting one-off retries or
test-specific workarounds. Prefer architecturally clean solutions that preserve
the declarative-to-imperative layering: clients call core APIs, core owns product
behavior, and imperative host/provider actions flow through the appropriate
agent or provisioner API. Keep rerunning the relevant focused and end-to-end
tests until the fix is demonstrated or the remaining blocker is clearly
identified with evidence.

For core Go work, use `task -t core/Taskfile.yaml test` or
`task -t core/Taskfile.yaml verify` rather than raw `go test ./core/...`.
The Taskfile applies the canonical pure-Go Dolt/Swarmion build tags and avoids
accidentally selecting local native ICU/zstd dependencies. Use
`task -t core/Taskfile.yaml deps:security` for `govulncheck` and
`task -t core/Taskfile.yaml deps:freshness` to compare critical runtime,
Swarmion, API, and cloud SDK modules with upstream.

For local client work use:

- macOS: `task -t clients/macos/Taskfile.yml run` or `task -t
  clients/macos/Taskfile.yml build`.
- iOS without Network Extension provisioning: `task -t clients/ios/Taskfile.yml
  run:no-tunnel DEVICE=<device-id>` or `task -t clients/ios/Taskfile.yml
  build:no-tunnel`.
- iOS with the Packet Tunnel extension: use `task -t
  clients/ios/Taskfile.yml run ALLOW_TUNNEL=1` or `task -t
  clients/ios/Taskfile.yml build ALLOW_TUNNEL=1` only when the Apple team has
  Network Extensions provisioning.

## Provisioner Images

Build provisioner images only through `/Users/al3x/code/protos/code/backend/cloud-provisioning/Taskfile.yml`.
Do not invoke `linuxkit build` directly for cloud images.

The provider boot settings are encoded in the CUE target contracts and consumed
by the Taskfile:

- Scaleway: UEFI, LinuxKit `iso-efi`, output `scaleway-efi.iso`.
- Hetzner: BIOS, LinuxKit `raw-bios`, output `hetzner-bios.img`.
- Local macOS dev/test: direct kernel boot plus EFI initrd ISO, 10 GB local disk.

Use `task -t cloud-provisioning/Taskfile.yml image-contracts` to inspect the
current contracts, and `task -t cloud-provisioning/Taskfile.yml cloud` to build
cloud images in the canonical order.

Hetzner server snapshots inherit the temporary upload server disk size. The
upload helper must be an available x86 server type with a disk no larger than
40 GB, or the upload must fail instead of producing an oversized snapshot. Keep
the helper selection on disk size before price so Protos images remain bootable
on Hetzner's smallest x86 VM sizes. Do not document Hetzner server snapshots as
5-10 GB unless the upload mechanism changes away from server snapshots.

Do not switch Hetzner back to `iso-efi` without revalidating the full upload
and boot path: snapshots made from that artifact are presented as an unreadable
CD-ROM. `raw-efi` currently reaches an EFI shell without a bootable filesystem
mapping. The validated Hetzner path is BIOS boot from the `raw-bios` disk image.
