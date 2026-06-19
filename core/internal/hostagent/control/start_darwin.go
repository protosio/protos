//go:build darwin

package control

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
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
	if err := startWithSudo(ctx, binary, logPath, uid, gid); err == nil {
		return nil
	}
	return startWithAppleScript(ctx, binary, logPath, uid, gid)
}

func startWithSudo(ctx context.Context, binary string, logPath string, uid int, gid int) error {
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(checkCtx, "sudo", "-n", binary, "--help").CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			return fmt.Errorf("host-agent sudo rule is not usable: %w", err)
		}
		return fmt.Errorf("host-agent sudo rule is not usable: %w: %s", err, message)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	args := []string{
		"-n",
		binary,
		"--socket-mode", "0660",
		"--socket-uid", fmt.Sprintf("%d", uid),
		"--socket-gid", fmt.Sprintf("%d", gid),
	}
	cmd := exec.Command("sudo", args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if logFile, closeLog := hostAgentLogFile(logPath); logFile != nil {
		defer closeLog()
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start host agent with sudo: %w", err)
	}
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("release host agent process: %w", err)
	}
	return nil
}

func startWithAppleScript(ctx context.Context, binary string, logPath string, uid int, gid int) error {
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

func hostAgentLogFile(logPath string) (*os.File, func()) {
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, func() {}
	}
	return file, func() {
		_ = file.Close()
	}
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
