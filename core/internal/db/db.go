package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/dolthub/vitess/go/vt/sqlparser"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	libp2ppeer "github.com/libp2p/go-libp2p/core/peer"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionadmin "github.com/nustiueudinastea/swarmion/runtime/adminrpc"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime/app"
	swarmiondoltrepo "github.com/nustiueudinastea/swarmion/runtime/doltrepo"
	cueschema "github.com/nustiueudinastea/swarmion/schema-engines/cue"
	declarativeschema "github.com/nustiueudinastea/swarmion/schema-engines/declarative"
	libp2ptransport "github.com/nustiueudinastea/swarmion/transports/libp2p"
	"github.com/protosio/protos/internal/config"
	protoscontracts "github.com/protosio/protos/internal/db/contracts/sql/protos"
	"github.com/protosio/protos/internal/util"
	"github.com/rs/xid"
	"google.golang.org/grpc"
)

const (
	swarmionNamespaceTemplate      = "/protos/db/%s"
	swarmionAdminNamespaceTemplate = "/protos/db/%s/admin"
	swarmionPortOffset             = 1
	swarmionStateDirName           = ".swarmion"

	committedWriteMaxAttempts       = 20
	committedWriteCheckpointTimeout = 45 * time.Second
	checkpointCatchUpMaxAttempts    = 4
	initFromPeerRetryBudget         = 45 * time.Second
	initFromPeerRetryInitialBackoff = time.Second
	initFromPeerRetryMaxBackoff     = 5 * time.Second
)

var (
	errSwarmionCheckpointCatchUpRetryable  = errors.New("swarmion checkpoint catch-up retryable")
	errSwarmionCheckpointedWriteIncomplete = errors.New("swarmion checkpointed write incomplete")
)

type Signer interface {
	Sign(commit string) (string, error)
	Verify(commit string, signature string, publicKey string) error
	PublicKey() string
	GetID() string
	Private() []byte
}

type swarmionSigner struct {
	Signer
	publicKey string
}

func newSwarmionSigner(signer Signer, publicKey libp2pcrypto.PubKey) (swarmionSigner, error) {
	if signer == nil {
		return swarmionSigner{}, fmt.Errorf("signer is nil")
	}
	if publicKey == nil {
		return swarmionSigner{}, fmt.Errorf("swarmion public key is nil")
	}
	publicKeyBytes, err := libp2pcrypto.MarshalPublicKey(publicKey)
	if err != nil {
		return swarmionSigner{}, fmt.Errorf("marshal swarmion public key: %w", err)
	}
	return swarmionSigner{
		Signer:    signer,
		publicKey: base64.StdEncoding.EncodeToString(publicKeyBytes),
	}, nil
}

func (s swarmionSigner) PublicKey() string {
	return s.publicKey
}

func (s swarmionSigner) Verify(commit string, signature string, publicKey string) error {
	if s.Signer == nil {
		return fmt.Errorf("signer is nil")
	}
	if err := s.Signer.Verify(commit, signature, publicKey); err == nil {
		return nil
	}
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return s.Signer.Verify(commit, signature, publicKey)
	}
	pubKey, err := libp2pcrypto.UnmarshalPublicKey(publicKeyBytes)
	if err != nil {
		return s.Signer.Verify(commit, signature, publicKey)
	}
	rawKey, err := pubKey.Raw()
	if err != nil {
		return fmt.Errorf("extract raw swarmion public key: %w", err)
	}
	return s.Signer.Verify(commit, signature, base64.StdEncoding.EncodeToString(rawKey))
}

type Commit struct {
	Hash            string
	Committer       string
	Email           string
	Date            time.Time
	Message         string
	SignerPublicKey string
	ParentHashes    []string
	Refs            []string
}

type DB struct {
	app     *swarmionapp.App
	network *libp2ptransport.Network
	sqldb   *sql.DB

	name       string
	workingDir string
	signer     Signer

	mu                   sync.Mutex
	opMu                 contextMutex
	initialized          bool
	watchCancel          context.CancelFunc
	tableChangeCallbacks *util.Map[string, tableChangeCallback]
	runtimeCallbacks     *util.Map[string, Notifier]
	replicationNoticeSig string
	notificationMu       sync.Mutex
	notificationDepth    int
	pendingNotifyAll     bool
	pendingNotifyTables  map[string]struct{}
}

type contextMutex struct {
	once sync.Once
	ch   chan struct{}
}

func (m *contextMutex) init() {
	m.once.Do(func() {
		m.ch = make(chan struct{}, 1)
		m.ch <- struct{}{}
	})
}

func (m *contextMutex) Lock() {
	_ = m.LockContext(context.Background())
}

func (m *contextMutex) LockContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	m.init()
	select {
	case <-m.ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *contextMutex) Unlock() {
	m.init()
	select {
	case m.ch <- struct{}{}:
	default:
		panic("db operation mutex unlocked while not locked")
	}
}

//go:embed migrations/*.sql
var rootDir embed.FS

func Open(workDir string, dbName string, signer Signer) (*DB, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}
	if dbName == "" {
		return nil, fmt.Errorf("db name is empty")
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workdir: %w", err)
	}

	db := &DB{
		name:                 dbName,
		workingDir:           absWorkDir,
		signer:               signer,
		tableChangeCallbacks: util.NewMap[string, tableChangeCallback](),
		runtimeCallbacks:     util.NewMap[string, Notifier](),
	}

	if err := quarantineIncompleteDatabase(absWorkDir, dbName); err != nil {
		return nil, err
	}
	if databaseExists(absWorkDir, dbName) {
		if err := db.openSwarmion(context.Background(), nil); err != nil {
			return nil, fmt.Errorf("failed to open swarmion db: %w", err)
		}
		if err := db.runMigrations(context.Background()); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return db, nil
}

func (db *DB) Init() error {
	if err := quarantineIncompleteDatabase(db.workingDir, db.name); err != nil {
		return err
	}
	if err := db.openSwarmion(context.Background(), nil); err != nil {
		return fmt.Errorf("failed to init swarmion db: %w", err)
	}

	if err := db.runMigrations(context.Background()); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func databaseExists(workDir string, dbName string) bool {
	_, err := os.Stat(filepath.Join(workDir, dbName, ".dolt", "repo_state.json"))
	return err == nil
}

func quarantineIncompleteDatabase(workDir string, dbName string) error {
	dbDir := filepath.Join(workDir, dbName)
	doltDir := filepath.Join(dbDir, ".dolt")
	if stat, err := os.Stat(doltDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect database directory %q: %w", doltDir, err)
	} else if !stat.IsDir() {
		return nil
	}
	if _, err := os.Stat(filepath.Join(doltDir, "repo_state.json")); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect database repo state: %w", err)
	}
	quarantineDir := filepath.Join(
		workDir,
		fmt.Sprintf("%s.incomplete.%d", dbName, time.Now().UnixNano()),
	)
	if err := os.Rename(dbDir, quarantineDir); err != nil {
		return fmt.Errorf("quarantine incomplete database %q: %w", dbDir, err)
	}
	util.GetLogger("db").Warnf("quarantined incomplete database %q to %q", dbDir, quarantineDir)
	return nil
}

func (db *DB) openSwarmion(ctx context.Context, bootstrapPeers []string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.app != nil {
		return nil
	}

	privateKey, err := libp2pcrypto.UnmarshalEd25519PrivateKey(db.signer.Private())
	if err != nil {
		return fmt.Errorf("failed to create swarmion private key: %w", err)
	}

	logger := log.New(io.Discard, "", log.LstdFlags|log.Lmicroseconds|log.LUTC)
	listenPort := swarmionListenPort()
	network, err := libp2ptransport.New(ctx, libp2ptransport.Config{
		ListenAddrs: []string{
			fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
			fmt.Sprintf("/ip6/::/tcp/%d", listenPort),
		},
		BootstrapPeers:       append([]string(nil), bootstrapPeers...),
		PrivateKey:           privateKey,
		TargetConnectedPeers: 32,
		Logger:               logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create swarmion transport: %w", err)
	}
	swarmionSigner, err := newSwarmionSigner(db.signer, privateKey.GetPublic())
	if err != nil {
		_ = network.Close()
		return fmt.Errorf("failed to create swarmion signer: %w", err)
	}

	swarmManifest, hasSwarmManifest, err := db.loadSwarmManifest()
	if err != nil {
		_ = network.Close()
		return err
	}
	appConfig := swarmionapp.Config{
		Repository: swarmiondoltrepo.Config{
			Dir:         db.workingDir,
			Name:        db.name,
			CommitName:  db.signer.GetID(),
			CommitEmail: db.signer.GetID() + "@protos.local",
			Signer:      swarmionSigner,
		},
		BootstrapPeers:                  append([]string(nil), bootstrapPeers...),
		Namespace:                       fmt.Sprintf(swarmionNamespaceTemplate, db.name),
		AdminNamespace:                  fmt.Sprintf(swarmionAdminNamespaceTemplate, db.name),
		HeartbeatInterval:               5 * time.Second,
		CheckpointMaterializationPolicy: swarmionapp.CheckpointMaterializationManualLazy,
		SchemaEngine:                    cueschema.New(protoscontracts.Catalog, declarativeschema.New(protoscontracts.Catalog)),
		OnWriteNotification:             db.handleWriteNotification,
		Logger:                          logger,
	}
	if hasSwarmManifest {
		appConfig.SwarmManifest = swarmManifest
	}
	app, err := swarmionapp.Open(ctx, appConfig, network)
	if err != nil {
		_ = network.Close()
		return fmt.Errorf("failed to open swarmion runtime: %w", err)
	}
	if err := db.persistSwarmManifest(ctx, app); err != nil {
		_ = app.Close()
		return err
	}

	db.app = app
	db.network = network
	db.sqldb = app.SQLDB()
	configureEmbeddedSQLDB(db.sqldb)
	db.initialized = true
	watchCtx, watchCancel := context.WithCancel(context.Background())
	db.watchCancel = watchCancel
	db.startSwarmionWatchers(watchCtx, app)
	return nil
}

func configureEmbeddedSQLDB(sqldb *sql.DB) {
	if sqldb == nil {
		return
	}
	sqldb.SetMaxOpenConns(1)
	sqldb.SetMaxIdleConns(1)
	sqldb.SetConnMaxLifetime(0)
	sqldb.SetConnMaxIdleTime(0)
}

func (db *DB) swarmionManifestPath() string {
	if db == nil {
		return ""
	}
	return filepath.Join(db.workingDir, swarmionStateDirName, db.name+".swarm-manifest.json")
}

func (db *DB) loadSwarmManifest() (swarmionprotocol.SwarmManifest, bool, error) {
	path := db.swarmionManifestPath()
	if path == "" {
		return swarmionprotocol.SwarmManifest{}, false, fmt.Errorf("db is nil")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return swarmionprotocol.SwarmManifest{}, false, nil
		}
		return swarmionprotocol.SwarmManifest{}, false, fmt.Errorf("read swarmion manifest %q: %w", path, err)
	}
	var manifest swarmionprotocol.SwarmManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return swarmionprotocol.SwarmManifest{}, false, fmt.Errorf("parse swarmion manifest %q: %w", path, err)
	}
	return manifest, true, nil
}

func (db *DB) persistSwarmManifest(ctx context.Context, app *swarmionapp.App) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if app == nil {
		return fmt.Errorf("swarmion app is not initialized")
	}
	status, err := app.SwarmManifestStatus(ctx)
	if err != nil {
		return fmt.Errorf("read swarmion manifest status: %w", err)
	}
	if !status.Complete {
		notifyLog.Warnf("swarmion manifest for %s is not complete; not persisting an initial boundary yet", db.name)
		return nil
	}
	path := db.swarmionManifestPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create swarmion manifest directory: %w", err)
	}
	data, err := json.MarshalIndent(status.Manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode swarmion manifest: %w", err)
	}
	data = append(data, '\n')
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write swarmion manifest %q: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace swarmion manifest %q: %w", path, err)
	}
	return nil
}

func (db *DB) startSwarmionWatchers(ctx context.Context, app *swarmionapp.App) {
	if db == nil || app == nil {
		return
	}
	if events, err := app.WatchStatus(ctx); err == nil {
		go db.forwardSwarmionStatusEvents(events)
	} else {
		notifyLog.Warnf("failed to watch swarmion status: %s", err.Error())
	}
	if events, err := app.WatchCheckpointRoots(ctx); err == nil {
		go db.forwardSwarmionCheckpointRootEvents(events)
	} else {
		notifyLog.Warnf("failed to watch swarmion checkpoint roots: %s", err.Error())
	}
}

func (db *DB) forwardSwarmionStatusEvents(events <-chan swarmionapp.StatusEvent) {
	for event := range events {
		switch event.Kind {
		case swarmionapp.StatusEventCheckpointRootChanged,
			swarmionapp.StatusEventTentativeRootChanged,
			swarmionapp.StatusEventFatalChanged,
			swarmionapp.StatusEventStateProvidersChanged:
			db.triggerRuntimeChangeCallbacks()
		}
	}
}

func (db *DB) forwardSwarmionCheckpointRootEvents(events <-chan swarmionapp.CheckpointRootEvent) {
	for event := range events {
		db.triggerPublishedTableChangeCallbacks(event.ChangedTables...)
	}
}

func (db *DB) runMigrations(ctx context.Context) error {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return fmt.Errorf("db is not initialized")
	}
	if _, err := sqldb.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sqddl_history (
filename VARCHAR(255) NOT NULL,
checksum VARCHAR(255) NOT NULL,
started_at TIMESTAMP NULL,
time_taken_ns BIGINT NOT NULL,
success BOOLEAN NOT NULL,
PRIMARY KEY (filename)
)`); err != nil {
		return fmt.Errorf("ensure migration history: %w", err)
	}

	migrationsDir, err := fs.Sub(rootDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations dir: %w", err)
	}
	entries, err := fs.ReadDir(migrationsDir, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	filenames := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, ".undo.sql") || !strings.HasSuffix(name, ".sql") {
			continue
		}
		filenames = append(filenames, name)
	}
	sort.Strings(filenames)

	appliedAny := false
	for _, filename := range filenames {
		applied, err := migrationApplied(ctx, sqldb, filename)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := db.applyMigration(ctx, migrationsDir, filename); err != nil {
			return err
		}
		appliedAny = true
	}
	if appliedAny {
		commit, err := db.commitStaged(ctx, "run migrations", true)
		if err != nil {
			return fmt.Errorf("commit migrations: %w", err)
		}
		if err := db.waitForCommittedRoot(ctx, commit, "run migrations"); err != nil {
			return fmt.Errorf("checkpoint migrations: %w", err)
		}
	}

	return nil
}

func (db *DB) applyMigration(ctx context.Context, migrationsDir fs.FS, filename string) error {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return fmt.Errorf("db is not initialized")
	}
	start := time.Now()
	contents, err := fs.ReadFile(migrationsDir, filename)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}
	statements, err := sqlparser.SplitStatementToPieces(string(contents))
	if err != nil {
		return fmt.Errorf("split migration %s: %w", filename, err)
	}
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := sqldb.ExecContext(ctx, statement); err != nil {
			if ignorableMigrationError(statement, err) {
				continue
			}
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	checksum := sha256.Sum256(contents)
	if _, err := sqldb.ExecContext(
		ctx,
		`INSERT INTO sqddl_history (filename, checksum, started_at, time_taken_ns, success)
VALUES (?, ?, NOW(), ?, true)
ON DUPLICATE KEY UPDATE checksum = VALUES(checksum), started_at = VALUES(started_at), time_taken_ns = VALUES(time_taken_ns), success = VALUES(success)`,
		filename,
		hex.EncodeToString(checksum[:]),
		time.Since(start).Nanoseconds(),
	); err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}
	return nil
}

func ignorableMigrationError(statement string, err error) bool {
	if err == nil {
		return false
	}
	statement = strings.ToUpper(statement)
	message := strings.ToLower(err.Error())
	if strings.Contains(statement, "ALTER TABLE") &&
		strings.Contains(statement, "ADD COLUMN") &&
		strings.Contains(message, "already exists") {
		return true
	}
	if strings.Contains(statement, "CREATE TABLE") &&
		strings.Contains(message, "already exists") {
		return true
	}
	return strings.Contains(statement, "CREATE INDEX") &&
		(strings.Contains(message, "already exists") || strings.Contains(message, "duplicate key name"))
}

func migrationApplied(ctx context.Context, sqldb *sql.DB, filename string) (bool, error) {
	var success bool
	err := sqldb.QueryRowContext(ctx, "SELECT success FROM sqddl_history WHERE filename = ?", filename).Scan(&success)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read migration history for %s: %w", filename, err)
	}
	return success, nil
}

func (db *DB) Close() error {
	if db == nil {
		return nil
	}
	db.opMu.Lock()
	defer db.opMu.Unlock()

	db.mu.Lock()
	app := db.app
	watchCancel := db.watchCancel
	db.app = nil
	db.network = nil
	db.sqldb = nil
	db.initialized = false
	db.watchCancel = nil
	db.mu.Unlock()
	if watchCancel != nil {
		watchCancel()
	}
	if app == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := db.evictSwarmionPeer(ctx, app, app.PeerID()); err != nil {
		notifyLog.Debugf("failed to evict local swarmion peer before close: %s", err.Error())
	}
	cancel()
	return app.Close()
}

func (db *DB) Initialized() bool {
	if db == nil {
		return false
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.initialized
}

func (db *DB) InitFromPeer(peerID string, bootstrapPeers []string) error {
	return db.InitFromPeerContext(context.Background(), peerID, bootstrapPeers)
}

func (db *DB) InitFromPeerContext(ctx context.Context, peerID string, bootstrapPeers []string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(bootstrapPeers) == 0 {
		return fmt.Errorf("cannot initialize swarmion db from peer %s without bootstrap addresses", peerID)
	}
	attemptCtx, cancel := context.WithTimeout(ctx, initFromPeerRetryBudget)
	defer cancel()

	var lastErr error
	backoff := initFromPeerRetryInitialBackoff
	for attempt := 1; ; attempt++ {
		if err := quarantineIncompleteDatabase(db.workingDir, db.name); err != nil {
			return err
		}
		err := db.openSwarmion(attemptCtx, bootstrapPeers)
		if err == nil {
			db.triggerAllTableChangeCallbacks()
			if attempt > 1 {
				notifyLog.Infof("initialized swarmion db from peer %s after %d attempts", peerID, attempt)
			}
			return nil
		}
		lastErr = err
		if !retryableSwarmionBootstrapError(err) {
			return fmt.Errorf("failed to initialize swarmion db from peer %s: %w", peerID, err)
		}
		if attemptCtx.Err() != nil {
			return fmt.Errorf("failed to initialize swarmion db from peer %s after %d attempts: %w", peerID, attempt, lastErr)
		}
		notifyLog.Warnf("retryable swarmion bootstrap failure from peer %s on attempt %d: %s", peerID, attempt, err.Error())
		if err := db.quarantineBootstrapRetryDatabaseDir(); err != nil {
			return fmt.Errorf("reset retryable swarmion bootstrap attempt from peer %s: %w", peerID, err)
		}
		select {
		case <-attemptCtx.Done():
			return fmt.Errorf("failed to initialize swarmion db from peer %s after %d attempts: %w", peerID, attempt, lastErr)
		case <-time.After(backoff):
		}
		if backoff < initFromPeerRetryMaxBackoff {
			backoff *= 2
			if backoff > initFromPeerRetryMaxBackoff {
				backoff = initFromPeerRetryMaxBackoff
			}
		}
	}
}

func retryableSwarmionBootstrapError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "fetch checkpoint history from bootstrap state") &&
		strings.Contains(message, "sync_checkpoint_history") &&
		strings.Contains(message, "no connected providers")
}

func (db *DB) quarantineBootstrapRetryDatabaseDir() error {
	if db == nil || db.Initialized() {
		return nil
	}
	dbDir := filepath.Join(db.workingDir, db.name)
	if _, err := os.Stat(dbDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect bootstrap database directory %q: %w", dbDir, err)
	}
	quarantineDir := filepath.Join(
		db.workingDir,
		fmt.Sprintf("%s.bootstrap-retry.%d", db.name, time.Now().UnixNano()),
	)
	if err := os.Rename(dbDir, quarantineDir); err != nil {
		return fmt.Errorf("quarantine retryable bootstrap database %q: %w", dbDir, err)
	}
	notifyLog.Warnf("quarantined retryable bootstrap database %q to %q", dbDir, quarantineDir)
	return nil
}

func (db *DB) EnableGRPCServers(*grpc.Server) error {
	return nil
}

func (db *DB) AddPeer(string, *grpc.ClientConn) error {
	return nil
}

func (db *DB) RemovePeer(peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if db == nil || peerID == "" {
		return nil
	}
	if !db.Initialized() {
		return nil
	}

	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return db.evictSwarmionPeer(ctx, app, peerID)
}

func (db *DB) PrepareSwarmionShutdown(ctx context.Context) error {
	if db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}
	return db.evictSwarmionPeer(ctx, app, app.PeerID())
}

func (db *DB) ReconcileRemovedSwarmionPeers(ctx context.Context, activePeerIDs map[string]struct{}) error {
	if db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}

	active := make(map[string]struct{}, len(activePeerIDs))
	for peerID := range activePeerIDs {
		peerID = strings.TrimSpace(peerID)
		if peerID != "" {
			active[peerID] = struct{}{}
		}
	}

	peerStatuses, err := app.PeerStatus(ctx)
	if err != nil {
		return fmt.Errorf("read swarmion peer status for removed-peer reconciliation: %w", err)
	}

	var failures []string
	for _, peerStatus := range peerStatuses {
		peerID := strings.TrimSpace(peerStatus.PeerID)
		if peerID == "" {
			continue
		}
		if _, found := active[peerID]; found {
			continue
		}
		if err := db.evictSwarmionPeer(ctx, app, peerID); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("reconcile removed swarmion peers: %s", strings.Join(failures, "; "))
	}
	return nil
}

func (db *DB) evictSwarmionPeer(ctx context.Context, app swarmionPeerEvictor, peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if app == nil || peerID == "" {
		return nil
	}
	if db != nil {
		if err := db.closeSwarmionTransportPeer(peerID); err != nil {
			notifyLog.Debugf("failed to close swarmion transport peer %s before eviction: %s", peerID, err.Error())
		}
	}
	if _, err := app.EvictPeer(ctx, swarmionapp.PeerEvictionRequest{PeerID: peerID}); err != nil {
		return fmt.Errorf("evict swarmion peer %s: %w", peerID, err)
	}
	return nil
}

func (db *DB) closeSwarmionTransportPeer(peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if db == nil || peerID == "" {
		return nil
	}
	db.mu.Lock()
	network := db.network
	db.mu.Unlock()
	if network == nil || network.ID() == peerID {
		return nil
	}
	host := network.Host()
	if host == nil || host.Network() == nil {
		return nil
	}
	pid, err := libp2ppeer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("decode swarmion peer id %s: %w", peerID, err)
	}
	return host.Network().ClosePeer(pid)
}

func (db *DB) ConnectPeer(peerID string, publicIP string) error {
	return db.ConnectPeerIPs(peerID, []string{publicIP})
}

func (db *DB) ConnectPeerIPs(peerID string, ips []string) error {
	if strings.TrimSpace(peerID) == "" {
		return nil
	}
	if !db.Initialized() {
		return nil
	}

	db.mu.Lock()
	network := db.network
	db.mu.Unlock()
	if network == nil {
		return nil
	}

	listenPort := swarmionListenPort()
	if listenPort == 0 {
		return nil
	}

	addrs := swarmionPeerAddrs(peerID, ips, listenPort)
	if len(addrs) == 0 {
		return nil
	}

	var errs []error
	for _, addr := range addrs {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := network.Connect(ctx, addr)
		cancel()
		if err == nil {
			return nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", addr, err))
	}
	return errors.Join(errs...)
}

func swarmionListenPort() int {
	if config.Get().P2PPort <= 0 {
		return 0
	}
	return config.Get().P2PPort + swarmionPortOffset
}

func swarmionPeerAddrs(peerID string, ips []string, port int) []string {
	seen := map[string]struct{}{}
	var addrs []string
	for _, rawIP := range ips {
		rawIP = strings.TrimSpace(rawIP)
		if rawIP == "" {
			continue
		}
		ip := net.ParseIP(rawIP)
		if ip == nil {
			continue
		}
		var addr string
		if ip.To4() == nil {
			addr = fmt.Sprintf("/ip6/%s/tcp/%d/p2p/%s", ip.String(), port, peerID)
		} else {
			addr = fmt.Sprintf("/ip4/%s/tcp/%d/p2p/%s", ip.String(), port, peerID)
		}
		if _, found := seen[addr]; found {
			continue
		}
		seen[addr] = struct{}{}
		addrs = append(addrs, addr)
	}
	return addrs
}

func (db *DB) ListenMultiaddrs() []string {
	if db == nil {
		return nil
	}
	db.mu.Lock()
	network := db.network
	db.mu.Unlock()
	if network == nil {
		return nil
	}
	return network.ListenMultiaddrs()
}

func (db *DB) DialableListenMultiaddrs(ips []string) []string {
	if db == nil {
		return nil
	}
	var addrs []string
	if db.signer != nil {
		addrs = append(addrs, swarmionPeerAddrs(db.signer.GetID(), ips, swarmionListenPort())...)
	}
	addrs = append(addrs, db.ListenMultiaddrs()...)
	return dedupeMultiaddrs(addrs)
}

func dedupeMultiaddrs(addrs []string) []string {
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}
		if _, found := seen[addr]; found {
			continue
		}
		seen[addr] = struct{}{}
		deduped = append(deduped, addr)
	}
	return deduped
}

func (db *DB) SwarmionStatus() (swarmionapp.Status, bool) {
	if db == nil {
		return swarmionapp.Status{}, false
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return swarmionapp.Status{}, false
	}
	return app.Status(), true
}

// SwarmionRootStatus returns the current, revisitable root lifecycle for a
// published write. Callers must not treat a parked status as a durable
// rejection; a later canonical head may classify the same root differently.
func (db *DB) SwarmionRootStatus(ctx context.Context, rootHash string) (swarmionapp.BranchRootStatus, error) {
	if db == nil {
		return swarmionapp.BranchRootStatus{}, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return swarmionapp.BranchRootStatus{}, fmt.Errorf("db is not initialized")
	}
	return app.RootStatus(ctx, swarmionapp.BranchRootStatusRequest{RootHash: rootHash})
}

func (db *DB) CatchUpCheckpoint(ctx context.Context, reason string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := db.catchUpCheckpointStrict(ctx, reason); err != nil {
		if errors.Is(err, errSwarmionCheckpointCatchUpRetryable) {
			notifyLog.Debugf("deferred swarmion checkpoint catch-up for %q after retryable response: %s", reason, err.Error())
			return nil
		}
		return err
	}
	return nil
}

func (db *DB) CatchUpCheckpointStrict(ctx context.Context, reason string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.catchUpCheckpointStrict(ctx, reason)
}

func (db *DB) catchUpCheckpointStrict(ctx context.Context, reason string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "protos checkpoint read"
	}
	if err := catchUpSwarmionCheckpoint(ctx, app, reason); err != nil {
		return fmt.Errorf("catch up swarmion checkpoint view: %w", err)
	}
	return nil
}

func IsRetryableCheckpointCatchUp(err error) bool {
	return errors.Is(err, errSwarmionCheckpointCatchUpRetryable)
}

func catchUpSwarmionCheckpoint(ctx context.Context, app interface {
	CatchUpCheckpoint(context.Context, string) (swarmionadmin.CheckpointCatchUpResponse, error)
}, reason string) error {
	if app == nil {
		return nil
	}
	var lastErr error
	for attempt := 0; attempt < checkpointCatchUpMaxAttempts; attempt++ {
		err := catchUpSwarmionCheckpointOnce(ctx, app, reason)
		if err == nil || !errors.Is(err, errSwarmionCheckpointCatchUpRetryable) {
			return err
		}
		lastErr = err
		if attempt == checkpointCatchUpMaxAttempts-1 {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 250 * time.Millisecond):
		}
	}
	return lastErr
}

func catchUpSwarmionCheckpointOnce(ctx context.Context, app interface {
	CatchUpCheckpoint(context.Context, string) (swarmionadmin.CheckpointCatchUpResponse, error)
}, reason string) error {
	resp, err := app.CatchUpCheckpoint(ctx, reason)
	if opErr := checkpointCatchUpOperationalError(resp); opErr != nil {
		if err != nil {
			return fmt.Errorf("%w: %w", opErr, err)
		}
		return opErr
	}
	if err != nil {
		return err
	}
	return nil
}

func checkpointCatchUpOperationalError(resp swarmionadmin.CheckpointCatchUpResponse) error {
	status := strings.TrimSpace(resp.Status)
	switch swarmionadmin.CheckpointCatchUpStatus(status) {
	case swarmionadmin.CheckpointCatchUpStatusBlockedFatal:
		return fmt.Errorf("checkpoint catch-up blocked by fatal protocol state: %s", checkpointCatchUpReason(resp))
	case swarmionadmin.CheckpointCatchUpStatusFailed:
		return fmt.Errorf("checkpoint catch-up failed: %s", checkpointCatchUpReason(resp))
	case swarmionadmin.CheckpointCatchUpStatusTargetChanged, swarmionadmin.CheckpointCatchUpStatusRetryable:
		return fmt.Errorf("%w: status=%s reason=%s", errSwarmionCheckpointCatchUpRetryable, status, checkpointCatchUpReason(resp))
	case swarmionadmin.CheckpointCatchUpStatusNoTarget,
		swarmionadmin.CheckpointCatchUpStatusNoSnapshot,
		swarmionadmin.CheckpointCatchUpStatusAlreadyCurrent,
		swarmionadmin.CheckpointCatchUpStatusComplete:
		return nil
	}
	if resp.BlockedByFatal {
		return fmt.Errorf("checkpoint catch-up blocked by fatal protocol state: %s", checkpointCatchUpReason(resp))
	}
	if resp.TargetChanged || resp.Retryable {
		return fmt.Errorf("%w: status=%s reason=%s", errSwarmionCheckpointCatchUpRetryable, status, checkpointCatchUpReason(resp))
	}
	return nil
}

func checkpointCatchUpReason(resp swarmionadmin.CheckpointCatchUpResponse) string {
	for _, value := range []string{resp.BlockingReason, resp.Detail, resp.Message} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	if strings.TrimSpace(resp.Status) != "" {
		return resp.Status
	}
	return "no details"
}

func (db *DB) SwarmionCompatibility(ctx context.Context) ([]swarmionapp.ManifestCompatibility, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil, fmt.Errorf("swarmion app is not initialized")
	}
	return app.Compatibility(ctx)
}

func (db *DB) SwarmionPeerStatus(ctx context.Context) ([]swarmionapp.PeerStatus, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil, fmt.Errorf("swarmion app is not initialized")
	}
	return app.PeerStatus(ctx)
}

func (db *DB) SwarmionPeerRemovalReadiness(ctx context.Context, peerID string) (swarmionapp.PeerRemovalReadinessResponse, error) {
	if db == nil {
		return swarmionapp.PeerRemovalReadinessResponse{}, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return swarmionapp.PeerRemovalReadinessResponse{}, fmt.Errorf("swarmion app is not initialized")
	}
	return app.PeerRemovalReadiness(ctx, swarmionapp.PeerRemovalReadinessRequest{PeerID: peerID})
}

func (db *DB) SwarmionContentSyncTrace() ([]string, bool) {
	if db == nil {
		return nil, false
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil, false
	}
	return app.ContentSyncTrace(), true
}

func (db *DB) GetSqlDB() *sql.DB {
	if db == nil {
		return nil
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.sqldb
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := db.opMu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer db.opMu.Unlock()

	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return sqldb.ExecContext(ctx, query, args...)
}

// ReadRows runs a query and keeps the database operation lock until consume
// returns and the result rows have been closed.
func (db *DB) ReadRows(ctx context.Context, query string, args []any, consume func(*sql.Rows) error) error {
	if db == nil {
		return fmt.Errorf("db is not initialized")
	}
	if consume == nil {
		return fmt.Errorf("row consumer is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := db.opMu.LockContext(ctx); err != nil {
		return err
	}
	defer db.opMu.Unlock()

	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return fmt.Errorf("db is not initialized")
	}
	rows, err := sqldb.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if err := consume(rows); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

type lockedSQLDB struct {
	db *DB
}

func (q lockedSQLDB) sqlDB() (*sql.DB, error) {
	if q.db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	sqldb := q.db.GetSqlDB()
	if sqldb == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return sqldb, nil
}

func (q lockedSQLDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	sqldb, err := q.sqlDB()
	if err != nil {
		return nil, err
	}
	return sqldb.QueryContext(ctx, query, args...)
}

func (q lockedSQLDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	sqldb, err := q.sqlDB()
	if err != nil {
		return nil, err
	}
	return sqldb.ExecContext(ctx, query, args...)
}

func (q lockedSQLDB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	sqldb, err := q.sqlDB()
	if err != nil {
		return nil, err
	}
	return sqldb.PrepareContext(ctx, query)
}

func (db *DB) GetLastCommit(branch string) (Commit, error) {
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}
	query := fmt.Sprintf("SELECT commit_hash, committer, email, date, message, parents, refs FROM dolt_log('%s', '--parents', '--decorate=short') LIMIT 1;", escapeSQL(branch))
	commits, err := db.getCommits(query)
	if err != nil {
		return Commit{}, err
	}
	if len(commits) == 0 {
		return Commit{}, fmt.Errorf("no commits found")
	}
	return commits[0], nil
}

func (db *DB) GetAllCommits() ([]Commit, error) {
	return db.GetCommits("main")
}

func (db *DB) GetCommits(branch string) ([]Commit, error) {
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}
	query := fmt.Sprintf("SELECT commit_hash, committer, email, date, message, parents, refs FROM dolt_log('%s', '--parents', '--decorate=short');", escapeSQL(branch))
	return db.getCommits(query)
}

func (db *DB) getCommits(query string) ([]Commit, error) {
	var commits []Commit
	if err := db.ReadRows(context.Background(), query, nil, func(rows *sql.Rows) error {
		for rows.Next() {
			var commit Commit
			var parents sql.NullString
			var refs sql.NullString
			if err := rows.Scan(&commit.Hash, &commit.Committer, &commit.Email, &commit.Date, &commit.Message, &parents, &refs); err != nil {
				return fmt.Errorf("failed to scan commit: %w", err)
			}
			commit.SignerPublicKey = ExtractCommitSignerPublicKey(commit.Message)
			commit.ParentHashes = splitCommitList(parents.String)
			commit.Refs = splitCommitList(refs.String)
			commits = append(commits, commit)
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to read commits: %w", err)
	}
	return commits, nil
}

func splitCommitList(value string) []string {
	var items []string
	for _, raw := range strings.Split(value, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

type stagedCommitResult struct {
	Committed                bool
	Hash                     string
	EventID                  string
	PublishedRootHash        string
	WriteBaseRootHash        string
	WorkspaceHeadRootHash    string
	WorkspaceStagedRootHash  string
	WorkspaceWorkingRootHash string
}

func (r stagedCommitResult) hasPublishedContent() bool {
	return r.Committed && (strings.TrimSpace(r.Hash) != "" || strings.TrimSpace(r.PublishedRootHash) != "")
}

func (db *DB) commitStaged(ctx context.Context, message string, allowNoop bool) (stagedCommitResult, error) {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return stagedCommitResult{}, fmt.Errorf("db is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(message) == "" {
		message = "swarmion commit"
	}

	rows, err := sqldb.QueryContext(ctx, "CALL swarmion_commit_info('-Am', ?)", message)
	if err != nil {
		return stagedCommitResult{}, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return stagedCommitResult{}, err
	}
	values := make([]any, len(cols))
	scan := make([]any, len(cols))
	for i := range values {
		scan[i] = &values[i]
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return stagedCommitResult{}, err
		}
		if allowNoop {
			return stagedCommitResult{}, nil
		}
		return stagedCommitResult{}, fmt.Errorf("nothing to commit")
	}
	if err := rows.Scan(scan...); err != nil {
		return stagedCommitResult{}, err
	}
	result := parseStagedCommitResult(cols, values)
	if !result.Committed && !allowNoop {
		return result, fmt.Errorf("nothing to commit")
	}
	return result, nil
}

func parseStagedCommitResult(cols []string, values []any) stagedCommitResult {
	var result stagedCommitResult
	for i, col := range cols {
		if i >= len(values) {
			break
		}
		switch strings.ToLower(strings.TrimSpace(col)) {
		case "committed":
			result.Committed = commitInfoBool(values[i])
		case "hash":
			result.Hash = commitInfoString(values[i])
		case "event_id":
			result.EventID = commitInfoString(values[i])
		case "published_root_hash":
			result.PublishedRootHash = commitInfoString(values[i])
		case "write_base_root_hash":
			result.WriteBaseRootHash = commitInfoString(values[i])
		case "workspace_head_root_hash":
			result.WorkspaceHeadRootHash = commitInfoString(values[i])
		case "workspace_staged_root_hash":
			result.WorkspaceStagedRootHash = commitInfoString(values[i])
		case "workspace_working_root_hash":
			result.WorkspaceWorkingRootHash = commitInfoString(values[i])
		}
	}
	return result
}

func commitInfoString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func commitInfoBool(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int8:
		return v != 0
	case int16:
		return v != 0
	case int32:
		return v != 0
	case int64:
		return v != 0
	case uint:
		return v != 0
	case uint8:
		return v != 0
	case uint16:
		return v != 0
	case uint32:
		return v != 0
	case uint64:
		return v != 0
	case []byte:
		return commitInfoBool(string(v))
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "t", "true", "y", "yes":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

func (db *DB) waitForCommittedRoot(ctx context.Context, commit stagedCommitResult, reason string) error {
	if !commit.hasPublishedContent() {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(ctx, committedWriteCheckpointTimeout)
	defer cancel()

	return db.waitForCommittedRootObserved(waitCtx, commit, reason)
}

func (db *DB) waitForCommittedRootObserved(ctx context.Context, commit stagedCommitResult, reason string) error {
	if db == nil || !commit.hasPublishedContent() {
		return nil
	}
	db.mu.Lock()
	app := db.app
	db.mu.Unlock()
	if app == nil {
		return nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "protos committed write"
	}

	var lastErr error
	for {
		catchUpErr := catchUpSwarmionCheckpoint(ctx, app, reason)
		if catchUpErr != nil && !errors.Is(catchUpErr, errSwarmionCheckpointCatchUpRetryable) {
			return fmt.Errorf("catch up swarmion checkpoint view for published write: %w", catchUpErr)
		}

		status, statusErr := app.RootStatus(ctx, swarmionapp.BranchRootStatusRequest{RootHash: commit.PublishedRootHash})
		if statusErr != nil {
			return fmt.Errorf("read published root status: %w", statusErr)
		}
		reached, checkpointErr := stagedCommitCheckpointReached(status, commit)
		if checkpointErr != nil {
			return checkpointErr
		}
		if reached {
			return nil
		}
		if catchUpErr != nil {
			lastErr = catchUpErr
		} else if lastErr == nil || !isRetryableCommittedWriteError(lastErr) {
			lastErr = stagedCommitCheckpointWaitError(status, commit)
		}

		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("published write did not reach local checkpoint for %q: %w", reason, lastErr)
			}
			return fmt.Errorf("published write did not reach local checkpoint for %q: %w", reason, ctx.Err())
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func stagedCommitCheckpointReached(status swarmionapp.BranchRootStatus, commit stagedCommitResult) (bool, error) {
	if !commit.hasPublishedContent() {
		return false, nil
	}
	expectedRoot := swarmionprotocol.ParseRootHash(commit.PublishedRootHash)
	if expectedRoot.IsZero() {
		return false, fmt.Errorf(
			"%w: published_root=%q",
			errSwarmionCheckpointedWriteIncomplete,
			commit.PublishedRootHash,
		)
	}
	if status.RootHash != expectedRoot.String() {
		return false, nil
	}
	return status.Durable, nil
}

func stagedCommitCheckpointWaitError(status swarmionapp.BranchRootStatus, commit stagedCommitResult) error {
	lifecycle := swarmionapp.RootLifecycleFromStatus(status)
	return fmt.Errorf(
		"expected event_id=%s published_root=%s commit=%s root_status=%s revisitable=%t checkpointed=%t durable=%t parked_reason=%s pending_reason=%s",
		commit.EventID,
		commit.PublishedRootHash,
		commit.Hash,
		lifecycle.State,
		lifecycle.Revisitable,
		status.Checkpointed,
		status.Durable,
		status.ParkedReason,
		status.PendingReason,
	)
}

func (db *DB) RegisterTableChangeCallback(tableName string, notifier Notifier) {
	if db == nil || notifier == nil {
		return
	}
	if db.tableChangeCallbacks == nil {
		db.tableChangeCallbacks = util.NewMap[string, tableChangeCallback]()
	}
	guid := xid.New()
	db.tableChangeCallbacks.Set(guid.String(), tableChangeCallback{
		tableName: tableName,
		notifier:  notifier,
	})
}

func (db *DB) RegisterRuntimeChangeCallback(notifier Notifier) {
	if db == nil || notifier == nil {
		return
	}
	if db.runtimeCallbacks == nil {
		db.runtimeCallbacks = util.NewMap[string, Notifier]()
	}
	guid := xid.New()
	db.runtimeCallbacks.Set(guid.String(), notifier)
}

func (db *DB) WatchChanges(ctx context.Context) (<-chan ChangeEvent, func()) {
	ch := make(chan ChangeEvent, 1)
	if db == nil {
		close(ch)
		return ch, func() {}
	}
	notifier := &changeWatchNotifier{ch: ch}
	guid := xid.New().String()
	if db.tableChangeCallbacks == nil {
		db.tableChangeCallbacks = util.NewMap[string, tableChangeCallback]()
	}
	db.tableChangeCallbacks.Set(guid, tableChangeCallback{notifier: notifier})

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			db.tableChangeCallbacks.Delete(guid)
		})
	}
	if ctx != nil {
		go func() {
			<-ctx.Done()
			cancel()
		}()
	}
	return ch, cancel
}

type changeWatchNotifier struct {
	ch chan ChangeEvent
}

func (n *changeWatchNotifier) Notify() {
	n.NotifyChange(nil)
}

func (n *changeWatchNotifier) NotifyChange(tableNames []string) {
	event := ChangeEvent{TableNames: append([]string(nil), tableNames...)}
	select {
	case n.ch <- event:
	default:
	}
}

func (db *DB) handleWriteNotification(_ context.Context, notification swarmionapp.WriteNotification) error {
	if !notification.Accepted {
		return nil
	}
	return nil
}

func (db *DB) triggerPublishedTableChangeCallbacks(tableNames ...string) {
	if len(tableNames) == 0 {
		db.triggerAllTableChangeCallbacks()
		return
	}
	db.triggerTableChangeCallbacks(tableNames...)
}

func (db *DB) triggerTableChangeCallbacks(tableNames ...string) {
	if db == nil {
		return
	}
	if len(tableNames) == 0 {
		db.triggerRuntimeChangeCallbacks()
		return
	}
	if db.deferTableChangeCallbacks(tableNames...) {
		return
	}
	db.dispatchTableChangeCallbacks(true, tableNames...)
}

func (db *DB) triggerAllTableChangeCallbacks() {
	if db == nil {
		return
	}
	if db.deferTableChangeCallbacks() {
		return
	}
	db.dispatchTableChangeCallbacks(true)
}

func (db *DB) triggerRuntimeChangeCallbacks() {
	if db == nil {
		return
	}
	db.dispatchRuntimeChangeCallbacks(true)
}

func (db *DB) dispatchRuntimeChangeCallbacks(async bool) {
	seen := map[uintptr]struct{}{}
	notify := func(notifier Notifier) {
		if id, ok := notifierIdentity(notifier); ok {
			if _, found := seen[id]; found {
				return
			}
			seen[id] = struct{}{}
		}
		if async {
			notifyChangeAsync(notifier, nil)
		} else {
			notifySafely(notifier, nil)
		}
	}

	if db.tableChangeCallbacks != nil {
		for _, callback := range db.tableChangeCallbacks.Snapshot() {
			if callback.tableName != "" {
				continue
			}
			notify(callback.notifier)
		}
	}
	if db.runtimeCallbacks != nil {
		for _, notifier := range db.runtimeCallbacks.Snapshot() {
			notify(notifier)
		}
	}
}

func (db *DB) dispatchTableChangeCallbacks(async bool, tableNames ...string) {
	if db.tableChangeCallbacks == nil {
		return
	}
	tableSet := make(map[string]struct{}, len(tableNames))
	for _, tableName := range tableNames {
		tableSet[tableName] = struct{}{}
	}

	seen := map[uintptr]struct{}{}
	for _, callback := range db.tableChangeCallbacks.Snapshot() {
		if len(tableSet) > 0 && callback.tableName != "" {
			if _, found := tableSet[callback.tableName]; !found {
				continue
			}
		}
		if id, ok := notifierIdentity(callback.notifier); ok {
			if _, found := seen[id]; found {
				continue
			}
			seen[id] = struct{}{}
		}
		if async {
			notifyChangeAsync(callback.notifier, tableNames)
		} else {
			notifySafely(callback.notifier, tableNames)
		}
	}
}

func (db *DB) DeferNotifications() func(async bool) {
	if db == nil {
		return func(bool) {}
	}
	db.notificationMu.Lock()
	db.notificationDepth++
	db.notificationMu.Unlock()

	var once sync.Once
	return func(async bool) {
		once.Do(func() {
			tableNames, flush := db.releaseDeferredNotifications()
			if flush {
				db.dispatchTableChangeCallbacks(async, tableNames...)
			}
		})
	}
}

func (db *DB) deferTableChangeCallbacks(tableNames ...string) bool {
	db.notificationMu.Lock()
	defer db.notificationMu.Unlock()
	if db.notificationDepth == 0 {
		return false
	}
	db.addPendingNotificationLocked(tableNames...)
	return true
}

func (db *DB) releaseDeferredNotifications() ([]string, bool) {
	db.notificationMu.Lock()
	defer db.notificationMu.Unlock()
	if db.notificationDepth > 0 {
		db.notificationDepth--
	}
	if db.notificationDepth > 0 {
		return nil, false
	}
	if db.pendingNotifyAll {
		db.pendingNotifyAll = false
		db.pendingNotifyTables = nil
		return nil, true
	}
	if len(db.pendingNotifyTables) == 0 {
		return nil, false
	}
	tableNames := make([]string, 0, len(db.pendingNotifyTables))
	for tableName := range db.pendingNotifyTables {
		tableNames = append(tableNames, tableName)
	}
	sort.Strings(tableNames)
	db.pendingNotifyTables = nil
	return tableNames, true
}

func (db *DB) addPendingNotificationLocked(tableNames ...string) {
	if len(tableNames) == 0 {
		db.pendingNotifyAll = true
		db.pendingNotifyTables = nil
		return
	}
	if db.pendingNotifyAll {
		return
	}
	if db.pendingNotifyTables == nil {
		db.pendingNotifyTables = map[string]struct{}{}
	}
	for _, tableName := range tableNames {
		if tableName != "" {
			db.pendingNotifyTables[tableName] = struct{}{}
		}
	}
}

func escapeSQL(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

//
// Read operations
//

func SelectOne[T any](db *DB, mc QueryMapper[T]) (T, error) {
	return SelectOneContext(context.Background(), db, mc)
}

func SelectOneContext[T any](ctx context.Context, db *DB, mc QueryMapper[T]) (T, error) {
	if db == nil {
		return *new(T), fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	query, mapper := mc()
	lockStart := time.Now()
	if err := db.opMu.LockContext(ctx); err != nil {
		return *new(T), fmt.Errorf("failed to select one: %w", err)
	}
	defer db.opMu.Unlock()
	if elapsed := time.Since(lockStart); elapsed > time.Second {
		notifyLog.Debugf("select one waited %s for db operation lock", elapsed)
	}

	var res T
	var err error
	for attempt := 1; attempt <= 2; attempt++ {
		res, err = sq.FetchOneContext(ctx, lockedSQLDB{db: db}, query.SetDialect(sq.DialectMySQL), mapper)
		if err == nil {
			return res, nil
		}
		if attempt == 2 || !isRetryableCommittedWriteError(err) {
			return res, fmt.Errorf("failed to select one: %w", err)
		}
		if waitErr := waitBeforeCommittedWriteRetryContext(ctx, attempt); waitErr != nil {
			return res, fmt.Errorf("failed to select one: %w", waitErr)
		}
	}
	return res, fmt.Errorf("failed to select one: %w", err)
}

func SelectMultiple[T any](db *DB, mc QueryMapper[T]) ([]T, error) {
	return SelectMultipleContext(context.Background(), db, mc)
}

func SelectMultipleContext[T any](ctx context.Context, db *DB, mc QueryMapper[T]) ([]T, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	query, mapper := mc()
	lockStart := time.Now()
	if err := db.opMu.LockContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to select multiple: %w", err)
	}
	defer db.opMu.Unlock()
	if elapsed := time.Since(lockStart); elapsed > time.Second {
		notifyLog.Debugf("select multiple waited %s for db operation lock", elapsed)
	}

	var res []T
	var err error
	for attempt := 1; attempt <= 2; attempt++ {
		res, err = sq.FetchAllContext(ctx, lockedSQLDB{db: db}, query.SetDialect(sq.DialectMySQL), mapper)
		if err == nil {
			return res, nil
		}
		if attempt == 2 || !isRetryableCommittedWriteError(err) {
			return nil, fmt.Errorf("failed to select multiple: %w", err)
		}
		if waitErr := waitBeforeCommittedWriteRetryContext(ctx, attempt); waitErr != nil {
			return nil, fmt.Errorf("failed to select multiple: %w", waitErr)
		}
	}
	return nil, fmt.Errorf("failed to select multiple: %w", err)
}

//
// Edit operations
//

// Insert inserts a new entry in the database using the sq query builder. It
// returns after the root is published locally; checkpoint and durable status
// remain asynchronous and can be queried with SwarmionRootStatus.
func Insert(db *DB, mappers ...InsertMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWrite("insert", "insert", false, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range mappers {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

// InsertPublished commits and publishes a declarative write, but does not wait
// for the local durable checkpoint view to include that write. Use this for
// user-facing desired-state and durable feedback writes whose effects are
// observed by a reconciler or task stream.
func InsertPublished(db *DB, mappers ...InsertMapper) error {
	return InsertPublishedContext(context.Background(), db, mappers...)
}

func InsertPublishedContext(ctx context.Context, db *DB, mappers ...InsertMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWriteContext(ctx, "insert", "insert", false, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range mappers {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

// Update publishes the committed root locally and leaves checkpoint/durable
// observation asynchronous. Callers that need those stronger outcomes must
// track the returned root through the runtime status API.
func Update(db *DB, mappers ...UpdateMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWrite("update", "update", true, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range mappers {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

func UpdatePublishedContext(ctx context.Context, db *DB, mappers ...UpdateMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWriteContext(ctx, "update", "update", true, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range mappers {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

// UpdateAndInsertPublished commits update and insert mappers as one published
// write, without synchronously waiting for local checkpoint visibility.
func UpdateAndInsertPublished(db *DB, updates []UpdateMapper, inserts []InsertMapper) error {
	return UpdateAndInsertPublishedContext(context.Background(), db, updates, inserts)
}

func UpdateAndInsertPublishedContext(ctx context.Context, db *DB, updates []UpdateMapper, inserts []InsertMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWriteContext(ctx, "update", "update", true, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range updates {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		for _, mapper := range inserts {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

// Delete publishes the committed root locally and leaves checkpoint/durable
// observation asynchronous. It does not treat a revisitable parked status as
// a write failure.
func Delete(db *DB, mappers ...DeleteMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWrite("delete", "delete", true, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range mappers {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

func DeletePublishedContext(ctx context.Context, db *DB, mappers ...DeleteMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWriteContext(ctx, "delete", "delete", true, false, func(ctx context.Context, sqldb *sql.DB) error {
		staged := false
		for _, mapper := range mappers {
			if err := execWriteMapperContext(ctx, sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return writeApplyError{err: err, staged: staged}
			}
			staged = true
		}
		return nil
	})
}

type writeApplyError struct {
	err    error
	staged bool
}

func (e writeApplyError) Error() string {
	if e.err == nil {
		return ""
	}
	return e.err.Error()
}

func (e writeApplyError) Unwrap() error {
	return e.err
}

func (db *DB) committedWrite(operation string, commitMessage string, allowNoop bool, waitForCheckpoint bool, apply func(context.Context, *sql.DB) error) error {
	return db.committedWriteContext(context.Background(), operation, commitMessage, allowNoop, waitForCheckpoint, apply)
}

func (db *DB) committedWriteContext(ctx context.Context, operation string, commitMessage string, allowNoop bool, waitForCheckpoint bool, apply func(context.Context, *sql.DB) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var lastErr error
	for attempt := 1; attempt <= committedWriteMaxAttempts; attempt++ {
		lockStart := time.Now()
		if err := db.opMu.LockContext(ctx); err != nil {
			return fmt.Errorf("failed to %s: %w", operation, err)
		}
		locked := true
		unlock := func() {
			if locked {
				db.opMu.Unlock()
				locked = false
			}
		}
		if elapsed := time.Since(lockStart); elapsed > time.Second {
			notifyLog.Debugf("committed %s waited %s for db operation lock on attempt %d", operation, elapsed, attempt)
		}

		sqldb := db.GetSqlDB()
		if sqldb == nil {
			unlock()
			return fmt.Errorf("db is not initialized")
		}
		applyStart := time.Now()
		if err := apply(ctx, sqldb); err != nil {
			lastErr = err
			if attempt == committedWriteMaxAttempts || !isRetryableCommittedWriteError(err) {
				unlock()
				return fmt.Errorf("failed to %s: %w", operation, err)
			}
			notifyLog.Debugf("retrying committed %s after apply failure on attempt %d/%d: %s", operation, attempt, committedWriteMaxAttempts, err.Error())
			needsCatchUp := retryableApplyRequiresReset(err)
			if needsCatchUp {
				// The local working-set reset only touches local SQL/staging, so
				// it stays under the operation lock.
				if resetErr := db.resetWorkingSetForRetry(operation, lastErr); resetErr != nil {
					unlock()
					return fmt.Errorf("failed to %s: %w", operation, resetErr)
				}
			}
			unlock()
			// The checkpoint catch-up does network/admin RPCs; it runs without the
			// operation lock so it cannot starve local readers/writers (APIC reads,
			// task watchers) while it waits on peers.
			if needsCatchUp {
				if catchUpErr := db.catchUpCheckpointForRetry(operation, lastErr); catchUpErr != nil {
					return fmt.Errorf("failed to %s: %w", operation, catchUpErr)
				}
			}
			if waitErr := waitBeforeCommittedWriteRetryContext(ctx, attempt); waitErr != nil {
				return fmt.Errorf("failed to %s: %w", operation, waitErr)
			}
			continue
		}
		if elapsed := time.Since(applyStart); elapsed > time.Second {
			notifyLog.Debugf("committed %s apply phase took %s on attempt %d", operation, elapsed, attempt)
		}

		commitStart := time.Now()
		commit, err := db.commitStaged(ctx, commitMessage, allowNoop)
		if elapsed := time.Since(commitStart); elapsed > time.Second {
			notifyLog.Debugf("committed %s publish phase took %s on attempt %d", operation, elapsed, attempt)
		}
		if err == nil {
			unlock()
			if waitForCheckpoint {
				checkpointCtx, cancel := context.WithTimeout(ctx, committedWriteCheckpointTimeout)
				err = db.waitForCommittedRootObserved(checkpointCtx, commit, commitMessage)
				cancel()
				if err != nil {
					return fmt.Errorf("failed to %s: %w", operation, err)
				}
			}
			return nil
		}
		if allowNoop && isNoopCommitError(err) {
			unlock()
			return nil
		}
		lastErr = err
		if attempt == committedWriteMaxAttempts || !isRetryableCommittedWriteError(err) {
			unlock()
			return fmt.Errorf("failed to %s: %w", operation, err)
		}
		notifyLog.Debugf("retrying committed %s after commit failure on attempt %d/%d: %s", operation, attempt, committedWriteMaxAttempts, err.Error())

		// Local working-set reset stays under the operation lock.
		if resetErr := db.resetWorkingSetForRetry(operation, lastErr); resetErr != nil {
			unlock()
			return fmt.Errorf("failed to %s: %w", operation, resetErr)
		}
		unlock()
		// Network checkpoint catch-up runs without the operation lock.
		if catchUpErr := db.catchUpCheckpointForRetry(operation, lastErr); catchUpErr != nil {
			return fmt.Errorf("failed to %s: %w", operation, catchUpErr)
		}
		if waitErr := waitBeforeCommittedWriteRetryContext(ctx, attempt); waitErr != nil {
			return fmt.Errorf("failed to %s: %w", operation, waitErr)
		}
	}

	return fmt.Errorf("failed to %s: %w", operation, lastErr)
}

func isNoopCommitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "nothing to commit")
}

func retryableApplyRequiresReset(err error) bool {
	if err == nil {
		return false
	}
	var applyErr writeApplyError
	if errors.As(err, &applyErr) && !applyErr.staged && isTransientWorkspaceAccessError(applyErr.err) {
		return false
	}
	return true
}

func waitBeforeCommittedWriteRetryContext(ctx context.Context, attempt int) error {
	if attempt < 1 {
		attempt = 1
	}
	delay := 100 * time.Millisecond
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= time.Second {
			delay = time.Second
			break
		}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// resetWorkingSetForRetry hard-resets the local Dolt working set after a
// retryable write failure. It performs only local SQL/staging work and is meant
// to run while the operation lock is held.
func (db *DB) resetWorkingSetForRetry(operation string, cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if resetErr := db.resetWorkingSet(ctx); resetErr != nil {
		return errors.Join(cause, resetErr)
	}
	return nil
}

// catchUpCheckpointForRetry performs the network checkpoint catch-up after a
// retryable write failure. It does network/admin RPCs and MUST run without the
// operation lock held so that the wait on peers cannot starve local readers or
// writers. A retryable catch-up response is treated as deferred (non-fatal): the
// surrounding write loop will retry the whole operation.
func (db *DB) catchUpCheckpointForRetry(operation string, cause error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	catchUpErr := db.catchUpCheckpointStrict(ctx, operation+" retry after stale write")
	if errors.Is(catchUpErr, errSwarmionCheckpointCatchUpRetryable) {
		notifyLog.Debugf("deferred swarmion checkpoint catch-up for %q after retryable response: %s", operation, catchUpErr.Error())
		return nil
	}
	if catchUpErr != nil {
		return errors.Join(cause, catchUpErr)
	}
	return nil
}

func (db *DB) resetWorkingSet(ctx context.Context) error {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return fmt.Errorf("db is not initialized")
	}
	if _, err := sqldb.ExecContext(ctx, "CALL DOLT_RESET('--hard')"); err != nil {
		return fmt.Errorf("reset failed write: %w", err)
	}
	return nil
}

func isRetryableCommittedWriteError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "stale write context") ||
		strings.Contains(lower, "replay-base conflict") ||
		strings.Contains(lower, "checkpoint target changed before catch-up") ||
		strings.Contains(lower, "conflicts with protocol root") ||
		isTransientWorkspaceAccessError(err)
}

func isTransientWorkspaceAccessError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "re-open tentative dolt db") ||
		strings.Contains(lower, "database is locked") ||
		strings.Contains(lower, "database is read only") ||
		strings.Contains(lower, "cannot update manifest")
}

func execWriteMapperContext(ctx context.Context, sqldb *sql.DB, query sq.Query) error {
	statement, args, err := sq.ToSQL(sq.DialectMySQL, query, nil)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, err := sqldb.ExecContext(ctx, statement, args...); err != nil {
		return err
	}
	return nil
}
