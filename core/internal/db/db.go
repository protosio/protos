package db

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/dolthub/vitess/go/vt/sqlparser"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/nustiueudinastea/swarmion/runtime/sqlread"
	cueschema "github.com/nustiueudinastea/swarmion/schema-engines/cue"
	declarativeschema "github.com/nustiueudinastea/swarmion/schema-engines/declarative"
	swarmiontransport "github.com/nustiueudinastea/swarmion/transports"
	protoscontracts "github.com/protosio/protos/internal/db/contracts/sql/protos"
	"github.com/protosio/protos/internal/util"
	"github.com/rs/xid"
	"google.golang.org/grpc"
)

const (
	swarmionNamespaceTemplate      = "/protos/db/%s"
	swarmionAdminNamespaceTemplate = "/protos/db/%s/admin"
	swarmionStateDirName           = ".swarmion"

	ordinaryWriteSafeRetryMaxAttempts = 3
	sqlViewReadyRetryMaxAttempts      = 20
	ordinaryUncertainReceiptTimeout   = 10 * time.Second
	committedWriteCheckpointTimeout   = 45 * time.Second
	initFromPeerRetryBudget           = 45 * time.Second
	initFromPeerRetryInitialBackoff   = time.Second
	initFromPeerRetryMaxBackoff       = 5 * time.Second
	automaticBootstrapRetryBackoff    = 250 * time.Millisecond
	automaticBootstrapRetryMaxBackoff = 5 * time.Second
)

var (
	errSwarmionCheckpointCatchUpRetryable = errors.New("swarmion checkpoint catch-up retryable")
	errSwarmionPublishedWriteIncomplete   = errors.New("swarmion published write receipt incomplete")
	errPublishedWriteRetryExhausted       = errors.New("published write retry budget exhausted")

	// ErrEventReceiptPending identifies a published event that did not reach a
	// materialized durable checkpoint before its bounded wait ended.
	ErrEventReceiptPending = errors.New("swarmion event receipt pending")
	// ErrEventReceiptParked identifies a revisitable, exact-event parked
	// classification. It is not a permanent protocol rejection.
	ErrEventReceiptParked = errors.New("swarmion event receipt parked")
	// ErrOperationReceiptUnavailable means Swarmion cannot yet prove whether a
	// stable operation key is absent. Callers must wait for bootstrap/lineage
	// recovery and resolve the same operation again; publishing is unsafe.
	ErrOperationReceiptUnavailable = errors.New("swarmion operation receipt unavailable")
	// ErrPublishedWriteReceiptIdentityConflict identifies an impossible
	// disagreement between an uncertain commit's exact event receipt and the
	// receipt resolved from its stable operation key. The operation must not be
	// replayed or reported successful.
	ErrPublishedWriteReceiptIdentityConflict = errors.New("published write receipt identity conflict")
	// ErrPublishedWriteNoChange means an operation-backed mutation supplied no
	// executable statements, so no idempotency identity could be consumed.
	// A non-empty stable operation that changes no content instead uses
	// Swarmion's strict no-change policy and returns an exact same-root receipt.
	ErrPublishedWriteNoChange = errors.New("published write operation made no content change")
	// ErrMigrationSchemaConflict identifies a pre-contract schema object whose name
	// matches an embedded migration object but whose definition does not. The
	// migration operation is not started and no receipt is created.
	ErrMigrationSchemaConflict = errors.New("migration schema object conflicts with embedded definition")
)

type Signer interface {
	Sign(commit string) (string, error)
	Verify(commit string, signature string, publicKey string) error
	PublicKey() string
	GetID() string
	Private() []byte
}

type swarmionSigningIdentity struct {
	privateKey libp2pcrypto.PrivKey
	publicKey  []byte
}

func newSwarmionSigningIdentity(signer Signer) (swarmionSigningIdentity, error) {
	if signer == nil {
		return swarmionSigningIdentity{}, fmt.Errorf("signer is nil")
	}
	privateKey, err := libp2pcrypto.UnmarshalEd25519PrivateKey(signer.Private())
	if err != nil {
		return swarmionSigningIdentity{}, fmt.Errorf("unmarshal swarmion private key: %w", err)
	}
	publicKeyBytes, err := libp2pcrypto.MarshalPublicKey(privateKey.GetPublic())
	if err != nil {
		return swarmionSigningIdentity{}, fmt.Errorf("marshal swarmion public key: %w", err)
	}
	return swarmionSigningIdentity{
		privateKey: privateKey,
		publicKey:  append([]byte(nil), publicKeyBytes...),
	}, nil
}

func (s swarmionSigningIdentity) PublicKeyBytes() []byte {
	return append([]byte(nil), s.publicKey...)
}

func (s swarmionSigningIdentity) SignBytes(ctx context.Context, payload []byte) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.privateKey == nil {
		return nil, fmt.Errorf("swarmion signing identity is not initialized")
	}
	signature, err := s.privateKey.Sign(payload)
	if err != nil {
		return nil, fmt.Errorf("sign swarmion payload: %w", err)
	}
	return signature, nil
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

// RepositoryReadiness distinguishes a genuinely fresh database from an
// existing repository whose complete Swarmion Open is still pending or has
// failed. Initialized=false alone is not authority to accept a fresh Init RPC.
type RepositoryReadiness struct {
	Initialized        bool
	ExistingRepository bool
	BootstrapPending   bool
	BootstrapError     error
}

type eventReceiptContentDissentObservationKey struct {
	eventID                   string
	publishedRootHash         string
	durableCheckpointCommitID string
	durableCheckpointRootHash string
	queryableRootHash         string
}

type DB struct {
	host    *swarmionapp.Host
	runtime *swarmionapp.DatabaseRuntime
	link    swarmiontransport.Link
	sqldb   *sql.DB

	name       string
	workingDir string
	signer     Signer

	mu                   sync.Mutex
	openMu               contextMutex
	opMu                 contextMutex
	initialized          bool
	existingRepository   bool
	runtimeOpenedAt      time.Time
	runtimeGeneration    uint64
	watchCancel          context.CancelFunc
	backgroundWork       backgroundWorkRegistry
	tableChangeCallbacks *util.Map[string, tableChangeCallback]
	runtimeCallbacks     *util.Map[string, Notifier]
	replicationNoticeSig string
	notificationMu       sync.Mutex
	notificationDepth    int
	pendingNotifyAll     bool
	pendingNotifyTables  map[string]struct{}

	eventReceiptContentDissentObservations atomic.Uint64
	eventReceiptContentDissentSeen         sync.Map
	transactionMetrics                     transactionMetrics

	// Focused transaction and lifecycle tests use these seams to synchronize a
	// protocol-head movement, emulate typed publication outcomes, and inject
	// receipt, status, or migration-finalization outcomes. They are
	// never configured by production code.
	observePublishedWriteForTest           func(context.Context, PublishedWriteReceipt) (EventReceiptObservation, error)
	runMigrationsForTest                   func(context.Context) error
	waitSQLViewReadyForTest                func(context.Context) error
	waitMutationReadyForTest               func(context.Context) error
	waitPublishedWriteAppliedForTest       func(context.Context, PublishedWriteReceipt, string) (EventReceiptObservation, error)
	beforeOperationRecoveryReplaceForTest  func(PublishedWriteOperation) error
	beforeOperationCleanupAdmissionForTest func()
	executeOperationForTest                func(context.Context, *swarmionapp.DatabaseRuntime, swarmionapp.OperationRequest) swarmionapp.OperationResult
	waitPublishedWriteRetryForTest         func(context.Context, int, swarmionapp.OperationRetryReason) error

	bootstrapRetryMu     sync.Mutex
	bootstrapRetryCancel context.CancelFunc
	bootstrapRetryDone   chan struct{}
	bootstrapRetryErr    error

	operationRecoveryMu sync.Mutex
}

type contextMutex struct {
	once sync.Once
	ch   chan struct{}
}

// backgroundWorkRegistry serializes lifecycle-owned goroutine admission with
// shutdown. Once sealed, no Add can race the close-side Wait; a later runtime
// open resets the registry only after the preceding close joined every worker.
type backgroundWorkRegistry struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	sealed   bool
	active   int
	admitted uint64
}

func (registry *backgroundWorkRegistry) reset() error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.active != 0 {
		return fmt.Errorf("reset background work registry with %d active workers", registry.active)
	}
	registry.sealed = false
	return nil
}

func (registry *backgroundWorkRegistry) begin() (func(), bool) {
	registry.mu.Lock()
	if registry.sealed {
		registry.mu.Unlock()
		return nil, false
	}
	registry.active++
	registry.admitted++
	registry.wg.Add(1)
	registry.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			registry.mu.Lock()
			registry.active--
			registry.mu.Unlock()
			registry.wg.Done()
		})
	}, true
}

func (registry *backgroundWorkRegistry) seal() {
	registry.mu.Lock()
	registry.sealed = true
	registry.mu.Unlock()
}

func (registry *backgroundWorkRegistry) wait() {
	registry.wg.Wait()
}

func (registry *backgroundWorkRegistry) admissionCount() uint64 {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	return registry.admitted
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

func Open(workDir string, dbName string, signer Signer, link swarmiontransport.Link) (*DB, error) {
	if signer == nil {
		return nil, fmt.Errorf("signer is nil")
	}
	if link == nil {
		return nil, fmt.Errorf("swarmion transport link is nil")
	}
	if dbName == "" {
		return nil, fmt.Errorf("db name is empty")
	}

	absWorkDir, err := filepath.Abs(workDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workdir: %w", err)
	}

	existingRepository := databaseExists(absWorkDir, dbName)
	db := &DB{
		name:                 dbName,
		workingDir:           absWorkDir,
		signer:               signer,
		link:                 link,
		existingRepository:   existingRepository,
		tableChangeCallbacks: util.NewMap[string, tableChangeCallback](),
		runtimeCallbacks:     util.NewMap[string, Notifier](),
	}

	if existingRepository {
		if err := db.openAndFinalizeSwarmion(context.Background(), nil); err != nil {
			if errors.Is(err, swarmionapp.ErrBootstrapNotReady) {
				logBootstrapNotReady("deferred returning database open until a routed bootstrap peer is available", err)
				db.startAutomaticBootstrapRetry()
				return db, nil
			}
			return nil, fmt.Errorf("failed to open swarmion db: %w", err)
		}
	}

	return db, nil
}

func (db *DB) Init() error {
	if err := db.openAndFinalizeSwarmion(context.Background(), nil); err != nil {
		return fmt.Errorf("failed to init swarmion db: %w", err)
	}
	return nil
}

func databaseExists(workDir string, dbName string) bool {
	_, err := os.Stat(filepath.Join(workDir, dbName, ".dolt", "repo_state.json"))
	return err == nil
}

// openAndFinalizeSwarmion serializes the complete database-readiness
// lifecycle. OpenHost establishes the SQL projection, but the backend does not
// expose the database as initialized until its embedded migrations have also
// completed. A migration failure closes only this Swarmion scope; the borrowed
// application transport remains owned and usable by its caller.
func (db *DB) openAndFinalizeSwarmion(ctx context.Context, bootstrapPeers []string) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := db.openMu.LockContext(ctx); err != nil {
		return err
	}
	defer db.openMu.Unlock()

	db.mu.Lock()
	alreadyInitialized := db.initialized && db.runtime != nil && db.sqldb != nil
	db.mu.Unlock()
	if alreadyInitialized {
		return nil
	}
	if err := db.openSwarmionLocked(ctx, bootstrapPeers); err != nil {
		return err
	}
	if err := db.finalizeSwarmionOpenLocked(ctx); err != nil {
		migrationErr := fmt.Errorf("run migrations after opening swarmion runtime: %w", err)
		if closeErr := db.closeSwarmionRuntimeLocked(); closeErr != nil {
			return errors.Join(migrationErr, fmt.Errorf("close swarmion runtime after migration failure: %w", closeErr))
		}
		return migrationErr
	}
	return nil
}

// openSwarmionLocked opens the scoped Swarmion runtime while openMu is held.
// Callers that need a ready backend must use openAndFinalizeSwarmion.
func (db *DB) openSwarmionLocked(ctx context.Context, bootstrapPeers []string) error {
	db.mu.Lock()
	if db.runtime != nil {
		db.mu.Unlock()
		return nil
	}
	link := db.link
	db.mu.Unlock()
	if link == nil {
		return fmt.Errorf("swarmion transport link is nil")
	}

	logger := log.New(io.Discard, "", log.LstdFlags|log.Lmicroseconds|log.LUTC)
	identity, err := newSwarmionSigningIdentity(db.signer)
	if err != nil {
		return fmt.Errorf("failed to create swarmion signing identity: %w", err)
	}

	swarmManifest, hasSwarmManifest, err := db.loadSwarmManifest()
	if err != nil {
		return err
	}
	databaseID := swarmionapp.DatabaseID(db.name)
	databaseConfig := swarmionapp.DatabaseConfig{
		ID: databaseID,
		Repository: swarmionapp.RepositoryConfig{
			Dir:         db.workingDir,
			Name:        db.name,
			CommitName:  db.signer.GetID(),
			CommitEmail: db.signer.GetID() + "@protos.local",
		},
		BootstrapPeers:      append([]string(nil), bootstrapPeers...),
		Namespace:           fmt.Sprintf(swarmionNamespaceTemplate, db.name),
		AdminNamespace:      fmt.Sprintf(swarmionAdminNamespaceTemplate, db.name),
		HeartbeatInterval:   5 * time.Second,
		SchemaEngine:        cueschema.New(protoscontracts.Catalog, declarativeschema.New(protoscontracts.Catalog)),
		OnWriteNotification: db.handleWriteNotification,
		Logger:              logger,
	}
	if hasSwarmManifest {
		databaseConfig.SwarmManifest = swarmManifest
	}
	host, err := swarmionapp.OpenHost(ctx, swarmionapp.HostConfig{
		Identity:          identity,
		LogicalPeerTarget: 32,
		Databases:         []swarmionapp.DatabaseConfig{databaseConfig},
		Logger:            logger,
	}, link)
	if err != nil {
		return fmt.Errorf("failed to open swarmion runtime: %w", err)
	}
	runtime, ok := host.Database(databaseID)
	if !ok || runtime == nil {
		_ = host.Close()
		return fmt.Errorf("opened swarmion host without database runtime %q", databaseID)
	}
	if err := db.persistSwarmManifest(ctx, runtime); err != nil {
		_ = host.Close()
		return err
	}
	readDB, err := sqlread.Open(runtime)
	if err != nil {
		_ = host.Close()
		return fmt.Errorf("open read-only Swarmion database/sql adapter: %w", err)
	}
	if err := db.backgroundWork.reset(); err != nil {
		_ = readDB.Close()
		_ = host.Close()
		return fmt.Errorf("prepare Swarmion background work: %w", err)
	}

	db.mu.Lock()
	if db.runtimeGeneration == ^uint64(0) {
		db.mu.Unlock()
		_ = readDB.Close()
		_ = host.Close()
		return fmt.Errorf("swarmion runtime generation exhausted")
	}
	db.runtimeGeneration++
	db.host = host
	db.runtime = runtime
	db.runtimeOpenedAt = time.Now()
	db.sqldb = readDB
	configureEmbeddedSQLDB(db.sqldb)
	db.initialized = false
	db.watchCancel = nil
	db.mu.Unlock()
	return nil
}

func (db *DB) finalizeSwarmionOpenLocked(ctx context.Context) error {
	if err := db.recoverPublishedWriteOperations(ctx); err != nil {
		return err
	}
	db.mu.Lock()
	runMigrationsForTest := db.runMigrationsForTest
	db.mu.Unlock()
	if runMigrationsForTest != nil {
		if err := runMigrationsForTest(ctx); err != nil {
			return err
		}
	} else if err := db.runMigrations(ctx); err != nil {
		return err
	}

	db.mu.Lock()
	if db.runtime == nil || db.sqldb == nil {
		db.mu.Unlock()
		return fmt.Errorf("swarmion runtime closed before migration finalization")
	}
	runtime := db.runtime
	watchCtx, watchCancel := context.WithCancel(context.Background())
	db.watchCancel = watchCancel
	db.initialized = true
	db.mu.Unlock()
	db.startSwarmionWatchers(watchCtx, runtime)
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

func (db *DB) persistSwarmManifest(ctx context.Context, runtime *swarmionapp.DatabaseRuntime) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if runtime == nil {
		return fmt.Errorf("swarmion database runtime is not initialized")
	}
	status, err := runtime.Diagnostics().SwarmManifestStatus(ctx)
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

func (db *DB) startSwarmionWatchers(ctx context.Context, runtime *swarmionapp.DatabaseRuntime) {
	if db == nil || runtime == nil {
		return
	}
	statusDone, admitted := db.backgroundWork.begin()
	if !admitted {
		return
	}
	go func() {
		defer statusDone()
		db.runSwarmionStatusSubscription(ctx, func(ctx context.Context) (statusEventSubscription, error) {
			return runtime.SubscribeStatus(ctx)
		})
	}()

	checkpointDone, admitted := db.backgroundWork.begin()
	if !admitted {
		return
	}
	go func() {
		defer checkpointDone()
		db.runSwarmionCheckpointSubscription(ctx, func(ctx context.Context) (checkpointEventSubscription, error) {
			return runtime.SubscribeCheckpointRoots(ctx)
		})
	}()
}

type statusEventSubscription interface {
	Events() <-chan swarmionapp.StatusEvent
	Close()
}

type checkpointEventSubscription interface {
	Events() <-chan swarmionapp.CheckpointEvent
	Close()
}

const (
	swarmionSubscriptionRetryInitial = 50 * time.Millisecond
	swarmionSubscriptionRetryMaximum = 2 * time.Second
)

func (db *DB) runSwarmionStatusSubscription(
	ctx context.Context,
	subscribe func(context.Context) (statusEventSubscription, error),
) {
	recovering := false
	backoff := swarmionSubscriptionRetryInitial
	for {
		subscription, err := subscribe(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			notifyLog.Warnf("failed to subscribe to Swarmion status: %s", err.Error())
			db.triggerRuntimeChangeCallbacks()
			// This worker is bound to one DatabaseRuntime. If that owner reports
			// itself closed while the local watch context is still live, invalidate
			// consumers but do not spin forever against the same closed handle.
			if errors.Is(err, swarmionapp.ErrDatabaseRuntimeClosed) {
				return
			}
			recovering = true
			if !waitForSwarmionSubscriptionRetry(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, swarmionSubscriptionRetryMaximum)
			continue
		}
		unexpectedTerminal := db.forwardSwarmionStatusEvents(ctx, subscription.Events(), recovering)
		subscription.Close()
		if !unexpectedTerminal || ctx.Err() != nil {
			return
		}
		recovering = true
		if !waitForSwarmionSubscriptionRetry(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, swarmionSubscriptionRetryMaximum)
	}
}

func (db *DB) runSwarmionCheckpointSubscription(
	ctx context.Context,
	subscribe func(context.Context) (checkpointEventSubscription, error),
) {
	recovering := false
	backoff := swarmionSubscriptionRetryInitial
	for {
		subscription, err := subscribe(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			notifyLog.Warnf("failed to subscribe to Swarmion checkpoints: %s", err.Error())
			db.triggerPublishedTableChangeCallbacks()
			if errors.Is(err, swarmionapp.ErrDatabaseRuntimeClosed) {
				return
			}
			recovering = true
			if !waitForSwarmionSubscriptionRetry(ctx, backoff) {
				return
			}
			backoff = min(backoff*2, swarmionSubscriptionRetryMaximum)
			continue
		}
		unexpectedTerminal := db.forwardSwarmionCheckpointRootEvents(ctx, subscription.Events(), recovering)
		subscription.Close()
		if !unexpectedTerminal || ctx.Err() != nil {
			return
		}
		recovering = true
		if !waitForSwarmionSubscriptionRetry(ctx, backoff) {
			return
		}
		backoff = min(backoff*2, swarmionSubscriptionRetryMaximum)
	}
}

func waitForSwarmionSubscriptionRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (db *DB) forwardSwarmionStatusEvents(ctx context.Context, events <-chan swarmionapp.StatusEvent, invalidateInitial ...bool) bool {
	invalidateFirst := len(invalidateInitial) != 0 && invalidateInitial[0]
	for event := range events {
		if err := event.Validate(); err != nil {
			notifyLog.Warnf("Swarmion status subscription violated its event contract: %s", err.Error())
			db.triggerRuntimeChangeCallbacks()
			return true
		}
		if terminalErr := event.Terminal(); terminalErr != nil {
			// A forwarded reason is descriptive only. Normal shutdown is proven by
			// this worker's own context, never by source-controlled metadata.
			if ctx.Err() != nil {
				return false
			}
			notifyLog.Warnf("Swarmion status subscription terminated unexpectedly; retrying: %s", terminalErr.Error())
			db.triggerRuntimeChangeCallbacks()
			return true
		}
		if event.Initial {
			if invalidateFirst {
				db.triggerRuntimeChangeCallbacks()
				invalidateFirst = false
			}
			continue
		}
		// A sequence gap carries a complete replacement snapshot but no adjacent
		// change hints. Treat it as a full runtime invalidation.
		db.triggerRuntimeChangeCallbacks()
	}
	// Closing without a terminal event loses the source-specific handoff.
	if ctx.Err() != nil {
		return false
	}
	db.triggerRuntimeChangeCallbacks()
	return true
}

func (db *DB) forwardSwarmionCheckpointRootEvents(ctx context.Context, events <-chan swarmionapp.CheckpointEvent, invalidateInitial ...bool) bool {
	invalidateFirst := len(invalidateInitial) != 0 && invalidateInitial[0]
	for event := range events {
		if err := event.Validate(); err != nil {
			notifyLog.Warnf("Swarmion checkpoint subscription violated its event contract: %s", err.Error())
			db.triggerPublishedTableChangeCallbacks()
			return true
		}
		if terminalErr := event.Terminal(); terminalErr != nil {
			if ctx.Err() != nil {
				return false
			}
			notifyLog.Warnf("Swarmion checkpoint subscription terminated unexpectedly; retrying: %s", terminalErr.Error())
			db.triggerPublishedTableChangeCallbacks()
			return true
		}
		if event.Initial {
			if invalidateFirst {
				db.triggerPublishedTableChangeCallbacks()
				invalidateFirst = false
			}
			continue
		}
		if event.SequenceGap || !event.ChangedTablesComplete {
			db.triggerPublishedTableChangeCallbacks()
			continue
		}
		if len(event.ChangedTables) != 0 {
			db.triggerPublishedTableChangeCallbacks(event.ChangedTables...)
		}
	}
	if ctx.Err() != nil {
		return false
	}
	db.triggerPublishedTableChangeCallbacks()
	return true
}

const migrationHistoryCreateStatement = `CREATE TABLE IF NOT EXISTS sqddl_history (
filename VARCHAR(255) NOT NULL,
checksum VARCHAR(255) NOT NULL,
started_at TIMESTAMP NULL,
time_taken_ns BIGINT NOT NULL,
success BOOLEAN NOT NULL,
PRIMARY KEY (filename)
)`

func (db *DB) runMigrations(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	sqldb := db.GetSqlDB()
	if sqldb == nil {
		return fmt.Errorf("db is not initialized")
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
	operation, err := db.migrationBatchPublishedWriteOperation(migrationsDir, filenames)
	if err != nil {
		return err
	}
	operationID := operation.Key()
	resolved, err := db.LookupPublishedWriteOperation(ctx, operation)
	if err != nil {
		return fmt.Errorf("resolve migration batch %q: %w", operationID, err)
	}
	switch resolved.Disposition() {
	case swarmionapp.OperationAccepted:
		published, err := PublishedWriteReceiptFromResult(operation, resolved)
		if err != nil {
			return fmt.Errorf("recover migration batch %q receipt: %w", operationID, err)
		}
		observation, err := db.WaitForPublishedWriteApplied(ctx, published, "recover migrations "+operationID)
		if err != nil {
			return fmt.Errorf("checkpoint recovered migrations: %w", err)
		}
		if err := db.validateMigrationHistoryAtCheckpoint(
			ctx,
			operationID,
			published.EventID,
			observation.Status.DurableCheckpointCommitID,
			migrationsDir,
			filenames,
		); err != nil {
			return err
		}
		if err := db.removePublishedWriteOperation(operation); err != nil {
			return fmt.Errorf("retire recovered migration batch %q: %w", operationID, err)
		}
		return nil
	case swarmionapp.OperationRecoveryRequired:
		return fmt.Errorf("%w: migration batch %q", ErrOperationReceiptUnavailable, operationID)
	case swarmionapp.OperationRetryPermitted:
		// It is safe to stage the batch only after Swarmion has authoritatively
		// proved this operation absent from the recovered lineage.
	case swarmionapp.OperationFailedClosed:
		return fmt.Errorf("resolve migration batch %q failed closed: %s", operationID, operationDiagnosticText(resolved.Diagnostic()))
	default:
		return fmt.Errorf("resolve migration batch %q returned unknown disposition %q", operationID, resolved.Disposition())
	}

	pendingFilenames := make([]string, 0, len(filenames))
	for _, filename := range filenames {
		contents, err := fs.ReadFile(migrationsDir, filename)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", filename, err)
		}
		expectedChecksum := sha256.Sum256(contents)
		applied, err := migrationApplied(ctx, sqldb, filename, hex.EncodeToString(expectedChecksum[:]))
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		pendingFilenames = append(pendingFilenames, filename)
	}
	if len(pendingFilenames) == 0 {
		if err := db.removePublishedWriteOperation(operation); err != nil {
			return fmt.Errorf("retire empty migration batch %q: %w", operationID, err)
		}
		return nil
	}

	published, publishErr := db.executePublishedWriteOperationContext(
		ctx,
		operation,
		"migration batch",
		func(ctx context.Context, executor sqlContextExecer) error {
			if err := appendMigrationStatementIfNeeded(ctx, sqldb, executor, migrationHistoryCreateStatement); err != nil {
				return fmt.Errorf("ensure migration history: %w", err)
			}
			for _, filename := range pendingFilenames {
				if err := db.applyMigration(ctx, sqldb, executor, migrationsDir, filename); err != nil {
					return err
				}
			}
			return nil
		},
	)
	if errors.Is(publishErr, ErrPublishedWriteReceiptIdentityConflict) {
		return fmt.Errorf("commit migrations: %w", publishErr)
	}
	if publishErr != nil && !published.HasExactEventIdentity() {
		return fmt.Errorf("commit migrations: %w", publishErr)
	}
	if !published.HasExactEventIdentity() {
		return fmt.Errorf("commit migrations: migration batch %q did not return an exact published event receipt", operationID)
	}

	if publishErr != nil {
		notifyLog.Warnf(
			"tracking published migration batch after receipt-return error operation=%s event_id=%s published_root=%s error=%s",
			operationID,
			published.EventID,
			published.PublishedRootHash,
			publishErr.Error(),
		)
	}
	observation, err := db.WaitForPublishedWriteApplied(ctx, published, "run migrations "+operationID)
	if err != nil {
		return fmt.Errorf("checkpoint migrations: %w", err)
	}
	if err := db.validateMigrationHistoryAtCheckpoint(
		ctx,
		operationID,
		published.EventID,
		observation.Status.DurableCheckpointCommitID,
		migrationsDir,
		filenames,
	); err != nil {
		return err
	}
	if err := db.removePublishedWriteOperation(operation); err != nil {
		return fmt.Errorf("retire applied migration batch %q: %w", operationID, err)
	}

	return nil
}

func (db *DB) migrationBatchPublishedWriteOperation(migrationsDir fs.FS, filenames []string) (PublishedWriteOperation, error) {
	intent, err := encodeMigrationCatalogIntent(migrationsDir, filenames)
	if err != nil {
		return PublishedWriteOperation{}, err
	}
	return db.migrationPublishedWriteOperation(intent)
}

func encodeMigrationCatalogIntent(migrationsDir fs.FS, filenames []string) ([]byte, error) {
	canonicalFilenames := append([]string(nil), filenames...)
	sort.Strings(canonicalFilenames)
	if uint64(len(canonicalFilenames)) > uint64(^uint32(0)) {
		return nil, fmt.Errorf("migration catalog contains too many files")
	}
	var intent bytes.Buffer
	intent.WriteString("PROTOS-MIGRATION-CATALOG")
	intent.WriteByte(0)
	intent.WriteByte(1)
	if err := binary.Write(&intent, binary.BigEndian, uint32(len(canonicalFilenames))); err != nil {
		return nil, fmt.Errorf("encode migration catalog file count: %w", err)
	}
	for _, filename := range canonicalFilenames {
		if uint64(len(filename)) > uint64(^uint32(0)) {
			return nil, fmt.Errorf("migration filename is too long")
		}
		contents, err := fs.ReadFile(migrationsDir, filename)
		if err != nil {
			return nil, fmt.Errorf("read migration %s for operation identity: %w", filename, err)
		}
		checksum := sha256.Sum256(contents)
		if err := binary.Write(&intent, binary.BigEndian, uint32(len(filename))); err != nil {
			return nil, fmt.Errorf("encode migration filename length: %w", err)
		}
		intent.WriteString(filename)
		intent.Write(checksum[:])
	}
	return intent.Bytes(), nil
}

func (db *DB) validateMigrationHistoryAtCheckpoint(
	ctx context.Context,
	operationID string,
	eventID string,
	durableCheckpointCommitID string,
	migrationsDir fs.FS,
	filenames []string,
) error {
	checkpointCommitID := strings.TrimSpace(durableCheckpointCommitID)
	if swarmionprotocol.ParseCheckpointCommitID(checkpointCommitID).IsZero() {
		return fmt.Errorf("migration batch %q has no durable checkpoint commit for invariant validation", operationID)
	}
	for _, filename := range filenames {
		contents, err := fs.ReadFile(migrationsDir, filename)
		if err != nil {
			return fmt.Errorf("read migration %s for durable invariant: %w", filename, err)
		}
		expectedChecksum := sha256.Sum256(contents)
		expectedChecksumHex := hex.EncodeToString(expectedChecksum[:])
		var (
			found    bool
			checksum string
			success  bool
		)
		err = db.ReadRowsAsOf(
			ctx,
			checkpointCommitID,
			"SELECT checksum, success FROM sqddl_history AS OF ? WHERE filename = ?",
			[]any{filename},
			func(rows *sql.Rows) error {
				if !rows.Next() {
					return nil
				}
				found = true
				return rows.Scan(&checksum, &success)
			},
		)
		if err != nil {
			return fmt.Errorf("query durable sqddl_history invariant for %q: %w", filename, err)
		}
		if !found {
			return fmt.Errorf(
				"migration invariant conflict operation=%s event_id=%s checkpoint=%s: sqddl_history row %q is absent",
				operationID,
				eventID,
				checkpointCommitID,
				filename,
			)
		}
		if !success || !strings.EqualFold(strings.TrimSpace(checksum), expectedChecksumHex) {
			return fmt.Errorf(
				"migration invariant conflict operation=%s event_id=%s checkpoint=%s: sqddl_history row %q got checksum=%q success=%t want checksum=%q success=true",
				operationID,
				eventID,
				checkpointCommitID,
				filename,
				checksum,
				success,
				expectedChecksumHex,
			)
		}
	}
	return nil
}

func (db *DB) applyMigration(
	ctx context.Context,
	sqldb *sql.DB,
	executor sqlContextExecer,
	migrationsDir fs.FS,
	filename string,
) error {
	if sqldb == nil {
		return fmt.Errorf("migration SQL inspector is not configured")
	}
	if executor == nil {
		return fmt.Errorf("migration SQL executor is not configured")
	}
	contents, err := fs.ReadFile(migrationsDir, filename)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", filename, err)
	}
	statements, err := sqlparser.SplitStatementToPieces(string(contents))
	if err != nil {
		return fmt.Errorf("split migration %s: %w", filename, err)
	}
	// The operation helper accepts a static statement list. Start the timer in
	// that list so time_taken_ns measures SQL execution, not this compilation
	// pass.
	if _, err := executor.ExecContext(ctx, `SET @protos_migration_started_at = NOW(6)`); err != nil {
		return fmt.Errorf("start migration timer %s: %w", filename, err)
	}
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if err := appendMigrationStatementIfNeeded(ctx, sqldb, executor, statement); err != nil {
			return fmt.Errorf("%s: %w", filename, err)
		}
	}

	checksum := sha256.Sum256(contents)
	if _, err := executor.ExecContext(
		ctx,
		`INSERT INTO sqddl_history (filename, checksum, started_at, time_taken_ns, success)
VALUES (?, ?, @protos_migration_started_at, TIMESTAMPDIFF(MICROSECOND, @protos_migration_started_at, NOW(6)) * 1000, true)
ON DUPLICATE KEY UPDATE checksum = VALUES(checksum), started_at = VALUES(started_at), time_taken_ns = VALUES(time_taken_ns), success = VALUES(success)`,
		filename,
		hex.EncodeToString(checksum[:]),
	); err != nil {
		return fmt.Errorf("record migration %s: %w", filename, err)
	}
	return nil
}

type migrationColumnSchema struct {
	Name       string
	ColumnType string
	Nullable   bool
	Default    sql.NullString
	Extra      string
	Charset    sql.NullString
	Collation  sql.NullString
	Ordinal    int
}

type migrationIndexFieldSchema struct {
	Column string
	Length int64
	Order  string
}

type migrationIndexSchema struct {
	Found     bool
	Unique    bool
	IndexType string
	Fields    []migrationIndexFieldSchema
}

// appendMigrationStatementIfNeeded compiles duplicate-tolerant pre-contract DDL into
// the static operation body. An existing object is skipped only after its
// definition is shown to be compatible; a same-name mismatch fails before
// Execute can accept the migration operation.
func appendMigrationStatementIfNeeded(
	ctx context.Context,
	sqldb *sql.DB,
	executor sqlContextExecer,
	statement string,
) error {
	if sqldb == nil {
		return fmt.Errorf("migration SQL inspector is not configured")
	}
	if executor == nil {
		return fmt.Errorf("migration SQL executor is not configured")
	}
	parsed, parseErr := sqlparser.Parse(statement)
	if parseErr == nil {
		var definitions []*sqlparser.DDL
		switch parsed := parsed.(type) {
		case *sqlparser.DDL:
			definitions = []*sqlparser.DDL{parsed}
		case *sqlparser.AlterTable:
			definitions = make([]*sqlparser.DDL, 0, len(parsed.Statements))
			for _, definition := range parsed.Statements {
				if definition == nil {
					continue
				}
				copyDefinition := *definition
				if strings.TrimSpace(copyDefinition.Table.Name.String()) == "" {
					copyDefinition.Table = parsed.Table
				}
				definitions = append(definitions, &copyDefinition)
			}
		}
		if len(definitions) != 0 {
			allHandledAndCompatible := true
			for _, definition := range definitions {
				handled, compatible, err := migrationDDLAlreadyCompatible(ctx, sqldb, definition)
				if err != nil {
					return err
				}
				if !handled || !compatible {
					allHandledAndCompatible = false
				}
			}
			if allHandledAndCompatible {
				return nil
			}
		}
	}
	_, err := executor.ExecContext(ctx, statement)
	return err
}

func migrationDDLAlreadyCompatible(
	ctx context.Context,
	sqldb *sql.DB,
	ddl *sqlparser.DDL,
) (handled bool, compatible bool, err error) {
	switch {
	case ddl.Action == sqlparser.CreateStr && ddl.TableSpec != nil:
		compatible, err = migrationTableAlreadyCompatible(ctx, sqldb, ddl)
		return true, compatible, err
	case ddl.IndexSpec != nil && ddl.IndexSpec.Action == sqlparser.CreateStr:
		compatible, err = migrationIndexAlreadyCompatible(ctx, sqldb, ddl)
		return true, compatible, err
	case ddl.Action == sqlparser.AlterStr && ddl.ColumnAction == sqlparser.AddStr && ddl.TableSpec != nil:
		compatible, err = migrationAddedColumnAlreadyCompatible(ctx, sqldb, ddl)
		return true, compatible, err
	default:
		return false, false, nil
	}
}

func migrationTableAlreadyCompatible(ctx context.Context, sqldb *sql.DB, ddl *sqlparser.DDL) (bool, error) {
	tableName := strings.TrimSpace(ddl.Table.Name.String())
	if tableName == "" || ddl.TableSpec == nil {
		return false, fmt.Errorf("%w: CREATE TABLE has no inspectable table definition", ErrMigrationSchemaConflict)
	}
	actualColumns, err := loadMigrationTableColumns(ctx, sqldb, tableName)
	if err != nil {
		return false, fmt.Errorf("inspect migration table %q: %w", tableName, err)
	}
	if len(actualColumns) == 0 {
		return false, nil
	}
	if len(ddl.TableSpec.Constraints) != 0 {
		return false, fmt.Errorf(
			"%w: table %q exists and its embedded constraints cannot be proven equivalent",
			ErrMigrationSchemaConflict,
			tableName,
		)
	}
	if len(actualColumns) != len(ddl.TableSpec.Columns) {
		return false, fmt.Errorf(
			"%w: table %q has %d columns; embedded migration requires %d",
			ErrMigrationSchemaConflict,
			tableName,
			len(actualColumns),
			len(ddl.TableSpec.Columns),
		)
	}
	for index, expected := range ddl.TableSpec.Columns {
		if err := compareMigrationColumn(tableName, expected, actualColumns[index]); err != nil {
			return false, err
		}
	}

	expectedPrimary, err := expectedMigrationTablePrimaryKey(ddl.TableSpec)
	if err != nil {
		return false, fmt.Errorf("%w: table %q primary key: %w", ErrMigrationSchemaConflict, tableName, err)
	}
	actualPrimary, err := loadMigrationIndex(ctx, sqldb, tableName, "PRIMARY")
	if err != nil {
		return false, fmt.Errorf("inspect migration table %q primary key: %w", tableName, err)
	}
	if err := compareMigrationIndex(tableName, "PRIMARY", expectedPrimary, actualPrimary); err != nil {
		return false, err
	}

	// CREATE TABLE can contain named secondary indexes. Validate those required
	// definitions, while allowing unrelated additional indexes; standalone
	// CREATE INDEX statements are validated in their own migration step.
	for _, indexDefinition := range ddl.TableSpec.Indexes {
		if indexDefinition == nil || indexDefinition.Info == nil || indexDefinition.Info.Primary {
			continue
		}
		indexName := strings.TrimSpace(indexDefinition.Info.Name.String())
		if indexName == "" {
			return false, fmt.Errorf(
				"%w: table %q has an unnamed embedded secondary index that cannot be correlated safely",
				ErrMigrationSchemaConflict,
				tableName,
			)
		}
		expected, err := expectedMigrationTableIndex(indexDefinition)
		if err != nil {
			return false, fmt.Errorf("%w: table %q index %q: %w", ErrMigrationSchemaConflict, tableName, indexName, err)
		}
		actual, err := loadMigrationIndex(ctx, sqldb, tableName, indexName)
		if err != nil {
			return false, fmt.Errorf("inspect migration table %q index %q: %w", tableName, indexName, err)
		}
		if err := compareMigrationIndex(tableName, indexName, expected, actual); err != nil {
			return false, err
		}
	}
	return true, nil
}

func migrationIndexAlreadyCompatible(ctx context.Context, sqldb *sql.DB, ddl *sqlparser.DDL) (bool, error) {
	tableName := strings.TrimSpace(ddl.Table.Name.String())
	indexName := strings.TrimSpace(ddl.IndexSpec.ToName.String())
	if tableName == "" || indexName == "" {
		return false, fmt.Errorf(
			"%w: CREATE INDEX has no stable table/index name",
			ErrMigrationSchemaConflict,
		)
	}
	expected, err := expectedMigrationIndexSpec(ddl.IndexSpec)
	if err != nil {
		return false, fmt.Errorf("%w: index %q on table %q: %w", ErrMigrationSchemaConflict, indexName, tableName, err)
	}
	actual, err := loadMigrationIndex(ctx, sqldb, tableName, indexName)
	if err != nil {
		return false, fmt.Errorf("inspect migration index %q on table %q: %w", indexName, tableName, err)
	}
	if !actual.Found {
		return false, nil
	}
	if err := compareMigrationIndex(tableName, indexName, expected, actual); err != nil {
		return false, err
	}
	return true, nil
}

func migrationAddedColumnAlreadyCompatible(ctx context.Context, sqldb *sql.DB, ddl *sqlparser.DDL) (bool, error) {
	if ddl.TableSpec == nil || len(ddl.TableSpec.Columns) != 1 {
		return false, nil
	}
	tableName := strings.TrimSpace(ddl.Table.Name.String())
	expected := ddl.TableSpec.Columns[0]
	columns, err := loadMigrationTableColumns(ctx, sqldb, tableName)
	if err != nil {
		return false, fmt.Errorf("inspect migration table %q: %w", tableName, err)
	}
	for index, actual := range columns {
		if !strings.EqualFold(strings.TrimSpace(actual.Name), strings.TrimSpace(expected.Name.String())) {
			continue
		}
		if err := compareMigrationColumn(tableName, expected, actual); err != nil {
			return false, err
		}
		if ddl.ColumnOrder != nil {
			wantOrdinal := index + 1
			if ddl.ColumnOrder.First {
				wantOrdinal = 1
			} else if after := strings.TrimSpace(ddl.ColumnOrder.AfterColumn.String()); after != "" {
				wantOrdinal = 0
				for _, candidate := range columns {
					if strings.EqualFold(candidate.Name, after) {
						wantOrdinal = candidate.Ordinal + 1
						break
					}
				}
			}
			if wantOrdinal == 0 || actual.Ordinal != wantOrdinal {
				return false, fmt.Errorf(
					"%w: column %q on table %q has ordinal %d; embedded migration requires %d",
					ErrMigrationSchemaConflict,
					expected.Name.String(),
					tableName,
					actual.Ordinal,
					wantOrdinal,
				)
			}
		}
		return true, nil
	}
	return false, nil
}

func loadMigrationTableColumns(ctx context.Context, sqldb *sql.DB, tableName string) ([]migrationColumnSchema, error) {
	rows, err := queryWhenSQLViewReady(ctx, func() (*sql.Rows, error) {
		return sqldb.QueryContext(ctx, `SELECT column_name, column_type, is_nullable, column_default, extra, character_set_name, collation_name, ordinal_position
FROM information_schema.columns
WHERE table_schema = DATABASE() AND LOWER(table_name) = LOWER(?)
ORDER BY ordinal_position`, tableName)
	})
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := make([]migrationColumnSchema, 0)
	for rows.Next() {
		var (
			column   migrationColumnSchema
			nullable string
			extra    sql.NullString
		)
		if err := rows.Scan(
			&column.Name,
			&column.ColumnType,
			&nullable,
			&column.Default,
			&extra,
			&column.Charset,
			&column.Collation,
			&column.Ordinal,
		); err != nil {
			return nil, err
		}
		column.Nullable = strings.EqualFold(strings.TrimSpace(nullable), "YES")
		if extra.Valid {
			column.Extra = extra.String
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

func loadMigrationIndex(ctx context.Context, sqldb *sql.DB, tableName string, indexName string) (migrationIndexSchema, error) {
	rows, err := queryWhenSQLViewReady(ctx, func() (*sql.Rows, error) {
		return sqldb.QueryContext(ctx, `SELECT non_unique, column_name, sub_part, collation, index_type
FROM information_schema.statistics
WHERE table_schema = DATABASE() AND LOWER(table_name) = LOWER(?) AND LOWER(index_name) = LOWER(?)
ORDER BY seq_in_index`, tableName, indexName)
	})
	if err != nil {
		return migrationIndexSchema{}, err
	}
	defer rows.Close()
	actual := migrationIndexSchema{}
	for rows.Next() {
		var (
			nonUnique int64
			column    sql.NullString
			length    sql.NullInt64
			order     sql.NullString
			indexType sql.NullString
		)
		if err := rows.Scan(&nonUnique, &column, &length, &order, &indexType); err != nil {
			return migrationIndexSchema{}, err
		}
		if !column.Valid || strings.TrimSpace(column.String) == "" {
			return migrationIndexSchema{}, fmt.Errorf("functional index metadata is not supported")
		}
		if !actual.Found {
			actual.Found = true
			actual.Unique = nonUnique == 0
			if indexType.Valid {
				actual.IndexType = normalizeMigrationSQL(indexType.String)
			}
		} else if actual.Unique != (nonUnique == 0) {
			return migrationIndexSchema{}, fmt.Errorf("index uniqueness metadata is inconsistent")
		}
		field := migrationIndexFieldSchema{Column: normalizeMigrationSQL(column.String)}
		if length.Valid {
			field.Length = length.Int64
		}
		if order.Valid {
			field.Order = normalizeMigrationIndexOrder(order.String)
		}
		actual.Fields = append(actual.Fields, field)
	}
	if err := rows.Err(); err != nil {
		return migrationIndexSchema{}, err
	}
	return actual, nil
}

func compareMigrationColumn(tableName string, expected *sqlparser.ColumnDefinition, actual migrationColumnSchema) error {
	if expected == nil {
		return fmt.Errorf("%w: table %q has a nil embedded column definition", ErrMigrationSchemaConflict, tableName)
	}
	expectedName := strings.TrimSpace(expected.Name.String())
	if !strings.EqualFold(expectedName, strings.TrimSpace(actual.Name)) {
		return fmt.Errorf(
			"%w: table %q column %d is %q; embedded migration requires %q",
			ErrMigrationSchemaConflict,
			tableName,
			actual.Ordinal,
			actual.Name,
			expectedName,
		)
	}
	expectedType := expectedMigrationColumnType(expected.Type)
	actualType := normalizeMigrationColumnType(actual.ColumnType)
	if expectedType != actualType {
		return fmt.Errorf(
			"%w: column %q on table %q has type %q; embedded migration requires %q",
			ErrMigrationSchemaConflict,
			expectedName,
			tableName,
			actual.ColumnType,
			expectedType,
		)
	}
	expectedPrimary := strings.Contains(" "+normalizeMigrationSQL(sqlparser.String(&expected.Type))+" ", " primary key ")
	expectedNullable := !bool(expected.Type.NotNull) && !expectedPrimary
	if expectedNullable != actual.Nullable {
		return fmt.Errorf(
			"%w: column %q on table %q nullable=%t; embedded migration requires nullable=%t",
			ErrMigrationSchemaConflict,
			expectedName,
			tableName,
			actual.Nullable,
			expectedNullable,
		)
	}
	expectedDefault := ""
	if expected.Type.Default != nil {
		expectedDefault = normalizeMigrationDefault(sqlparser.String(expected.Type.Default))
	}
	actualDefault := ""
	if actual.Default.Valid {
		actualDefault = normalizeMigrationDefault(actual.Default.String)
	}
	if expectedDefault != actualDefault {
		return fmt.Errorf(
			"%w: column %q on table %q has default %q; embedded migration requires %q",
			ErrMigrationSchemaConflict,
			expectedName,
			tableName,
			actualDefault,
			expectedDefault,
		)
	}
	expectedExtra := ""
	if bool(expected.Type.Autoincrement) {
		expectedExtra = "auto_increment"
	}
	actualExtra := normalizeMigrationColumnExtra(actual.Extra)
	if expectedExtra != actualExtra {
		return fmt.Errorf(
			"%w: column %q on table %q has extra attributes %q; embedded migration requires %q",
			ErrMigrationSchemaConflict,
			expectedName,
			tableName,
			actualExtra,
			expectedExtra,
		)
	}
	if expected.Type.Charset != "" && !strings.EqualFold(expected.Type.Charset, actual.Charset.String) {
		return fmt.Errorf("%w: column %q on table %q has charset %q; embedded migration requires %q", ErrMigrationSchemaConflict, expectedName, tableName, actual.Charset.String, expected.Type.Charset)
	}
	if expected.Type.Collate != "" && !strings.EqualFold(expected.Type.Collate, actual.Collation.String) {
		return fmt.Errorf("%w: column %q on table %q has collation %q; embedded migration requires %q", ErrMigrationSchemaConflict, expectedName, tableName, actual.Collation.String, expected.Type.Collate)
	}
	if expected.Type.GeneratedExpr != nil || expected.Type.ForeignKeyDef != nil || expected.Type.Constraint != nil || expected.Type.OnUpdate != nil {
		return fmt.Errorf(
			"%w: column %q on table %q uses embedded attributes that migration preflight cannot prove equivalent",
			ErrMigrationSchemaConflict,
			expectedName,
			tableName,
		)
	}
	return nil
}

func expectedMigrationColumnType(columnType sqlparser.ColumnType) string {
	columnType.Default = nil
	columnType.OnUpdate = nil
	columnType.Comment = nil
	columnType.Null = false
	columnType.NotNull = false
	columnType.Autoincrement = false
	columnType.KeyOpt = 0
	columnType.Charset = ""
	columnType.Collate = ""
	columnType.Constraint = nil
	columnType.ForeignKeyDef = nil
	columnType.GeneratedExpr = nil
	return normalizeMigrationColumnType(sqlparser.String(&columnType))
}

func expectedMigrationTablePrimaryKey(spec *sqlparser.TableSpec) (migrationIndexSchema, error) {
	expected := migrationIndexSchema{Unique: true, IndexType: "btree"}
	for _, column := range spec.Columns {
		if column != nil && strings.Contains(" "+normalizeMigrationSQL(sqlparser.String(&column.Type))+" ", " primary key ") {
			expected.Found = true
			expected.Fields = append(expected.Fields, migrationIndexFieldSchema{Column: normalizeMigrationSQL(column.Name.String()), Order: "asc"})
		}
	}
	for _, index := range spec.Indexes {
		if index == nil || index.Info == nil || !index.Info.Primary {
			continue
		}
		if expected.Found {
			return migrationIndexSchema{}, fmt.Errorf("multiple primary-key definitions")
		}
		expected.Found = true
		fields, err := expectedMigrationIndexFields(index.Fields)
		if err != nil {
			return migrationIndexSchema{}, err
		}
		expected.Fields = fields
	}
	return expected, nil
}

func expectedMigrationTableIndex(index *sqlparser.IndexDefinition) (migrationIndexSchema, error) {
	fields, err := expectedMigrationIndexFields(index.Fields)
	if err != nil {
		return migrationIndexSchema{}, err
	}
	return migrationIndexSchema{
		Found:     true,
		Unique:    index.Info.Unique,
		IndexType: "",
		Fields:    fields,
	}, nil
}

func expectedMigrationIndexSpec(index *sqlparser.IndexSpec) (migrationIndexSchema, error) {
	fields, err := expectedMigrationIndexFields(index.Fields)
	if err != nil {
		return migrationIndexSchema{}, err
	}
	return migrationIndexSchema{
		Found:     true,
		Unique:    strings.EqualFold(strings.TrimSpace(index.Type), sqlparser.UniqueStr),
		IndexType: normalizeMigrationSQL(index.Using.String()),
		Fields:    fields,
	}, nil
}

func expectedMigrationIndexFields(fields []*sqlparser.IndexField) ([]migrationIndexFieldSchema, error) {
	expected := make([]migrationIndexFieldSchema, 0, len(fields))
	for _, field := range fields {
		if field == nil || field.Expression != nil || strings.TrimSpace(field.Column.String()) == "" {
			return nil, fmt.Errorf("functional or unnamed index fields are not supported")
		}
		entry := migrationIndexFieldSchema{
			Column: normalizeMigrationSQL(field.Column.String()),
			Order:  normalizeMigrationIndexOrder(field.Order),
		}
		if field.Length != nil {
			length, err := strconv.ParseInt(string(field.Length.Val), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid index prefix length %q", string(field.Length.Val))
			}
			entry.Length = length
		}
		expected = append(expected, entry)
	}
	return expected, nil
}

func compareMigrationIndex(tableName string, indexName string, expected migrationIndexSchema, actual migrationIndexSchema) error {
	if expected.Found != actual.Found {
		return fmt.Errorf(
			"%w: index %q on table %q presence=%t; embedded migration requires presence=%t",
			ErrMigrationSchemaConflict,
			indexName,
			tableName,
			actual.Found,
			expected.Found,
		)
	}
	if !expected.Found {
		return nil
	}
	if expected.Unique != actual.Unique || len(expected.Fields) != len(actual.Fields) {
		return fmt.Errorf(
			"%w: index %q on table %q is unique=%t fields=%v; embedded migration requires unique=%t fields=%v",
			ErrMigrationSchemaConflict,
			indexName,
			tableName,
			actual.Unique,
			actual.Fields,
			expected.Unique,
			expected.Fields,
		)
	}
	if expected.IndexType != "" && expected.IndexType != actual.IndexType {
		return fmt.Errorf(
			"%w: index %q on table %q has type %q; embedded migration requires %q",
			ErrMigrationSchemaConflict,
			indexName,
			tableName,
			actual.IndexType,
			expected.IndexType,
		)
	}
	for index := range expected.Fields {
		want := expected.Fields[index]
		got := actual.Fields[index]
		if want.Column != got.Column || want.Length != got.Length || (want.Order != "" && want.Order != got.Order) {
			return fmt.Errorf(
				"%w: index %q on table %q has fields %v; embedded migration requires %v",
				ErrMigrationSchemaConflict,
				indexName,
				tableName,
				actual.Fields,
				expected.Fields,
			)
		}
	}
	return nil
}

func normalizeMigrationColumnType(value string) string {
	normalized := normalizeMigrationSQL(value)
	switch normalized {
	case "bool", "boolean":
		return "tinyint(1)"
	case "integer":
		return "int"
	default:
		return normalized
	}
}

func normalizeMigrationDefault(value string) string {
	normalized := normalizeMigrationSQL(value)
	if normalized == "null" {
		return ""
	}
	if len(normalized) >= 2 && ((normalized[0] == '\'' && normalized[len(normalized)-1] == '\'') || (normalized[0] == '"' && normalized[len(normalized)-1] == '"')) {
		return normalized[1 : len(normalized)-1]
	}
	return normalized
}

func normalizeMigrationColumnExtra(value string) string {
	fields := strings.Fields(strings.ReplaceAll(normalizeMigrationSQL(value), "default_generated", ""))
	return strings.Join(fields, " ")
}

func normalizeMigrationIndexOrder(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "a", "asc":
		return "asc"
	case "d", "desc":
		return "desc"
	default:
		return normalizeMigrationSQL(value)
	}
}

func normalizeMigrationSQL(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func migrationApplied(ctx context.Context, sqldb *sql.DB, filename string, expectedChecksum string) (bool, error) {
	var (
		checksum string
		success  bool
	)
	rows, err := queryWhenSQLViewReady(ctx, func() (*sql.Rows, error) {
		return sqldb.QueryContext(ctx, "SELECT checksum, success FROM sqddl_history WHERE filename = ?", filename)
	})
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "sqddl_history") &&
			(strings.Contains(lower, "not found") || strings.Contains(lower, "does not exist") || strings.Contains(lower, "doesn't exist") || strings.Contains(lower, "unknown table")) {
			return false, nil
		}
		return false, fmt.Errorf("read migration history for %s: %w", filename, err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return false, fmt.Errorf("read migration history for %s: %w", filename, err)
		}
		return false, nil
	}
	if err := rows.Scan(&checksum, &success); err != nil {
		return false, fmt.Errorf("read migration history for %s: %w", filename, err)
	}
	if !strings.EqualFold(strings.TrimSpace(checksum), strings.TrimSpace(expectedChecksum)) {
		return false, fmt.Errorf(
			"migration checksum drift for %s: history=%q embedded=%q",
			filename,
			checksum,
			expectedChecksum,
		)
	}
	return success, nil
}

// startAutomaticBootstrapRetry resumes only an already-marked interrupted
// bootstrap. Open calls it exclusively after the public SDK returns the typed
// ErrBootstrapNotReady result for an existing repository. It never creates a
// foundation repository and never treats provider unavailability as absence.
func (db *DB) startAutomaticBootstrapRetry() {
	if db == nil {
		return
	}
	db.bootstrapRetryMu.Lock()
	if db.bootstrapRetryCancel != nil {
		db.bootstrapRetryMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	db.bootstrapRetryCancel = cancel
	db.bootstrapRetryDone = done
	db.bootstrapRetryErr = nil
	db.bootstrapRetryMu.Unlock()

	go func() {
		err := db.runAutomaticBootstrapRetry(ctx)
		if ctxErr := ctx.Err(); ctxErr != nil && errors.Is(err, ctxErr) {
			// Closing the backend deliberately cancels this worker. Keep that
			// lifecycle outcome distinct from a permanent bootstrap failure.
			err = nil
		}
		db.bootstrapRetryMu.Lock()
		if db.bootstrapRetryDone == done {
			db.bootstrapRetryCancel = nil
			db.bootstrapRetryDone = nil
			db.bootstrapRetryErr = err
		}
		db.bootstrapRetryMu.Unlock()
		close(done)
		if err != nil {
			notifyLog.Errorf("automatic interrupted-bootstrap recovery stopped: %s", err.Error())
		}
	}()
}

func (db *DB) stopAutomaticBootstrapRetry() {
	if db == nil {
		return
	}
	db.bootstrapRetryMu.Lock()
	cancel := db.bootstrapRetryCancel
	done := db.bootstrapRetryDone
	db.bootstrapRetryMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

// AutomaticBootstrapError reports a permanent error from autonomous recovery.
// A nil result means recovery is still pending, completed, or was never needed.
// Provider timeouts remain pending typed bootstrap outcomes and are not stored
// here as proof of absence.
func (db *DB) AutomaticBootstrapError() error {
	if db == nil {
		return nil
	}
	db.bootstrapRetryMu.Lock()
	defer db.bootstrapRetryMu.Unlock()
	return db.bootstrapRetryErr
}

func (db *DB) runAutomaticBootstrapRetry(ctx context.Context) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	link := db.link
	db.mu.Unlock()
	if link == nil {
		return fmt.Errorf("swarmion transport link is nil")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		snapshot, subscription, err := link.SubscribeRoutes(ctx)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return fmt.Errorf("subscribe to application-owned routes for interrupted bootstrap: %w", err)
		}
		err = db.retryInterruptedBootstrapFromRoutes(ctx, snapshot, subscription)
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		closeErr := subscription.Close(closeCtx)
		cancel()
		if closeErr != nil && !errors.Is(closeErr, swarmiontransport.ErrClosedLink) {
			notifyLog.Warnf("close interrupted-bootstrap route subscription: %s", closeErr.Error())
		}
		if errors.Is(err, swarmiontransport.ErrObservationOverflow) {
			continue
		}
		return err
	}
}

func (db *DB) retryInterruptedBootstrapFromRoutes(
	ctx context.Context,
	snapshot swarmiontransport.RouteSnapshot,
	subscription swarmiontransport.RouteSubscription,
) error {
	routes := make(map[swarmiontransport.PeerID]swarmiontransport.RouteState, len(snapshot.Routes))
	for peerID, route := range snapshot.Routes {
		routes[peerID] = route
	}
	backoff := automaticBootstrapRetryBackoff
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		candidates := routedBootstrapCandidates(db.link.LocalPeer(), routes)
		if len(candidates) > 0 {
			attemptCtx, cancel := context.WithTimeout(ctx, initFromPeerRetryBudget)
			err := db.openAndFinalizeSwarmion(attemptCtx, candidates)
			cancel()
			if err == nil {
				db.triggerAllTableChangeCallbacks()
				return nil
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if !errors.Is(err, swarmionapp.ErrBootstrapNotReady) {
				return fmt.Errorf("retry interrupted bootstrap: %w", err)
			}
			logBootstrapNotReady("routed peers were not yet ready to resume interrupted bootstrap", err)
		}

		waitCtx := ctx
		waitCancel := func() {}
		if len(candidates) > 0 {
			waitCtx, waitCancel = context.WithTimeout(ctx, backoff)
		}
		event, err := subscription.Next(waitCtx)
		waitCancel()
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			if len(candidates) > 0 && errors.Is(err, context.DeadlineExceeded) {
				backoff *= 2
				if backoff > automaticBootstrapRetryMaxBackoff {
					backoff = automaticBootstrapRetryMaxBackoff
				}
				continue
			}
			return err
		}
		peerID := event.Peer
		if peerID == "" {
			peerID = event.Route.Peer
		}
		if peerID == "" {
			continue
		}
		if event.Reachable {
			route := event.Route
			if route.Peer == "" {
				route.Peer = peerID
			}
			routes[peerID] = route
			backoff = automaticBootstrapRetryBackoff
		} else {
			delete(routes, peerID)
		}
	}
}

func routedBootstrapCandidates(
	local swarmiontransport.PeerID,
	routes map[swarmiontransport.PeerID]swarmiontransport.RouteState,
) []string {
	peers := make([]string, 0, len(routes))
	for peerID, route := range routes {
		if route.Peer != "" {
			peerID = route.Peer
		}
		peer := strings.TrimSpace(string(peerID))
		if peer == "" || peer == strings.TrimSpace(string(local)) {
			continue
		}
		peers = append(peers, peer)
	}
	sort.Strings(peers)
	result := make([]string, 0, len(peers))
	last := ""
	for _, peer := range peers {
		if peer == last {
			continue
		}
		last = peer
		// A p2p-only candidate deliberately supplies no physical endpoint.
		// It is valid only because the application Link reported an already
		// usable route; Swarmion's EnsureRoute call then becomes a no-op.
		result = append(result, "/p2p/"+peer)
	}
	return result
}

func (db *DB) Close() error {
	if db == nil {
		return nil
	}
	db.stopAutomaticBootstrapRetry()
	db.openMu.Lock()
	defer db.openMu.Unlock()
	return db.closeSwarmionRuntimeLocked()
}

// closeSwarmionRuntimeLocked closes only the runtime scoped to this DB while
// openMu is held. It deliberately does not stop or wait for automatic
// bootstrap recovery, which lets that goroutine fail closed after a migration
// error without waiting on itself. The borrowed application Link and physical
// host are never closed here.
func (db *DB) closeSwarmionRuntimeLocked() error {
	db.opMu.Lock()

	db.mu.Lock()
	host := db.host
	sqldb := db.sqldb
	watchCancel := db.watchCancel
	db.host = nil
	db.runtime = nil
	db.runtimeOpenedAt = time.Time{}
	db.sqldb = nil
	db.initialized = false
	db.watchCancel = nil
	db.mu.Unlock()
	// Seal while opMu is still held. An Execute that completed before this close
	// may attempt to schedule journal cleanup only after releasing opMu; that
	// admission must observe the sealed registry before close starts waiting.
	db.backgroundWork.seal()
	db.opMu.Unlock()
	if watchCancel != nil {
		watchCancel()
	}
	var closeErr error
	if sqldb != nil {
		closeErr = sqldb.Close()
	}
	if host != nil {
		closeErr = errors.Join(closeErr, host.Close())
	}
	db.backgroundWork.wait()
	return closeErr
}

func (db *DB) Initialized() bool {
	if db == nil {
		return false
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.initialized
}

// RepositoryReadiness reports whether fresh initialization is valid. An
// interrupted existing repository remains existing while automatic bootstrap
// is pending, and a permanent bootstrap error is reported separately rather
// than being mistaken for a fresh database.
func (db *DB) RepositoryReadiness() RepositoryReadiness {
	if db == nil {
		return RepositoryReadiness{}
	}
	db.mu.Lock()
	readiness := RepositoryReadiness{
		Initialized:        db.initialized,
		ExistingRepository: db.existingRepository,
	}
	db.mu.Unlock()
	db.bootstrapRetryMu.Lock()
	readiness.BootstrapPending = readiness.ExistingRepository && !readiness.Initialized && db.bootstrapRetryCancel != nil
	readiness.BootstrapError = db.bootstrapRetryErr
	db.bootstrapRetryMu.Unlock()
	return readiness
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
		openCtx, openCancel := context.WithCancel(attemptCtx)
		err := db.openAndFinalizeSwarmion(openCtx, bootstrapPeers)
		openCancel()
		if err == nil {
			db.triggerAllTableChangeCallbacks()
			if attempt > 1 {
				notifyLog.Infof("initialized swarmion db from peer %s after %d attempts", peerID, attempt)
			}
			return nil
		}
		lastErr = err
		if !errors.Is(err, swarmionapp.ErrBootstrapNotReady) {
			return fmt.Errorf("failed to initialize swarmion db from peer %s: %w", peerID, err)
		}
		logBootstrapNotReady(fmt.Sprintf("retryable swarmion bootstrap failure from peer %s on attempt %d", peerID, attempt), err)
		if attemptCtx.Err() != nil {
			return fmt.Errorf("failed to initialize swarmion db from peer %s after %d attempts: %w", peerID, attempt, lastErr)
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

func logBootstrapNotReady(message string, err error) {
	var notReady *swarmionapp.BootstrapNotReadyError
	if errors.As(err, &notReady) && notReady != nil {
		notifyLog.Warnf(
			"%s stage=%s missing_roots=%d missing_events=%d providers=%d",
			message,
			notReady.Stage,
			notReady.MissingRootCount,
			notReady.MissingEventCount,
			notReady.ProviderCount,
		)
		return
	}
	notifyLog.Warnf("%s", message)
}

func (db *DB) EnableGRPCServers(*grpc.Server) error {
	return nil
}

func (db *DB) AddPeer(string, *grpc.ClientConn) error {
	return nil
}

func (db *DB) PrepareSwarmionShutdown(ctx context.Context) error {
	if db == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Host.Close is the Swarmion session drain boundary. It unregisters only
	// Swarmion protocols and leaves the borrowed application transport alive.
	return db.Close()
}

func (db *DB) SwarmionStatus() (swarmionapp.Status, bool) {
	if db == nil {
		return swarmionapp.Status{}, false
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return swarmionapp.Status{}, false
	}
	status, err := app.Diagnostics().Status()
	return status, err == nil
}

// SwarmionRuntimeGeneration identifies the currently open runtime instance.
// It changes monotonically on every successful close/open replacement and is
// used only to bind application side-effect leases to the drain runtime that
// established their prerequisites.
func (db *DB) SwarmionRuntimeGeneration() (uint64, bool) {
	if db == nil {
		return 0, false
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.runtimeGeneration, db.runtime != nil && db.runtimeGeneration != 0
}

// SwarmionDetailedStatus returns the detached high-cardinality diagnostic
// snapshot. It is observational only and never grants operation authority.
func (db *DB) SwarmionDetailedStatus() (swarmionapp.DetailedStatus, bool) {
	if db == nil {
		return swarmionapp.DetailedStatus{}, false
	}
	db.mu.Lock()
	runtime := db.runtime
	db.mu.Unlock()
	if runtime == nil {
		return swarmionapp.DetailedStatus{}, false
	}
	status, err := runtime.Diagnostics().DetailedStatus()
	return status, err == nil
}

// RuntimeSnapshot returns a copy of Swarmion's current protocol state for
// diagnostics and regression tests without exposing its internal App.
func (db *DB) RuntimeSnapshot() (*swarmionprotocol.NodeState, bool) {
	if db == nil {
		return nil, false
	}
	db.mu.Lock()
	runtime := db.runtime
	db.mu.Unlock()
	if runtime == nil {
		return nil, false
	}
	snapshot, err := runtime.Diagnostics().Snapshot()
	return snapshot, err == nil
}

// EventReceiptState is the backend-facing lifecycle of one exact
// EventID/published-root pair. AppliedDurably is the operation protocol
// boundary; historical-root content coverage remains a separate field on the
// observation's Swarmion status.
type EventReceiptState string

const (
	EventReceiptStatePending          EventReceiptState = "pending"
	EventReceiptStateParkedConflict   EventReceiptState = "parked_conflict"
	EventReceiptStateDependencyParked EventReceiptState = "dependency_parked"
	EventReceiptStateStaleAnchor      EventReceiptState = "stale_anchor"
	EventReceiptStateAppliedDurably   EventReceiptState = "applied_durably"
)

// EventReceiptObservation pairs Swarmion's exact event status with the
// backend operation lifecycle. Receipt includes any exact checkpoint
// commit/root learned by this read. RootStatus is intentionally absent:
// root-level parking is ambiguous when multiple events publish one root and
// remains a separate diagnostic-only API.
type EventReceiptObservation struct {
	Receipt PublishedWriteReceipt
	Status  swarmionapp.ReceiptStatus
	State   EventReceiptState
}

// EventReceiptPendingError preserves the last exact observation when a
// bounded wait expires. Callers may persist Observation and resume tracking
// the same receipt without publishing the operation again.
type EventReceiptPendingError struct {
	Observation EventReceiptObservation
	Reason      string
	Cause       error
}

func (e *EventReceiptPendingError) Error() string {
	if e == nil {
		return ErrEventReceiptPending.Error()
	}
	status := e.Observation.Status
	message := fmt.Sprintf(
		"%s: reason=%q event_id=%s published_root=%s known=%t checkpointed=%t checkpoint=%s/%s durable_head=%s/%s content_coverage=%s",
		ErrEventReceiptPending,
		e.Reason,
		e.Observation.Receipt.EventID,
		e.Observation.Receipt.PublishedRootHash,
		status.Known,
		status.Checkpointed,
		status.CheckpointCommitID,
		status.CheckpointRootHash,
		status.DurableCheckpointCommitID,
		status.DurableCheckpointRootHash,
		status.ContentCoverage,
	)
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *EventReceiptPendingError) Is(target error) bool {
	return target == ErrEventReceiptPending
}

func (e *EventReceiptPendingError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// EventReceiptParkedError reports an exact-event parked classification. The
// classification is head-relative and revisitable, so callers must apply
// their existing conflict/retry policy without automatically republishing.
type EventReceiptParkedError struct {
	Observation EventReceiptObservation
	Reason      string
}

func (e *EventReceiptParkedError) Error() string {
	if e == nil {
		return ErrEventReceiptParked.Error()
	}
	return fmt.Sprintf(
		"%s: reason=%q event_id=%s published_root=%s state=%s parked_reason=%s revisitable=%t",
		ErrEventReceiptParked,
		e.Reason,
		e.Observation.Receipt.EventID,
		e.Observation.Receipt.PublishedRootHash,
		e.Observation.State,
		e.Observation.Status.ParkedReason,
		e.Observation.Status.Revisitable,
	)
}

func (e *EventReceiptParkedError) Is(target error) bool {
	return target == ErrEventReceiptParked
}

// EventReceiptMetrics contains backend observations that are operationally
// important to surface separately from Swarmion's own runtime counters.
type EventReceiptMetrics struct {
	ContentDissentObservations uint64
}

func (db *DB) EventReceiptMetrics() EventReceiptMetrics {
	if db == nil {
		return EventReceiptMetrics{}
	}
	return EventReceiptMetrics{
		ContentDissentObservations: db.eventReceiptContentDissentObservations.Load(),
	}
}

// SwarmionRootStatus returns the current, revisitable lifecycle for a
// published root. It is a snapshot/content diagnostic, not an operation-level
// durability receipt. Use SwarmionEventReceiptStatus or
// WaitForPublishedWriteApplied for persistence-sensitive operations.
func (db *DB) SwarmionRootStatus(ctx context.Context, rootHash string) (swarmionapp.BranchRootStatus, error) {
	if db == nil {
		return swarmionapp.BranchRootStatus{}, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return swarmionapp.BranchRootStatus{}, fmt.Errorf("db is not initialized")
	}
	return app.Diagnostics().RootStatus(ctx, swarmionapp.BranchRootStatusRequest{RootHash: rootHash})
}

// SwarmionEventReceiptStatus reports checkpoint application and historical
// content coverage for one exact EventID/published-root identity pair.
func (db *DB) SwarmionEventReceiptStatus(ctx context.Context, eventID string, publishedRootHash string) (swarmionapp.ReceiptStatus, error) {
	if db == nil {
		return swarmionapp.ReceiptStatus{}, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	expectedEventID, expectedRoot, err := validateEventReceiptIdentity(eventID, publishedRootHash)
	if err != nil {
		return swarmionapp.ReceiptStatus{}, err
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return swarmionapp.ReceiptStatus{}, fmt.Errorf("db is not initialized")
	}
	receipt := swarmionapp.EventReceipt{
		EventID:           expectedEventID.String(),
		PublishedRootHash: expectedRoot.String(),
	}
	tracking := swarmionapp.ReceiptTrackingRequest{Receipt: receipt}
	snapshot, err := app.ObserveReceipt(ctx, tracking)
	if err != nil {
		return swarmionapp.ReceiptStatus{}, err
	}
	if err := snapshot.ValidateFor(tracking); err != nil {
		return swarmionapp.ReceiptStatus{}, fmt.Errorf("validate swarmion receipt snapshot: %w", err)
	}
	status := snapshot.Event
	if err := validateEventReceiptStatus(expectedEventID, expectedRoot, status); err != nil {
		return swarmionapp.ReceiptStatus{}, err
	}
	return status, nil
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
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "protos checkpoint read"
	}
	if err := catchUpSwarmionCheckpoint(ctx, app.Maintenance(), reason); err != nil {
		return fmt.Errorf("catch up swarmion checkpoint view: %w", err)
	}
	return nil
}

func IsRetryableCheckpointCatchUp(err error) bool {
	return errors.Is(err, errSwarmionCheckpointCatchUpRetryable)
}

func catchUpSwarmionCheckpoint(ctx context.Context, app interface {
	ReconcileCheckpoint(context.Context, swarmionapp.CheckpointReconcileRequest) (swarmionapp.CheckpointReconcileResult, error)
}, reason string) error {
	if app == nil {
		return nil
	}
	// A target_changed response describes only this attempt. Callers must
	// re-query their exact receipt/status before deciding whether a fresh
	// catch-up request is still appropriate; this helper never starts an
	// automatic catch-up loop.
	return catchUpSwarmionCheckpointOnce(ctx, app, reason)
}

func catchUpSwarmionCheckpointOnce(ctx context.Context, app interface {
	ReconcileCheckpoint(context.Context, swarmionapp.CheckpointReconcileRequest) (swarmionapp.CheckpointReconcileResult, error)
}, reason string) error {
	result, err := app.ReconcileCheckpoint(ctx, swarmionapp.CheckpointReconcileRequest{Reason: reason})
	if opErr := checkpointReconcileOperationalError(result); opErr != nil {
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

func checkpointReconcileOperationalError(result swarmionapp.CheckpointReconcileResult) error {
	if err := result.Validate(); err != nil {
		return fmt.Errorf("checkpoint reconciliation returned an invalid result: %w", err)
	}
	switch result.State {
	case swarmionapp.CheckpointReconcileBlockedFatal:
		return fmt.Errorf("checkpoint catch-up blocked by fatal protocol state: %s", checkpointReconcileReason(result))
	case swarmionapp.CheckpointReconcileFailed:
		return fmt.Errorf("checkpoint catch-up failed: %s", checkpointReconcileReason(result))
	case swarmionapp.CheckpointReconcileTargetChanged, swarmionapp.CheckpointReconcileRetryable:
		return fmt.Errorf("%w: status=%s reason=%s", errSwarmionCheckpointCatchUpRetryable, result.State, checkpointReconcileReason(result))
	case swarmionapp.CheckpointReconcileNoTarget,
		swarmionapp.CheckpointReconcileNoSnapshot,
		swarmionapp.CheckpointReconcileAlreadyCurrent,
		swarmionapp.CheckpointReconcileComplete:
		return nil
	default:
		return fmt.Errorf("checkpoint catch-up returned unknown state %q", result.State)
	}
}

func checkpointReconcileReason(result swarmionapp.CheckpointReconcileResult) string {
	for _, value := range []string{result.BlockingReason, result.Detail} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	if result.State != "" {
		return string(result.State)
	}
	return "no details"
}

func (db *DB) SwarmionCompatibility(ctx context.Context) ([]swarmionapp.ManifestCompatibility, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return nil, fmt.Errorf("swarmion app is not initialized")
	}
	return app.Diagnostics().Compatibility(ctx)
}

func (db *DB) SwarmionPeerStatus(ctx context.Context) ([]swarmionapp.PeerStatus, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return nil, fmt.Errorf("swarmion database runtime is not initialized")
	}
	return app.Diagnostics().PeerStatus(ctx)
}

func (db *DB) SwarmionContentSyncTrace() ([]string, bool) {
	if db == nil {
		return nil, false
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return nil, false
	}
	trace, err := app.Diagnostics().ContentSyncTrace()
	return trace, err == nil
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
	receipt, err := db.executeOrdinaryPublishedWriteContext(
		ctx,
		"direct SQL",
		false,
		true,
		[]preparedWriteStatement{{SQL: query, Args: append([]any(nil), args...)}},
	)
	if err != nil {
		return nil, err
	}
	if !receipt.HasExactEventIdentity() {
		return nil, fmt.Errorf("direct SQL mutation returned no exact receipt")
	}
	return operationStatementResult{}, nil
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
	rows, err := queryWhenSQLViewReady(ctx, func() (*sql.Rows, error) {
		return sqldb.QueryContext(ctx, query, args...)
	})
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

// ReadRowsAsOf runs a read against one immutable Dolt checkpoint commit. The
// query must contain `AS OF ?`, and that marker must be its first bind
// parameter. The validated immutable checkpoint ID is rendered as an SQL
// literal before execution because Dolt resolves AS OF revisions during query
// planning rather than as an ordinary value expression. Remaining application
// values stay parameterized. This lets callers pair a business-invariant read
// with ReceiptStatus.DurableCheckpointCommitID instead of racing the live
// tentative SQL overlay.
func (db *DB) ReadRowsAsOf(ctx context.Context, checkpointCommitID string, query string, args []any, consume func(*sql.Rows) error) error {
	checkpointCommitID = strings.TrimSpace(checkpointCommitID)
	if swarmionprotocol.ParseCheckpointCommitID(checkpointCommitID).IsZero() {
		return fmt.Errorf("durable checkpoint commit ID is empty")
	}
	normalized := strings.ToLower(query)
	asOfIndex := strings.Index(normalized, " as of ?")
	if asOfIndex < 0 {
		return fmt.Errorf("as-of query must contain AS OF ?")
	}
	if firstBind := strings.Index(query, "?"); firstBind != asOfIndex+len(" as of ") {
		return fmt.Errorf("AS OF ? must be the first query bind parameter")
	}
	if strings.ContainsAny(checkpointCommitID, "'\\") {
		return fmt.Errorf("durable checkpoint commit ID contains invalid SQL literal characters")
	}
	query = query[:asOfIndex] + " AS OF '" + checkpointCommitID + "'" + query[asOfIndex+len(" as of ?"):]
	return db.ReadRows(ctx, query, args, consume)
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
	return queryWhenSQLViewReady(ctx, func() (*sql.Rows, error) {
		return sqldb.QueryContext(ctx, query, args...)
	})
}

func (q lockedSQLDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if q.db == nil {
		return nil, fmt.Errorf("db is not initialized")
	}
	return q.db.ExecContext(ctx, query, args...)
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

// PublishedWriteReceipt identifies the exact event/root pair accepted by
// Swarmion. EventID, not PublishedRootHash alone, is the operation identity.
// CheckpointCommitID and CheckpointRootHash are populated by a later exact
// event observation and may be persisted for restart recovery.
type PublishedWriteReceipt struct {
	Committed             bool   `json:"committed,omitempty"`
	OutcomeUncertain      bool   `json:"outcome_uncertain,omitempty"`
	Checkpointed          bool   `json:"checkpointed,omitempty"`
	EventID               string `json:"event_id"`
	PublishedRootHash     string `json:"published_root_hash"`
	AuthorPeerID          string `json:"author_peer_id,omitempty"`
	OperationIntentDigest string `json:"operation_intent_digest,omitempty"`
	CheckpointCommitID    string `json:"checkpoint_commit_id,omitempty"`
	CheckpointRootHash    string `json:"checkpoint_root_hash,omitempty"`
}

// PublishedWriteReceiptIdentityConflictError preserves the uncertain commit's
// exact receipt when operation lookup resolves a different event/root identity.
// This is a fail-closed protocol/API invariant error and never grants replay.
type PublishedWriteReceiptIdentityConflictError struct {
	Receipt                   PublishedWriteReceipt
	ResolvedEventID           string
	ResolvedPublishedRootHash string
	Cause                     error
}

func (e *PublishedWriteReceiptIdentityConflictError) Error() string {
	if e == nil {
		return ErrPublishedWriteReceiptIdentityConflict.Error()
	}
	message := fmt.Sprintf(
		"%s: uncertain=%s/%s resolved=%s/%s",
		ErrPublishedWriteReceiptIdentityConflict,
		e.Receipt.EventID,
		e.Receipt.PublishedRootHash,
		e.ResolvedEventID,
		e.ResolvedPublishedRootHash,
	)
	if e.Cause != nil {
		message += ": " + e.Cause.Error()
	}
	return message
}

func (e *PublishedWriteReceiptIdentityConflictError) Is(target error) bool {
	return target == ErrPublishedWriteReceiptIdentityConflict
}

func (e *PublishedWriteReceiptIdentityConflictError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// HasExactEventIdentity reports whether the receipt contains the inseparable
// EventID/published-root address needed for status tracking. An uncertain
// commit outcome can carry this address before publication is proven; callers
// must track it and must not replay the mutation merely because Committed is
// still false.
func (r PublishedWriteReceipt) HasExactEventIdentity() bool {
	_, _, err := validateEventReceiptIdentity(r.EventID, r.PublishedRootHash)
	return err == nil
}

// LookupPublishedWriteOperation resolves the original exact receipt before a
// caller considers running a logical write. The returned disposition is the
// only control-flow authority; Diagnostic remains non-authorizing detail.
func (db *DB) LookupPublishedWriteOperation(ctx context.Context, operation PublishedWriteOperation) (swarmionapp.OperationResult, error) {
	return db.resolvePublishedWriteOperation(ctx, operation, false)
}

// WaitPublishedWriteOperation waits only for replicated operation evidence to
// arrive. It never executes SQL or republishes the operation. Foreign history
// that remains incomplete is returned as unavailable when ctx ends.
func (db *DB) WaitPublishedWriteOperation(ctx context.Context, operation PublishedWriteOperation) (swarmionapp.OperationResult, error) {
	return db.resolvePublishedWriteOperation(ctx, operation, true)
}

// WaitSQLViewReady performs Swarmion's connection-free readiness wait. It is
// safe to use between rejected statement attempts and never executes SQL.
func (db *DB) WaitSQLViewReady(ctx context.Context) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if db.waitSQLViewReadyForTest != nil {
		return db.waitSQLViewReadyForTest(ctx)
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return fmt.Errorf("db is not initialized")
	}
	return app.WaitSQLViewReady(ctx)
}

// WaitMutationReady waits until Swarmion's current SQL projection can be
// represented by one exact write event. It is a preflight barrier only: it
// never retries or publishes a backend mutation.
func (db *DB) WaitMutationReady(ctx context.Context) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if db.waitMutationReadyForTest != nil {
		return db.waitMutationReadyForTest(ctx)
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return fmt.Errorf("db is not initialized")
	}
	return app.WaitMutationReady(ctx)
}

// WithPublishedWriteAbsenceLease runs one non-database side effect only while
// the current runtime directly proves operation absent. The database operation
// lock keeps that exact runtime owner live until action returns, so Close or a
// replacement runtime cannot revoke and rebind the proof mid-effect. action
// must not call back into DB methods because they share the same operation
// lock. An already accepted operation returns its exact receipt without running
// action.
func (db *DB) WithPublishedWriteAbsenceLease(
	ctx context.Context,
	operation PublishedWriteOperation,
	expectedRuntimeGeneration uint64,
	name string,
	action func() error,
) (PublishedWriteReceipt, bool, error) {
	if db == nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if action == nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("published write absence lease %q has no action", name)
	}
	if err := operation.Validate(); err != nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("published write absence lease %q: %w", name, err)
	}
	if err := db.persistPublishedWriteOperation(operation); err != nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("persist published write absence lease %q: %w", name, err)
	}
	if err := db.opMu.LockContext(ctx); err != nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("lock published write absence lease %q: %w", name, err)
	}
	defer db.opMu.Unlock()

	db.mu.Lock()
	runtime := db.runtime
	runtimeGeneration := db.runtimeGeneration
	db.mu.Unlock()
	if runtime == nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("%w: published write absence lease %q has no live runtime", ErrOperationReceiptUnavailable, name)
	}
	if expectedRuntimeGeneration != 0 && runtimeGeneration != expectedRuntimeGeneration {
		return PublishedWriteReceipt{}, false, fmt.Errorf(
			"%w: published write absence lease %q runtime generation changed from %d to %d",
			ErrOperationReceiptUnavailable,
			name,
			expectedRuntimeGeneration,
			runtimeGeneration,
		)
	}
	if localAuthor := strings.TrimSpace(runtime.PeerID()); localAuthor == "" || localAuthor != strings.TrimSpace(operation.AuthorPeerID()) {
		return PublishedWriteReceipt{}, false, fmt.Errorf(
			"%w: published write absence lease %q author=%q runtime_author=%q",
			ErrOperationReceiptUnavailable,
			name,
			operation.AuthorPeerID(),
			localAuthor,
		)
	}
	result, err := db.resolvePublishedWriteOnRuntimeLocked(ctx, runtime, operation, "side-effect preflight")
	if err != nil {
		return PublishedWriteReceipt{}, false, fmt.Errorf("resolve published write absence lease %q: %w", name, err)
	}
	switch result.Disposition() {
	case swarmionapp.OperationAccepted:
		receipt, err := publishedWriteReceiptFromAcceptedResult(operation, result)
		return receipt, true, err
	case swarmionapp.OperationRetryPermitted:
		reason, ok := result.RetryReason()
		if !ok || reason != swarmionapp.RetryResolvedAbsent {
			return PublishedWriteReceipt{}, false, fmt.Errorf(
				"%w: published write absence lease %q returned retry reason %q",
				ErrOperationReceiptUnavailable,
				name,
				reason,
			)
		}
		// Re-read the dynamic result at the last instruction before the effect.
		// DB Close cannot begin while opMu is held, but this also fails closed if
		// the runtime owner was closed directly by an internal test or fault path.
		if result.Disposition() != swarmionapp.OperationRetryPermitted {
			return PublishedWriteReceipt{}, false, fmt.Errorf("%w: published write absence lease %q was revoked before its action", ErrOperationReceiptUnavailable, name)
		}
		if err := action(); err != nil {
			return PublishedWriteReceipt{}, false, err
		}
		return PublishedWriteReceipt{}, false, nil
	case swarmionapp.OperationRecoveryRequired:
		return PublishedWriteReceipt{}, false, fmt.Errorf("%w: published write absence lease %q requires recovery", ErrOperationReceiptUnavailable, name)
	case swarmionapp.OperationFailedClosed:
		// Diagnostic is descriptive and must not add authority-bearing nodes to
		// this application error graph.
		return PublishedWriteReceipt{}, false, fmt.Errorf("published write absence lease %q failed closed: %s", name, operationDiagnosticText(result.Diagnostic()))
	default:
		return PublishedWriteReceipt{}, false, fmt.Errorf("published write absence lease %q returned unknown disposition %q", name, result.Disposition())
	}
}

// PublishedWriteReceiptFromResult extracts an exact accepted receipt from the
// normalized backend result. Recovery-required and retryable results are never
// promoted to accepted by diagnostic inspection.
func PublishedWriteReceiptFromResult(operation PublishedWriteOperation, result swarmionapp.OperationResult) (PublishedWriteReceipt, error) {
	return publishedWriteReceiptFromAcceptedResult(operation, result)
}

type sqlContextExecer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type operationStatementCollector struct {
	statements []swarmionapp.Statement
}

type operationStatementResult struct{}

func (operationStatementResult) LastInsertId() (int64, error) { return 0, nil }
func (operationStatementResult) RowsAffected() (int64, error) { return 0, nil }

func (c *operationStatementCollector) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.statements = append(c.statements, swarmionapp.Statement{
		Query: query,
		Args:  append([]any(nil), args...),
	})
	return operationStatementResult{}, nil
}

func (db *DB) resolvePublishedWriteOnRuntimeLocked(
	ctx context.Context,
	runtime *swarmionapp.DatabaseRuntime,
	operation PublishedWriteOperation,
	source string,
) (swarmionapp.OperationResult, error) {
	if runtime == nil {
		return swarmionapp.OperationResult{}, ErrOperationReceiptUnavailable
	}
	result := runtime.ResolveOperation(ctx, operation.Recovery)
	recovery, err := recoveryFromOperationResult(operation, result)
	if err != nil {
		if diagnostic := result.Diagnostic(); diagnostic != nil {
			notifyLog.Warnf("operation resolution returned malformed recovery operation=%s diagnostic=%s", operation.Key(), diagnostic.Error())
		}
		return swarmionapp.OperationResult{}, err
	}
	if err := db.enrichOperationRecoveryFromResult(operation, result, recovery, source); err != nil {
		return result, err
	}
	return result, nil
}

func (db *DB) executePublishedWriteAttemptContext(
	ctx context.Context,
	operation PublishedWriteOperation,
	name string,
	allowNoop bool,
	requireReceipt bool,
	expectedRuntimeGeneration uint64,
	retryGuard *publishedWriteRetryGuard,
	apply func(context.Context, sqlContextExecer) error,
) (swarmionapp.OperationResult, bool, *swarmionapp.DatabaseRuntime, error) {
	if db == nil {
		return swarmionapp.OperationResult{}, false, nil, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if apply == nil {
		return swarmionapp.OperationResult{}, false, nil, fmt.Errorf("published write operation %q has no SQL body", name)
	}
	if err := operation.Validate(); err != nil {
		return swarmionapp.OperationResult{}, false, nil, fmt.Errorf("published write operation %q: %w", name, err)
	}
	// This write is mandatory even for callers that already persisted the
	// operation in replicated business state. It closes the ordinary-SQL gap and
	// guarantees a process-local exact recovery record before runtime admission.
	if err := db.persistPublishedWriteOperation(operation); err != nil {
		return swarmionapp.OperationResult{}, false, nil, fmt.Errorf("persist published write operation %q before execution: %w", name, err)
	}

	lockStart := time.Now()
	if err := db.opMu.LockContext(ctx); err != nil {
		return swarmionapp.OperationResult{}, false, nil, fmt.Errorf("lock published write operation %q: %w", name, err)
	}
	defer db.opMu.Unlock()
	if elapsed := time.Since(lockStart); elapsed > time.Second {
		notifyLog.Debugf("published write operation %s waited %s for db operation lock", name, elapsed)
	}

	db.mu.Lock()
	runtime := db.runtime
	runtimeGeneration := db.runtimeGeneration
	db.mu.Unlock()
	if runtime == nil {
		if retryGuard != nil {
			return retryGuard.result, false, nil, fmt.Errorf("%w: published %s write runtime closed before authorized retry", ErrOperationReceiptUnavailable, name)
		}
		return swarmionapp.OperationResult{}, false, nil, fmt.Errorf("db is not initialized")
	}
	if expectedRuntimeGeneration != 0 && runtimeGeneration != expectedRuntimeGeneration {
		return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf(
			"%w: published %s write runtime generation changed from %d to %d",
			ErrOperationReceiptUnavailable,
			name,
			expectedRuntimeGeneration,
			runtimeGeneration,
		)
	}
	if retryGuard != nil {
		guardDisposition := retryGuard.result.Disposition()
		guardReason, guardReasonOK := retryGuard.result.RetryReason()
		if runtime != retryGuard.runtime || guardDisposition != swarmionapp.OperationRetryPermitted || !guardReasonOK || guardReason != retryGuard.reason {
			return retryGuard.result, false, runtime, fmt.Errorf(
				"%w: published %s write retry authority changed before the next execution (disposition=%s reason=%q runtime_match=%t)",
				ErrOperationReceiptUnavailable,
				name,
				guardDisposition,
				guardReason,
				runtime == retryGuard.runtime,
			)
		}
	}
	localAuthor := strings.TrimSpace(runtime.PeerID())
	if localAuthor == "" {
		return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("%w: operation=%s local author is unavailable", ErrOperationReceiptUnavailable, name)
	}
	if requestedAuthor := strings.TrimSpace(operation.AuthorPeerID()); requestedAuthor != localAuthor {
		resolved, resolveErr := db.LookupPublishedWriteOperation(ctx, operation)
		if resolved.Disposition() == swarmionapp.OperationAccepted {
			return resolved, false, runtime, nil
		}
		if resolved.Disposition() == swarmionapp.OperationRetryPermitted {
			return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf(
				"foreign published write operation %q for author %q returned authoritative absence; refusing local publication",
				name,
				requestedAuthor,
			)
		}
		return swarmionapp.OperationResult{}, false, runtime, errors.Join(resolveErr, fmt.Errorf("%w: foreign operation=%s author=%s", ErrOperationReceiptUnavailable, name, requestedAuthor))
	}
	preflight, preflightErr := db.resolvePublishedWriteOnRuntimeLocked(ctx, runtime, operation, "pre-execution")
	if preflightErr != nil {
		return preflight, false, runtime, fmt.Errorf("resolve published write operation %q immediately before execution: %w", name, preflightErr)
	}
	switch preflight.Disposition() {
	case swarmionapp.OperationAccepted:
		if requireReceipt {
			if _, err := publishedWriteReceiptFromAcceptedResult(operation, preflight); err != nil {
				return swarmionapp.OperationResult{}, false, runtime, err
			}
		}
		db.transactionMetrics.operationTransactionsAlreadyAccepted.Add(1)
		return preflight, false, runtime, nil
	case swarmionapp.OperationRetryPermitted:
		preflightReason, ok := preflight.RetryReason()
		if !ok || preflightReason != swarmionapp.RetryResolvedAbsent {
			return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf(
				"%w: published write operation %q preflight returned non-absence retry reason %q",
				ErrOperationReceiptUnavailable,
				name,
				preflightReason,
			)
		}
	case swarmionapp.OperationRecoveryRequired, swarmionapp.OperationFailedClosed:
		return preflight, false, runtime, nil
	default:
		return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("published write operation %q preflight returned unknown disposition %q", name, preflight.Disposition())
	}

	collector := &operationStatementCollector{}
	if err := apply(ctx, collector); err != nil {
		return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("prepare published write operation %q: %w", name, err)
	}
	if len(collector.statements) == 0 {
		if allowNoop {
			return swarmionapp.OperationResult{}, true, runtime, nil
		}
		return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("%w: operation=%s has no statements", ErrPublishedWriteNoChange, name)
	}

	request := swarmionapp.OperationRequest{
		Identity:   operation.Identity,
		Statements: append([]swarmionapp.Statement(nil), collector.statements...),
	}
	var readyErr error
	if db.waitMutationReadyForTest != nil {
		readyErr = db.waitMutationReadyForTest(ctx)
	} else {
		readyErr = runtime.WaitMutationReady(ctx)
	}
	if readyErr != nil {
		return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("wait to execute published write operation %q: %w", name, readyErr)
	}
	db.transactionMetrics.operationTransactionsAttempted.Add(1)
	var result swarmionapp.OperationResult
	if db.executeOperationForTest != nil {
		result = db.executeOperationForTest(ctx, runtime, request)
	} else {
		result = runtime.Execute(ctx, request)
	}
	recovery, recoveryErr := recoveryFromOperationResult(operation, result)
	if recoveryErr != nil {
		if diagnostic := result.Diagnostic(); diagnostic != nil {
			notifyLog.Warnf("operation execution returned malformed recovery operation=%s diagnostic=%s", name, diagnostic.Error())
		}
		return swarmionapp.OperationResult{}, false, runtime, recoveryErr
	}
	var recoveryPersistErr error
	if persistErr := db.persistPublishedWriteOperation(PublishedWriteOperation{Identity: operation.Identity, Recovery: recovery}); persistErr != nil {
		recoveryPersistErr = fmt.Errorf("persist operation recovery after execution: %w", persistErr)
	}

	recordExecutedCommit := func() {
		db.transactionMetrics.transactionsStarted.Add(1)
		db.transactionMetrics.commitsAttempted.Add(1)
		db.transactionMetrics.commitsSucceeded.Add(1)
	}

	switch result.Disposition() {
	case swarmionapp.OperationAccepted:
		acceptance, ok := result.Acceptance()
		if !ok {
			return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("accepted published write operation %q has no acceptance history", name)
		}
		switch acceptance {
		case swarmionapp.AcceptanceExecuted:
			recordExecutedCommit()
			db.transactionMetrics.operationTransactionsExecuted.Add(1)
		case swarmionapp.AcceptanceAlreadyAccepted:
			db.transactionMetrics.operationTransactionsAlreadyAccepted.Add(1)
		case swarmionapp.AcceptanceNoChange:
			recordExecutedCommit()
			db.transactionMetrics.operationTransactionsNoChange.Add(1)
			db.transactionMetrics.noopCommitOutcomes.Add(1)
		}
		if requireReceipt {
			if _, err := publishedWriteReceiptFromAcceptedResult(operation, result); err != nil {
				return swarmionapp.OperationResult{}, false, runtime, err
			}
		}
		if recoveryPersistErr != nil {
			// The complete receipt-less recovery address was synced before Execute.
			// Losing only the post-accept enrichment must not turn accepted evidence
			// into a retryable application error and mint a second random identity.
			db.transactionMetrics.operationRecoveryPersistenceFailures.Add(1)
			acceptedReceipt, _ := result.Receipt()
			notifyLog.Warnf(
				"accepted published write retained its pre-execute recovery record after enrichment failure operation=%s event_id=%s error=%s",
				name,
				acceptedReceipt.EventID,
				recoveryPersistErr.Error(),
			)
		}
		return result, false, runtime, nil

	case swarmionapp.OperationRetryPermitted:
		db.transactionMetrics.operationTransactionsFailed.Add(1)
		retryReason, ok := result.RetryReason()
		if !ok {
			return swarmionapp.OperationResult{}, false, runtime, fmt.Errorf("retry-permitted published write operation %q has no retry reason", name)
		}
		if retryReason == swarmionapp.RetryWorkspaceDirty {
			db.transactionMetrics.operationWorkspaceDirtyOutcomes.Add(1)
		}
		if retryReason == swarmionapp.RetryStaleWriteContext {
			db.transactionMetrics.staleWriteContextOutcomes.Add(1)
		}
		if retryReason == swarmionapp.RetryProjectionTooWide {
			db.transactionMetrics.staleWriteContextOutcomes.Add(1)
			db.transactionMetrics.projectionTooWideOutcomes.Add(1)
		}
		if errors.Is(result.Diagnostic(), swarmionapp.ErrContentConflict) {
			db.transactionMetrics.typedConflicts.Add(1)
		}
		return result, false, runtime, recoveryPersistErr

	case swarmionapp.OperationRecoveryRequired:
		db.transactionMetrics.operationTransactionsFailed.Add(1)
		db.transactionMetrics.operationTransactionLifecycleOpaqueFailures.Add(1)
		if receipt, _ := publishedWriteReceiptFromRecoveryRequired(operation, result); receipt.HasExactEventIdentity() {
			db.transactionMetrics.uncertainEventReceiptsAfterCommitErr.Add(1)
		}
		return result, false, runtime, recoveryPersistErr

	default:
		db.transactionMetrics.operationTransactionsFailed.Add(1)
		db.transactionMetrics.operationTransactionLifecycleOpaqueFailures.Add(1)
		return result, false, runtime, recoveryPersistErr
	}
}

type publishedWriteRetryGuard struct {
	runtime *swarmionapp.DatabaseRuntime
	result  swarmionapp.OperationResult
	reason  swarmionapp.OperationRetryReason
}

// executePublishedWriteOperationContext runs one idempotent operation through
// the public Execute contract. It retries only direct, same-call
// OperationRetryPermitted evidence and requires an exact receipt, including
// for a body that produces no content change.
func (db *DB) executePublishedWriteOperationContext(
	ctx context.Context,
	operation PublishedWriteOperation,
	name string,
	apply func(context.Context, sqlContextExecer) error,
) (PublishedWriteReceipt, error) {
	return db.executePublishedWriteTransactionWithSafeRetryContext(ctx, operation, name, false, true, apply)
}

// executePublishedWriteOperationForRuntimeGenerationContext is the
// generation-bound form used by workflows whose permission to continue was
// established by an earlier runtime-owned lifecycle result. A non-zero
// generation must still be current at every Execute attempt; a replacement
// runtime can never inherit that permission.
func (db *DB) executePublishedWriteOperationForRuntimeGenerationContext(
	ctx context.Context,
	operation PublishedWriteOperation,
	name string,
	expectedRuntimeGeneration uint64,
	apply func(context.Context, sqlContextExecer) error,
) (PublishedWriteReceipt, error) {
	if expectedRuntimeGeneration == 0 {
		return PublishedWriteReceipt{}, fmt.Errorf("published write operation %q has no runtime generation", name)
	}
	return db.executePublishedWriteTransactionForRuntimeGenerationWithSafeRetryContext(
		ctx,
		operation,
		name,
		false,
		true,
		expectedRuntimeGeneration,
		apply,
	)
}

func (db *DB) executePublishedWriteTransactionContext(
	ctx context.Context,
	operation PublishedWriteOperation,
	name string,
	allowNoop bool,
	requireReceiptAfterCommit bool,
	expectedRuntimeGeneration uint64,
	retryGuard *publishedWriteRetryGuard,
	apply func(context.Context, sqlContextExecer) error,
) (swarmionapp.OperationResult, bool, *swarmionapp.DatabaseRuntime, error) {
	return db.executePublishedWriteAttemptContext(
		ctx,
		operation,
		name,
		allowNoop,
		requireReceiptAfterCommit,
		expectedRuntimeGeneration,
		retryGuard,
		apply,
	)
}

func validateEventReceiptIdentity(eventID string, publishedRootHash string) (swarmionprotocol.EventID, swarmionprotocol.RootHash, error) {
	parsedEventID := swarmionprotocol.ParseEventID(strings.TrimSpace(eventID))
	parsedRoot := swarmionprotocol.ParseRootHash(strings.TrimSpace(publishedRootHash))
	if parsedEventID.IsZero() || parsedRoot.IsZero() {
		return swarmionprotocol.EventID{}, swarmionprotocol.RootHash{}, fmt.Errorf(
			"%w: event_id=%q published_root=%q",
			errSwarmionPublishedWriteIncomplete,
			eventID,
			publishedRootHash,
		)
	}
	return parsedEventID, parsedRoot, nil
}

type eventReceiptRuntime interface {
	ObserveReceipt(context.Context, swarmionapp.ReceiptTrackingRequest) (swarmionapp.ReceiptSnapshot, error)
	WaitReceipt(context.Context, swarmionapp.ReceiptWaitRequest) (swarmionapp.ReceiptWaitResult, error)
}

// ObservePublishedWriteReceipt reads only the exact event receipt for operation
// lifecycle decisions. RootStatus remains available separately for content and
// root diagnostics, but it must not classify an operation because several
// events may publish the same root.
func (db *DB) ObservePublishedWriteReceipt(ctx context.Context, receipt PublishedWriteReceipt) (EventReceiptObservation, error) {
	if db == nil {
		return EventReceiptObservation{}, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return EventReceiptObservation{}, fmt.Errorf("db is not initialized")
	}
	return observePublishedWriteReceipt(ctx, app, receipt)
}

func observePublishedWriteReceipt(ctx context.Context, runtime eventReceiptRuntime, receipt PublishedWriteReceipt) (EventReceiptObservation, error) {
	observation := EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}
	expectedEventID, expectedRoot, err := validateEventReceiptIdentity(receipt.EventID, receipt.PublishedRootHash)
	if err != nil {
		return observation, err
	}
	observation.Receipt.EventID = expectedEventID.String()
	observation.Receipt.PublishedRootHash = expectedRoot.String()
	if runtime == nil {
		return observation, fmt.Errorf("swarmion event receipt runtime is not initialized")
	}

	trackedReceipt := swarmionapp.EventReceipt{
		EventID:           expectedEventID.String(),
		PublishedRootHash: expectedRoot.String(),
	}
	tracking := swarmionapp.ReceiptTrackingRequest{Receipt: trackedReceipt}
	snapshot, err := runtime.ObserveReceipt(ctx, tracking)
	if err != nil {
		return observation, fmt.Errorf("read swarmion event receipt status: %w", err)
	}
	if err := snapshot.ValidateFor(tracking); err != nil {
		return observation, fmt.Errorf("validate swarmion receipt snapshot: %w", err)
	}
	return eventReceiptObservationFromStatus(observation.Receipt, snapshot.Event)
}

func eventReceiptObservationFromStatus(receipt PublishedWriteReceipt, status swarmionapp.ReceiptStatus) (EventReceiptObservation, error) {
	observation := EventReceiptObservation{Receipt: receipt, Status: status, State: EventReceiptStatePending}
	expectedEventID, expectedRoot, err := validateEventReceiptIdentity(receipt.EventID, receipt.PublishedRootHash)
	if err != nil {
		return observation, err
	}
	if err := validateEventReceiptStatus(expectedEventID, expectedRoot, status); err != nil {
		return observation, err
	}
	observation.Receipt.EventID = expectedEventID.String()
	observation.Receipt.PublishedRootHash = expectedRoot.String()
	if status.Checkpointed {
		observation.Receipt.Checkpointed = true
		observation.Receipt.CheckpointCommitID = status.CheckpointCommitID
		observation.Receipt.CheckpointRootHash = status.CheckpointRootHash
	}
	if status.AppliedDurably {
		observation.State = EventReceiptStateAppliedDurably
		return observation, nil
	}
	if parkedState, parked := exactEventReceiptParkedState(status); parked {
		observation.State = parkedState
	}
	return observation, nil
}

func validateEventReceiptStatus(expectedEventID swarmionprotocol.EventID, expectedRoot swarmionprotocol.RootHash, status swarmionapp.ReceiptStatus) error {
	if swarmionprotocol.ParseEventID(status.EventID) != expectedEventID ||
		swarmionprotocol.ParseRootHash(status.ExpectedPublishedRootHash) != expectedRoot {
		return fmt.Errorf(
			"%w: event status identity mismatch requested=%s/%s returned=%s/%s",
			errSwarmionPublishedWriteIncomplete,
			expectedEventID,
			expectedRoot,
			status.EventID,
			status.ExpectedPublishedRootHash,
		)
	}
	if status.Checkpointed {
		if swarmionprotocol.ParseCheckpointCommitID(status.CheckpointCommitID).IsZero() ||
			swarmionprotocol.ParseRootHash(status.CheckpointRootHash).IsZero() {
			return fmt.Errorf(
				"%w: checkpointed event %s/%s has incomplete checkpoint=%s/%s",
				errSwarmionPublishedWriteIncomplete,
				expectedEventID,
				expectedRoot,
				status.CheckpointCommitID,
				status.CheckpointRootHash,
			)
		}
	}
	if status.AppliedDurably {
		if !status.Checkpointed ||
			swarmionprotocol.ParseCheckpointCommitID(status.DurableCheckpointCommitID).IsZero() ||
			swarmionprotocol.ParseRootHash(status.DurableCheckpointRootHash).IsZero() {
			return fmt.Errorf(
				"%w: durably applied event %s/%s has incomplete checkpoint lineage",
				errSwarmionPublishedWriteIncomplete,
				expectedEventID,
				expectedRoot,
			)
		}
		switch status.ContentCoverage {
		case swarmionapp.BranchEventContentCoverageCovered,
			swarmionapp.BranchEventContentCoverageDissent,
			swarmionapp.BranchEventContentCoverageUnavailable:
		default:
			return fmt.Errorf(
				"%w: durably applied event %s/%s has invalid content coverage %q",
				errSwarmionPublishedWriteIncomplete,
				expectedEventID,
				expectedRoot,
				status.ContentCoverage,
			)
		}
	}
	if status.Durable != (status.AppliedDurably && status.ContentCoverage == swarmionapp.BranchEventContentCoverageCovered) {
		return fmt.Errorf(
			"%w: inconsistent event receipt durable=%t applied_durably=%t content_coverage=%q",
			errSwarmionPublishedWriteIncomplete,
			status.Durable,
			status.AppliedDurably,
			status.ContentCoverage,
		)
	}
	if status.Parked {
		if !status.Known || status.Checkpointed || status.AppliedDurably || !status.Revisitable {
			return fmt.Errorf(
				"%w: inconsistent exact-event parking parked=%t known=%t checkpointed=%t applied_durably=%t revisitable=%t",
				errSwarmionPublishedWriteIncomplete,
				status.Parked,
				status.Known,
				status.Checkpointed,
				status.AppliedDurably,
				status.Revisitable,
			)
		}
		switch status.ParkedReason {
		case swarmionapp.BranchRootParkedReasonConflict,
			swarmionapp.BranchRootParkedReasonDependencyParked,
			swarmionapp.BranchRootParkedReasonStaleAnchor:
		default:
			return fmt.Errorf(
				"%w: exact event %s/%s has invalid parked reason %q",
				errSwarmionPublishedWriteIncomplete,
				expectedEventID,
				expectedRoot,
				status.ParkedReason,
			)
		}
	} else if status.ParkedReason != "" || status.Revisitable {
		return fmt.Errorf(
			"%w: non-parked exact event %s/%s has parked_reason=%q revisitable=%t",
			errSwarmionPublishedWriteIncomplete,
			expectedEventID,
			expectedRoot,
			status.ParkedReason,
			status.Revisitable,
		)
	}
	return nil
}

func exactEventReceiptParkedState(status swarmionapp.ReceiptStatus) (EventReceiptState, bool) {
	if !status.Parked {
		return EventReceiptStatePending, false
	}
	switch status.ParkedReason {
	case swarmionapp.BranchRootParkedReasonDependencyParked:
		return EventReceiptStateDependencyParked, true
	case swarmionapp.BranchRootParkedReasonStaleAnchor:
		return EventReceiptStateStaleAnchor, true
	default:
		return EventReceiptStateParkedConflict, true
	}
}

// WaitForPublishedWriteApplied waits for event-level durable application. A
// caller deadline is honored; without one, the backend's 45-second receipt
// budget is applied. Covered, dissent, and unavailable content observations
// all terminate once AppliedDurably is true. Unresolved pending work remains
// on the exact receipt wait; this path never starts checkpoint catch-up.
func (db *DB) WaitForPublishedWriteApplied(ctx context.Context, receipt PublishedWriteReceipt, reason string) (EventReceiptObservation, error) {
	if db == nil {
		return EventReceiptObservation{}, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if waitForTest := db.waitPublishedWriteAppliedForTest; waitForTest != nil {
		return waitForTest(ctx, receipt, reason)
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, committedWriteCheckpointTimeout)
		defer cancel()
	}
	db.mu.Lock()
	app := db.runtime
	db.mu.Unlock()
	if app == nil {
		return EventReceiptObservation{}, fmt.Errorf("db is not initialized")
	}
	observation, err := waitForPublishedWriteApplied(ctx, app, receipt, reason)
	if err == nil {
		db.recordTerminalEventReceiptObservation(reason, observation)
	}
	return observation, err
}

func waitForPublishedWriteApplied(ctx context.Context, runtime eventReceiptRuntime, receipt PublishedWriteReceipt, reason string) (EventReceiptObservation, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "protos published write"
	}
	observation := EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}
	expectedEventID, expectedRoot, err := validateEventReceiptIdentity(receipt.EventID, receipt.PublishedRootHash)
	if err != nil {
		return observation, err
	}
	trackedReceipt := swarmionapp.EventReceipt{
		EventID:           expectedEventID.String(),
		PublishedRootHash: expectedRoot.String(),
	}
	tracking := swarmionapp.ReceiptTrackingRequest{Receipt: trackedReceipt}
	condition := swarmionapp.ReceiptConditionAppliedDurably
	result, waitErr := runtime.WaitReceipt(ctx, swarmionapp.ReceiptWaitRequest{
		Tracking:  tracking,
		Condition: condition,
	})
	if result.Snapshot.Receipt != (swarmionapp.EventReceipt{}) {
		if validationErr := result.Snapshot.ValidateFor(tracking); validationErr != nil {
			return observation, fmt.Errorf("validate swarmion receipt wait snapshot: %w", validationErr)
		}
		if validationErr := validateReceiptWaitBoundary(result, condition, result.Snapshot.Event.AppliedDurably, waitErr); validationErr != nil {
			return observation, fmt.Errorf("swarmion receipt wait returned an inconsistent applied-durability result: %w", validationErr)
		}
	} else if result.Satisfied {
		return observation, fmt.Errorf("swarmion receipt wait reported applied durability without an exact snapshot")
	}
	status := result.Snapshot.Event
	// WaitReceipt returns its latest exact observation with a context
	// error. Preserve that checkpoint/parking evidence in the backend's typed
	// outcome without performing checkpoint catch-up or republishing the event.
	if status.EventID != "" || status.ExpectedPublishedRootHash != "" {
		if validationErr := validateEventReceiptStatus(expectedEventID, expectedRoot, status); validationErr != nil {
			return observation, validationErr
		}
		observation.Status = status
		if status.Checkpointed {
			observation.Receipt.Checkpointed = true
			observation.Receipt.CheckpointCommitID = status.CheckpointCommitID
			observation.Receipt.CheckpointRootHash = status.CheckpointRootHash
		}
		if status.AppliedDurably && waitErr == nil && result.Satisfied {
			observation.State = EventReceiptStateAppliedDurably
			return observation, nil
		}
		if status.AppliedDurably {
			observation.State = EventReceiptStateAppliedDurably
		}
		if parkedState, parked := exactEventReceiptParkedState(status); parked {
			observation.State = parkedState
			parkedErr := &EventReceiptParkedError{Observation: observation, Reason: reason}
			if waitErr != nil {
				return observation, errors.Join(parkedErr, waitErr)
			}
			return observation, parkedErr
		}
	}
	if waitErr == nil {
		waitErr = fmt.Errorf("event receipt wait returned before applied_durably")
	}
	return observation, &EventReceiptPendingError{Observation: observation, Reason: reason, Cause: waitErr}
}

func (db *DB) recordTerminalEventReceiptObservation(reason string, observation EventReceiptObservation) {
	if db == nil || observation.State != EventReceiptStateAppliedDurably || observation.Status.ContentCoverage != swarmionapp.BranchEventContentCoverageDissent {
		return
	}
	key := eventReceiptContentDissentObservationKey{
		eventID:                   strings.TrimSpace(observation.Receipt.EventID),
		publishedRootHash:         strings.TrimSpace(observation.Receipt.PublishedRootHash),
		durableCheckpointCommitID: strings.TrimSpace(observation.Status.DurableCheckpointCommitID),
		durableCheckpointRootHash: strings.TrimSpace(observation.Status.DurableCheckpointRootHash),
		queryableRootHash:         strings.TrimSpace(observation.Status.QueryableRootHash),
	}
	if _, loaded := db.eventReceiptContentDissentSeen.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	count := db.eventReceiptContentDissentObservations.Add(1)
	proof := observation.Status.DurableProofObservation
	var eventContained, proofRan, proofUnavailable, proofCovered, proofConflict bool
	var proofMergedRoot string
	if proof != nil {
		eventContained = proof.EventContained
		proofRan = proof.ProofRan
		proofUnavailable = proof.ProofUnavailable
		proofCovered = proof.Covered
		proofConflict = proof.Conflict
		proofMergedRoot = proof.MergedRootHash
	}
	notifyLog.Infof(
		"swarmion event receipt content_dissent diagnostic=true reason=%q event_id=%s published_root=%s checkpoint_commit=%s checkpoint_root=%s durable_head_commit=%s durable_head_root=%s queryable_root=%s proof_event_contained=%t proof_ran=%t proof_unavailable=%t proof_covered=%t proof_conflict=%t proof_merged_root=%s backend_dissent_observations=%d",
		reason,
		observation.Receipt.EventID,
		observation.Receipt.PublishedRootHash,
		observation.Status.CheckpointCommitID,
		observation.Status.CheckpointRootHash,
		observation.Status.DurableCheckpointCommitID,
		observation.Status.DurableCheckpointRootHash,
		observation.Status.QueryableRootHash,
		eventContained,
		proofRan,
		proofUnavailable,
		proofCovered,
		proofConflict,
		proofMergedRoot,
		count,
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
		if attempt == 2 || !isTransientWorkspaceAccessError(err) {
			return res, fmt.Errorf("failed to select one: %w", err)
		}
		if waitErr := waitBeforeDatabaseRetryContext(ctx, attempt); waitErr != nil {
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
		if attempt == 2 || !isTransientWorkspaceAccessError(err) {
			return nil, fmt.Errorf("failed to select multiple: %w", err)
		}
		if waitErr := waitBeforeDatabaseRetryContext(ctx, attempt); waitErr != nil {
			return nil, fmt.Errorf("failed to select multiple: %w", waitErr)
		}
	}
	return nil, fmt.Errorf("failed to select multiple: %w", err)
}

//
// Edit operations
//

type preparedWriteStatement struct {
	SQL  string `json:"sql"`
	Args []any  `json:"args,omitempty"`
}

func prepareWriteStatement(query sq.Query) (preparedWriteStatement, error) {
	statement, args, err := sq.ToSQL(sq.DialectMySQL, query, nil)
	if err != nil {
		return preparedWriteStatement{}, err
	}
	return preparedWriteStatement{SQL: statement, Args: args}, nil
}

func executePreparedWriteStatements(ctx context.Context, executor sqlContextExecer, statements []preparedWriteStatement) error {
	for _, statement := range statements {
		if _, err := executor.ExecContext(ctx, statement.SQL, statement.Args...); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) newOrdinaryPublishedWriteOperation(name string, statements []preparedWriteStatement) (PublishedWriteOperation, error) {
	encoded, err := json.Marshal(statements)
	if err != nil {
		return PublishedWriteOperation{}, fmt.Errorf("encode ordinary %s write intent: %w", name, err)
	}
	return db.NewPublishedWriteOperation(OperationSchemaOrdinarySQLWrite, []byte(name), encoded)
}

func (db *DB) executeOrdinaryPublishedWriteContext(
	ctx context.Context,
	name string,
	allowNoop bool,
	requireReceiptAfterCommit bool,
	statements []preparedWriteStatement,
) (PublishedWriteReceipt, error) {
	if len(statements) == 0 {
		if allowNoop {
			return PublishedWriteReceipt{}, nil
		}
		return PublishedWriteReceipt{}, fmt.Errorf("ordinary %s write has no SQL statements", name)
	}
	operation, err := db.newOrdinaryPublishedWriteOperation(name, statements)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	receipt, executeErr := db.executePublishedWriteTransactionWithSafeRetryContext(
		ctx,
		operation,
		name,
		allowNoop,
		requireReceiptAfterCommit,
		func(ctx context.Context, executor sqlContextExecer) error {
			return executePreparedWriteStatements(ctx, executor, statements)
		},
	)
	if errors.Is(executeErr, ErrPublishedWriteReceiptIdentityConflict) {
		return receipt, executeErr
	}
	if receipt.HasExactEventIdentity() && !receipt.Committed {
		if waitErr := db.waitForOrdinaryPublishedWriteKnown(ctx, receipt); waitErr != nil {
			notifyLog.Warnf(
				"ordinary write retained an exact uncertain receipt but local acceptance remained unresolved operation=%s event_id=%s published_root=%s error=%s",
				name,
				receipt.EventID,
				receipt.PublishedRootHash,
				waitErr.Error(),
			)
			// Preserve the exact receipt in both the return value and the typed
			// pending error. The event may already be accepted, so this outcome
			// never grants permission to replay the SQL body.
			return receipt, errors.Join(executeErr, waitErr)
		}
		receipt.Committed = true
		receipt.OutcomeUncertain = false
		if executeErr != nil {
			notifyLog.Warnf(
				"ordinary write local publication resolved after response error operation=%s event_id=%s published_root=%s error=%s",
				name,
				receipt.EventID,
				receipt.PublishedRootHash,
				executeErr.Error(),
			)
		}
		return receipt, nil
	}
	return receipt, executeErr
}

func prepareInsertWriteStatements(label string, mappers []InsertMapper) ([]preparedWriteStatement, error) {
	statements := make([]preparedWriteStatement, 0, len(mappers))
	for _, mapper := range mappers {
		statement, err := prepareWriteStatement(mapper().SetDialect(sq.DialectMySQL))
		if err != nil {
			return nil, fmt.Errorf("prepare %s: %w", label, err)
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func prepareUpdateWriteStatements(label string, mappers []UpdateMapper) ([]preparedWriteStatement, error) {
	statements := make([]preparedWriteStatement, 0, len(mappers))
	for _, mapper := range mappers {
		statement, err := prepareWriteStatement(mapper().SetDialect(sq.DialectMySQL))
		if err != nil {
			return nil, fmt.Errorf("prepare %s: %w", label, err)
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func prepareDeleteWriteStatements(label string, mappers []DeleteMapper) ([]preparedWriteStatement, error) {
	statements := make([]preparedWriteStatement, 0, len(mappers))
	for _, mapper := range mappers {
		statement, err := prepareWriteStatement(mapper().SetDialect(sq.DialectMySQL))
		if err != nil {
			return nil, fmt.Errorf("prepare %s: %w", label, err)
		}
		statements = append(statements, statement)
	}
	return statements, nil
}

func (db *DB) waitForOrdinaryPublishedWriteKnown(ctx context.Context, receipt PublishedWriteReceipt) error {
	if ctx == nil {
		ctx = context.Background()
	}
	waitCtx, cancel := context.WithTimeout(ctx, ordinaryUncertainReceiptTimeout)
	defer cancel()
	observation := EventReceiptObservation{Receipt: receipt, State: EventReceiptStatePending}

	if observeForTest := db.observePublishedWriteForTest; observeForTest != nil {
		observed, err := observeForTest(waitCtx, receipt)
		if err == nil && observed.Status.Known {
			return nil
		}
		if observed.Receipt.HasExactEventIdentity() {
			observation = observed
		}
		if err == nil {
			err = waitCtx.Err()
			if err == nil {
				err = fmt.Errorf(
					"exact event receipt %s/%s remained unknown after a bounded acceptance wait",
					receipt.EventID,
					receipt.PublishedRootHash,
				)
			}
		}
		return &EventReceiptPendingError{
			Observation: observation,
			Reason:      "resolve uncertain ordinary write local publication",
			Cause:       err,
		}
	}

	db.mu.Lock()
	runtime := db.runtime
	db.mu.Unlock()
	if runtime == nil {
		return &EventReceiptPendingError{
			Observation: observation,
			Reason:      "resolve uncertain ordinary write local publication",
			Cause:       fmt.Errorf("db is not initialized"),
		}
	}
	expectedEventID, expectedRoot, err := validateEventReceiptIdentity(receipt.EventID, receipt.PublishedRootHash)
	if err != nil {
		return err
	}
	tracking := swarmionapp.ReceiptTrackingRequest{Receipt: swarmionapp.EventReceipt{
		EventID:           expectedEventID.String(),
		PublishedRootHash: expectedRoot.String(),
	}}
	condition := swarmionapp.ReceiptConditionLocallyAccepted
	result, waitErr := runtime.WaitReceipt(waitCtx, swarmionapp.ReceiptWaitRequest{
		Tracking:  tracking,
		Condition: condition,
	})
	if result.Snapshot.Receipt != (swarmionapp.EventReceipt{}) {
		if validationErr := result.Snapshot.ValidateFor(tracking); validationErr != nil {
			return &EventReceiptPendingError{
				Observation: observation,
				Reason:      "resolve uncertain ordinary write local publication",
				Cause:       fmt.Errorf("validate swarmion receipt wait snapshot: %w", validationErr),
			}
		}
		if validationErr := validateReceiptWaitBoundary(result, condition, result.Snapshot.Event.Known, waitErr); validationErr != nil {
			return &EventReceiptPendingError{
				Observation: observation,
				Reason:      "resolve uncertain ordinary write local publication",
				Cause:       fmt.Errorf("swarmion receipt wait returned an inconsistent local-acceptance result: %w", validationErr),
			}
		}
		observed, observationErr := eventReceiptObservationFromStatus(receipt, result.Snapshot.Event)
		if observationErr != nil {
			return &EventReceiptPendingError{
				Observation: observation,
				Reason:      "resolve uncertain ordinary write local publication",
				Cause:       observationErr,
			}
		}
		observation = observed
	} else if result.Satisfied {
		return &EventReceiptPendingError{
			Observation: observation,
			Reason:      "resolve uncertain ordinary write local publication",
			Cause:       fmt.Errorf("swarmion receipt wait reported local acceptance without an exact snapshot"),
		}
	}
	if waitErr == nil && result.Satisfied {
		return nil
	}
	if waitErr == nil {
		waitErr = fmt.Errorf(
			"exact event receipt %s/%s remained unknown after a bounded acceptance wait",
			receipt.EventID,
			receipt.PublishedRootHash,
		)
	}
	return &EventReceiptPendingError{
		Observation: observation,
		Reason:      "resolve uncertain ordinary write local publication",
		Cause:       waitErr,
	}
}

// validateReceiptWaitBoundary accepts Swarmion's terminal-handoff exception:
// a watch may close after observing the requested boundary, in which case its
// latest snapshot satisfies the predicate while Result.Satisfied remains false
// and the typed terminal error remains authoritative. A normal successful wait
// must have no error and must explicitly report the predicate as satisfied.
func validateReceiptWaitBoundary(
	result swarmionapp.ReceiptWaitResult,
	condition swarmionapp.ReceiptCondition,
	snapshotSatisfies bool,
	waitErr error,
) error {
	if result.Condition != condition {
		return fmt.Errorf("condition=%q, want %q", result.Condition, condition)
	}
	if result.Satisfied {
		if waitErr != nil {
			return fmt.Errorf("satisfied wait also returned an error: %w", waitErr)
		}
		if !snapshotSatisfies {
			return fmt.Errorf("satisfied wait snapshot does not meet condition %q", condition)
		}
		return nil
	}
	if waitErr == nil {
		return fmt.Errorf("unsatisfied wait returned without an error")
	}
	return nil
}

// InsertWithReceiptContext publishes an insert and returns the exact root
// receipt for callers that need to observe its later lifecycle.
func InsertWithReceiptContext(ctx context.Context, db *DB, mappers ...InsertMapper) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	statements, err := prepareInsertWriteStatements("insert", mappers)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	return db.executeOrdinaryPublishedWriteContext(ctx, "insert", false, true, statements)
}

// UpdateWithReceiptContext publishes an update and returns the exact root
// receipt for callers that need to observe its later lifecycle.
func UpdateWithReceiptContext(ctx context.Context, db *DB, mappers ...UpdateMapper) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	statements, err := prepareUpdateWriteStatements("update", mappers)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	return db.executeOrdinaryPublishedWriteContext(ctx, "update", true, true, statements)
}

// UpdateAndInsertWithReceiptContext commits update and insert mappers as one
// published write and returns the exact event/root receipt. Ordinary product
// callers should wrap it with UpdateAndInsertWithAvailabilityContext.
// Restart-sensitive operations must instead use
// UpdateAndInsertWithOperationReceiptContext with a stable key and intent
// digest retained in replicated operation state.
func UpdateAndInsertWithReceiptContext(ctx context.Context, db *DB, updates []UpdateMapper, inserts []InsertMapper) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	updateStatements, err := prepareUpdateWriteStatements("update", updates)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	insertStatements, err := prepareInsertWriteStatements("insert", inserts)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	statements := append(updateStatements, insertStatements...)
	return db.executeOrdinaryPublishedWriteContext(ctx, "update and insert", true, true, statements)
}

// UpdateAndInsertWithOperationReceiptContext publishes all mappers as one
// operation-correlated SQL transaction. A found operation skips every mapper
// and returns the original exact event receipt.
func UpdateAndInsertWithOperationReceiptContext(
	ctx context.Context,
	db *DB,
	operation PublishedWriteOperation,
	updates []UpdateMapper,
	inserts []InsertMapper,
) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	updateStatements, err := prepareUpdateWriteStatements("operation update", updates)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	insertStatements, err := prepareInsertWriteStatements("operation insert", inserts)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	statements := append(updateStatements, insertStatements...)
	return db.executePublishedWriteOperationContext(ctx, operation, "update and insert", func(ctx context.Context, executor sqlContextExecer) error {
		return executePreparedWriteStatements(ctx, executor, statements)
	})
}

// InsertAndUpdateWithOperationReceiptContext publishes inserts before updates
// as one operation-correlated SQL transaction. It exists for phase-authority
// CAS workflows where an immutable INSERT ... SELECT must first prove the
// exact pre-state and the following update is conditional on that fact.
func InsertAndUpdateWithOperationReceiptContext(
	ctx context.Context,
	db *DB,
	operation PublishedWriteOperation,
	inserts []InsertMapper,
	updates []UpdateMapper,
) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	insertStatements, err := prepareInsertWriteStatements("operation conditional insert", inserts)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	updateStatements, err := prepareUpdateWriteStatements("operation conditional update", updates)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	statements := append(insertStatements, updateStatements...)
	return db.executePublishedWriteOperationContext(ctx, operation, "insert and update", func(ctx context.Context, executor sqlContextExecer) error {
		return executePreparedWriteStatements(ctx, executor, statements)
	})
}

// InsertAndUpdateWithOperationReceiptForRuntimeGenerationContext is the
// generation-bound variant for a phase transition authorized by a lifecycle
// result from the same DatabaseRuntime generation.
func InsertAndUpdateWithOperationReceiptForRuntimeGenerationContext(
	ctx context.Context,
	db *DB,
	operation PublishedWriteOperation,
	expectedRuntimeGeneration uint64,
	inserts []InsertMapper,
	updates []UpdateMapper,
) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	insertStatements, err := prepareInsertWriteStatements("operation conditional insert", inserts)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	updateStatements, err := prepareUpdateWriteStatements("operation conditional update", updates)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	statements := append(insertStatements, updateStatements...)
	return db.executePublishedWriteOperationForRuntimeGenerationContext(
		ctx,
		operation,
		"insert and update",
		expectedRuntimeGeneration,
		func(ctx context.Context, executor sqlContextExecer) error {
			return executePreparedWriteStatements(ctx, executor, statements)
		},
	)
}

// DeleteAndInsertWithOperationReceiptContext publishes deletes and immutable
// marker inserts as one operation-correlated SQL transaction. It is intended
// for side-effect workflows whose stable recovery marker must be inseparable
// from the business mutation that the marker describes.
func DeleteAndInsertWithOperationReceiptContext(
	ctx context.Context,
	db *DB,
	operation PublishedWriteOperation,
	deletes []DeleteMapper,
	inserts []InsertMapper,
) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	deleteStatements, err := prepareDeleteWriteStatements("operation delete", deletes)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	insertStatements, err := prepareInsertWriteStatements("operation marker insert", inserts)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	statements := append(deleteStatements, insertStatements...)
	return db.executePublishedWriteOperationContext(ctx, operation, "delete and insert", func(ctx context.Context, executor sqlContextExecer) error {
		return executePreparedWriteStatements(ctx, executor, statements)
	})
}

// DeleteAndInsertWithOperationReceiptForRuntimeGenerationContext is the
// generation-bound variant for final mutations authorized by an earlier
// lifecycle result from the same DatabaseRuntime generation.
func DeleteAndInsertWithOperationReceiptForRuntimeGenerationContext(
	ctx context.Context,
	db *DB,
	operation PublishedWriteOperation,
	expectedRuntimeGeneration uint64,
	deletes []DeleteMapper,
	inserts []InsertMapper,
) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	deleteStatements, err := prepareDeleteWriteStatements("operation delete", deletes)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	insertStatements, err := prepareInsertWriteStatements("operation marker insert", inserts)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	statements := append(deleteStatements, insertStatements...)
	return db.executePublishedWriteOperationForRuntimeGenerationContext(
		ctx,
		operation,
		"delete and insert",
		expectedRuntimeGeneration,
		func(ctx context.Context, executor sqlContextExecer) error {
			return executePreparedWriteStatements(ctx, executor, statements)
		},
	)
}

// DeleteWithReceiptContext publishes a delete and returns the exact root
// receipt for callers that need to observe its later lifecycle.
func DeleteWithReceiptContext(ctx context.Context, db *DB, mappers ...DeleteMapper) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	statements, err := prepareDeleteWriteStatements("delete", mappers)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	return db.executeOrdinaryPublishedWriteContext(ctx, "delete", true, true, statements)
}

// DeleteWithOperationReceiptContext publishes every mapper as one
// operation-correlated SQL transaction and returns the authoritative exact
// receipt from the typed publication outcome.
func DeleteWithOperationReceiptContext(
	ctx context.Context,
	db *DB,
	operation PublishedWriteOperation,
	mappers ...DeleteMapper,
) (PublishedWriteReceipt, error) {
	if db == nil {
		return PublishedWriteReceipt{}, fmt.Errorf("db is nil")
	}
	statements, err := prepareDeleteWriteStatements("operation delete", mappers)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	return db.executePublishedWriteOperationContext(ctx, operation, "delete", func(ctx context.Context, executor sqlContextExecer) error {
		return executePreparedWriteStatements(ctx, executor, statements)
	})
}

func (db *DB) executePublishedWriteTransactionWithSafeRetryContext(
	ctx context.Context,
	operation PublishedWriteOperation,
	name string,
	allowNoop bool,
	requireReceiptAfterCommit bool,
	apply func(context.Context, sqlContextExecer) error,
) (PublishedWriteReceipt, error) {
	return db.executePublishedWriteTransactionForRuntimeGenerationWithSafeRetryContext(
		ctx,
		operation,
		name,
		allowNoop,
		requireReceiptAfterCommit,
		0,
		apply,
	)
}

func (db *DB) executePublishedWriteTransactionForRuntimeGenerationWithSafeRetryContext(
	ctx context.Context,
	operation PublishedWriteOperation,
	name string,
	allowNoop bool,
	requireReceiptAfterCommit bool,
	expectedRuntimeGeneration uint64,
	apply func(context.Context, sqlContextExecer) error,
) (PublishedWriteReceipt, error) {
	var retryGuard *publishedWriteRetryGuard
	for attempt := 1; ; attempt++ {
		result, noOp, resultRuntime, callErr := db.executePublishedWriteTransactionContext(
			ctx,
			operation,
			name,
			allowNoop,
			requireReceiptAfterCommit,
			expectedRuntimeGeneration,
			retryGuard,
			apply,
		)
		if noOp {
			return PublishedWriteReceipt{}, callErr
		}
		switch result.Disposition() {
		case swarmionapp.OperationAccepted:
			receipt, err := publishedWriteReceiptFromAcceptedResult(operation, result)
			if err != nil {
				return PublishedWriteReceipt{}, err
			}
			exactReceipt := receipt
			if !requireReceiptAfterCommit {
				receipt.EventID = ""
				receipt.PublishedRootHash = ""
			}
			if result.Diagnostic() != nil {
				notifyLog.Warnf(
					"accepted published write retained a terminal diagnostic operation=%s event_id=%s error=%s",
					name,
					receipt.EventID,
					result.Diagnostic(),
				)
			}
			db.schedulePublishedWriteOperationCleanup(operation, exactReceipt)
			return receipt, callErr
		case swarmionapp.OperationRecoveryRequired:
			receipt, receiptErr := publishedWriteReceiptFromRecoveryRequired(operation, result)
			if diagnostic := result.Diagnostic(); diagnostic != nil {
				notifyLog.Warnf("published write requires recovery operation=%s diagnostic=%s", name, diagnostic.Error())
			}
			return receipt, errors.Join(callErr, receiptErr, ErrOperationReceiptUnavailable)
		case swarmionapp.OperationFailedClosed:
			return PublishedWriteReceipt{}, errors.Join(
				callErr,
				fmt.Errorf("published %s write failed closed: %s", name, operationDiagnosticText(result.Diagnostic())),
			)
		case swarmionapp.OperationRetryPermitted:
			// Continue only on the direct result returned under the current runtime
			// lease. Neither Diagnostic nor any reachable error node is inspected.
			if callErr != nil {
				return PublishedWriteReceipt{}, callErr
			}
		default:
			return PublishedWriteReceipt{}, errors.Join(callErr, fmt.Errorf("published %s write returned unknown disposition %q", name, result.Disposition()))
		}
		if attempt >= ordinaryWriteSafeRetryMaxAttempts {
			return PublishedWriteReceipt{}, fmt.Errorf(
				"%w: published %s write remained safely rejected after %d attempts: %v",
				errPublishedWriteRetryExhausted,
				name,
				attempt,
				operationDiagnosticText(result.Diagnostic()),
			)
		}
		retryReason, ok := result.RetryReason()
		if !ok {
			return PublishedWriteReceipt{}, fmt.Errorf("retry-permitted published %s write has no retry reason", name)
		}
		if resultRuntime == nil {
			return PublishedWriteReceipt{}, fmt.Errorf("%w: retry-permitted published %s write has no owning runtime", ErrOperationReceiptUnavailable, name)
		}
		retryGuard = &publishedWriteRetryGuard{runtime: resultRuntime, result: result, reason: retryReason}
		if db.waitPublishedWriteRetryForTest != nil {
			if waitErr := db.waitPublishedWriteRetryForTest(ctx, attempt, retryReason); waitErr != nil {
				return PublishedWriteReceipt{}, fmt.Errorf("wait before safely retrying published %s write: %w", name, waitErr)
			}
			continue
		}
		if retryReason == swarmionapp.RetryProjectionTooWide {
			if readyErr := db.WaitMutationReady(ctx); readyErr != nil {
				return PublishedWriteReceipt{}, fmt.Errorf("wait for published %s write projection readiness: %w", name, readyErr)
			}
			continue
		}
		if waitErr := waitBeforeDatabaseRetryContext(ctx, attempt); waitErr != nil {
			return PublishedWriteReceipt{}, fmt.Errorf("wait before safely retrying published %s write: %w", name, waitErr)
		}
	}
}

func operationDiagnosticText(err error) string {
	if err == nil {
		return "<none>"
	}
	return err.Error()
}

func queryWhenSQLViewReady(ctx context.Context, query func() (*sql.Rows, error)) (*sql.Rows, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for attempt := 1; attempt <= sqlViewReadyRetryMaxAttempts; attempt++ {
		rows, err := query()
		if err == nil || !errors.Is(err, swarmionapp.ErrSQLViewNotReady) {
			return rows, err
		}
		if attempt == sqlViewReadyRetryMaxAttempts {
			return nil, fmt.Errorf("SQL view remained not ready after %d bounded query attempts: %w", attempt, err)
		}
		if waitErr := waitBeforeDatabaseRetryContext(ctx, attempt); waitErr != nil {
			return nil, errors.Join(err, waitErr)
		}
	}
	return nil, fmt.Errorf("SQL view query retry loop exhausted")
}

func execWhenSQLViewReady(ctx context.Context, exec func() (sql.Result, error)) (sql.Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for attempt := 1; attempt <= sqlViewReadyRetryMaxAttempts; attempt++ {
		result, err := exec()
		if err == nil || !errors.Is(err, swarmionapp.ErrSQLViewNotReady) {
			return result, err
		}
		if attempt == sqlViewReadyRetryMaxAttempts {
			return nil, fmt.Errorf("SQL view remained not ready after %d bounded statement attempts: %w", attempt, err)
		}
		if waitErr := waitBeforeDatabaseRetryContext(ctx, attempt); waitErr != nil {
			return nil, errors.Join(err, waitErr)
		}
	}
	return nil, fmt.Errorf("SQL view statement retry loop exhausted")
}

func waitBeforeDatabaseRetryContext(ctx context.Context, attempt int) error {
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
