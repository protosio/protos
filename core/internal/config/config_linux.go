package config

var config = &Config{
	WorkDir:         "/var/lib/protos",
	P2PPort:         10500,
	Runtime:         "containerd",
	RuntimeEndpoint: "/run/containerd/containerd.sock",
	InternalDomain:  "protos.internal",
	ExternalDNS:     "8.8.8.8:53",
}
