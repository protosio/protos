//go:build !linux

package p2p

import (
	"context"
	"fmt"
	"time"
)

func probeAppHTTPFromSandbox(context.Context, string, string, time.Duration, int) ([]byte, error) {
	return nil, fmt.Errorf("app HTTP probes are only supported on Linux instances")
}
