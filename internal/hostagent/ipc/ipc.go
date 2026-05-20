package ipc

import "os"

const (
	DefaultSocketPath = "/var/run/protos-vm-hostagent.sock"
	SocketEnv         = "PROTOS_VM_HOSTAGENT_SOCKET"
)

func SocketPath() string {
	if socket := os.Getenv(SocketEnv); socket != "" {
		return socket
	}
	return DefaultSocketPath
}
