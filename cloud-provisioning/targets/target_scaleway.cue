package cloudkit

import "list"

#protosImage: #imageContract & {
	provider: "scaleway"
	boot: {
		firmware: "uefi"
		notes: ["Scaleway Protos images are uploaded as EFI ISO images."]
	}
	linuxkit: {
		arch: "amd64"
		formats: ["iso-efi"]
		outputFiles: ["scaleway-efi.iso"]
	}
	devices: {
		root: "/dev/vda"
		data: "/dev/sda"
	}
	upload: {
		kind: "scaleway-image-from-iso"
		notes: ["The ISO artifact is compact; Scaleway owns the final image size after import."]
	}
}

//
// kernel
//
kernel: #kernel & {
	cmdline: "console=tty0 console=ttyS0,115200n8 earlyprintk=ttyS0,115200 consoleblank=0 root=/dev/vda"
}

//
// init
//
init: #init_base

//
// onboot
//
#metadata_scaleway: #metadata & {
	command: ["/usr/bin/metadata", "scaleway"]
}
#format_scaleway: #container & {
	name:  "format"
	image: "linuxkit/format:4f779c0b0d0ba145b7f03211b5cbf59fbbe12e54"
	command: ["/usr/bin/format", "-type", "btrfs", "-label", "PROTOSDATA", "-verbose", "/dev/sda"]
}
onboot: #onboot & list.FlattenN([#onboot_common, #metadata_scaleway, #format_scaleway, #mount, #rngd_boot], 1)

services: #services & [#getty, #rngd_service, #logwrite, #protos]

files: #files
