module github.com/protosio/protos/network/modules/wireguard

go 1.26.4

toolchain go1.26.5

require (
	filippo.io/edwards25519 v1.2.0
	github.com/containernetworking/plugins v1.9.1
	github.com/ebitengine/purego v0.10.0
	github.com/google/nftables v0.3.0
	github.com/protosio/protos v0.0.0
	github.com/tmc/apple v0.6.3
	github.com/vishvananda/netlink v1.3.1
	golang.org/x/net v0.57.0
	golang.org/x/sys v0.47.0
	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
)

require (
	github.com/cilium/ebpf v0.16.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/jsimonetti/rtnetlink v1.4.2 // indirect
	github.com/mdlayher/genetlink v1.4.0 // indirect
	github.com/mdlayher/netlink v1.11.2 // indirect
	github.com/mdlayher/socket v0.6.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
)

replace github.com/protosio/protos => ../../..
