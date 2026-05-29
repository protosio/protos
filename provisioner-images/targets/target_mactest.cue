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

services: #services & [#getty, #protos]

//
// files
//
files: #files
