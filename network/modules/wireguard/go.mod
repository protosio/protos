module github.com/protosio/protos/network/modules/wireguard

go 1.26.3

require (
	filippo.io/edwards25519 v1.2.0
	github.com/containernetworking/cni v1.3.0
	github.com/containernetworking/plugins v1.9.1
	github.com/protosio/protos v0.0.0
	github.com/vishvananda/netlink v1.3.1
	golang.zx2c4.com/wireguard v0.0.0-20250521234502-f333402bd9cb
	golang.zx2c4.com/wireguard/wgctrl v0.0.0-20241231184526-a9ab2273dd10
)

require (
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/coreos/go-iptables v0.8.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/jsimonetti/rtnetlink v1.4.2 // indirect
	github.com/mdlayher/genetlink v1.4.0 // indirect
	github.com/mdlayher/netlink v1.11.2 // indirect
	github.com/mdlayher/socket v0.6.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/safchain/ethtool v0.7.0 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/vishvananda/netns v0.0.5 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.54.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.44.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	sigs.k8s.io/knftables v0.0.21 // indirect
)

replace github.com/protosio/protos => ../../..
