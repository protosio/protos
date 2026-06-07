//go:build darwin

package control

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type StartOptions struct {
	BinaryPath string
	LogPath    string
	SocketUID  int
	SocketGID  int
}

func Start(ctx context.Context, opts StartOptions) error {
	binary, err := resolveBinary(opts.BinaryPath)
	if err != nil {
		return err
	}
	logPath := strings.TrimSpace(opts.LogPath)
	if logPath == "" {
		logPath = "/Library/Logs/ProtosHostAgent.log"
	}
	uid := opts.SocketUID
	if uid < 0 {
		uid = os.Getuid()
	}
	gid := opts.SocketGID
	if gid < 0 {
		gid = os.Getgid()
	}
	command := fmt.Sprintf(
		"/usr/bin/nohup %s --socket-mode 0660 --socket-uid %d --socket-gid %d >> %s 2>&1 &",
		shellQuote(binary),
		uid,
		gid,
		shellQuote(logPath),
	)
	script := fmt.Sprintf(
		`do shell script "%s" with administrator privileges`,
		appleScriptString(command),
	)
	out, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", script).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return fmt.Errorf("start host agent: %w", err)
		}
		return fmt.Errorf("start host agent: %w: %s", err, message)
	}
	return nil
}

func resolveBinary(value string) (string, error) {
	candidates := []string{
		value,
		os.Getenv("PROTOS_HOSTAGENT_BIN"),
	}
	if executable, err := os.Executable(); err == nil {
		candidates = append(candidates, ancestorCandidates(filepath.Dir(executable))...)
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, ancestorCandidates(cwd)...)
	}
	candidates = append(candidates,
		"/usr/local/bin/protos-hostagent",
		"/opt/homebrew/bin/protos-hostagent",
	)
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		normalized, err := filepath.Abs(candidate)
		if err != nil {
			normalized = filepath.Clean(candidate)
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		info, err := os.Stat(normalized)
		if err == nil && !info.IsDir() {
			return normalized, nil
		}
	}
	return "", fmt.Errorf("could not find protos-hostagent; set PROTOS_HOSTAGENT_BIN")
}

func ancestorCandidates(dir string) []string {
	out := make([]string, 0, 16)
	for i := 0; i < 16; i++ {
		out = append(out, filepath.Join(dir, "bin", "protos-hostagent"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return out
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}

func appleScriptString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return value
}
