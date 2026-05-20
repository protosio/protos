package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"

	"github.com/protosio/protos/internal/config"
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
		runVM        bool
		manifestPath string
		hostSocket   string
		socketMode   string
		socketUID    int
		socketGID    int
		workDir      string
		logLevel     string
	)
	flag.BoolVar(&runVM, "run-vm", false, "run one VM from a manifest and block until it exits")
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

func runDaemon(hostSocket, socketMode string, socketUID, socketGID int, workDir string) error {
	if err := os.MkdirAll(workDir, 0700); err != nil {
		return fmt.Errorf("create work dir %s: %w", workDir, err)
	}
	config.New(workDir, nil)

	networkServer, closeNetwork, err := hostWireGuardModule()
	if err != nil {
		return err
	}
	defer func() {
		_ = closeNetwork()
	}()

	hostServer := hostagentdaemon.NewServer(networkServer)
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
	defer hostEndpoint.close()

	endpoints := []*grpcEndpoint{hostEndpoint}
	log.Infof("Serving host agent on %s", hostSocket)

	errCh := make(chan error, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint := endpoint
		go func() {
			errCh <- endpoint.serve()
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		log.Infof("Received %s, stopping", sig)
		for _, endpoint := range endpoints {
			endpoint.server.GracefulStop()
		}
		return nil
	case err := <-errCh:
		for _, endpoint := range endpoints {
			endpoint.server.GracefulStop()
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

func hostWireGuardModule() (networkmodule.Module, func() error, error) {
	mod, err := wireguardmodule.New()
	if err != nil {
		return nil, nil, err
	}
	return mod, mod.Close, nil
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
