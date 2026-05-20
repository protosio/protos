# provisioner-images

LinuxKit based provisioner image creator for Protos.

## Build

```
task mactest
task macdev
task scaleway
task pkg
```

The macOS targets emit `kernel+initrd` files for direct boot experiments and
an EFI initrd ISO for the local macOS provisioner. The provisioner prefers the
EFI ISO when it is present.

## Sign the host agent

```
codesign --entitlements ../cmd/protos-hostagent/protos-hostagent.entitlements -f -s - <protos-hostagent binary path>
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
cd /var/lib/src/backend
```

To start protosd run:

```
go run ./cmd/protosd --loglevel debug
```
