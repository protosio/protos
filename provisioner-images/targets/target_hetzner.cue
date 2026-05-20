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
#metadata_hetzner: #metadata & {
	command: ["/usr/bin/metadata", "hetzner"]
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

files: #files
