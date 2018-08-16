package util

import (
	"fmt"
	"net"
)

// PortType defines a port type, that can hold TCP or UDP
type PortType string

// TCP port type
var TCP = PortType("TCP")

// UDP port type
var UDP = PortType("UDP")

//Port defines a struct that holds information about a port
type Port struct {
	Nr   int
	Type PortType
}

// GetLocalIPs returns the locally configured IP address
func GetLocalIPs() ([]string, error) {
	ips := []string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, fmt.Errorf("Failed to retrieve the local network interfaces: %s", err.Error())
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return ips, fmt.Errorf("Failed to retrieve IPs for %s: %s", i.Name, err.Error())
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip.String() != "127.0.0.1" {
				ips = append(ips, ip.String())
			}
		}
	}

	return ips, nil
}
