package ipc

import "os"

const (
	DefaultSocketPath = "/var/run/protos-hostagent.sock"
	SocketEnv         = "PROTOS_HOSTAGENT_SOCKET"
)

func SocketPath() string {
	if socket := os.Getenv(SocketEnv); socket != "" {
		return socket
	}
	return DefaultSocketPath
}
