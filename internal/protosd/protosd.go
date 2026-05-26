package protosd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	stdruntime "runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"

	"github.com/protosio/protos/apic"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/banner"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/membership"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	appruntime "github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
	"github.com/protosio/protos/provisioners/hetzner"
	"github.com/protosio/protos/provisioners/local_macos"
	"github.com/protosio/protos/provisioners/scaleway"
)

const DNSPort = 53

var log = util.GetLogger("protosd")

type Capabilities struct {
	API        bool
	Provision  bool
	Network    bool
	AppRuntime bool
}

func DefaultCapabilities() Capabilities {
	caps := Capabilities{
		API:     true,
		Network: true,
	}
	if stdruntime.GOOS == "darwin" {
		caps.Provision = true
		return caps
	}
	caps.AppRuntime = true
	return caps
}

func ParseCapabilities(value string) (Capabilities, error) {
	if env := strings.TrimSpace(os.Getenv("PROTOS_CAPABILITIES")); env != "" && strings.TrimSpace(value) == "" {
		value = env
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return DefaultCapabilities(), nil
	}

	caps := Capabilities{}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		switch field {
		case "":
		case "none":
			caps = Capabilities{}
		case "default":
			caps = DefaultCapabilities()
		case "all":
			caps = Capabilities{API: true, Provision: true, Network: true, AppRuntime: true}
		case "api":
			caps.API = true
		case "provision", "provisioner", "machine-provisioner":
			caps.Provision = true
		case "network", "host-network":
			caps.Network = true
		case "app-runtime", "apps", "runtime":
			caps.AppRuntime = true
		case "no-api", "!api":
			caps.API = false
		case "no-provision", "no-provisioner", "!provision", "!provisioner":
			caps.Provision = false
		case "no-network", "!network":
			caps.Network = false
		case "no-app-runtime", "!app-runtime":
			caps.AppRuntime = false
		default:
			return caps, fmt.Errorf("unknown protosd capability %q", field)
		}
	}
	return caps, nil
}

func (c Capabilities) String() string {
	enabled := []string{"db", "p2p"}
	if c.API {
		enabled = append(enabled, "api")
	}
	if c.Provision {
		enabled = append(enabled, "provisioner")
	}
	if c.Network {
		enabled = append(enabled, "network")
	}
	if c.AppRuntime {
		enabled = append(enabled, "app-runtime")
	}
	return strings.Join(enabled, ",")
}

type Options struct {
	DataDir      string
	Capabilities string
}

type Node struct {
	cfg          config.Config
	version      string
	capabilities Capabilities
	stoppers     map[string]func() error
	localKey     *pcrypto.Key
	initOnce     sync.Once
	initCh       chan struct{}

	DB             *db.DB
	Manager        *user.Manager
	KeyManager     *pcrypto.Manager
	AppManager     *app.Manager
	NetworkManager *network.Manager
	CloudManager   *provisioners.Manager
	P2PManager     *p2p.P2P
	appRuntime     appruntime.RuntimePlatform
}

func StartUp(configFile string, version *semver.Version, opts Options) {
	node, err := NewNode(configFile, version, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer node.closeDB()

	if err := node.Start(); err != nil {
		log.Fatal(err)
	}
	node.Wait()
}

func NewNode(configFile string, version *semver.Version, opts Options) (*Node, error) {
	cfg := config.Load(configFile, version)
	if strings.TrimSpace(opts.DataDir) != "" {
		cfg.WorkDir = opts.DataDir
	}
	if cfg.WorkDir == "" {
		cfg.WorkDir = defaultWorkDir()
	}
	workDir, err := resolvePath(cfg.WorkDir)
	if err != nil {
		return nil, err
	}
	cfg.WorkDir = workDir
	config.Get().WorkDir = workDir

	capabilityValue := strings.TrimSpace(opts.Capabilities)
	if capabilityValue == "" {
		capabilityValue = strings.TrimSpace(os.Getenv("PROTOS_CAPABILITIES"))
	}
	if capabilityValue == "" && len(cfg.Capabilities) > 0 {
		capabilityValue = strings.Join(cfg.Capabilities, ",")
	}
	caps, err := ParseCapabilities(capabilityValue)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create Protos directory %q: %w", cfg.WorkDir, err)
	}

	lkey, err := pcrypto.GetLocalKey(cfg.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get local key: %w", err)
	}

	dbcli, err := db.Open(cfg.WorkDir, config.DBName, lkey)
	if err != nil {
		return nil, err
	}

	keyManager := pcrypto.CreateManager(dbcli)
	userManager := user.CreateManager(dbcli, keyManager)
	node := &Node{
		cfg:          cfg,
		version:      version.String(),
		capabilities: caps,
		stoppers:     map[string]func() error{},
		localKey:     lkey,
		initCh:       make(chan struct{}),
		DB:           dbcli,
		KeyManager:   keyManager,
		Manager:      userManager,
	}
	if dbcli.Initialized() {
		node.markInitialized()
	}
	return node, nil
}

func (n *Node) Start() error {
	log.Infof("starting Protos daemon with capabilities: %s", n.capabilities.String())
	banner.PrintBanner(n.cfg)

	var err error
	if n.capabilities.Network {
		n.NetworkManager, err = network.NewManager()
		if err != nil {
			return err
		}
	}

	n.appRuntime = appruntime.Create(n.NetworkManager, n.cfg.RuntimeEndpoint)
	n.AppManager = app.CreateManager(n.localKey.GetID(), n.appRuntime, n.DB)

	n.P2PManager, err = p2p.NewManager(n.localKey, n.AppManager, n.DB, n.cfg.P2PPort)
	if err != nil {
		return fmt.Errorf("failed to create p2p manager: %w", err)
	}

	n.CloudManager, err = provisioners.CreateManager(
		n.DB,
		n.Manager,
		n.KeyManager,
		n.P2PManager,
		hetzner.NewFactory(),
		scaleway.NewFactory(),
		localmacos.NewFactory(),
	)
	if err != nil {
		return fmt.Errorf("failed to create provisioner manager: %w", err)
	}

	if n.capabilities.API {
		apiStopper, err := apic.StartGRPCServer(n.cfg.WorkDir, n.version, n.APIServices())
		if err != nil {
			return err
		}
		n.stoppers["api"] = apiStopper
	}

	p2pStopper, err := n.P2PManager.StartServer()
	if err != nil {
		return err
	}
	n.stoppers["p2p"] = p2pStopper

	if n.capabilities.Network {
		if err := n.NetworkManager.Init(n.localKey, n.cfg.InternalDomain); err != nil {
			return fmt.Errorf("failed to initialize network reconciler: %w", err)
		}
		n.stoppers["network"] = func() error {
			log.Info("bringing down network")
			return n.NetworkManager.Down()
		}
		dnsStopper := dns.StartServer(n.localKey, n.dnsPort(), n.cfg.ExternalDNS, n.cfg.InternalDomain, n.AppManager)
		n.stoppers["dns"] = dnsStopper
		if err := n.configureLocalResolver(); err != nil {
			return err
		}
	}

	if n.capabilities.AppRuntime {
		if n.appRuntime == nil {
			return fmt.Errorf("app runtime capability is enabled but no runtime platform is available on %s", stdruntime.GOOS)
		}
		if err := n.appRuntime.Init(); err != nil {
			return err
		}
	}

	dbNotifier := &DBNotifier{
		database:     n.DB,
		cm:           n.CloudManager,
		um:           n.Manager,
		am:           n.AppManager,
		nm:           n.NetworkManager,
		p2pm:         n.P2PManager,
		capabilities: n.capabilities,
	}
	for _, registration := range []struct {
		model    any
		notifier db.Notifier
	}{
		{model: db.CLOUD_MACHINE_METADATA{}, notifier: dbNotifier},
		{model: db.MACHINE{}, notifier: dbNotifier},
		{model: db.PEER{}, notifier: dbNotifier},
		{model: db.USER{}, notifier: dbNotifier},
		{model: db.USER_DEVICE_METADATA{}, notifier: dbNotifier},
		{model: db.APP{}, notifier: dbNotifier},
	} {
		if err := n.DB.RegisterNotifier(registration.model, registration.notifier); err != nil {
			return fmt.Errorf("failed to register database notifier: %w", err)
		}
	}
	if n.capabilities.AppRuntime {
		if err := n.DB.RegisterNotifier(db.APP{}, n.AppManager); err != nil {
			return fmt.Errorf("failed to register app notifier: %w", err)
		}
	}

	log.Info("Started all servers successfully")
	if n.DB.Initialized() {
		dbNotifier.Notify()
	} else {
		log.Info("DB not initialized. Waiting for local init or remote init")
	}
	n.stoppers["db-reconcile"] = db.StartPeriodicNotifier(dbNotifier, 5*time.Second)
	return nil
}

func (n *Node) APIServices() *apic.Services {
	return &apic.Services{
		DB:             n.DB,
		Manager:        n.Manager,
		KeyManager:     n.KeyManager,
		AppManager:     n.AppManager,
		NetworkManager: n.NetworkManager,
		CloudManager:   n.CloudManager,
		P2PManager:     n.P2PManager,
		CanProvision:   n.capabilities.Provision,
		InitFunc:       n.Init,
		ReleaseFetch:   n.GetProtosAvailableReleases,
	}
}

func (n *Node) Init(username string, name string, organization string) error {
	log.Debug("Performing initialization")
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to init. Could not retrieve hostname: %w", err)
	}
	if err := n.DB.Init(); err != nil {
		return fmt.Errorf("failed to init. Error while initializing db: %w", err)
	}

	adminUser, err := n.Manager.CreateUser(username, name, true)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	if err := n.Manager.AddDevice(adminUser.Username, hostname, n.localKey); err != nil {
		return fmt.Errorf("failed to add user. Error while creating user device: %w", err)
	}
	n.markInitialized()
	return nil
}

func (n *Node) GetProtosAvailableReleases() (release.Releases, error) {
	var releases release.Releases
	resp, err := http.Get(config.ReleasesURL)
	if err != nil {
		return releases, errors.Wrapf(err, "Failed to retrieve releases from '%s'", config.ReleasesURL)
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return releases, errors.Wrap(err, "Failed to JSON decode the releases response")
	}
	if len(releases.Releases) == 0 {
		return releases, errors.Errorf("Something went wrong. Parsed 0 releases from '%s'", config.ReleasesURL)
	}
	return releases, nil
}

func (n *Node) Wait() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigs)

	sig := <-sigs
	log.Infof("Received OS signal %s. Terminating", sig.String())
	n.Stop()
	log.Info("Shutdown completed")
}

func (n *Node) Stop() {
	for _, stopper := range n.stoppers {
		if err := stopper(); err != nil {
			log.Error(err)
		}
	}
}

func (n *Node) closeDB() {
	if n.DB != nil {
		_ = n.DB.Close()
	}
}

func (n *Node) markInitialized() {
	n.initOnce.Do(func() {
		close(n.initCh)
	})
}

func (n *Node) dnsPort() int {
	if stdruntime.GOOS == "darwin" {
		return config.LocalDNSPort
	}
	return DNSPort
}

func (n *Node) configureLocalResolver() error {
	if stdruntime.GOOS == "darwin" {
		return nil
	}
	if n.localKey == nil {
		return fmt.Errorf("cannot configure local resolver without a local key")
	}
	resolver := fmt.Sprintf("nameserver %s\n", n.localKey.IPv6Address().String())
	if err := os.WriteFile("/etc/resolv.conf", []byte(resolver), 0644); err != nil {
		return fmt.Errorf("configure local resolver: %w", err)
	}
	log.Infof("Configured local resolver to use Protos DNS at %s", n.localKey.IPv6Address().String())
	return nil
}

type DBNotifier struct {
	database     *db.DB
	cm           *provisioners.Manager
	um           *user.Manager
	am           *app.Manager
	nm           *network.Manager
	p2pm         *p2p.P2P
	capabilities Capabilities
	mu           sync.Mutex
}

func (dbn *DBNotifier) Notify() {
	dbn.mu.Lock()
	defer dbn.mu.Unlock()

	if !dbn.database.Initialized() {
		return
	}

	catchUpCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	if err := dbn.database.CatchUpFinalized(catchUpCtx, "protosd reconcile desired state"); err != nil {
		cancel()
		log.Error(fmt.Errorf("failed to catch up finalized DB state: %w", err))
		return
	}
	cancel()

	if dbn.capabilities.Provision {
		if err := dbn.cm.ReconcileDesiredInstances(); err != nil {
			log.Error(fmt.Errorf("failed to reconcile desired instances: %w", err))
		}
	}

	peerIDs, err := db.GetPeerIDs(dbn.database)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve peer membership: %w", err))
		return
	}

	instances, err := dbn.cm.GetInstances(true)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve instances: %w", err))
		return
	}
	instances = membership.FilterInstances(instances, peerIDs)

	witnessInstances, err := dbn.cm.GetInstances(false)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve witness instances: %w", err))
		return
	}
	witnessInstances = membership.FilterInstances(witnessInstances, peerIDs)

	userDevices, err := dbn.um.GetAllDevices(false)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve user devices: %w", err))
		return
	}
	userDevices = membership.FilterDevices(userDevices, peerIDs)

	appRoutes := []network.AppRoute{}
	if dbn.am != nil {
		apps, err := dbn.am.GetAll()
		if err != nil {
			log.Error(fmt.Errorf("failed to retrieve apps for network routing: %w", err))
			return
		}
		for _, application := range apps {
			if application.IP == nil {
				continue
			}
			appRoutes = append(appRoutes, network.AppRoute{
				InstanceID: application.InstanceID,
				IP:         application.IP,
			})
		}
	}

	if dbn.capabilities.Network && dbn.nm != nil {
		if err := dbn.nm.ConfigurePeers(instances, userDevices, appRoutes); err != nil {
			log.Error(fmt.Errorf("failed to configure network peers: %w", err))
			return
		}
	}

	if dbn.capabilities.AppRuntime && dbn.am != nil {
		dbn.am.Notify()
	}

	err = dbn.p2pm.ConfigurePeers(membership.Machines(instances, userDevices))
	if err != nil {
		log.Error(fmt.Errorf("failed to configure p2p peers: %w", err))
		return
	}

	if err := dbn.database.ReconcileWitnesses(context.Background(), membership.WitnessCandidates(witnessInstances, userDevices)); err != nil {
		log.Error(fmt.Errorf("failed to reconcile swarmion witnesses: %w", err))
	}
}

func defaultWorkDir() string {
	if stdruntime.GOOS != "darwin" {
		return "/var/lib/protos"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".protos"
	}
	return filepath.Join(home, ".protos")
}

func resolvePath(path string) (string, error) {
	if path == "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve home directory: %w", err)
	}
	if path == "~" {
		path = home
	} else if strings.HasPrefix(path, "~/") {
		path = filepath.Join(home, path[2:])
	}
	return filepath.Abs(path)
}
