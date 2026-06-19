package cloudkit

#runtime: {
	mkdir?: [...string]
	cgroups?: [...string]
	mounts?: [...string]
	interfaces?: [...string]
	namespace?: string
}
#cmount: {
	type:         string
	source?:      string
	destination?: string
	options: [...string]
}
#device: {
	path:   string
	type:   string
	major?: string
	minor?: string
	mode?:  string
}
#container: {
	name:               string
	image:              string
	cgroupsPath?:       string
	pid?:               string
	uts?:               string
	net?:               string
	ipc?:               string
	rootfsPropagation?: string
	runtime?:           #runtime
	command?: [...string]
	env?: [...string]
	binds?: [...string]
	capabilities?: [...string]
	mounts?: [...#cmount]
	devices?: [...#device]
}
#kernel: {
	image:   "linuxkit/kernel:6.12.59"
	cmdline: string
}
#init: [...string]
#onboot: [...#container]
#services: [...#container]
#file: {
	path:      string
	contents?: string
	source?:   string
	mode?:     string
	optional?: bool
}
#files: [...#file]

#imageContract: {
	provider: string
	boot: {
		firmware: "bios" | "uefi" | "direct-kernel"
		notes: [...string]
	}
	linuxkit: {
		arch: "amd64" | "arm64"
		formats: [...string]
		// imageSize is passed to linuxkit --size when the target needs a fixed
		// backing disk. Empty means the LinuxKit format controls its own size.
		imageSize: *"" | string
		outputFiles: [...string]
	}
	devices: {
		root: string
		data?: string
	}
	upload?: {
		kind: string
		preferredLocations?: [...string]
		// Some clouds derive snapshot size from an upload helper VM rather than
		// the raw artifact. Use the smallest available helper disk for those.
		preferSmallestSnapshotDisk: bool | *false
		targetSnapshotDiskGiB?: int
		notes: [...string]
	}
}

#protosImage: #imageContract

//
// init base
//
#init_base: #init & [
	"linuxkit/init:b5506cc74a6812dc40982cacfd2f4328f8a4b12a",
	"linuxkit/runc:9442aa234715e751a16144f1d4ae3fd1a00fd492",
	"linuxkit/containerd:ba19f64efd3331a8fd0a33e00eabd14f6ee1780e",
	"linuxkit/ca-certificates:256f1950df59f2f209e9f0b81374177409eb11de",
]

//
// parent containers
//
#rngd: #container & {
	image: "linuxkit/rngd:984eb580ecb63986f07f626b61692a97aacd7198"
}

//
// boot containers
//
#sysfs: #container & {
	name:  "sysfs"
	image: "linuxkit/sysfs:6d5bd933762f6b216744c711c6e876756cee9600"
}
#sysctl: #container & {
	name:  "sysctl"
	image: "linuxkit/sysctl:43ac1d39da329c3567fcb9689e5ca99de6d169b6"
}
#modprobe: #container & {
	name:  "modprobe"
	image: "linuxkit/modprobe:4248cdc3494779010e7e7488fc17b6fd45b73aeb"
	command: ["modprobe", "btrfs"]
}
#dhcpcd: #container & {
	name:  "dhcpcd"
	image: "linuxkit/dhcpcd:b87e9ececac55a65eaa592f4dd8b4e0c3009afdb"
	command: ["/sbin/dhcpcd", "--nobackground", "-4", "-f", "/dhcpcd.conf", "-1"]
	binds: [
		"/etc/protos-dhcpcd.conf:/dhcpcd.conf:ro",
	]
}
#static_network: #container & {
	name:  "static-network"
	image: "linuxkit/dhcpcd:b87e9ececac55a65eaa592f4dd8b4e0c3009afdb"
	net:   "host"
	capabilities: ["all"]
	command: ["/bin/sh", "/usr/bin/protos-static-network"]
	binds: [
		"/usr/bin/protos-static-network:/usr/bin/protos-static-network:ro",
		"/run/config/protos/network:/run/config/protos/network:ro",
		"/etc/resolv.conf:/etc/resolv.conf",
	]
}
#format: #container & {
	name:  "format"
	image: "linuxkit/format:4f779c0b0d0ba145b7f03211b5cbf59fbbe12e54"
	command: ["/usr/bin/format", "-type", "btrfs", "-label", "PROTOSDATA", "-verbose", "/dev/vda"]
}
#mount: #container & {
	name:  "mount"
	image: "linuxkit/mount:bd1c3bb45e48e68e47a9456d1669f7119f855184"
	command: ["/usr/bin/mountie", "-label", "PROTOSDATA", "/var/lib"]
}
#swap: #container & {
	name:  "swap"
	image: "linuxkit/swap:7e19e5e69370e82b90a110093441abbf5e70638b"
	command: ["/swap.sh", "--path", "/var/lib/swap", "--size", "2048M"]
}
#metadata: #container & {
	name:  "metadata"
	image: "linuxkit/metadata:4bbf406678d376e1ae9c9efae6ca2421f63fb4ff"
}
#rngd_boot: #rngd & {
	name: "rngd1"
	command: ["/sbin/rngd", "-1"]
}
#onboot_common: [#sysctl, #modprobe, #dhcpcd]
#onboot_common_no_network: [#sysctl, #modprobe]

//
// services
//
#getty: #container & {
	name:  "getty"
	image: "linuxkit/getty:a86d74c8f89be8956330c3b115b0b1f2e09ef6e0"
	env: ["INSECURE=true"]
}
#rngd_service: #rngd & {
	name: "rngd"
}
#sshd: #container & {
	name:  "sshd"
	image: "linuxkit/sshd:08e5d4a46603eff485d5d1b14001cc932a530858"
}
#logwrite: #container & {
	name:  "logwrite"
	image: "linuxkit/logwrite:24e6a76c2d45a7679d4f53db9ea377373b543dc7"
}
#kmsg: #container & {
	name:  "kmsg"
	image: "linuxkit/kmsg:c4d8d509cf496faa21c184ae7fdff6fddc6e186d"
}
#protos: #container & {
	name:  "protos"
	image: "protosio/protosd:local"
	pid:   "host"
	net:   "host"
	env: [
		"PROTOS_NETWORK_MODULE=wireguard",
		"PROTOS_CONTAINERD_SNAPSHOTTER=native",
		"TMPDIR=/var/lib/protos/tmp",
	]
	capabilities: ["all"]
	binds: [
		"/var/lib/containerd:/var/lib/containerd",
		"/dev:/dev",
		"/etc/resolv.conf:/etc/resolv.conf",
		"/etc/ssl/certs/ca-certificates.crt:/etc/ssl/certs/ca-certificates.crt",
		"/run/config/protos:/run/config/protos:ro",
		"/run/containerd/containerd.sock:/run/containerd/containerd.sock",
		"/sys:/sys",
		"/var/log:/var/log",
	]
	mounts: [
		#cmount & {
			type:        "bind"
			source:      "/tmp"
			destination: "/tmp"
			options: ["rbind", "rshared"]
		},
		#cmount & {
			type:        "bind"
			source:      "/var/lib/protos"
			destination: "/var/lib/protos"
			options: ["rbind", "rshared"]
		},
	]
	runtime: #runtime & {
		mkdir: ["/var/lib/protos", "/var/lib/protos/tmp", "/run/containerd", "/var/log"]
	}
}

//
// files
//
#file_ip_forwarding: #file & {
	path:     "/etc/sysctl.d/protos.conf"
	contents: "net.ipv4.ip_forward = 1"
}
#file_static_network: #file & {
	path: "/usr/bin/protos-static-network"
	mode: "0755"
	contents: """
        #!/bin/sh
        set -eu

        CONFIG_DIR=/run/config/protos/network
        IFACE=$(cat "$CONFIG_DIR/interface" 2>/dev/null || echo eth0)
        ADDRESS=$(cat "$CONFIG_DIR/address")
        GATEWAY=$(cat "$CONFIG_DIR/gateway" 2>/dev/null || true)
        DNS=$(cat "$CONFIG_DIR/dns" 2>/dev/null || true)

        if [ -z "$ADDRESS" ]; then
          echo "static network address is missing" >&2
          exit 1
        fi

        ip link set "$IFACE" up
        ip addr flush dev "$IFACE" scope global || true
        ip addr add "$ADDRESS" dev "$IFACE"

        if [ -n "$GATEWAY" ]; then
          ip route replace default via "$GATEWAY" dev "$IFACE"
        fi

        if [ -n "$DNS" ]; then
          RESOLV_CONF=/etc/resolv.conf
          if [ -L "$RESOLV_CONF" ]; then
            RESOLV_TARGET=$(readlink "$RESOLV_CONF")
            case "$RESOLV_TARGET" in
              /*) RESOLV_CONF="$RESOLV_TARGET" ;;
              *) RESOLV_CONF="$(dirname "$RESOLV_CONF")/$RESOLV_TARGET" ;;
            esac
          fi
          mkdir -p "$(dirname "$RESOLV_CONF")"
          : > "$RESOLV_CONF"
          for server in $DNS; do
            echo "nameserver $server" >> "$RESOLV_CONF"
          done
        fi

        ip addr show dev "$IFACE"
        ip route show default || true
        """
}
#file_dhcpcd_conf: #file & {
	path: "/etc/protos-dhcpcd.conf"
	contents: """
        # Configure both classic eth* and predictable en*/ens*/enp* NIC names.
        allowinterfaces eth* en* ens* enp*

        hostname
        clientid
        persistent
        option rapid_commit
        option domain_name_servers, domain_name, domain_search, host_name
        option classless_static_routes
        option ntp_servers
        option interface_mtu
        require dhcp_server_identifier
        slaac private
        nodelay
        noarp
        waitip 4
        """
}
#files: [#file_ip_forwarding, #file_static_network, #file_dhcpcd_conf]
