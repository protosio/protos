package util

import (
	"fmt"
	"net"
	"strconv"
)

// PortType defines a port type, that can hold TCP or UDP
type PortType string

// TCP port type
var TCP = PortType("TCP")

// UDP port type
var UDP = PortType("UDP")

// SCTP port type
var SCTP = PortType("SCTP")

//Port defines a struct that holds information about a port
type Port struct {
	Nr   int
	Type PortType
}

// MarshalJSON is a customer JSON marshallers for Port
func (port *Port) MarshalJSON() ([]byte, error) {
	portStr := fmt.Sprintf("\"%s/%s\"", strconv.Itoa(port.Nr), string(port.Type))
	return []byte(portStr), nil
}

// GetLocalIPs returns the locally configured IP address
func GetLocalIPs() ([]string, error) {
	ips := []string{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips, fmt.Errorf("failed to retrieve the local network interfaces: %s", err.Error())
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			return ips, fmt.Errorf("failed to retrieve IPs for %s: %s", i.Name, err.Error())
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

// AllNetworkIPs receives an IP network and returns a list with all its IPs
func AllNetworkIPs(ipnet net.IPNet) []net.IP {
	ip, _, err := net.ParseCIDR(ipnet.String())
	if err != nil {
		log.Fatal(err)
	}

	var ips []net.IP
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		dup := make(net.IP, len(ip))
		copy(dup, ip)
		ips = append(ips, dup)
	}
	// remove network address and broadcast address
	return ips[1 : len(ips)-1]
}

// IPinList checks if an IP is in a list of IPs
func IPinList(ip net.IP, ips []net.IP) bool {
	for _, nip := range ips {
		if ip.Equal(nip) {
			return true
		}
	}
	return false
}

//  http://play.golang.org/p/m8TNTtygK0
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
