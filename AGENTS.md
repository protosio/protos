# Protos Backend Agent Notes

## Provisioner Images

Build provisioner images only through `/Users/al3x/code/protos/code/backend/provisioner-images/Taskfile.yml`.
Do not invoke `linuxkit build` directly for cloud images.

The provider boot settings are encoded in the CUE target contracts and consumed
by the Taskfile:

- Scaleway: UEFI, LinuxKit `iso-efi`, output `scaleway-efi.iso`.
- Hetzner: BIOS, LinuxKit `raw-bios`, output `hetzner-bios.img`.
- Local macOS dev/test: direct kernel boot plus EFI initrd ISO, 10 GB local disk.

Use `task -t provisioner-images/Taskfile.yml image-contracts` to inspect the
current contracts, and `task -t provisioner-images/Taskfile.yml cloud` to build
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
