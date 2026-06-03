package cloudkit

import "list"

#protosImage: #imageContract & {
	provider: "local-macos-test"
	boot: {
		firmware: "direct-kernel"
		notes: ["The local test VM uses kernel+initrd directly, with an EFI initrd ISO available for fallback."]
	}
	linuxkit: {
		arch: "arm64"
		formats: ["kernel+initrd", "iso-efi-initrd"]
		imageSize: "10G"
		outputFiles: ["mactest-kernel", "mactest-initrd.img", "mactest-efi-initrd.iso", "mactest-disk.img"]
	}
	devices: {
		root: "/dev/vda"
	}
}

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
