//go:build linux

package p2p

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

func probeAppHTTPFromSandbox(ctx context.Context, appID string, target string, timeout time.Duration, maxBytes int) ([]byte, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return nil, fmt.Errorf("app ID is required")
	}
	targetURL, err := url.ParseRequestURI(strings.TrimSpace(target))
	if err != nil {
		return nil, fmt.Errorf("invalid probe URL: %w", err)
	}
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return nil, fmt.Errorf("unsupported probe URL scheme %q", targetURL.Scheme)
	}

	pidFile := fmt.Sprintf("/run/containerd/io.containerd.runtime.v2.task/protos/%s/init.pid", appID)
	inner := fmt.Sprintf("wget -q -T %d -O - %s | head -c %d", int(timeout.Seconds()), shellQuote(targetURL.String()), maxBytes)
	command := fmt.Sprintf(
		"pid=$(cat %s 2>/dev/null) && [ -n \"$pid\" ] && nsenter -t \"$pid\" -m -n -- /bin/sh -lc %s",
		shellQuote(pidFile),
		shellQuote(inner),
	)
	cmdCtx, cancel := context.WithTimeout(ctx, timeout+2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(cmdCtx, "nsenter", "-t", "1", "-m", "--", "/bin/sh", "-lc", command).CombinedOutput()
	if cmdCtx.Err() != nil {
		return nil, cmdCtx.Err()
	}
	if err != nil {
		return nil, fmt.Errorf("probe app %s HTTP: %w: %s", appID, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}
