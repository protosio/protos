package cloudkit

import "list"

#protosImage: #imageContract & {
	provider: "hetzner"
	boot: {
		firmware: "bios"
		notes: [
			"Hetzner server snapshots boot as disks through a BIOS-style path.",
			"Use LinuxKit raw-bios here: iso-efi snapshots are treated like an unreadable CD-ROM, and raw-efi currently lands in an EFI shell with no bootable filesystem mapping.",
		]
	}
	linuxkit: {
		arch: "amd64"
		formats: ["raw-bios"]
		outputFiles: ["hetzner-bios.img"]
	}
	devices: {
		root: "/dev/sda"
		data: "/dev/sdb"
	}
	upload: {
		kind: "hetzner-server-snapshot"
		preferredLocations: ["ash"]
		preferSmallestSnapshotDisk: true
		targetSnapshotDiskGiB: 40
		notes: [
			"Hetzner snapshots inherit the upload helper server disk size.",
			"The upload helper must be an available x86 server type with a disk no larger than 40 GiB.",
			"If the selected location only has larger x86 helpers available, fail the upload instead of creating an oversized snapshot.",
			"Keep helper selection ordered by disk size before price so deployable snapshots do not grow silently.",
		]
	}
}

//
// kernel
//
kernel: #kernel & {
	cmdline: "console=tty0 console=ttyS0,115200n8 earlyprintk=ttyS0,115200 consoleblank=0 root=/dev/sda"
}

//
// init
//
init: #init_base

//
// onboot
//
#metadata_hetzner: #container & {
	name:         "metadata"
	image:        "linuxkit/dhcpcd:b87e9ececac55a65eaa592f4dd8b4e0c3009afdb"
	net:          "host"
	capabilities: ["all"]
	command:      ["/bin/sh", "/usr/bin/protos-hetzner-metadata"]
	binds: [
		"/usr/bin/protos-hetzner-metadata:/usr/bin/protos-hetzner-metadata:ro",
		"/run:/run",
	]
}
#format_hetzner: #container & {
	name:  "format"
	image: "linuxkit/format:4f779c0b0d0ba145b7f03211b5cbf59fbbe12e54"
	command: ["/usr/bin/format", "-type", "btrfs", "-label", "PROTOSDATA", "-verbose", "/dev/sdb"]
}
onboot: #onboot & list.FlattenN([#onboot_common, #metadata_hetzner, #format_hetzner, #mount, #rngd_boot], 1)

services: #services & [#getty, #rngd_service, #protos]

#file_hetzner_metadata: #file & {
	path: "/usr/bin/protos-hetzner-metadata"
	mode: "0755"
	contents: """
        #!/bin/sh
        set -eu

        BASE=http://169.254.169.254/hetzner/v1
        CONFIG_DIR=/run/config

        fetch_once() {
          wget -q -T 3 -O - "$1" 2>/dev/null || true
        }

        fetch_retry() {
          URL=$1
          TRIES=${2:-30}
          I=0
          while [ "$I" -lt "$TRIES" ]; do
            BODY=$(fetch_once "$URL")
            if [ -n "$BODY" ]; then
              printf '%s\n' "$BODY"
              return 0
            fi
            I=$((I + 1))
            sleep 2
          done
          return 1
        }

        write_public_keys() {
          KEYS=$(fetch_retry "$BASE/metadata/public-keys" 30 || true)
          if [ -z "$KEYS" ]; then
            return 1
          fi
          echo "$KEYS" | sed -n 's/^- //p; /^ssh-/p; /^ecdsa-/p; /^sk-/p' > "$CONFIG_DIR/ssh/authorized_keys"
          [ -s "$CONFIG_DIR/ssh/authorized_keys" ]
        }

        run_userdata() {
          USERDATA=$(mktemp)
          if fetch_retry "$BASE/userdata" 30 > "$USERDATA" || fetch_retry "http://169.254.169.254/latest/user-data" 30 > "$USERDATA"; then
            if [ -s "$USERDATA" ]; then
              cp "$USERDATA" "$CONFIG_DIR/userdata"
              if [ "$(head -c 2 "$USERDATA")" = "#!" ]; then
                sh "$USERDATA"
              fi
            fi
          fi
          rm -f "$USERDATA"
        }

        mkdir -p "$CONFIG_DIR/ssh" "$CONFIG_DIR/protos"

        HOSTNAME=$(fetch_once "$BASE/metadata/hostname")
        if [ -n "$HOSTNAME" ]; then
          echo "$HOSTNAME" > "$CONFIG_DIR/hostname"
          hostname "$HOSTNAME" || true
        fi

        PUBLIC_IPV4=$(fetch_once "$BASE/metadata/public-ipv4")
        if [ -n "$PUBLIC_IPV4" ]; then
          echo "$PUBLIC_IPV4" > "$CONFIG_DIR/public_ipv4"
        fi

        INSTANCE_ID=$(fetch_once "$BASE/metadata/instance-id")
        if [ -n "$INSTANCE_ID" ]; then
          echo "$INSTANCE_ID" > "$CONFIG_DIR/instance_id"
        fi

        write_public_keys || true
        run_userdata || true

        if [ -s "$CONFIG_DIR/ssh/authorized_keys" ]; then
          chmod 0600 "$CONFIG_DIR/ssh/authorized_keys"
        fi
        """
}

files: list.Concat([#files, [#file_hetzner_metadata]])
