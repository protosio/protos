package db

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"encoding/hex"
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
	"github.com/protosio/protos/internal/config"
	protoscontracts "github.com/protosio/protos/internal/db/contracts/sql/protos"
	"github.com/protosio/protos/internal/util"
	"github.com/rs/xid"
	"google.golang.org/grpc"
	swarmionapp "swarmion.dev/runtime/app"
	swarmiondoltrepo "swarmion.dev/runtime/doltrepo"
	cueschema "swarmion.dev/schema-engines/cue"
	declarativeschema "swarmion.dev/schema-engines/declarative"
	libp2ptransport "swarmion.dev/transports/libp2p"
)

const (
	swarmionNamespaceTemplate      = "/protos/db/%s"
	swarmionAdminNamespaceTemplate = "/protos/db/%s/admin"
	swarmionPortOffset             = 1

	committedWriteMaxAttempts = 3
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
	Hash         string
	Committer    string
	Email        string
	Date         time.Time
	Message      string
	ParentHashes []string
	Refs         []string
}

type DB struct {
	app     *swarmionapp.App
	network *libp2ptransport.Network
	sqldb   *sql.DB

	name       string
	workingDir string
	signer     Signer

	mu                   sync.Mutex
	opMu                 sync.Mutex
	initialized          bool
	watchCancel          context.CancelFunc
	witnessMu            sync.Mutex
	witnessRankRequests  map[string]pendingWitnessRankRequest
	tableChangeCallbacks *util.Map[string, tableChangeCallback]
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

	app, err := swarmionapp.Open(ctx, swarmionapp.Config{
		Repository: swarmiondoltrepo.Config{
			Dir:         db.workingDir,
			Name:        db.name,
			CommitName:  db.signer.GetID(),
			CommitEmail: db.signer.GetID() + "@protos.local",
			Signer:      swarmionSigner,
		},
		BootstrapPeers:                 append([]string(nil), bootstrapPeers...),
		Namespace:                      fmt.Sprintf(swarmionNamespaceTemplate, db.name),
		AdminNamespace:                 fmt.Sprintf(swarmionAdminNamespaceTemplate, db.name),
		HeartbeatInterval:              5 * time.Second,
		FinalizedMaterializationPolicy: swarmionapp.FinalizedMaterializationEager,
		AutomaticEpochPolicies:         true,
		WitnessSelectionLimit:          protosWitnessSelectionLimit,
		SchemaEngine:                   cueschema.New(protoscontracts.Catalog, declarativeschema.New(protoscontracts.Catalog)),
		OnWriteNotification:            db.handleWriteNotification,
		Logger:                         logger,
	}, network)
	if err != nil {
		_ = network.Close()
		return fmt.Errorf("failed to open swarmion runtime: %w", err)
	}

	db.app = app
	db.network = network
	db.sqldb = app.SQLDB()
	db.initialized = true
	watchCtx, watchCancel := context.WithCancel(context.Background())
	db.watchCancel = watchCancel
	db.startSwarmionWatchers(watchCtx, app)
	return nil
}

func (db *DB) startSwarmionWatchers(ctx context.Context, app *swarmionapp.App) {
	if db == nil || app == nil {
		return
	}
	if events, err := app.WatchFinalizedRoots(ctx); err == nil {
		go db.forwardFinalizedRootEvents(events)
	} else {
		notifyLog.Warnf("failed to watch swarmion finalized roots: %s", err.Error())
	}
	if events, err := app.WatchStatus(ctx); err == nil {
		go db.forwardSwarmionStatusEvents(events)
	} else {
		notifyLog.Warnf("failed to watch swarmion status: %s", err.Error())
	}
}

func (db *DB) forwardFinalizedRootEvents(events <-chan swarmionapp.FinalizedRootEvent) {
	for event := range events {
		db.triggerTableChangeCallbacks(event.ChangedTables...)
	}
}

func (db *DB) forwardSwarmionStatusEvents(events <-chan swarmionapp.StatusEvent) {
	for event := range events {
		switch event.Kind {
		case swarmionapp.StatusEventFinalizedRootChanged,
			swarmionapp.StatusEventTentativeRootChanged,
			swarmionapp.StatusEventFatalChanged,
			swarmionapp.StatusEventActiveWitnessesChanged,
			swarmionapp.StatusEventEligibleWitnessesChanged,
			swarmionapp.StatusEventStateProvidersChanged:
			db.triggerTableChangeCallbacks()
		}
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
		if _, err := db.commitStaged(ctx, "run migrations", true); err != nil {
			return fmt.Errorf("commit migrations: %w", err)
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
	if len(bootstrapPeers) == 0 {
		return fmt.Errorf("cannot initialize swarmion db from peer %s without bootstrap addresses", peerID)
	}
	if err := db.openSwarmion(context.Background(), bootstrapPeers); err != nil {
		return fmt.Errorf("failed to initialize swarmion db from peer %s: %w", peerID, err)
	}
	db.triggerTableChangeCallbacks()
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
	if peerID == app.PeerID() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := app.EvictPeer(ctx, peerID); err != nil {
		errText := err.Error()
		if strings.Contains(errText, "still an active witness") ||
			strings.Contains(errText, "still witness-eligible") ||
			strings.Contains(errText, "cannot evict local peer") {
			return nil
		}
		return fmt.Errorf("evict swarmion peer %s: %w", peerID, err)
	}
	return nil
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

func (db *DB) CatchUpFinalized(ctx context.Context, reason string) error {
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
		reason = "protos finalized read"
	}
	if _, err := app.CatchUpFinalized(ctx, reason); err != nil {
		return fmt.Errorf("catch up swarmion finalized view: %w", err)
	}
	return nil
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
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return sqldb.ExecContext(ctx, query, args...)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return sqldb.QueryContext(ctx, query, args...)
}

func (db *DB) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return sqldb.PrepareContext(ctx, query)
}

func (db *DB) ExecSQLAndCommit(statement string, commitMsg string) (string, error) {
	if strings.TrimSpace(statement) == "" {
		return "", fmt.Errorf("sql statement is empty")
	}

	db.opMu.Lock()
	defer db.opMu.Unlock()

	if _, err := db.ExecContext(context.Background(), statement); err != nil {
		return "", fmt.Errorf("failed to exec sql statement: %w", err)
	}
	commit, err := db.commitStaged(context.Background(), commitMsg, false)
	if err != nil {
		return "", err
	}

	return commit, nil
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
	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve commits: %w", err)
	}
	defer rows.Close()

	var commits []Commit
	for rows.Next() {
		var commit Commit
		var parents sql.NullString
		var refs sql.NullString
		if err := rows.Scan(&commit.Hash, &commit.Committer, &commit.Email, &commit.Date, &commit.Message, &parents, &refs); err != nil {
			return nil, fmt.Errorf("failed to scan commit: %w", err)
		}
		commit.ParentHashes = splitCommitList(parents.String)
		commit.Refs = splitCommitList(refs.String)
		commits = append(commits, commit)
	}
	if err := rows.Err(); err != nil {
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

func (db *DB) commitStaged(ctx context.Context, message string, allowNoop bool) (string, error) {
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return "", fmt.Errorf("db is not initialized")
	}
	if strings.TrimSpace(message) == "" {
		message = "swarmion commit"
	}

	call := "CALL swarmion_commit('-Am', ?)"
	if allowNoop {
		call = "CALL swarmion_commit_info('-Am', ?)"
	}

	rows, err := sqldb.QueryContext(ctx, call, message)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	values := make([]any, len(cols))
	scan := make([]any, len(cols))
	for i := range values {
		scan[i] = &values[i]
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", nil
	}
	if err := rows.Scan(scan...); err != nil {
		return "", err
	}
	for i, col := range cols {
		if strings.EqualFold(col, "hash") {
			return fmt.Sprint(values[i]), nil
		}
	}
	return "", nil
}

func (db *DB) RegisterTableChangeCallback(tableName string, notifier Notifier) {
	if db == nil || notifier == nil {
		return
	}
	guid := xid.New()
	db.tableChangeCallbacks.Set(guid.String(), tableChangeCallback{
		tableName: tableName,
		notifier:  notifier,
	})
}

func (db *DB) WatchChanges(ctx context.Context) (<-chan ChangeEvent, func()) {
	ch := make(chan ChangeEvent, 1)
	if db == nil {
		close(ch)
		return ch, func() {}
	}
	notifier := &changeWatchNotifier{ch: ch}
	guid := xid.New().String()
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
	if !notification.Accepted || len(notification.ChangedTables) == 0 {
		return nil
	}
	db.triggerTableChangeCallbacks(notification.ChangedTables...)
	return nil
}

func (db *DB) triggerTableChangeCallbacks(tableNames ...string) {
	if db == nil {
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
		notifyChangeAsync(callback.notifier, tableNames)
	}
}

func escapeSQL(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

//
// Read operations
//

func SelectOne[T any](db *DB, mc QueryMapper[T]) (T, error) {
	query, mapper := mc()
	res, err := sq.FetchOne(db, query.SetDialect(sq.DialectMySQL), mapper)
	if err != nil {
		return res, fmt.Errorf("failed to select one: %w", err)
	}
	return res, nil
}

func SelectMultiple[T any](db *DB, mc QueryMapper[T]) ([]T, error) {
	query, mapper := mc()
	res, err := sq.FetchAll(db, query.SetDialect(sq.DialectMySQL), mapper)
	if err != nil {
		return nil, fmt.Errorf("failed to select multiple: %w", err)
	}
	return res, nil
}

//
// Edit operations
//

// Insert inserts a new entry in the database using the sq query builder
func Insert(db *DB, mappers ...InsertMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWrite("insert", "insert", false, func(sqldb *sql.DB) error {
		for _, mapper := range mappers {
			if err := execWriteMapper(sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return err
			}
		}
		return nil
	})
}

func Update(db *DB, mappers ...UpdateMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWrite("update", "update", true, func(sqldb *sql.DB) error {
		for _, mapper := range mappers {
			if err := execWriteMapper(sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return err
			}
		}
		return nil
	})
}

func Delete(db *DB, mappers ...DeleteMapper) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	return db.committedWrite("delete", "delete", true, func(sqldb *sql.DB) error {
		for _, mapper := range mappers {
			if err := execWriteMapper(sqldb, mapper().SetDialect(sq.DialectMySQL)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (db *DB) committedWrite(operation string, commitMessage string, allowNoop bool, apply func(*sql.DB) error) error {
	db.opMu.Lock()
	defer db.opMu.Unlock()

	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return fmt.Errorf("db is not initialized")
	}

	var lastErr error
	for attempt := 1; attempt <= committedWriteMaxAttempts; attempt++ {
		if err := apply(sqldb); err != nil {
			return fmt.Errorf("failed to %s: %w", operation, err)
		}

		_, err := db.commitStaged(context.Background(), commitMessage, allowNoop)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == committedWriteMaxAttempts || !isRetryableCommittedWriteError(err) {
			return fmt.Errorf("failed to %s: %w", operation, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		resetErr := db.resetWorkingSet(ctx)
		catchUpErr := db.CatchUpFinalized(ctx, operation+" retry after stale write")
		cancel()
		if resetErr != nil {
			return fmt.Errorf("failed to %s: %w", operation, errors.Join(lastErr, resetErr))
		}
		if catchUpErr != nil {
			lastErr = errors.Join(lastErr, catchUpErr)
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}

	return fmt.Errorf("failed to %s: %w", operation, lastErr)
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
		strings.Contains(lower, "finalized target changed before catch-up") ||
		strings.Contains(lower, "conflicts with protocol root")
}

func execWriteMapper(sqldb *sql.DB, query sq.Query) error {
	statement, args, err := sq.ToSQL(sq.DialectMySQL, query, nil)
	if err != nil {
		return err
	}
	if _, err := sqldb.ExecContext(context.Background(), statement, args...); err != nil {
		return err
	}
	return nil
}
