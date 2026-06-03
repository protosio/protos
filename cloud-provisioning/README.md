# Cloud Provisioning

LinuxKit based provisioner image creator for Protos.

## Build

```
task mactest
task macdev
task scaleway
task hetzner
task cloud
task image-contracts
task pkg
```

Use `Taskfile.yml` for image builds. Do not run `linuxkit build` directly for
cloud images, because the Taskfile reads the provider boot contract from CUE
and verifies the LinuxKit format before building.

The cloud boot choices are encoded in the target CUE files:

| Target | Firmware | LinuxKit format | Artifact |
| --- | --- | --- | --- |
| Scaleway | UEFI | `iso-efi` | `targets/output/scaleway/scaleway-efi.iso` |
| Hetzner | BIOS | `raw-bios` | `targets/output/hetzner/hetzner-bios.img` |

The Hetzner upload path creates a server snapshot, and Hetzner snapshots inherit
the upload helper server disk size. The provisioner therefore chooses the
smallest available x86 helper disk in the selected location before considering
price, and refuses to upload when that helper would be larger than 40 GB. This
keeps Protos snapshots bootable on Hetzner's smallest x86 VM sizes. The LinuxKit
raw BIOS artifact itself remains compact. The 5-10 GB target applies to local
Protos-managed disks and to raw artifacts where the provider does not impose a
larger minimum.

Hetzner must use `raw-bios` with the current server-snapshot upload flow:
`iso-efi` snapshots are presented as an unreadable CD-ROM, and `raw-efi` lands
in an EFI shell without a bootable filesystem mapping. Keep those observations
encoded in CUE before changing the target format again.

Local macOS test and development disks are fixed at 10 GB by the CUE contract
and the Taskfile. Increase that only when the VM workload actually needs more
space.

The macOS targets emit `kernel+initrd` files for direct boot experiments and
an EFI initrd ISO for the local macOS provisioner. The provisioner prefers the
EFI ISO when it is present.

## Sign the host agent

```
codesign --entitlements ../core/cmd/protos-hostagent/protos-hostagent.entitlements -f -s - <protos-hostagent binary path>
```

## Run the host agent

For the root-backed development path, run the host agent:

```
sudo protos-hostagent
```

It serves VM control and network reconciliation on
`/var/run/protos-hostagent.sock`. Set `PROTOS_HOSTAGENT_SOCKET` to override the
socket path.
When started with `sudo`, the socket is chowned to `SUDO_UID`/`SUDO_GID` so
the invoking user can run `protosd` unprivileged. A LaunchDaemon install should
set `--socket-uid` and `--socket-gid` explicitly.

## Connect via SSH or getty and switch to the root namespace

```
nsenter --target 1 --mount --uts --ipc --net --pid
```

## Development

Steps for setting up a development environment for Protos

### Sync code

Start mutagen sync session:

```
task syncstart
```

###

Once you SSH into the new dev build, export the following env variables:

Optional:
```
rm -rf /var/lib/src/* && rm -rf /var/lib/protos/*
```

Prepare environment:
```
mkdir -p /var/lib/go/tmp
mkdir -p /var/lib/go/cache
export GOPATH=/var/lib/go
export PATH=${PATH}:/usr/local/go/bin
export TMPDIR=${GOPATH}/tmp
export GOCACHE=${GOPATH}/cache
```

Run mutagen sync on the host, then switch dirs:

```
cd /var/lib/src/backend/core
```

To start protosd run:

```
go run ./cmd/protosd --loglevel debug
```
