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
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Masterminds/semver/v3"
	libp2phost "github.com/libp2p/go-libp2p/core/host"
	"github.com/pkg/errors"

	"github.com/protosio/protos/apic"
	"github.com/protosio/protos/internal/app"
	"github.com/protosio/protos/internal/banner"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/dns"
	"github.com/protosio/protos/internal/invitations"
	"github.com/protosio/protos/internal/invitations/mdns"
	"github.com/protosio/protos/internal/membership"
	"github.com/protosio/protos/internal/network"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/provisioners"
	"github.com/protosio/protos/internal/release"
	appruntime "github.com/protosio/protos/internal/runtime"
	"github.com/protosio/protos/internal/swarmionlink"
	"github.com/protosio/protos/internal/tasks"
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
	cfg           config.Config
	version       string
	capabilities  Capabilities
	stoppers      map[string]func() error
	stopMu        sync.Mutex
	stopped       bool
	localKey      *pcrypto.Key
	peerHost      libp2phost.Host
	dbCloseOnce   sync.Once
	hostCloseOnce sync.Once
	initOnce      sync.Once
	initCh        chan struct{}

	DB                 *db.DB
	Manager            *user.Manager
	KeyManager         *pcrypto.Manager
	AppManager         *app.Manager
	NetworkManager     *network.Manager
	ProvisionerManager *provisioners.Manager
	P2PManager         *p2p.P2P
	TaskManager        *tasks.Manager
	appRuntime         appruntime.RuntimePlatform

	networkMu          sync.Mutex
	networkLifecycleMu sync.Mutex
	networkDesired     bool
	networkEnabled     bool
	networkState       string
	networkMessage     string
	networkExternalDNS string
	networkDNSStopper  func() error
	dbNotifier         *DBNotifier
	inviteManager      *invitations.Manager
}

func StartUp(configFile string, version *semver.Version, opts Options) {
	node, err := NewNode(configFile, version, opts)
	if err != nil {
		log.Fatal(err)
	}
	defer node.Close()

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
	log.Infof("resolved Protos workdir: %s", workDir)

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

	log.Infof("loading local key from workdir: %s", cfg.WorkDir)
	lkey, err := pcrypto.GetLocalKey(cfg.WorkDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get local key: %w", err)
	}

	log.Infof("creating application-owned libp2p host on port %d", cfg.P2PPort)
	peerHost, err := p2p.NewHost(lkey, cfg.P2PPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create application p2p host: %w", err)
	}
	cleanupHost := func() {
		_ = peerHost.Close()
	}

	registry, err := swarmionlink.NewRegistry(peerHost)
	if err != nil {
		cleanupHost()
		return nil, fmt.Errorf("failed to create shared p2p protocol registry: %w", err)
	}

	// Reserve every Protos application protocol through the same registry before
	// Swarmion opens its borrowed session. This makes ownership deterministic and
	// collision-safe across both consumers.
	p2pManager, err := p2p.NewManagerWithRegistry(peerHost, registry, nil, nil, cfg.P2PPort)
	if err != nil {
		cleanupHost()
		return nil, fmt.Errorf("failed to register application p2p protocols: %w", err)
	}
	cleanupP2P := func() {
		_ = p2pManager.Close()
		cleanupHost()
	}

	log.Infof("opening Protos database in workdir: %s", cfg.WorkDir)
	dbcli, err := db.Open(cfg.WorkDir, config.DBName, lkey, registry.Link())
	if err != nil {
		cleanupP2P()
		return nil, err
	}
	if err := p2pManager.SetExternalDB(dbcli); err != nil {
		_ = dbcli.Close()
		cleanupP2P()
		return nil, fmt.Errorf("failed to attach database to application p2p manager: %w", err)
	}
	log.Infof("opened Protos database; initialized=%t", dbcli.Initialized())

	keyManager := pcrypto.CreateManager(dbcli)
	userManager := user.CreateManager(dbcli, keyManager)
	inviteManager, err := invitations.NewManager(mdns.NewChannel())
	if err != nil {
		_ = dbcli.Close()
		cleanupP2P()
		return nil, fmt.Errorf("failed to create invite manager: %w", err)
	}
	node := &Node{
		cfg:            cfg,
		version:        version.String(),
		capabilities:   caps,
		stoppers:       map[string]func() error{},
		localKey:       lkey,
		peerHost:       peerHost,
		initCh:         make(chan struct{}),
		DB:             dbcli,
		KeyManager:     keyManager,
		Manager:        userManager,
		TaskManager:    tasks.NewManager(dbcli),
		P2PManager:     p2pManager,
		networkDesired: caps.Network,
		inviteManager:  inviteManager,
	}
	node.stoppers["p2p-server"] = p2pManager.StopServer
	node.stoppers["p2p-scope"] = p2pManager.Close
	node.stoppers["p2p-host"] = node.closePeerHost
	if caps.Network {
		node.networkState = networkRuntimeStateDisabled
	} else {
		node.networkState = networkRuntimeStateUnsupported
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
	n.AppManager = app.CreateManager(n.localKey.GetID(), n.appRuntime, n.DB, n.TaskManager)

	if n.P2PManager == nil {
		return fmt.Errorf("application p2p manager is not configured")
	}
	n.P2PManager.SetAppManager(n.AppManager)
	if n.TaskManager != nil {
		n.TaskManager.SetExecutorPeerID(n.P2PManager.PeerID())
	}
	n.P2PManager.SetTaskManager(n.TaskManager)
	if !n.DB.Initialized() {
		initOriginPublicKey, err := readInitOriginPublicKey()
		if err != nil {
			return err
		}
		if initOriginPublicKey != "" {
			if err := n.P2PManager.SetInitPeerPublicKey(initOriginPublicKey); err != nil {
				return fmt.Errorf("failed to configure init peer: %w", err)
			}
		}
	}
	if n.capabilities.Network {
		n.P2PManager.SetNetworkInspector(n)
	}

	n.ProvisionerManager, err = provisioners.CreateManager(
		n.DB,
		n.Manager,
		n.KeyManager,
		n.P2PManager,
		n.TaskManager,
		hetzner.NewFactory(),
		scaleway.NewFactory(),
		localmacos.NewFactory(),
	)
	if err != nil {
		return fmt.Errorf("failed to create provisioner manager: %w", err)
	}
	n.ProvisionerManager.SetProvisionerMutationEnabled(n.capabilities.Provision)

	if n.capabilities.API {
		apiStopper, err := apic.StartGRPCServer(n.cfg.WorkDir, n.version, n.APIServices())
		if err != nil {
			return err
		}
		n.stoppers["api"] = apiStopper
	}
	n.stoppers["invitations"] = func() error {
		if n.inviteManager != nil {
			n.inviteManager.Stop()
		}
		return nil
	}

	p2pStopper, err := n.P2PManager.StartServer()
	if err != nil {
		return err
	}
	n.stoppers["p2p-server"] = p2pStopper

	if n.capabilities.Network {
		normalizedDNS, err := network.NormalizeDNSServer(n.cfg.ExternalDNS)
		if err != nil {
			return fmt.Errorf("invalid external DNS server %q: %w", n.cfg.ExternalDNS, err)
		}
		n.networkExternalDNS = normalizedDNS
		n.stoppers["network"] = func() error {
			return n.DisableNetwork(context.Background())
		}
		if err := n.enableNetworkAtStartup(context.Background()); err != nil {
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
		if imageManager, ok := n.appRuntime.(p2p.ImageManager); ok {
			n.P2PManager.SetImageManager(imageManager)
			n.AppManager.SetImageResolver(n.P2PManager)
		}
	}

	dbNotifier := &DBNotifier{
		database:     n.DB,
		cm:           n.ProvisionerManager,
		um:           n.Manager,
		am:           n.AppManager,
		network:      n,
		p2pm:         n.P2PManager,
		capabilities: n.capabilities,
		externalDNS:  n.networkExternalDNS,
	}
	n.dbNotifier = dbNotifier
	for _, registration := range []struct {
		model    any
		notifier db.Notifier
	}{
		{model: db.CLOUD_MACHINE_METADATA{}, notifier: dbNotifier},
		{model: db.MACHINE{}, notifier: dbNotifier},
		{model: db.PEER{}, notifier: dbNotifier},
		{model: db.EXIT_ROUTE{}, notifier: dbNotifier},
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
		n.DB.RegisterRuntimeChangeCallback(n.AppManager)
	}
	if n.TaskManager != nil {
		n.stoppers["task-runner"] = n.TaskManager.Start(context.Background(), 2*time.Second)
	}

	log.Info("Started all servers successfully")
	if n.DB.Initialized() {
		dbNotifier.Notify()
		if n.capabilities.AppRuntime && n.AppManager != nil {
			n.AppManager.Notify()
		}
	} else {
		log.Info("DB not initialized. Waiting for local init or remote init")
	}
	n.stoppers["db-reconcile"] = db.StartPeriodicNotifier(dbNotifier, 5*time.Second)
	// App-runtime reconciliation gets its own periodic notifier so that
	// convergence of the imperative app lifecycle does not depend on the database
	// notifier's control-plane path (checkpoint catch-up and network/peer
	// reconfiguration), which can fail under cleanup/load. This is the safety net
	// that recovers any reconcile notification dropped while a reconcile task was
	// already in flight.
	if n.capabilities.AppRuntime && n.AppManager != nil {
		n.stoppers["app-reconcile"] = db.StartPeriodicNotifier(n.AppManager, 5*time.Second)
	}
	return nil
}

func (n *Node) APIServices() *apic.Services {
	return &apic.Services{
		DB:                 n.DB,
		Manager:            n.Manager,
		KeyManager:         n.KeyManager,
		AppManager:         n.AppManager,
		NetworkManager:     n.NetworkManager,
		NetworkControl:     n,
		ProvisionerManager: n.ProvisionerManager,
		P2PManager:         n.P2PManager,
		TaskManager:        n.TaskManager,
		Invites:            n.inviteManager,
		CanProvision:       n.capabilities.Provision,
		WorkDir:            n.cfg.WorkDir,
		Capabilities:       n.capabilities.String(),
		P2PPort:            n.cfg.P2PPort,
		InitFunc:           n.Init,
		MarkInitialized:    n.markInitialized,
		ReleaseFetch:       n.GetProtosAvailableReleases,
	}
}

func (n *Node) Init(username string, name string, organisation string) error {
	log.Debug("Performing initialization")
	if err := rejectFreshInitializationForRepository(n.DB.RepositoryReadiness()); err != nil {
		return err
	}
	flushNotifications := n.DB.DeferNotifications()
	defer flushNotifications(true)

	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to init. Could not retrieve hostname: %w", err)
	}
	if err := n.DB.Init(); err != nil {
		return fmt.Errorf("failed to init. Error while initializing db: %w", err)
	}
	if _, err := db.EnsureOrganisation(n.DB, organisation); err != nil {
		return fmt.Errorf("failed to create organisation: %w", err)
	}

	adminUser, err := n.Manager.CreateUser(username, name, true)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	if err := n.Manager.AddDevice(adminUser.Username, hostname, n.localKey); err != nil {
		return fmt.Errorf("failed to add user. Error while creating user device: %w", err)
	}
	n.markInitialized()
	flushNotifications(false)
	return nil
}

func rejectFreshInitializationForRepository(readiness db.RepositoryReadiness) error {
	if readiness.ExistingRepository && !readiness.Initialized {
		if readiness.BootstrapPending {
			return fmt.Errorf("failed to init: existing repository is awaiting bootstrap recovery")
		}
		if readiness.BootstrapError != nil {
			return fmt.Errorf("failed to init: existing repository recovery failed: %w", readiness.BootstrapError)
		}
		return fmt.Errorf("failed to init: existing repository is not ready")
	}
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

func readInitOriginPublicKey() (string, error) {
	data, err := os.ReadFile(config.InitOriginPublicKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read init origin public key: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
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
	n.stopMu.Lock()
	if n.stopped {
		n.stopMu.Unlock()
		return
	}
	n.stopped = true
	stoppers := make(map[string]func() error, len(n.stoppers))
	for name, stopper := range n.stoppers {
		stoppers[name] = stopper
	}
	database := n.DB
	n.stopMu.Unlock()

	stopNodeComponents(stoppers, func() {
		if database == nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		n.dbCloseOnce.Do(func() {
			if err := database.PrepareSwarmionShutdown(ctx); err != nil {
				log.Debugf("failed to prepare swarmion shutdown: %s", err.Error())
			}
		})
	})
}

func stopNodeComponents(stoppers map[string]func() error, prepareSwarmionShutdown func()) {
	if stopper := stoppers["task-runner"]; stopper != nil {
		if err := stopper(); err != nil {
			log.Error(err)
		}
	}

	names := make([]string, 0, len(stoppers))
	for name := range stoppers {
		if name != "task-runner" && name != "p2p-scope" && name != "p2p-host" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		if err := stoppers[name](); err != nil {
			log.Error(err)
		}
	}

	// All writers and ingress paths are now stopped. Drain and close Swarmion
	// before unregistering the application scope from the shared host.
	prepareSwarmionShutdown()
	if stopper := stoppers["p2p-scope"]; stopper != nil {
		if err := stopper(); err != nil {
			log.Error(err)
		}
	}
	// The physical host is application-owned. It is always the final network
	// resource closed, after Swarmion/DB and every scoped Protos handler.
	if stopper := stoppers["p2p-host"]; stopper != nil {
		if err := stopper(); err != nil {
			log.Error(err)
		}
	}
}

func (n *Node) Close() {
	n.Stop()
	n.closeDB()
}

func (n *Node) closeDB() {
	if n == nil {
		return
	}
	n.dbCloseOnce.Do(func() {
		if n.DB != nil {
			_ = n.DB.Close()
		}
	})
}

func (n *Node) closePeerHost() error {
	if n == nil {
		return nil
	}
	var closeErr error
	n.hostCloseOnce.Do(func() {
		if n.peerHost != nil {
			closeErr = n.peerHost.Close()
		}
	})
	return closeErr
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
	network      networkRuntime
	p2pm         *p2p.P2P
	capabilities Capabilities
	externalDNS  string
	mu           sync.Mutex
}

type networkRuntime interface {
	NetworkEnabled() bool
	ConfigureNetworkPeers([]provisioners.InstanceInfo, []user.UserDevice, []network.AppRoute, []network.ExitRoute) error
}

func (dbn *DBNotifier) Notify() {
	dbn.mu.Lock()
	defer dbn.mu.Unlock()

	if !dbn.database.Initialized() {
		return
	}

	catchUpCtx, catchUpCancel := context.WithTimeout(context.Background(), 10*time.Second)
	catchUpErr := dbn.database.CatchUpCheckpointStrict(catchUpCtx, "database notifier reconcile")
	catchUpCancel()
	if catchUpErr != nil {
		if db.IsRetryableCheckpointCatchUp(catchUpErr) {
			log.Debugf("deferred database notifier reconcile after retryable checkpoint catch-up: %s", catchUpErr.Error())
		} else {
			log.Error(fmt.Errorf("failed to catch up checkpoint before database notifier reconcile: %w", catchUpErr))
		}
		return
	}

	if dbn.capabilities.Provision {
		if err := dbn.cm.QueueDesiredInstanceReconciles(context.Background()); err != nil {
			log.Error(fmt.Errorf("failed to queue desired instance reconciliation: %w", err))
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
	instances = provisioners.ActiveInstances(instances)

	replicationInstances, err := dbn.cm.GetInstances(false)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve replication instances: %w", err))
		return
	}
	replicationInstances = membership.FilterInstances(replicationInstances, peerIDs)
	replicationInstances = provisioners.ActiveInstances(replicationInstances)

	userDevices, err := dbn.um.GetAllDevices(false)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve user devices: %w", err))
		return
	}
	userDevices = membership.FilterDevices(userDevices, peerIDs)

	appRoutes := []network.AppRoute{}
	if dbn.am != nil {
		activeInstanceIDs := make(map[string]struct{}, len(instances))
		for _, instance := range instances {
			activeInstanceIDs[instance.ID] = struct{}{}
		}
		apps, err := dbn.am.GetAll()
		if err != nil {
			log.Error(fmt.Errorf("failed to retrieve apps for network routing: %w", err))
			return
		}
		for _, application := range apps {
			if application.IP == nil {
				continue
			}
			if _, found := activeInstanceIDs[application.InstanceID]; !found {
				continue
			}
			appRoutes = append(appRoutes, network.AppRoute{
				InstanceID: application.InstanceID,
				IP:         application.IP,
			})
		}
	}

	exitRoutes, err := network.GetExitRoutes(dbn.database)
	if err != nil {
		log.Error(fmt.Errorf("failed to retrieve exit routes: %w", err))
		return
	}
	if dbn.capabilities.Network && dbn.network != nil && dbn.network.NetworkEnabled() {
		dbn.configureDNSForwarder(exitRoutes)
	}

	if dbn.capabilities.Network && dbn.network != nil && dbn.network.NetworkEnabled() {
		if err := dbn.network.ConfigureNetworkPeers(instances, userDevices, appRoutes, exitRoutes); err != nil {
			log.Error(fmt.Errorf("failed to configure network peers: %w", err))
			return
		}
	}

	err = dbn.p2pm.ConfigurePeers(membership.Machines(instances, userDevices))
	if err != nil {
		log.Error(fmt.Errorf("failed to configure p2p peers: %w", err))
		return
	}

	if err := dbn.database.ReconcileReplicationPeers(context.Background(), membership.ReplicationCandidates(replicationInstances, userDevices)); err != nil {
		log.Error(fmt.Errorf("failed to reconcile swarmion replication metadata: %w", err))
	}
	if dbn.capabilities.AppRuntime && dbn.am != nil {
		dbn.am.Notify()
	}
}

func activeMembershipPeerIDs(instances []provisioners.InstanceInfo, devices []user.UserDevice) map[string]struct{} {
	peerIDs := make(map[string]struct{}, len(instances)+len(devices))
	for _, instance := range instances {
		if !provisioners.IsActiveInstance(instance) {
			continue
		}
		peerID, err := instance.GetPeerID()
		if err == nil && strings.TrimSpace(peerID) != "" {
			peerIDs[peerID] = struct{}{}
		}
	}
	for _, device := range devices {
		peerID, err := db.PeerIDFromPublicKeyString(device.GetPublicKey())
		if err == nil && strings.TrimSpace(peerID) != "" {
			peerIDs[peerID] = struct{}{}
		}
	}
	return peerIDs
}

func (dbn *DBNotifier) configureDNSForwarder(exitRoutes []network.ExitRoute) {
	if dbn.um == nil {
		return
	}
	dnsServer := dbn.externalDNS
	currentDevice, ok, err := dbn.um.GetCurrentDeviceIfExists()
	if err != nil {
		log.Debugf("failed to resolve current device for DNS exit route: %s", err.Error())
		dns.SetExternalServer(dnsServer)
		return
	}
	if !ok {
		dns.SetExternalServer(dnsServer)
		return
	}
	for _, route := range exitRoutes {
		if route.DeviceID != currentDevice.ID || network.NormalizeExitRouteStatus(route.DesiredStatus) != network.ExitRouteStatusActive {
			continue
		}
		if network.ExitRouteUsesFullTunnel(route) && route.DNSServer != "" {
			dnsServer = route.DNSServer
		}
		break
	}
	dns.SetExternalServer(dnsServer)
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
