package cloudkit

import "list"

//
// kernel
//
kernel: #kernel & {
	cmdline: "console=hvc0 irqfixup"
}

//
// init
//
init: #init_base

//
// onboot
//
#metadata_mactest: #metadata & {
	command: ["/usr/bin/metadata", "cdrom"]
}
onboot: #onboot & list.FlattenN([#onboot_common_no_network, #metadata_mactest, #static_network, #format, #mount, #swap], 1)

//
// services
//
#sshd_mactest: #sshd & {
	binds: [
		"/run/config/ssh/authorized_keys:/root/.ssh/authorized_keys",
		"/var/log:/var/log",
		"/var/lib/protos:/var/lib/protos",
	]
	runtime: #runtime & {
		mkdir: ["/var/log", "/var/lib/protos"]
	}
}
services: #services & [#getty, #sshd_mactest, #protos]

//
// files
//
#file_authorized_keys: #file & {
	path:     "/root/.ssh/authorized_keys"
	source:   "~/.ssh/protos.pub"
	mode:     "0600"
	optional: true
}
files: list.Concat([#files, [#file_authorized_keys]])
