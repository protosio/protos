package util

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
