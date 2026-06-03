package cloudkit

import "list"

#protosImage: #imageContract & {
	provider: "local-macos-dev"
	boot: {
		firmware: "direct-kernel"
		notes: ["The local development VM uses kernel+initrd directly, with an EFI initrd ISO available for fallback."]
	}
	linuxkit: {
		arch: "arm64"
		formats: ["kernel+initrd", "iso-efi-initrd"]
		imageSize: "10G"
		outputFiles: ["macdev-kernel", "macdev-initrd.img", "macdev-efi-initrd.iso", "macdev-disk.img"]
	}
	devices: {
		root: "/dev/vda"
	}
}

//
// kernel
//
kernel: #kernel & {
    cmdline: "console=tty0 console=ttyS0 console=ttyAMA0 console=ttysclp0 console=hvc0"
}

//
// init
//
init: #init_base

//
// onboot
//
onboot: #onboot & list.FlattenN([#onboot_common, #format, #mount], 1)

//
// services
//
#macdev: #container & {
    name:  "macdev"
    image: "protosio/dev:a1a785cb56d0eb32199646e6f6a08ed524c366bb"
    pid: "host"
    capabilities: ["all"]
    binds: [
        "/root/.ssh:/root/.ssh",
        "/etc/resolv.conf:/etc/resolv.conf",
        "/tmp:/tmp",
        "/etc:/hostroot/etc",
        "/var/lib:/var/lib:rshared,rbind",
        "/dev:/dev",
        "/run/containerd/containerd.sock:/run/containerd/containerd.sock",
    ]
    rootfsPropagation: "shared"
    devices: [
        #device & {
            path: "all"
            type: "b"
        }
    ]
    runtime: #runtime & {
        mkdir: ["/root/.ssh", "/tmp", "/var/lib", "/run/containerd"]
    }
}
services: #services & [#getty, #macdev]

//
// files
//
#file_authorized_keys: #file & {
        path:     "/root/.ssh/authorized_keys"
        source:   "~/.ssh/protos.pub"
        mode:     "0600"
        optional: true
    },
files: list.Concat([#files, [#file_authorized_keys]])
