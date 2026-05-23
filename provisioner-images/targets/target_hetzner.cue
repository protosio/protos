package cloudkit

import "list"

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

//
// services
//
#sshd_hetzner: #sshd & {
	binds: [
		"/run/config/ssh/authorized_keys:/root/.ssh/authorized_keys",
		"/var/log:/var/log",
		"/var/lib/protos:/var/lib/protos",
	]
	runtime: #runtime & {
		mkdir: ["/var/log", "/var/lib/protos"]
	}
}
services: #services & [#getty, #rngd_service, #sshd_hetzner, #protos]

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

        mkdir -p "$CONFIG_DIR/ssh"

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

        write_public_keys || run_userdata

        if [ ! -s "$CONFIG_DIR/ssh/authorized_keys" ]; then
          echo "Hetzner metadata did not provide authorized_keys" >&2
          exit 1
        fi
        chmod 0600 "$CONFIG_DIR/ssh/authorized_keys"
        """
}

files: list.Concat([#files, [#file_hetzner_metadata]])
