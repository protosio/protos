package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/protosio/protos/internal/config"
	hostagentclient "github.com/protosio/protos/internal/hostagent/client"
	hostagentdaemon "github.com/protosio/protos/internal/hostagent/daemon"
	hostagentipc "github.com/protosio/protos/internal/hostagent/ipc"
	hostagentpb "github.com/protosio/protos/internal/hostagent/proto"
	"github.com/protosio/protos/internal/localvm"
	networkmodule "github.com/protosio/protos/internal/network/module"
	"github.com/protosio/protos/internal/util"
	wireguardmodule "github.com/protosio/protos/network/modules/wireguard"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var log = util.GetLogger("protos-hostagent")

func main() {
	var (
		runVM                         bool
		manifestPath                  string
		hostSocket                    string
		socketMode                    string
		socketUID                     int
		socketGID                     int
		workDir                       string
		logLevel                      string
		stopExisting                  bool
		cleanupVMRunnerManifestPrefix string
		vmnetSelftest                 bool
	)
	flag.BoolVar(&vmnetSelftest, "vmnet-selftest", false, "open an isolation-off shared vmnet interface, report its parameters, and exit (requires root)")
	flag.BoolVar(&runVM, "run-vm", false, "run one VM from a manifest and block until it exits")
	flag.BoolVar(&stopExisting, "stop-existing", false, "stop existing host agent daemon processes and exit")
	flag.StringVar(&cleanupVMRunnerManifestPrefix, "cleanup-vm-runners-manifest-prefix", "", "stop VM runner processes whose manifest path has this prefix and exit")
	flag.StringVar(&manifestPath, "manifest", "", "path to a local macOS VM manifest")
	flag.StringVar(&hostSocket, "socket", hostagentipc.SocketPath(), "Unix socket path for host agent IPC")
	flag.StringVar(&workDir, "work-dir", defaultWorkDir(), "root-owned host agent state directory")
	flag.StringVar(&socketMode, "socket-mode", "0600", "Unix socket permissions")
	flag.IntVar(&socketUID, "socket-uid", -1, "Unix socket owner uid; defaults to SUDO_UID when available")
	flag.IntVar(&socketGID, "socket-gid", -1, "Unix socket owner gid; defaults to SUDO_GID when available")
	flag.StringVar(&logLevel, "loglevel", "info", "log level: debug, info, warn, error")
	flag.Parse()

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatal(err)
	}
	util.SetLogLevel(level)

	if vmnetSelftest {
		if err := localvm.VMNetSelftest(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if cleanupVMRunnerManifestPrefix != "" {
		if err := stopVMRunnersByManifestPrefix(cleanupVMRunnerManifestPrefix); err != nil {
			log.Fatal(err)
		}
		return
	}

	if stopExisting {
		if err := stopExistingDaemons(hostSocket); err != nil {
			log.Fatal(err)
		}
		return
	}

	if manifestPath != "" {
		if !runVM {
			log.Warn("Running a VM directly with -manifest is deprecated; use --run-vm -manifest for child mode")
		}
		if err := localvm.Run(manifestPath); err != nil {
			log.Fatal(err)
		}
		return
	}

	if err := runDaemon(hostSocket, socketMode, socketUID, socketGID, workDir); err != nil {
		log.Fatal(err)
	}
}

func stopExistingDaemons(hostSocket string) error {
	if err := requestExistingDaemonShutdown(hostSocket); err == nil {
		if waitForNoHostAgentDaemons(time.Minute) {
			log.Info("Stopped host agent daemon via shutdown RPC")
			return nil
		}
		log.Warn("Host agent shutdown RPC did not stop all daemon processes; falling back to signal")
	} else {
		log.Debugf("Host agent shutdown RPC unavailable: %s", err.Error())
	}

	pids, err := hostAgentDaemonPIDs()
	if err != nil {
		return err
	}
	if len(pids) == 0 {
		log.Info("No existing host agent daemons found")
		return nil
	}

	for _, pid := range pids {
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find host agent process %d: %w", pid, err)
		}
		log.Infof("Stopping host agent daemon pid=%d", pid)
		if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("stop host agent process %d: %w", pid, err)
		}
	}

	deadline := time.Now().Add(time.Minute)
	for _, pid := range pids {
		for time.Now().Before(deadline) {
			if !processExists(pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if processExists(pid) {
			process, err := os.FindProcess(pid)
			if err != nil {
				return fmt.Errorf("find host agent process %d for force stop: %w", pid, err)
			}
			log.Warnf("Force stopping host agent daemon pid=%d", pid)
			if err := process.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
				return fmt.Errorf("force stop host agent process %d: %w", pid, err)
			}
		}
	}

	return nil
}

func requestExistingDaemonShutdown(hostSocket string) error {
	if strings.TrimSpace(hostSocket) == "" {
		return fmt.Errorf("host agent socket path is empty")
	}
	client, err := hostagentclient.NewWithSocket(hostSocket)
	if err != nil {
		return fmt.Errorf("connect to host agent: %w", err)
	}
	defer client.Close()
	return client.Shutdown()
}

func waitForNoHostAgentDaemons(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pids, err := hostAgentDaemonPIDs()
		if err != nil {
			return false
		}
		if len(pids) == 0 {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	pids, err := hostAgentDaemonPIDs()
	return err == nil && len(pids) == 0
}

func hostAgentDaemonPIDs() ([]int, error) {
	out, err := exec.Command("/bin/ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	self := os.Getpid()
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid == self {
			continue
		}
		command := strings.Join(fields[1:], " ")
		exe := filepath.Base(fields[1])
		if exe != "protos-hostagent" {
			continue
		}
		if strings.Contains(command, "--run-vm") || strings.Contains(command, "-manifest") || strings.Contains(command, "--stop-existing") {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

type vmRunnerProcess struct {
	pid          int
	manifestPath string
}

func stopVMRunnersByManifestPrefix(prefix string) error {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return fmt.Errorf("manifest prefix is required")
	}
	runners, err := hostAgentVMRunnerProcessesByManifestPrefix(prefix)
	if err != nil {
		return err
	}
	if len(runners) == 0 {
		log.Infof("No VM runners found for manifest prefix %s", prefix)
		return nil
	}

	for _, runner := range runners {
		process, err := os.FindProcess(runner.pid)
		if err != nil {
			return fmt.Errorf("find VM runner process %d: %w", runner.pid, err)
		}
		log.Infof("Stopping VM runner pid=%d manifest=%s", runner.pid, runner.manifestPath)
		if err := process.Signal(syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("stop VM runner process %d: %w", runner.pid, err)
		}
	}

	deadline := time.Now().Add(15 * time.Second)
	for _, runner := range runners {
		for time.Now().Before(deadline) {
			if !processExists(runner.pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		if processExists(runner.pid) {
			process, err := os.FindProcess(runner.pid)
			if err != nil {
				return fmt.Errorf("find VM runner process %d for force stop: %w", runner.pid, err)
			}
			log.Warnf("Force stopping VM runner pid=%d manifest=%s", runner.pid, runner.manifestPath)
			if err := process.Signal(syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
				return fmt.Errorf("force stop VM runner process %d: %w", runner.pid, err)
			}
		}
	}

	return nil
}

func hostAgentVMRunnerProcessesByManifestPrefix(prefix string) ([]vmRunnerProcess, error) {
	out, err := exec.Command("/bin/ps", "-axo", "pid=,command=").Output()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	self := os.Getpid()
	var runners []vmRunnerProcess
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid == self {
			continue
		}
		exe := filepath.Base(fields[1])
		if exe != "protos-hostagent" {
			continue
		}
		args := fields[2:]
		manifest := vmRunnerManifestArg(args)
		if manifest == "" || !pathHasPrefixVariant(manifest, prefix) {
			continue
		}
		runners = append(runners, vmRunnerProcess{pid: pid, manifestPath: manifest})
	}
	return runners, nil
}

func vmRunnerManifestArg(args []string) string {
	for i, arg := range args {
		if arg == "-manifest" || arg == "--manifest" {
			if i+1 < len(args) {
				return args[i+1]
			}
			return ""
		}
		if strings.HasPrefix(arg, "-manifest=") {
			return strings.TrimPrefix(arg, "-manifest=")
		}
		if strings.HasPrefix(arg, "--manifest=") {
			return strings.TrimPrefix(arg, "--manifest=")
		}
	}
	return ""
}

func pathHasPrefixVariant(path string, prefix string) bool {
	for _, pathVariant := range pathAliasVariants(path) {
		for _, prefixVariant := range pathAliasVariants(prefix) {
			if strings.HasPrefix(pathVariant, prefixVariant) {
				return true
			}
		}
	}
	return false
}

func pathAliasVariants(path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	cleaned := filepath.Clean(path)
	variants := []string{cleaned}
	if strings.HasPrefix(cleaned, "/tmp/") {
		variants = append(variants, "/private"+cleaned)
	}
	if strings.HasPrefix(cleaned, "/private/tmp/") {
		variants = append(variants, strings.TrimPrefix(cleaned, "/private"))
	}
	return uniqueStrings(variants)
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func runDaemon(hostSocket, socketMode string, socketUID, socketGID int, workDir string) error {
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return fmt.Errorf("create work dir %s: %w", workDir, err)
	}
	config.New(workDir, nil)

	networkServer, err := hostWireGuardModule()
	if err != nil {
		return err
	}

	hostServer := hostagentdaemon.NewServer(networkServer)
	shutdownCh := make(chan struct{}, 1)
	hostServer.SetShutdownFunc(func() {
		select {
		case shutdownCh <- struct{}{}:
		default:
		}
	})
	defer func() {
		if err := hostServer.Close(); err != nil {
			log.Warnf("Host agent cleanup failed: %v", err)
		}
	}()
	uid, gid := socketOwner(socketUID, socketGID)

	mode, err := parseFileMode(socketMode)
	if err != nil {
		return err
	}

	hostEndpoint, err := serveUnixGRPC(hostSocket, mode, uid, gid, func(server *grpc.Server) {
		hostagentpb.RegisterHostAgentServer(server, hostServer)
	})
	if err != nil {
		return err
	}

	endpoints := []*grpcEndpoint{hostEndpoint}
	defer func() {
		for _, endpoint := range endpoints {
			endpoint.close()
		}
	}()
	log.Infof("Serving host agent on %s", hostSocket)

	errCh := make(chan error, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint := endpoint
		go func() {
			errCh <- endpoint.serve()
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		log.Infof("Received %s, stopping", sig)
		for _, endpoint := range endpoints {
			endpoint.server.Stop()
		}
		return hostServer.Close()
	case <-shutdownCh:
		log.Info("Shutdown RPC requested, stopping")
		for _, endpoint := range endpoints {
			endpoint.server.Stop()
		}
		return hostServer.Close()
	case err := <-errCh:
		for _, endpoint := range endpoints {
			endpoint.server.Stop()
		}
		cleanupErr := hostServer.Close()
		if errors.Is(err, grpc.ErrServerStopped) {
			err = nil
		}
		if err != nil && cleanupErr != nil {
			return fmt.Errorf("%w; cleanup failed: %w", err, cleanupErr)
		}
		if cleanupErr != nil {
			return cleanupErr
		}
		return err
	}
}

type grpcEndpoint struct {
	socket   string
	listener net.Listener
	server   *grpc.Server
}

func serveUnixGRPC(socketPath string, mode os.FileMode, uid, gid int, register func(*grpc.Server)) (*grpcEndpoint, error) {
	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		return nil, fmt.Errorf("create socket directory: %w", err)
	}
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("listen on %s: %w", socketPath, err)
	}
	if err := os.Chmod(socketPath, mode); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("chmod %s: %w", socketPath, err)
	}
	if uid >= 0 || gid >= 0 {
		if err := os.Chown(socketPath, uid, gid); err != nil {
			_ = listener.Close()
			return nil, fmt.Errorf("chown %s: %w", socketPath, err)
		}
	}

	server := grpc.NewServer()
	register(server)
	return &grpcEndpoint{socket: socketPath, listener: listener, server: server}, nil
}

func (e *grpcEndpoint) serve() error {
	return e.server.Serve(e.listener)
}

func (e *grpcEndpoint) close() {
	e.server.Stop()
	_ = e.listener.Close()
	_ = os.Remove(e.socket)
}

func hostWireGuardModule() (networkmodule.Module, error) {
	mod, err := wireguardmodule.New()
	if err != nil {
		return nil, err
	}
	return mod, nil
}

func parseFileMode(mode string) (os.FileMode, error) {
	parsed, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid socket mode %q: %w", mode, err)
	}
	return os.FileMode(parsed), nil
}

func socketOwner(uid, gid int) (int, int) {
	if uid < 0 {
		uid = envInt("SUDO_UID", -1)
	}
	if gid < 0 {
		gid = envInt("SUDO_GID", -1)
	}
	return uid, gid
}

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func defaultWorkDir() string {
	if workDir := os.Getenv("PROTOS_HOSTAGENT_WORKDIR"); workDir != "" {
		return workDir
	}
	if runtime.GOOS == "darwin" {
		return "/Library/Application Support/Protos/hostagent"
	}
	return "/var/lib/protos/hostagent"
}
