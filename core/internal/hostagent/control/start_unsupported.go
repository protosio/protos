//go:build !darwin

package control

import (
	"context"
	"fmt"
	"runtime"
)

type StartOptions struct {
	BinaryPath string
	LogPath    string
	SocketUID  int
	SocketGID  int
}

func Start(context.Context, StartOptions) error {
	return fmt.Errorf("host-agent startup is not supported on %s", runtime.GOOS)
}

func Stop(context.Context) error {
	return fmt.Errorf("host-agent shutdown is not supported on %s", runtime.GOOS)
}
