package provisioners

import (
	"context"
	stdsql "database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bokwoon95/sq"
	swarmionprotocol "github.com/nustiueudinastea/swarmion/protocol"
	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/pkg/errors"
	"github.com/protosio/protos/internal/config"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/p2p"
	"github.com/protosio/protos/internal/p2p/proto"
	"github.com/protosio/protos/internal/pcrypto"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
	"github.com/protosio/protos/internal/user"
	"github.com/protosio/protos/internal/util"
	"golang.org/x/crypto/ssh"
)

var log = util.GetLogger("provisioners")

// ErrInstanceDeleteInvariantConflict means a later durable write made the
// provision visible again after its delete event was applied.
var ErrInstanceDeleteInvariantConflict = errors.New("instance delete invariant conflict")

// ErrInstanceLifecycleConflict means a requested lifecycle transition races
// an already-replicated delete intent or its active delete task. Callers must
// resolve ownership explicitly rather than reopening the peer route.
var ErrInstanceLifecycleConflict = errors.New("instance lifecycle conflict")

// ErrInstanceInitializationRecoveryRequired means a provider-backed instance
// has crossed the peer-discovery/admission boundary and can no longer be
// compensated by deleting its provider resources or replicated identity. The
// retained record is the recovery authority; callers must resume or explicitly
// resolve that same instance instead of replaying deployment.
var ErrInstanceInitializationRecoveryRequired = errors.New("instance initialization recovery required")

// instanceDeletePublicationWithoutReceiptError marks the final database
// operation boundary when Swarmion returned no exact receipt. The task layer
// may replay it only when the wrapped error is the explicit typed
// not-accepted-safe-to-retry outcome.
type instanceDeletePublicationWithoutReceiptError struct {
	Cause error
}

func (err *instanceDeletePublicationWithoutReceiptError) Error() string {
	if err == nil || err.Cause == nil {
		return "instance delete publication returned no receipt"
	}
	return "instance delete publication returned no receipt: " + err.Cause.Error()
}

func (err *instanceDeletePublicationWithoutReceiptError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func instanceDeletePublicationWithoutReceipt(cause error) error {
	if cause == nil {
		cause = fmt.Errorf("operation returned neither a receipt nor an error")
	}
	var existing *instanceDeletePublicationWithoutReceiptError
	if errors.As(cause, &existing) {
		return cause
	}
	return &instanceDeletePublicationWithoutReceiptError{Cause: cause}
}

// Type represents a specific cloud (AWS, GCP, DigitalOcean etc.)
type Type string

func (ct Type) String() string {
	return string(ct)
}

// CreateManager creates and returns a cloud manager.
//
// Concrete provisioners are passed in by the caller so internal/provisioners owns the
// orchestration contract without importing implementation packages.
func CreateManager(db *db.DB, um *user.Manager, sm *pcrypto.Manager, p2p *p2p.P2P, taskManager *tasks.Manager, provisioners ...ProvisionerFactory) (*Manager, error) {
	if db == nil || um == nil || sm == nil || p2p == nil || taskManager == nil {
		return nil, fmt.Errorf("failed to create cloud manager: none of the inputs can be nil")
	}

	manager := &Manager{
		db:           db,
		um:           um,
		sm:           sm,
		p2p:          p2p,
		provisioners: newProvisionerRegistry(),
		tasks:        taskManager,
		lifecycleSig: map[string]string{},
	}
	for _, provisioner := range provisioners {
		manager.RegisterProvisioner(provisioner)
	}
	if err := manager.registerTaskStreams(); err != nil {
		return nil, err
	}

	return manager, nil
}

// Manager manages provisioners and instances.
type Manager struct {
	db                       *db.DB
	um                       *user.Manager
	sm                       *pcrypto.Manager
	p2p                      *p2p.P2P
	provisioners             *provisionerRegistry
	tasks                    *tasks.Manager
	lifecycleMu              sync.Mutex
	lifecycleSig             map[string]string
	instanceLifecycleMu      sync.Map
	deleteReceiptTracker     instanceDeleteReceiptTracker
	peerDrainRuntime         replicationPeerDrainRuntime
	peerRouteFence           replicationPeerRouteFence
	providerMutationDisabled bool
	// Recovery-only test seams prove foreign pending/parked handling without
	// synthesizing protocol state. Production always uses the DB methods.
	lookupDeleteRecoveryOperation func(context.Context, db.PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error)
	observeDeleteRecoveryReceipt  func(context.Context, db.PublishedWriteReceipt) (db.EventReceiptObservation, error)
	publishDeleteOperation        func(context.Context, db.PublishedWriteOperation, InstanceInfo) (db.PublishedWriteReceipt, error)
	lookupPeerDrainAuthorization  func(context.Context, db.PublishedWriteOperation) (swarmionapp.BranchOperationReceipt, error)
	publishPeerDrainAuthorization func(context.Context, db.PublishedWriteOperation, InstanceInfo, tasks.OperationFact) (db.PublishedWriteReceipt, error)
	waitPeerDrainAuthorization    func(context.Context, db.PublishedWriteReceipt, string) (db.EventReceiptObservation, error)
	verifyPeerDrainAuthorization  func(context.Context, string, instancePeerDrainAuthorization, tasks.OperationFact) error
	// afterInstanceDeletePublished is an internal fault-injection seam for the
	// accepted-event-to-immutable-receipt-fact crash boundary.
	afterInstanceDeletePublished func(db.PublishedWriteReceipt)
	// Phase-boundary fault injection proves that restart resolves P before
	// repeating drain and that provider deletion never runs without durable P.
	afterPeerDrainAuthorized func(db.PublishedWriteReceipt)
	afterProviderDelete      func(string)
	// Deployment-only seams keep the admission/compensation boundary directly
	// testable. Production uses the application-owned P2P manager and user store.
	currentDeviceForInstance        func() (user.UserDevice, error)
	discoverPeerForInstance         func(context.Context, string) (*p2p.DiscoveredPeer, error)
	addPeerForInstance              func(InstanceInfo) (*p2p.Client, error)
	initializePeerForInstance       func(context.Context, *p2p.Client, *proto.InitRequest) (*proto.InitResponse, error)
	originBootstrapAddrsForInstance func(string, string) []string
	// Deployment admission seams make the exact publication boundary directly
	// testable. An exact unresolved receipt must never trigger a compensating
	// delete or a second enqueue publication.
	insertDeploymentPlaceholder func(context.Context, ...db.InsertMapper) (db.PublishedWriteConfirmation, error)
	deleteDeploymentPlaceholder func(context.Context, ...db.DeleteMapper) (db.PublishedWriteConfirmation, error)
	enqueueDeploymentTask       func(context.Context, tasks.EnqueueOptions[deployInstanceTaskPayload]) (tasks.Record, error)
}

// SetProviderMutationEnabled prevents task-stream registration on a
// non-provisioning replica from becoming provider-side execution authority.
func (cm *Manager) SetProviderMutationEnabled(enabled bool) {
	if cm == nil {
		return
	}
	cm.providerMutationDisabled = !enabled
}

func (cm *Manager) lockInstanceLifecycle(instanceID string) func() {
	if cm == nil {
		return func() {}
	}
	instanceID = strings.TrimSpace(instanceID)
	value, _ := cm.instanceLifecycleMu.LoadOrStore(instanceID, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

func instanceInitializationRecoveryRequired(instanceID string, stage string, cause error) error {
	instanceID = strings.TrimSpace(instanceID)
	stage = strings.TrimSpace(stage)
	if cause == nil {
		cause = fmt.Errorf("instance initialization stopped")
	}
	return fmt.Errorf("%w: instance=%s stage=%s: %w", ErrInstanceInitializationRecoveryRequired, instanceID, stage, cause)
}

func (cm *Manager) deploymentCurrentDevice() (user.UserDevice, error) {
	if cm.currentDeviceForInstance != nil {
		return cm.currentDeviceForInstance()
	}
	return cm.um.GetCurrentDevice()
}

func (cm *Manager) deploymentDiscoverPeer(ctx context.Context, target string) (*p2p.DiscoveredPeer, error) {
	if cm.discoverPeerForInstance != nil {
		return cm.discoverPeerForInstance(ctx, target)
	}
	return cm.p2p.DiscoverPeer(ctx, target)
}

func (cm *Manager) deploymentAddPeer(instance InstanceInfo) (*p2p.Client, error) {
	if cm.addPeerForInstance != nil {
		return cm.addPeerForInstance(instance)
	}
	return cm.p2p.AddPeer(instance)
}

func (cm *Manager) deploymentInitializePeer(ctx context.Context, client *p2p.Client, request *proto.InitRequest) (*proto.InitResponse, error) {
	if cm.initializePeerForInstance != nil {
		return cm.initializePeerForInstance(ctx, client, request)
	}
	return client.Init(ctx, request)
}

func (cm *Manager) deploymentOriginBootstrapAddrs(originPublicKey string, peerPublicIP string) []string {
	if cm.originBootstrapAddrsForInstance != nil {
		return cm.originBootstrapAddrsForInstance(originPublicKey, peerPublicIP)
	}
	return cm.originSwarmionBootstrapAddrs(originPublicKey, peerPublicIP)
}

type replicationPeerDrainRuntime interface {
	Available() bool
	Prepare(context.Context, string, []db.ReplicationCandidate) error
	Begin(context.Context, string, string) (swarmionapp.PeerDrainStatus, error)
	Watch(context.Context, string, string) (<-chan swarmionapp.PeerDrainEvent, error)
	Finalize(context.Context, string, string) (swarmionapp.PeerDrainFinalizeResponse, error)
}

type replicationPeerRouteFence interface {
	FencePeer(p2p.Machine) (string, string, error)
	WithPeerFenceGeneration(context.Context, string, string, func() error) error
}

type databasePeerDrainRuntime struct{ database *db.DB }

func (runtime databasePeerDrainRuntime) Available() bool {
	if runtime.database == nil {
		return false
	}
	_, ok := runtime.database.SwarmionStatus()
	return ok
}

func (runtime databasePeerDrainRuntime) Prepare(ctx context.Context, peerID string, candidates []db.ReplicationCandidate) error {
	return runtime.database.PrepareReplicationPeerDrain(ctx, peerID, candidates)
}

func (runtime databasePeerDrainRuntime) Begin(ctx context.Context, peerID, generation string) (swarmionapp.PeerDrainStatus, error) {
	return runtime.database.BeginReplicationPeerDrain(ctx, peerID, generation)
}

func (runtime databasePeerDrainRuntime) Watch(ctx context.Context, peerID, generation string) (<-chan swarmionapp.PeerDrainEvent, error) {
	return runtime.database.WatchReplicationPeerDrain(ctx, peerID, generation)
}

func (runtime databasePeerDrainRuntime) Finalize(ctx context.Context, peerID, generation string) (swarmionapp.PeerDrainFinalizeResponse, error) {
	return runtime.database.FinalizeReplicationPeerDrain(ctx, peerID, generation)
}

func (cm *Manager) replicationPeerDrainRuntime() replicationPeerDrainRuntime {
	if cm != nil && cm.peerDrainRuntime != nil {
		return cm.peerDrainRuntime
	}
	if cm == nil || cm.db == nil {
		return nil
	}
	return databasePeerDrainRuntime{database: cm.db}
}

func (cm *Manager) replicationPeerRouteFence() replicationPeerRouteFence {
	if cm != nil && cm.peerRouteFence != nil {
		return cm.peerRouteFence
	}
	if cm == nil || cm.p2p == nil {
		return nil
	}
	return cm.p2p
}

type instanceDeleteReceiptTracker interface {
	WaitForPublishedWriteApplied(context.Context, db.PublishedWriteReceipt, string) (db.EventReceiptObservation, error)
	InstanceExistsAtCheckpoint(context.Context, string, string) (bool, error)
}

type swarmionInstanceDeleteReceiptTracker struct {
	database *db.DB
}

func (tracker swarmionInstanceDeleteReceiptTracker) WaitForPublishedWriteApplied(ctx context.Context, receipt db.PublishedWriteReceipt, reason string) (db.EventReceiptObservation, error) {
	return tracker.database.WaitForPublishedWriteApplied(ctx, receipt, reason)
}

func (tracker swarmionInstanceDeleteReceiptTracker) InstanceExistsAtCheckpoint(ctx context.Context, checkpointCommitID string, instanceID string) (bool, error) {
	instanceIDBytes, err := db.UUIDBytes(instanceID)
	if err != nil {
		return false, fmt.Errorf("invalid instance ID %q: %w", instanceID, err)
	}
	var count int
	err = tracker.database.ReadRowsAsOf(
		ctx,
		checkpointCommitID,
		"SELECT COUNT(*) FROM machines AS OF ? WHERE id = ?",
		[]any{instanceIDBytes},
		func(rows *stdsql.Rows) error {
			if !rows.Next() {
				return stdsql.ErrNoRows
			}
			return rows.Scan(&count)
		},
	)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (cm *Manager) deleteReceiptBackend() instanceDeleteReceiptTracker {
	if cm != nil && cm.deleteReceiptTracker != nil {
		return cm.deleteReceiptTracker
	}
	if cm == nil || cm.db == nil {
		return nil
	}
	return swarmionInstanceDeleteReceiptTracker{database: cm.db}
}

const instanceSSHKeysDir = "instance-ssh-keys"

func instanceSSHKeyPath(instanceID string) (string, error) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return "", fmt.Errorf("instance id is empty")
	}
	if strings.ContainsAny(instanceID, `/\`) {
		return "", fmt.Errorf("invalid instance id %q", instanceID)
	}
	return path.Join(config.Get().WorkDir, instanceSSHKeysDir, instanceID), nil
}

// GetInstanceSSHKey returns the locally persisted private SSH key used for
// provisioning an instance.
func (cm *Manager) GetInstanceSSHKey(id string) (string, error) {
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return "", err
	}
	keyPath, err := instanceSSHKeyPath(instance.ID)
	if err != nil {
		return "", err
	}
	key, err := os.ReadFile(keyPath)
	if err == nil {
		return string(key), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("read SSH key for instance '%s': %w", instance.Name, err)
	}
	localKey, localErr := cm.sm.GetLocalKey()
	if localErr != nil {
		return "", fmt.Errorf("read SSH key for instance '%s': %w; failed to load local SSH key: %w", instance.Name, err, localErr)
	}
	return localKey.EncodePrivateKeytoPEM(), nil
}

func (cm *Manager) deleteInstanceSSHKey(instanceID string) error {
	keyPath, err := instanceSSHKeyPath(instanceID)
	if err != nil {
		return err
	}
	if err := os.Remove(keyPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

//
// Cloud manager methods
//

// RegisterProvisioner adds a concrete provisioner implementation to the manager.
func (cm *Manager) RegisterProvisioner(factory ProvisionerFactory) {
	cm.provisioners.register(factory)
}

// AddProvisioner validates and saves a provisioner configuration.
func (cm *Manager) AddProvisioner(name string, provisionerType string, auth map[string]string) error {
	record := newProvisionerRecord(name, Type(provisionerType), auth)
	provisioner, err := cm.newProvisioner(record)
	if err != nil {
		return fmt.Errorf("failed to create provisioner: %w", err)
	}
	if err := provisioner.Init(); err != nil {
		return fmt.Errorf("failed to initialize provisioner: %w", err)
	}
	if err := cm.saveProviderRecord(provisioner.ProviderRecord()); err != nil {
		return fmt.Errorf("failed to save provisioner: %w", err)
	}
	return nil
}

// AddProvider validates and saves a cloud provider configuration.
func (cm *Manager) AddProvider(cloudName string, cloud string, auth map[string]string) error {
	return cm.AddProvisioner(cloudName, cloud, auth)
}

// SupportedProvisioners returns a list of supported provisioners.
func (cm *Manager) SupportedProvisioners() []string {
	provisionerTypes := cm.provisioners.types()
	supportedProvisioners := make([]string, 0, len(provisionerTypes))
	for _, provisionerType := range provisionerTypes {
		supportedProvisioners = append(supportedProvisioners, provisionerType.String())
	}
	return supportedProvisioners
}

// SupportedProviders returns a list of supported cloud providers.
func (cm *Manager) SupportedProviders() []string {
	return cm.SupportedProvisioners()
}

// ProvisionerAuthFields returns the authentication fields required by a provisioner type.
func (cm *Manager) ProvisionerAuthFields(provisionerType string) ([]string, error) {
	factory, found := cm.provisioners.factory(Type(provisionerType))
	if !found {
		return nil, fmt.Errorf("provisioner '%s' not supported", provisionerType)
	}
	return factory.AuthFields(), nil
}

// ProviderAuthFields returns the authentication fields required by a provider type.
func (cm *Manager) ProviderAuthFields(cloud string) ([]string, error) {
	return cm.ProvisionerAuthFields(cloud)
}

// GetProvisioner returns a provisioner instance from the db.
func (cm *Manager) GetProvisioner(id string) (Provisioner, error) {
	record, found, err := cm.findProviderRecord(id)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provisioner '%s': %w", id, err)
	}
	if !found {
		return nil, fmt.Errorf("could not find provisioner '%s'", id)
	}
	provisioner, err := cm.newProvisioner(record)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve provisioner '%s': %w", id, err)
	}
	return provisioner, nil
}

// GetProvisionerOrDefault returns a configured provisioner or a transient
// zero-auth built-in provisioner such as local_macos.
func (cm *Manager) GetProvisionerOrDefault(id string) (Provisioner, error) {
	provisioner, err := cm.GetProvisioner(id)
	if err == nil {
		return provisioner, nil
	}
	defaultProvisioner, defaultErr := cm.newDefaultProvisioner(id)
	if defaultErr != nil {
		return nil, err
	}
	return defaultProvisioner, nil
}

// GetProvider returns a cloud provider instance from the db.
func (cm *Manager) GetProvider(id string) (ProviderClient, error) {
	return cm.GetProvisioner(id)
}

func (cm *Manager) newDefaultProvisioner(id string) (Provisioner, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("provisioner id is empty")
	}
	factory, found := cm.provisioners.factory(Type(id))
	if !found {
		return nil, fmt.Errorf("provisioner '%s' not supported", id)
	}
	if len(factory.AuthFields()) > 0 {
		return nil, fmt.Errorf("provisioner '%s' requires credentials", id)
	}
	return cm.newProvisioner(newProvisionerRecord(id, Type(id), nil))
}

func (cm *Manager) ensureProviderForDeployment(id string) (Provisioner, error) {
	provisioner, err := cm.GetProvider(id)
	if err == nil {
		return provisioner, nil
	}
	defaultProvisioner, defaultErr := cm.newDefaultProvisioner(id)
	if defaultErr != nil {
		return nil, err
	}
	if err := defaultProvisioner.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize default provisioner '%s': %w", id, err)
	}
	if err := cm.saveProviderRecord(defaultProvisioner.ProviderRecord()); err != nil {
		return nil, fmt.Errorf("failed to save default provisioner '%s': %w", id, err)
	}
	return defaultProvisioner, nil
}

// DeleteProvisioner deletes a provisioner from the db.
func (cm *Manager) DeleteProvisioner(name string) error {
	record, found, err := cm.findProviderRecord(name)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("could not find provisioner '%s'", name)
	}

	_, err = db.DeleteWithAvailabilityContext(context.Background(), cm.db, createCloudProviderDeleteMapper(record.ID))
	if err != nil {
		return fmt.Errorf("failed to delete provisioner '%s': %w", name, err)
	}

	return nil
}

// DeleteProvider deletes a cloud provider from the db.
func (cm *Manager) DeleteProvider(name string) error {
	return cm.DeleteProvisioner(name)
}

// GetProvisioners returns all configured provisioners from the db.
func (cm *Manager) GetProvisioners() ([]Provisioner, error) {
	provisioners := []Provisioner{}
	records, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper(nil))
	if err != nil {
		return provisioners, fmt.Errorf("failed to retrieve provisioners: %w", err)
	}

	for _, record := range records {
		client, err := cm.newProvisioner(record)
		if err != nil {
			return provisioners, err
		}
		provisioners = append(provisioners, client)
	}

	return provisioners, nil
}

// GetProviders returns all the cloud providers from the db.
func (cm *Manager) GetProviders() ([]ProviderClient, error) {
	return cm.GetProvisioners()
}

func (cm *Manager) provisionerDeps() ProvisionerDeps {
	return ProvisionerDeps{SecretManager: cm.sm, WorkDir: config.Get().WorkDir}
}

func (cm *Manager) newProvisioner(record ProvisionerRecord) (Provisioner, error) {
	record = record.normalized()
	if record.Name == "" {
		return nil, fmt.Errorf("cloud provider name is empty")
	}
	factory, found := cm.provisioners.factory(record.Type)
	if !found {
		return nil, fmt.Errorf("cloud '%s' not supported", record.Type.String())
	}
	return factory.NewClient(record, cm.provisionerDeps())
}

func (cm *Manager) saveProviderRecord(record ProviderRecord) error {
	record = record.normalized()
	if record.Name == "" {
		return fmt.Errorf("cloud provider name is empty")
	}

	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	records, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{cpModel.NAME.EqString(record.Name)}))
	if err != nil {
		return err
	}
	if len(records) > 1 {
		return fmt.Errorf("found multiple cloud providers named '%s'", record.Name)
	}
	if len(records) == 1 {
		record.ID = records[0].ID
		_, err := db.UpdateWithAvailabilityContext(context.Background(), cm.db, createCloudProviderUpdateMapper(record))
		return err
	}

	if strings.TrimSpace(record.ID) == "" {
		record.ID = db.MustNewUUIDv7()
	} else if _, err := db.UUIDBytes(record.ID); err != nil {
		return fmt.Errorf("cloud provider id must be a UUID: %w", err)
	}
	_, err = db.InsertWithAvailabilityContext(context.Background(), cm.db, createCloudProviderInsertMapper(record))
	return err
}

func (cm *Manager) findProviderRecord(ref string) (ProviderRecord, bool, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ProviderRecord{}, false, fmt.Errorf("cloud provider name is empty")
	}

	if _, parseErr := db.UUIDBytes(ref); parseErr == nil {
		records, err := cm.findProviderRecordsByID(ref)
		if err != nil {
			return ProviderRecord{}, false, err
		}
		if len(records) == 1 {
			return records[0], true, nil
		}
		if len(records) > 1 {
			return ProviderRecord{}, false, fmt.Errorf("found multiple cloud providers with id '%s'", ref)
		}
	}

	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	records, err := db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{cpModel.NAME.EqString(ref)}))
	if err != nil {
		return ProviderRecord{}, false, err
	}
	if len(records) == 0 {
		return ProviderRecord{}, false, nil
	}
	if len(records) > 1 {
		return ProviderRecord{}, false, fmt.Errorf("found multiple cloud providers named '%s'", ref)
	}
	return records[0], true, nil
}

func (cm *Manager) findProviderRecordsByID(id string) ([]ProviderRecord, error) {
	cpModel := sq.New[db.CLOUD_PROVIDER]("")
	return db.SelectMultiple(cm.db, createCloudProviderQueryMapper([]sq.Predicate{db.UUIDEq(cpModel.ID, id)}))
}

//
// Instance related methods
//

// DeployInstance deploys an instance on the provided cloud
func (cm *Manager) DeployInstance(instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (InstanceInfo, error) {
	instance, _, err := cm.DeployInstanceWithConfirmation(context.Background(), instanceName, cloudName, cloudLocation, release, machineType)
	return instance, err
}

// DeployInstanceWithConfirmation exposes the exact task-enqueue confirmation
// while preserving DeployInstance for callers that do not render write stages.
func (cm *Manager) DeployInstanceWithConfirmation(ctx context.Context, instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (result InstanceInfo, task tasks.Record, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	provider, err := cm.ensureProviderForDeployment(cloudName)
	if err != nil {
		return InstanceInfo{}, tasks.Record{}, fmt.Errorf("could not retrieve cloud '%s': %w", cloudName, err)
	}
	if _, err := requireComputeProvisioner(provider); err != nil {
		return InstanceInfo{}, tasks.Record{}, err
	}
	if _, err := requireImageProvisioner(provider); err != nil {
		return InstanceInfo{}, tasks.Record{}, err
	}
	if _, err := requireVolumeProvisioner(provider); err != nil {
		return InstanceInfo{}, tasks.Record{}, err
	}
	if existing, existingErr := db.SelectOne(cm.db, createInstanceQueryByNameMapper(instanceName)); existingErr == nil && existing.ID != "" {
		return InstanceInfo{}, tasks.Record{}, fmt.Errorf("instance '%s' already exists", instanceName)
	}

	pendingID := newPendingInstanceID()
	lifecycleOwnerPeerID := cm.localLifecycleExecutorPeerID()
	instance := InstanceInfo{
		ID:                   pendingID,
		Name:                 instanceName,
		Kind:                 KindCloudVM,
		KindID:               provider.NameStr(),
		DesiredStatus:        ServerStateRunning,
		ReplicationPriority:  db.DefaultReplicationPriorityForMachine(KindCloudVM, provider.NameStr()),
		Location:             cloudLocation,
		Status:               ServerStateChanging,
		LifecycleOwnerPeerID: lifecycleOwnerPeerID,
	}
	if err := cm.assertInstanceLifecycleExecutor(instance, lifecycleOwnerPeerID); err != nil {
		return InstanceInfo{}, tasks.Record{}, fmt.Errorf("authorize desired instance %q: %w", instanceName, err)
	}
	mm, cmm := createInstanceInsertMapper(instance)
	if _, err := cm.insertDeploymentPlaceholderWithAvailability(ctx, mm, cmm); err != nil {
		if errors.Is(err, db.ErrPublishedWriteConfirmationUnresolved) {
			return instance, tasks.Record{}, fmt.Errorf("failed to resolve desired instance '%s' publication: %w", instanceName, err)
		}
		return InstanceInfo{}, tasks.Record{}, fmt.Errorf("failed to save desired instance '%s': %w", instanceName, err)
	}

	task, err = cm.enqueueDeploymentTaskWithContext(ctx, tasks.EnqueueOptions[deployInstanceTaskPayload]{
		Stream:      InstanceDeploymentTaskStream,
		SubjectType: taskSubjectInstance,
		SubjectID:   pendingID,
		OwnerPeerID: lifecycleOwnerPeerID,
		Title:       fmt.Sprintf("Deploy instance %s", instanceName),
		Message:     "queued",
		Payload: deployInstanceTaskPayload{
			PendingInstanceID: pendingID,
			InstanceName:      instanceName,
			CloudName:         cloudName,
			CloudLocation:     cloudLocation,
			Release:           release,
			MachineType:       machineType,
		},
	})
	if err != nil {
		if errors.Is(err, db.ErrPublishedWriteConfirmationUnresolved) {
			instance.Status = fmt.Sprintf("%s: %s", task.Status, task.Message)
			return instance, task, fmt.Errorf("failed to resolve deployment task publication for instance '%s': %w", instanceName, err)
		}
		im, cmmd := createInstanceDeleteMapper(pendingID)
		cleanupCtx, cancelCleanup := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancelCleanup()
		_, _ = cm.deleteDeploymentPlaceholderWithAvailability(cleanupCtx, im, cmmd)
		return InstanceInfo{}, tasks.Record{}, fmt.Errorf("failed to queue deployment for instance '%s': %w", instanceName, err)
	}
	instance.Status = fmt.Sprintf("%s: %s", task.Status, task.Message)
	log.Infof("Queued deployment task '%s' for desired instance '%s'", task.ID, instanceName)
	return instance, task, nil
}

func (cm *Manager) insertDeploymentPlaceholderWithAvailability(
	ctx context.Context,
	mappers ...db.InsertMapper,
) (db.PublishedWriteConfirmation, error) {
	if cm != nil && cm.insertDeploymentPlaceholder != nil {
		return cm.insertDeploymentPlaceholder(ctx, mappers...)
	}
	if cm == nil {
		return db.PublishedWriteConfirmation{}, fmt.Errorf("cloud manager is nil")
	}
	return db.InsertWithAvailabilityContext(ctx, cm.db, mappers...)
}

func (cm *Manager) deleteDeploymentPlaceholderWithAvailability(
	ctx context.Context,
	mappers ...db.DeleteMapper,
) (db.PublishedWriteConfirmation, error) {
	if cm != nil && cm.deleteDeploymentPlaceholder != nil {
		return cm.deleteDeploymentPlaceholder(ctx, mappers...)
	}
	if cm == nil {
		return db.PublishedWriteConfirmation{}, fmt.Errorf("cloud manager is nil")
	}
	return db.DeleteWithAvailabilityContext(ctx, cm.db, mappers...)
}

func (cm *Manager) enqueueDeploymentTaskWithContext(
	ctx context.Context,
	opts tasks.EnqueueOptions[deployInstanceTaskPayload],
) (tasks.Record, error) {
	if cm != nil && cm.enqueueDeploymentTask != nil {
		return cm.enqueueDeploymentTask(ctx, opts)
	}
	if cm == nil {
		return tasks.Record{}, fmt.Errorf("cloud manager is nil")
	}
	return tasks.EnqueueContext(ctx, cm.tasks, opts)
}

func (cm *Manager) deployInstanceImperative(ctx context.Context, progress func(int, string, any) error, pendingInstanceID string, instanceName string, cloudName string, cloudLocation string, release release.Release, machineType string) (result InstanceInfo, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}
	pendingInstance, err := cm.getInstanceRecord(pendingInstanceID)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("load deployment authority for pending instance %q: %w", pendingInstanceID, err)
	}
	if err := cm.assertInstanceLifecycleExecutor(pendingInstance, ""); err != nil {
		return InstanceInfo{}, err
	}
	lifecycleOwnerPeerID := pendingInstance.LifecycleOwnerPeerID
	// init cloud
	_ = progress(5, "initializing provisioner", nil)
	provider, err := cm.ensureProviderForDeployment(cloudName)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("could not retrieve cloud '%s': %w", cloudName, err)
	}
	computeProvider, err := requireComputeProvisioner(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	imageProvider, err := requireImageProvisioner(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	volumeProvider, err := requireVolumeProvisioner(provider)
	if err != nil {
		return InstanceInfo{}, err
	}
	if err := provider.Init(); err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to init cloud provider '%s'(%s) API: %w", cloudName, provider.TypeStr(), err)
	}

	var vmID string
	var volumeID string
	var instanceInfo InstanceInfo
	admissionEstablished := false
	defer func() {
		if err == nil || vmID == "" {
			return
		}
		if admissionEstablished {
			// DiscoverPeer has already authenticated and admitted a route on the
			// shared application host. From this point the provider resource and
			// replicated identity are one recoverable operation; compensation
			// must not tear either half down or permit a replay to create a second
			// VM.
			log.Warnf("Retaining admitted instance deployment '%s' (%s) for recovery: %s", instanceName, vmID, err.Error())
			return
		}
		log.Warnf("Cleaning up failed instance deployment '%s' (%s): %s", instanceName, vmID, err.Error())
		if instanceInfo.ID != "" {
			_ = cm.deleteInstanceSSHKey(instanceInfo.ID)
		}
		if stopErr := computeProvider.StopInstance(vmID, cloudLocation); stopErr != nil {
			log.Debugf("failed to stop partially deployed instance '%s' (%s): %s", instanceName, vmID, stopErr.Error())
		}
		if volumeID != "" {
			if deleteErr := volumeProvider.DeleteVolume(volumeID, cloudLocation); deleteErr != nil {
				log.Warnf("failed to delete partially deployed instance volume '%s' for '%s': %s", volumeID, instanceName, deleteErr.Error())
			}
		}
		if deleteErr := computeProvider.DeleteInstance(vmID, cloudLocation); deleteErr != nil {
			log.Warnf("failed to delete partially deployed instance '%s' (%s): %s", instanceName, vmID, deleteErr.Error())
		}
	}()

	// validate machine type
	_ = progress(10, "validating machine type", nil)
	supportedMachineTypes, err := computeProvider.SupportedMachines(cloudLocation)
	if err != nil {
		return InstanceInfo{}, err
	}
	if _, found := supportedMachineTypes[machineType]; !found {
		return InstanceInfo{}, errors.Errorf("Machine type '%s' is not valid for cloud provider '%s'. The following types are supported: \n%s", machineType, provider.TypeStr(), createMachineTypesString(supportedMachineTypes))
	}

	// add image
	_ = progress(20, "resolving image", nil)
	imageID := ""
	images, err := imageProvider.GetImages()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}
	if selectedID, selectedImage, found := SelectProtosImageForRef(images, cloudLocation, release.Version); found {
		imageID = selectedID
		log.Infof("selected Protos image '%s' (%s) for version '%s'", selectedImage.Name, selectedID, release.Version)
	}
	if imageID != "" {
		log.Infof("found Protos image version '%s' in your cloud account", release.Version)
	} else {
		// upload protos image
		if image, found := release.CloudImages[provider.TypeStr()]; found {
			log.Infof("Protos image version '%s' not in your infra cloud account. Adding it.", release.Version)
			_ = progress(25, "uploading image", map[string]string{"version": release.Version})
			imageID, err = imageProvider.AddImage(image.URL, image.Digest, release.Version, cloudLocation)
			if err != nil {
				return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
			}
		} else {
			return InstanceInfo{}, errors.Errorf("could not find a Protos version '%s' release for cloud '%s'", release.Version, provider.TypeStr())
		}
	}

	thisDevice, err := cm.deploymentCurrentDevice()
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get current device : %w", err)
	}

	// deploy a protos instance
	log.Infof("Deploying instance '%s' of type '%s', using Protos version '%s' (image id '%s')", instanceName, machineType, release.Version, imageID)
	_ = progress(35, "creating VM", map[string]string{"image_id": imageID})
	vmID, err = computeProvider.NewInstance(instanceName, imageID, thisDevice.GetPublicKey(), machineType, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to deploy Protos instance: %w", err)
	}
	log.Infof("Instance with ID '%s' deployed", vmID)

	// get instance info
	instanceInfo, err = computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get Protos instance info: %w", err)
	}
	if instanceInfo.ProviderResourceID == "" {
		instanceInfo.ProviderResourceID = vmID
	}
	instanceInfo.LifecycleOwnerPeerID = lifecycleOwnerPeerID
	if instanceInfo.Kind == "" {
		instanceInfo.Kind = KindCloudVM
	}
	if instanceInfo.KindID == "" {
		instanceInfo.KindID = provider.NameStr()
	}
	instanceInfo.ReplicationPriority = db.DefaultReplicationPriorityForMachine(instanceInfo.Kind, instanceInfo.KindID)
	if pendingInstanceID != "" {
		pendingUpdate := instanceInfo
		pendingUpdate.ID = pendingInstanceID
		pendingUpdate.PublicKey = ""
		pendingUpdate.Architecture = ""
		pendingUpdate.DesiredStatus = ServerStateRunning
		pendingUpdate.ReplicationPriority = db.DefaultReplicationPriorityForMachine(pendingUpdate.Kind, pendingUpdate.KindID)
		if updateErr := cm.updateDeploymentPlaceholder(ctx, pendingUpdate); updateErr != nil {
			return InstanceInfo{}, updateErr
		}
	}

	// create protos data volume
	log.Infof("creating data volume for Protos instance '%s'", instanceName)
	_ = progress(45, "creating data volume", nil)
	volumeID, err = volumeProvider.NewVolume(instanceName, 30000, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to create data volume: %w", err)
	}

	// attach volume to instance
	_ = progress(55, "attaching data volume", map[string]string{"volume_id": volumeID})
	err = volumeProvider.AttachVolume(volumeID, vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to attach volume to instance '%s': %w", instanceName, err)
	}

	// start protos instance
	log.Infof("Starting instance '%s'", instanceName)
	_ = progress(65, "starting VM", nil)
	err = computeProvider.StartInstance(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to start instance: %w", err)
	}

	// get instance info again
	instanceUpdate, err := computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, fmt.Errorf("failed to get instance info: %w", err)
	}
	instanceInfo.PublicIP = instanceUpdate.PublicIP
	instanceInfo.Volumes = instanceUpdate.Volumes
	if instanceUpdate.Status != "" {
		instanceInfo.Status = instanceUpdate.Status
	}
	if pendingInstanceID != "" {
		pendingUpdate := instanceInfo
		pendingUpdate.ID = pendingInstanceID
		pendingUpdate.PublicKey = ""
		pendingUpdate.Architecture = ""
		pendingUpdate.DesiredStatus = ServerStateRunning
		if updateErr := cm.updateDeploymentPlaceholder(ctx, pendingUpdate); updateErr != nil {
			return InstanceInfo{}, updateErr
		}
	}

	_ = progress(75, "discovering VM identity", map[string]string{"public_ip": instanceInfo.PublicIP})
	discoverCtx, discoverCancel := context.WithTimeout(ctx, 5*time.Minute)
	discoveredPeer, err := cm.deploymentDiscoverPeer(discoverCtx, instanceInfo.PublicIP)
	discoverCancel()
	if err != nil {
		return InstanceInfo{}, deploymentFailureError(computeProvider, vmID, cloudLocation, fmt.Errorf("failed to discover instance peer over libp2p: %w", err))
	}
	// A successful discovery has already admitted this identity and route to the
	// shared host. Set the boundary before inspecting the result so even a
	// malformed success cannot trigger destructive provider compensation.
	admissionEstablished = true
	if discoveredPeer == nil || strings.TrimSpace(discoveredPeer.PublicKey) == "" {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(pendingInstanceID, "discover_peer", fmt.Errorf("discovery returned no peer identity"))
	}
	instanceInfo.PublicKey = discoveredPeer.PublicKey
	if pendingInstanceID != "" {
		instanceInfo.ID = pendingInstanceID
	} else if _, parseErr := db.UUIDBytes(instanceInfo.ID); parseErr != nil {
		instanceInfo.ID = db.MustNewUUIDv7()
	}
	log.Infof("Discovered instance peer '%s' at '%s' with fingerprint '%s'", discoveredPeer.ID, discoveredPeer.Address, discoveredPeer.Fingerprint)
	if persistErr := cm.persistDiscoveredDeploymentIdentity(ctx, pendingInstanceID, instanceInfo); persistErr != nil {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "persist_discovered_identity", persistErr)
	}

	_ = progress(85, "initializing VM", map[string]string{"peer_id": discoveredPeer.ID})
	p2pClient, err := cm.deploymentAddPeer(instanceInfo)
	if err != nil {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "add_peer", err)
	}
	if p2pClient == nil {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "add_peer", errors.New("p2p client is nil"))
	}

	originSwarmionAddrs := cm.deploymentOriginBootstrapAddrs(thisDevice.GetPublicKey(), instanceInfo.PublicIP)

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := cm.deploymentInitializePeer(ctx, p2pClient, &proto.InitRequest{
		OriginDevice:          thisDevice.GetName(),
		OriginDevicePublicKey: thisDevice.GetPublicKey(),
		OriginSwarmionAddrs:   originSwarmionAddrs,
		InstanceName:          instanceName,
	})
	if err != nil {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "initialize_peer", err)
	}
	if resp == nil {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "initialize_peer", fmt.Errorf("initialization returned no response"))
	}

	instanceUpdate, err = computeProvider.GetInstanceInfo(vmID, cloudLocation)
	if err != nil {
		return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "refresh_provider_state", err)
	}

	// final save instance info
	instanceInfo.Architecture = resp.Architecture
	instanceInfo.Status = instanceUpdate.Status
	instanceInfo.DesiredStatus = ServerStateRunning
	instanceInfo.ReplicationPriority = db.DefaultReplicationPriorityForMachine(instanceInfo.Kind, instanceInfo.KindID)

	_ = progress(95, "saving VM identity", map[string]string{"peer_id": discoveredPeer.ID})
	if pendingInstanceID != "" {
		if err := cm.completeDeploymentInstance(ctx, pendingInstanceID, instanceInfo); err != nil {
			return InstanceInfo{}, instanceInitializationRecoveryRequired(instanceInfo.ID, "complete_deployment", err)
		}
	}

	log.Infof("Instance '%s' at '%s' is ready", instanceName, instanceInfo.PublicIP)

	return instanceInfo, nil
}

func deploymentFailureError(provider ComputeProvisioner, id string, location string, cause error) error {
	diagnosticsProvider, ok := provider.(DeploymentDiagnosticsProvider)
	if !ok {
		return cause
	}
	diagnostics, err := diagnosticsProvider.DeploymentDiagnostics(id, location)
	if err != nil {
		log.Debugf("failed to collect deployment diagnostics for '%s': %s", id, err.Error())
		return cause
	}
	diagnostics = strings.TrimSpace(diagnostics)
	if diagnostics == "" {
		return cause
	}
	return fmt.Errorf("%w\n\nVM diagnostics:\n%s", cause, diagnostics)
}

func (cm *Manager) InitInstance(instanceName string, kind string, kindID string, locationName string, ipString string) (err error) {
	lifecycleOwnerPeerID := cm.localLifecycleExecutorPeerID()
	instanceInfo := InstanceInfo{
		ID:                   db.MustNewUUIDv7(),
		PublicIP:             ipString,
		Name:                 instanceName,
		Kind:                 kind,
		KindID:               kindID,
		DesiredStatus:        ServerStateRunning,
		ReplicationPriority:  db.DefaultReplicationPriorityForMachine(kind, kindID),
		Location:             locationName,
		LifecycleOwnerPeerID: lifecycleOwnerPeerID,
	}
	if err := cm.assertInstanceLifecycleExecutor(instanceInfo, lifecycleOwnerPeerID); err != nil {
		return fmt.Errorf("authorize discovered instance %q: %w", instanceName, err)
	}

	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("String '%s' is not a valid IP address", ipString)
	}

	thisDevice, err := cm.deploymentCurrentDevice()
	if err != nil {
		return fmt.Errorf("failed to get current device : %w", err)
	}

	discoverCtx, discoverCancel := context.WithTimeout(context.Background(), 5*time.Minute)
	discoveredPeer, err := cm.deploymentDiscoverPeer(discoverCtx, instanceInfo.PublicIP)
	discoverCancel()
	if err != nil {
		return fmt.Errorf("failed to discover instance peer over libp2p: %w", err)
	}
	if discoveredPeer == nil || strings.TrimSpace(discoveredPeer.PublicKey) == "" {
		return instanceInitializationRecoveryRequired(instanceInfo.ID, "discover_peer", fmt.Errorf("discovery returned no peer identity"))
	}
	instanceInfo.PublicKey = discoveredPeer.PublicKey
	log.Infof("Discovered instance peer '%s' at '%s' with fingerprint '%s'", discoveredPeer.ID, discoveredPeer.Address, discoveredPeer.Fingerprint)

	machineMapper, machineMetadataMapper := createInstanceInsertMapper(instanceInfo)
	if _, err := db.InsertWithAvailabilityContext(context.Background(), cm.db, machineMapper, machineMetadataMapper, db.CreatePeerInsertMapper(instanceInfo.PublicKey)); err != nil {
		return instanceInitializationRecoveryRequired(instanceInfo.ID, "persist_discovered_identity", fmt.Errorf("failed to save instance '%s': %w", instanceName, err))
	}

	p2pClient, err := cm.deploymentAddPeer(instanceInfo)
	if err != nil {
		return instanceInitializationRecoveryRequired(instanceInfo.ID, "add_peer", err)
	}
	if p2pClient == nil {
		return instanceInitializationRecoveryRequired(instanceInfo.ID, "add_peer", errors.New("p2p client is nil"))
	}

	originSwarmionAddrs := cm.deploymentOriginBootstrapAddrs(thisDevice.GetPublicKey(), instanceInfo.PublicIP)

	// do the initialization
	log.Infof("Initializing instance '%s'", instanceName)
	resp, err := cm.deploymentInitializePeer(context.Background(), p2pClient, &proto.InitRequest{
		OriginDevice:          thisDevice.GetName(),
		OriginDevicePublicKey: thisDevice.GetPublicKey(),
		OriginSwarmionAddrs:   originSwarmionAddrs,
		InstanceName:          instanceName,
	})
	if err != nil {
		return instanceInitializationRecoveryRequired(instanceInfo.ID, "initialize_peer", err)
	}
	if resp == nil {
		return instanceInitializationRecoveryRequired(instanceInfo.ID, "initialize_peer", fmt.Errorf("initialization returned no response"))
	}

	instanceInfo.Architecture = resp.Architecture

	log.Infof("Instance '%s'(%s) initialized", instanceName, ipString)

	return nil
}

func (cm *Manager) originSwarmionBootstrapAddrs(originPublicKey string, peerPublicIP string) []string {
	ips := originBootstrapIPs(originPublicKey, peerPublicIP)
	addrs := cm.p2p.DialableListenMultiaddrs(ips)
	if len(addrs) == 0 {
		return cm.p2p.ListenMultiaddrs()
	}
	return addrs
}

func originBootstrapIPs(originPublicKey string, peerPublicIP string) []string {
	var ips []string
	if key, err := pcrypto.CreatePublicKeyFromBase64(originPublicKey); err == nil {
		ips = append(ips, key.IPv6Address().String())
	}
	if gateway := localMacOSNATGateway(peerPublicIP); gateway != "" {
		ips = append(ips, gateway)
	}
	return dedupeIPs(ips)
}

func localMacOSNATGateway(peerPublicIP string) string {
	ip := net.ParseIP(strings.TrimSpace(peerPublicIP))
	if ip == nil {
		return ""
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	if ip4[0] != 192 || ip4[1] != 168 || ip4[2] != 64 {
		return ""
	}
	return net.IPv4(ip4[0], ip4[1], ip4[2], 1).String()
}

func dedupeIPs(ips []string) []string {
	seen := map[string]struct{}{}
	deduped := make([]string, 0, len(ips))
	for _, ip := range ips {
		ip = strings.TrimSpace(ip)
		if ip == "" {
			continue
		}
		if _, found := seen[ip]; found {
			continue
		}
		seen[ip] = struct{}{}
		deduped = append(deduped, ip)
	}
	return deduped
}

func getProviderInstanceInfo(provider ComputeProvisioner, instance InstanceInfo) (InstanceInfo, string, error) {
	refs := []string{instance.ProviderResourceID, instance.ID}
	bestRef := firstNonEmptyString(refs...)
	var firstErr error
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		info, err := provider.GetInstanceInfo(ref, instance.Location)
		if err == nil {
			if info.ProviderResourceID == "" {
				info.ProviderResourceID = ref
			}
			return info, ref, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if strings.TrimSpace(instance.Name) == "" || instance.Name == instance.ID {
		return InstanceInfo{}, bestRef, firstErr
	}
	byName, nameErr := provider.GetInstanceInfo(instance.Name, instance.Location)
	if nameErr == nil {
		providerID := byName.ProviderResourceID
		if providerID == "" {
			providerID = byName.ID
		}
		byName.ProviderResourceID = providerID
		return byName, providerID, nil
	}
	if firstErr != nil {
		return InstanceInfo{}, bestRef, firstErr
	}
	return InstanceInfo{}, bestRef, nameErr
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// UpdateInstance updates an instance
func (cm *Manager) UpdateInstance(id string, ip string) error {
	unlock := cm.lockInstanceLifecycle(id)
	defer unlock()
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if err := cm.assertInstanceLifecycleExecutor(instance, ""); err != nil {
		return err
	}
	instance.PublicIP = ip
	im, cmm := createInstanceUpdateMapper(instance)
	_, err = db.UpdateWithAvailabilityContext(context.Background(), cm.db, im, cmm)
	if err != nil {
		return fmt.Errorf("failed to save instance '%s': %w", id, err)
	}
	persisted, err := cm.getInstanceRecord(instance.ID)
	if err != nil {
		return fmt.Errorf("verify saved instance '%s': %w", id, err)
	}
	if persisted.PublicIP != ip {
		return fmt.Errorf("%w: instance '%s' is delete-authorized and rejected the update", ErrInstanceLifecycleConflict, id)
	}

	return nil

}

// DeleteInstance queues deletion of an instance.
func (cm *Manager) DeleteInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.QueueDeleteInstance(ctx, id)
}

// DeleteInstanceLocal deletes only local database and peer state for an instance.
func (cm *Manager) DeleteInstanceLocal(ctx context.Context, id string) (tasks.Record, error) {
	return cm.QueueDeleteInstanceLocal(ctx, id)
}

func (cm *Manager) deleteInstanceImperative(
	ctx context.Context,
	progress func(int, string, any) error,
	id string,
	localOnly bool,
	operationID string,
	operationIdentity instanceDeleteOperationIdentity,
	peerDrainAuthorization *instancePeerDrainAuthorization,
	storedReceipt *instanceDeleteOperationReceipt,
	persistReceipt func(instanceDeleteOperationReceipt, int, string) error,
) error {
	ctx, cancel := instanceDeleteContext(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return err
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}
	if err := validateInstanceDeleteOperationIdentity(operationIdentity, operationID, id, localOnly); err != nil {
		return err
	}
	authorityInstance, authorityErr := cm.getInstanceRecord(id)
	if authorityErr != nil {
		if !errors.Is(authorityErr, stdsql.ErrNoRows) {
			return fmt.Errorf("load instance delete authority: %w", authorityErr)
		}
		authorityInstance = InstanceInfo{ID: id, LifecycleOwnerPeerID: operationIdentity.AuthorPeerID}
	}
	if err := cm.assertInstanceLifecycleExecutor(authorityInstance, operationIdentity.AuthorPeerID); err != nil {
		return err
	}
	if strings.TrimSpace(operationIdentity.AuthorPeerID) != strings.TrimSpace(authorityInstance.LifecycleOwnerPeerID) {
		return fmt.Errorf(
			"%w: instance=%s persisted_owner=%s delete_author=%s",
			ErrInstanceLifecycleOwnerConflict,
			id,
			authorityInstance.LifecycleOwnerPeerID,
			operationIdentity.AuthorPeerID,
		)
	}
	if storedReceipt != nil {
		if err := validateInstanceDeleteOperationReceipt(*storedReceipt, operationIdentity, operationID, id); err != nil {
			return err
		}
		if persistReceipt == nil {
			return fmt.Errorf("instance delete receipt persistence is not configured")
		}
		// Receipt recovery is status-first. A replicated receipt-bearing task
		// payload resumes this exact EventID/root and never republishes the delete.
		if err := cm.completeInstanceDeleteReceipt(ctx, *storedReceipt, persistReceipt); err != nil {
			return err
		}
		if err := cm.deleteInstanceSSHKey(id); err != nil {
			log.Warnf("failed to delete SSH key for instance '%s': %s", id, err.Error())
		}
		return nil
	}

	// Resolve before reading the instance or repeating any provider-side work.
	// A prior process may have lost the receipt immediately after Swarmion
	// accepted the final delete event.
	resolved, err := cm.db.LookupPublishedWriteOperation(ctx, operationIdentity.publishedWriteOperation())
	if err != nil {
		return fmt.Errorf("resolve instance delete operation %s: %w", operationID, err)
	}
	switch resolved.Resolution {
	case swarmionapp.BranchOperationReceiptFound:
		published, err := db.PublishedWriteReceiptFromOperation(resolved)
		if err != nil {
			return fmt.Errorf("recover instance delete operation %s: %w", operationID, err)
		}
		receipt := instanceDeleteReceiptFromPublished(operationID, operationIdentity, published)
		if err := persistReceipt(receipt, 92, "recovered published instance deletion"); err != nil {
			return fmt.Errorf("persist recovered instance delete receipt: %w", err)
		}
		if err := cm.completeInstanceDeleteReceipt(ctx, receipt, persistReceipt); err != nil {
			return err
		}
		if err := cm.deleteInstanceSSHKey(id); err != nil {
			log.Warnf("failed to delete SSH key for instance '%s': %s", id, err.Error())
		}
		return nil
	case swarmionapp.BranchOperationReceiptUnavailable:
		return fmt.Errorf("%w: instance delete operation=%s", db.ErrOperationReceiptUnavailable, operationID)
	case swarmionapp.BranchOperationReceiptAbsent:
		if !resolved.SafeToPublish {
			return fmt.Errorf("%w: instance delete operation=%s was absent without safe publication authority", db.ErrOperationReceiptUnavailable, operationID)
		}
		// Authoritative local absence permits the one and only first attempt.
	default:
		return fmt.Errorf("resolve instance delete operation %s returned unknown resolution %q", operationID, resolved.Resolution)
	}

	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if err := cm.assertInstanceLifecycleExecutor(instance, operationIdentity.AuthorPeerID); err != nil {
		return err
	}
	if (instance.Kind == KindCloudVM || instance.Kind == KindLocalVM) &&
		strings.TrimSpace(instance.PublicKey) == "" {
		// A provider-backed record without replicated peer identity may be an
		// untouched placeholder, or it may have crossed discovery immediately
		// before a crash. The database cannot distinguish those histories, so it
		// cannot prove provider destruction safe. Preserve both task and resource
		// for explicit recovery; never use absence of PublicKey as permission to
		// compensate. Local-only deletion is also unsafe because it would erase
		// the only replicated identity from which a later coordinated drain could
		// recover.
		return fmt.Errorf(
			"%w: instance=%s provider_resource=%s has no replicated peer identity",
			db.ErrReplicationPeerDrainPending,
			instance.ID,
			strings.TrimSpace(instance.ProviderResourceID),
		)
	}
	if strings.TrimSpace(instance.PublicKey) == "" && cm.tasks != nil {
		if task, found, taskErr := cm.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, instance.ID); taskErr == nil && found {
			if cancelErr := cm.tasks.Cancel(task.ID, "instance removed"); cancelErr != nil {
				log.Warnf("failed to cancel deployment task for instance '%s': %s", instance.Name, cancelErr.Error())
			}
		}
	}
	if strings.TrimSpace(instance.PublicKey) == "" {
		if err := progress(10, "marking instance deleting", map[string]string{"instance_id": instance.ID, "instance_name": instance.Name}); err != nil {
			return err
		}
		if err := cm.markInstanceDeleting(ctx, instance); err != nil {
			return err
		}
		instance.DesiredStatus = ServerStateDeleting
		return cm.executeInstanceDeleteAfterAuthorization(ctx, progress, id, localOnly, operationID, operationIdentity, instance, persistReceipt)
	}

	if peerDrainAuthorization == nil && cm.tasks != nil {
		if fact, found, factErr := cm.tasks.OperationFact(ctx, operationID, instancePeerDrainAuthorizedV1); factErr != nil {
			return fmt.Errorf("read peer-drain authorization fact for task %s: %w", operationID, factErr)
		} else if found {
			recovered, recoverErr := instancePeerDrainAuthorizationFromFact(fact)
			if recoverErr != nil {
				return recoverErr
			}
			peerDrainAuthorization = &recovered
		}
	}
	if peerDrainAuthorization == nil {
		if IsDeletingInstance(instance) {
			return fmt.Errorf("%w: legacy deleting instance %s has no immutable peer-drain authorization P", ErrInstanceDeleteInvariantConflict, instance.ID)
		}
		authorization, err := newInstancePeerDrainAuthorization(operationID, operationIdentity, instance, localOnly)
		if err != nil {
			return err
		}
		peerDrainAuthorization = &authorization
	}
	authorization := *peerDrainAuthorization
	if err := validateInstancePeerDrainAuthorization(authorization, operationID, operationIdentity, id, localOnly); err != nil {
		return err
	}
	if err := validateInstanceAgainstPeerDrainAuthorization(instance, authorization); err != nil {
		return err
	}
	fact, err := newInstancePeerDrainAuthorizationFact(authorization)
	if err != nil {
		return fmt.Errorf("build peer-drain authorization fact: %w", err)
	}

	resolvedReceipt, alreadyAccepted, err := cm.resolveInstancePeerDrainAuthorization(ctx, authorization)
	if err != nil {
		return err
	}
	continueDelete := func() error {
		instance.DesiredStatus = ServerStateDeleting
		return cm.executeInstanceDeleteAfterAuthorization(ctx, progress, id, localOnly, operationID, operationIdentity, instance, persistReceipt)
	}
	if alreadyAccepted {
		resolvedReceipt, err = cm.completeInstancePeerDrainAuthorization(ctx, authorization, fact, resolvedReceipt, true)
		if err != nil {
			return err
		}
		if cm.afterPeerDrainAuthorized != nil {
			cm.afterPeerDrainAuthorized(resolvedReceipt)
		}
		// P proves the immutable application authorization and prevents the
		// instance from being recreated through stale lifecycle work. Swarmion's
		// finalized tombstone is process-local, however, so recovery must still
		// establish a fresh route generation and complete Begin/Watch/Finalize
		// before provider I/O or D publication. Do not run Prepare here: the
		// provider may already be absent, and the scoped drain can safely finalize
		// a covered unknown peer from persisted checkpoint lineage.
		return cm.withInstancePeerDurableRemovalReady(ctx, instance, func() error {
			return continueDelete()
		})
	}

	// P is authoritatively absent. Keep the replicated desired status unchanged
	// while establishing fresh replacement coverage and a generation-matched
	// Swarmion drain. Finalize, P, provider I/O and D all remain under that lease.
	if IsDeletingInstance(instance) {
		return fmt.Errorf("%w: deleting instance %s has authoritative P absence", ErrInstanceDeleteInvariantConflict, instance.ID)
	}
	if status, ok := cm.db.SwarmionStatus(); !ok || strings.TrimSpace(status.PeerID) != strings.TrimSpace(authorization.AuthorPeerID) {
		return fmt.Errorf("%w: peer-drain authorization P task=%s requires original author %s", db.ErrOperationReceiptUnavailable, operationID, authorization.AuthorPeerID)
	}
	if err := progress(8, "preparing durable peer drain", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	if err := cm.prepareInstancePeerDrain(ctx, instance); err != nil {
		return err
	}
	if err := progress(30, "fencing p2p peer", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	if err := progress(40, "waiting for durable peer removal", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	return cm.withInstancePeerDurableRemovalReady(ctx, instance, func() error {
		unlock := cm.lockInstanceLifecycle(instance.ID)
		receipt, err := cm.completeInstancePeerDrainAuthorization(ctx, authorization, fact, db.PublishedWriteReceipt{}, false)
		if err != nil {
			unlock()
			return err
		}
		// The durable P fact closes the local QueueStart/UpdateInstance TOCTOU.
		// Release the in-process guard before provider I/O; subsequent lifecycle
		// writes are rejected by the SQL P tombstone while the route-generation
		// lease remains held through provider deletion and D publication.
		unlock()
		if cm.afterPeerDrainAuthorized != nil {
			cm.afterPeerDrainAuthorized(receipt)
		}
		return continueDelete()
	})
}

func (cm *Manager) executeInstanceDeleteAfterAuthorization(
	ctx context.Context,
	progress func(int, string, any) error,
	id string,
	localOnly bool,
	operationID string,
	operationIdentity instanceDeleteOperationIdentity,
	instance InstanceInfo,
	persistReceipt func(instanceDeleteOperationReceipt, int, string) error,
) error {
	if err := cm.assertInstanceLifecycleExecutor(instance, operationIdentity.AuthorPeerID); err != nil {
		return err
	}
	if err := progress(20, "removing instance apps", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	if err := cm.deleteAppsForInstance(ctx, instance.ID); err != nil {
		return err
	}

	if !localOnly && (instance.Kind == KindCloudVM || instance.Kind == KindLocalVM) {
		if err := ctx.Err(); err != nil {
			return err
		}
		provider, err := cm.GetProvider(instance.KindID)
		if err != nil {
			return fmt.Errorf("could not retrieve cloud '%s': %w", id, err)
		}

		computeProvider, err := requireComputeProvisioner(provider)
		if err != nil {
			return err
		}
		volumeProvider, err := requireVolumeProvisioner(provider)
		if err != nil {
			return err
		}
		if err := provider.Init(); err != nil {
			return fmt.Errorf("could not init cloud '%s': %w", id, err)
		}

		if err := progress(50, "loading provider instance", map[string]string{"instance_id": instance.ID, "provisioner": instance.KindID}); err != nil {
			return err
		}
		found := true
		vmInfo, providerInstanceID, err := getProviderInstanceInfo(computeProvider, instance)
		if err != nil {
			if errors.Is(err, ErrInstanceNotFound) {
				found = false
			} else {
				return fmt.Errorf("failed to get details for instance '%s': %w", id, err)
			}
		}

		// only delete cloud instance if found. Otherwise we proceed with removing it from local db
		if found {
			if vmInfo.Status == ServerStateRunning {
				log.Infof("Stopping instance '%s' (%s)", instance.Name, providerInstanceID)
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := progress(60, "stopping provider instance", map[string]string{"instance_id": instance.ID, "provider_instance_id": providerInstanceID}); err != nil {
					return err
				}
				err = computeProvider.StopInstance(providerInstanceID, instance.Location)
				if err != nil {
					log.Warnf("failed to stop instance '%s' before delete; attempting provider delete anyway: %s", id, err.Error())
				}
			}
			for _, vol := range vmInfo.Volumes {
				log.Infof("Deleting volume '%s' (%s) for instance '%s'", vol.Name, vol.VolumeID, id)
				if err := ctx.Err(); err != nil {
					return err
				}
				if err := progress(70, "deleting provider volume", map[string]string{"instance_id": instance.ID, "volume_id": vol.VolumeID}); err != nil {
					return err
				}
				err = volumeProvider.DeleteVolume(vol.VolumeID, instance.Location)
				if err != nil {
					return fmt.Errorf("could not delete volume '%s' for instance '%s': %w", vol.VolumeID, id, err)
				}
			}
			log.Infof("Deleting instance '%s' (%s)", instance.Name, providerInstanceID)
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := progress(80, "deleting provider instance", map[string]string{"instance_id": instance.ID, "provider_instance_id": providerInstanceID}); err != nil {
				return err
			}
			err = computeProvider.DeleteInstance(providerInstanceID, instance.Location)
			if err != nil {
				return fmt.Errorf("could not delete instance '%s': %w", id, err)
			}
			if cm.afterProviderDelete != nil {
				cm.afterProviderDelete(providerInstanceID)
			}
		}
	}

	if err := progress(90, "deleting instance records", map[string]string{"instance_id": instance.ID}); err != nil {
		return err
	}
	if err := cm.deleteInstanceRecords(ctx, operationID, operationIdentity, instance, persistReceipt); err != nil {
		return fmt.Errorf("failed to delete instance '%s': %w", id, err)
	}
	if err := cm.deleteInstanceSSHKey(instance.ID); err != nil {
		log.Warnf("failed to delete SSH key for instance '%s': %s", instance.Name, err.Error())
	}

	return nil
}

func instanceDeleteContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, 10*time.Minute)
}

func (cm *Manager) deleteInstanceRecords(
	ctx context.Context,
	operationID string,
	operationIdentity instanceDeleteOperationIdentity,
	instance InstanceInfo,
	persistReceipt func(instanceDeleteOperationReceipt, int, string) error,
) error {
	if cm == nil || cm.db == nil {
		return fmt.Errorf("provisioner manager database is not configured")
	}
	if strings.TrimSpace(instance.ID) == "" {
		return fmt.Errorf("instance ID is empty")
	}
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return fmt.Errorf("instance delete operation ID is empty")
	}
	if persistReceipt == nil {
		return fmt.Errorf("instance delete receipt persistence is not configured")
	}
	if err := cm.assertInstanceLifecycleExecutor(instance, operationIdentity.AuthorPeerID); err != nil {
		return err
	}
	effectFact, err := newInstanceDeleteEffectFact(operationID, operationIdentity)
	if err != nil {
		return fmt.Errorf("build instance delete effect fact: %w", err)
	}

	var (
		lastErr   error
		published db.PublishedWriteReceipt
	)
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			if lastErr != nil {
				return instanceDeletePublicationWithoutReceipt(fmt.Errorf("%w: previous publication error: %w", err, lastErr))
			}
			return instanceDeletePublicationWithoutReceipt(err)
		}
		publisher := cm.publishDeleteOperation
		if publisher == nil {
			publisher = func(ctx context.Context, operation db.PublishedWriteOperation, instance InstanceInfo) (db.PublishedWriteReceipt, error) {
				im, cmmd := createInstanceDeleteMapper(instance.ID)
				return db.DeleteAndInsertWithOperationReceiptContext(
					ctx,
					cm.db,
					operation,
					[]db.DeleteMapper{
						db.CreatePeerDeleteMapper(instance.PublicKey),
						createAppDeleteByInstanceMapper(instance.ID),
						im,
						cmmd,
					},
					[]db.InsertMapper{tasks.InsertOperationFactMapper(effectFact)},
				)
			}
		}
		var err error
		published, err = publisher(ctx, operationIdentity.publishedWriteOperation(), instance)
		if errors.Is(err, db.ErrPublishedWriteReceiptIdentityConflict) {
			return instanceDeletePublicationWithoutReceipt(err)
		}
		if errors.Is(err, db.ErrPublishedWriteNoChange) {
			// Stable operations consume their key even when SQL content is
			// unchanged. Reaching this error means the publisher supplied no
			// executable statements, not that the instance happened to be absent.
			return fmt.Errorf(
				"%w: instance %s delete operation %s supplied no executable statements: %w",
				ErrInstanceDeleteInvariantConflict,
				instance.ID,
				operationID,
				err,
			)
		}
		if published.HasExactEventIdentity() {
			if err != nil {
				log.Warnf(
					"tracking published instance delete after receipt-return error operation_id=%s instance_id=%s event_id=%s published_root=%s error=%s",
					operationID,
					instance.ID,
					published.EventID,
					published.PublishedRootHash,
					err.Error(),
				)
			}
			break
		}
		if err == nil {
			return instanceDeletePublicationWithoutReceipt(fmt.Errorf("instance delete did not publish an event receipt"))
		}
		if errors.Is(err, db.ErrOperationReceiptUnavailable) || !db.IsRetryablePublishedWriteError(err) {
			return instanceDeletePublicationWithoutReceipt(err)
		}
		lastErr = err
		sleep := time.Duration(attempt*2) * time.Second
		if sleep > 15*time.Second {
			sleep = 15 * time.Second
		}
		select {
		case <-ctx.Done():
		case <-time.After(sleep):
		}
	}

	if cm.afterInstanceDeletePublished != nil {
		cm.afterInstanceDeletePublished(published)
	}
	receipt := instanceDeleteReceiptFromPublished(operationID, operationIdentity, published)
	if err := persistReceipt(receipt, 92, "tracking published instance deletion"); err != nil {
		return fmt.Errorf(
			"persist instance delete receipt operation_id=%s event_id=%s published_root=%s: %w",
			receipt.OperationID,
			receipt.EventID,
			receipt.PublishedRootHash,
			err,
		)
	}
	return cm.completeInstanceDeleteReceipt(ctx, receipt, persistReceipt)
}

func instanceDeleteReceiptFromPublished(
	operationID string,
	identity instanceDeleteOperationIdentity,
	published db.PublishedWriteReceipt,
) instanceDeleteOperationReceipt {
	return instanceDeleteOperationReceipt{
		OperationID:           strings.TrimSpace(operationID),
		Operation:             instanceLifecycleOperationDelete,
		ExpectedInvariant:     identity.ExpectedInvariant,
		EventID:               published.EventID,
		PublishedRootHash:     published.PublishedRootHash,
		EventDigest:           published.EventDigest,
		AuthorSeq:             published.AuthorSeq,
		OperationIntentDigest: identity.IntentDigest,
		OperationAuthorPeerID: identity.AuthorPeerID,
		OutcomeUncertain:      published.OutcomeUncertain,
		CheckpointCommitID:    published.CheckpointCommitID,
		CheckpointRootHash:    published.CheckpointRootHash,
		Checkpointed:          published.Checkpointed,
	}
}

func validateInstanceDeleteOperationReceipt(
	receipt instanceDeleteOperationReceipt,
	identity instanceDeleteOperationIdentity,
	operationID string,
	instanceID string,
) error {
	operationID = strings.TrimSpace(operationID)
	instanceID = strings.TrimSpace(instanceID)
	if strings.TrimSpace(receipt.OperationID) == "" || receipt.OperationID != operationID {
		return fmt.Errorf("instance delete receipt operation ID %q does not match task %q", receipt.OperationID, operationID)
	}
	if receipt.Operation != instanceLifecycleOperationDelete {
		return fmt.Errorf("instance delete receipt operation is %q", receipt.Operation)
	}
	if receipt.ExpectedInvariant.Kind != instanceDeleteInvariantAbsent || strings.TrimSpace(receipt.ExpectedInvariant.InstanceID) == "" {
		return fmt.Errorf("instance delete receipt has invalid expected invariant: %+v", receipt.ExpectedInvariant)
	}
	if receipt.ExpectedInvariant.InstanceID != instanceID {
		return fmt.Errorf("instance delete receipt instance ID %q does not match task instance %q", receipt.ExpectedInvariant.InstanceID, instanceID)
	}
	if receipt.ExpectedInvariant != identity.ExpectedInvariant {
		return fmt.Errorf("instance delete receipt invariant does not match replicated operation identity")
	}
	if digest := strings.TrimSpace(receipt.OperationIntentDigest); digest == "" || digest != strings.TrimSpace(identity.IntentDigest) {
		return fmt.Errorf("instance delete receipt intent digest does not match replicated operation identity")
	}
	if author := strings.TrimSpace(receipt.OperationAuthorPeerID); author == "" || author != strings.TrimSpace(identity.AuthorPeerID) {
		return fmt.Errorf("instance delete receipt author does not match replicated operation identity")
	}
	eventID := strings.TrimSpace(receipt.EventID)
	eventIDBytes, eventIDErr := hex.DecodeString(eventID)
	if eventIDErr != nil || len(eventIDBytes) != 32 || hex.EncodeToString(eventIDBytes) != eventID {
		return fmt.Errorf("instance delete receipt has invalid event ID %q", receipt.EventID)
	}
	publishedRoot := strings.TrimSpace(receipt.PublishedRootHash)
	if swarmionprotocol.ParseRootHash(publishedRoot).IsZero() ||
		!swarmionprotocol.ParseCheckpointCommitID(publishedRoot).IsDoltCommitHash() {
		return fmt.Errorf("instance delete receipt has invalid published root %q", receipt.PublishedRootHash)
	}
	if eventID == "" || publishedRoot == "" {
		return fmt.Errorf("instance delete receipt is missing its exact event/root identity")
	}
	return nil
}

func (receipt instanceDeleteOperationReceipt) publishedWriteReceipt() db.PublishedWriteReceipt {
	return db.PublishedWriteReceipt{
		Committed:             !receipt.OutcomeUncertain,
		OutcomeUncertain:      receipt.OutcomeUncertain,
		Checkpointed:          receipt.Checkpointed,
		CommitHash:            receipt.CommitHash,
		EventID:               receipt.EventID,
		PublishedRootHash:     receipt.PublishedRootHash,
		EventDigest:           receipt.EventDigest,
		AuthorPeerID:          receipt.OperationAuthorPeerID,
		AuthorSeq:             receipt.AuthorSeq,
		OperationIntentDigest: receipt.OperationIntentDigest,
		CheckpointCommitID:    receipt.CheckpointCommitID,
		CheckpointRootHash:    receipt.CheckpointRootHash,
	}
}

func (receipt *instanceDeleteOperationReceipt) applyObservation(observation db.EventReceiptObservation) {
	if receipt == nil {
		return
	}
	receipt.CheckpointCommitID = observation.Receipt.CheckpointCommitID
	receipt.CheckpointRootHash = observation.Receipt.CheckpointRootHash
	receipt.Checkpointed = observation.Status.Checkpointed
	receipt.AppliedDurably = observation.Status.AppliedDurably
	if receipt.AppliedDurably {
		receipt.OutcomeUncertain = false
	}
	receipt.ContentCoverage = observation.Status.ContentCoverage
	receipt.ContentDurable = observation.Status.Durable
	receipt.DurableCheckpointCommitID = observation.Status.DurableCheckpointCommitID
	receipt.DurableCheckpointRootHash = observation.Status.DurableCheckpointRootHash
	receipt.QueryableRootHash = observation.Status.QueryableRootHash
	if observation.Status.DurableProofObservation == nil {
		receipt.Proof = nil
	} else {
		proof := *observation.Status.DurableProofObservation
		receipt.Proof = &proof
	}
}

func eventReceiptObservationFromError(err error) (db.EventReceiptObservation, bool) {
	var pendingErr *db.EventReceiptPendingError
	if errors.As(err, &pendingErr) {
		return pendingErr.Observation, true
	}
	var parkedErr *db.EventReceiptParkedError
	if errors.As(err, &parkedErr) {
		return parkedErr.Observation, true
	}
	return db.EventReceiptObservation{}, false
}

func (cm *Manager) completeInstanceDeleteReceipt(
	ctx context.Context,
	receipt instanceDeleteOperationReceipt,
	persistReceipt func(instanceDeleteOperationReceipt, int, string) error,
) error {
	tracker := cm.deleteReceiptBackend()
	if tracker == nil {
		return fmt.Errorf("instance delete receipt tracker is not configured")
	}
	if persistReceipt == nil {
		return fmt.Errorf("instance delete receipt persistence is not configured")
	}
	observation, err := tracker.WaitForPublishedWriteApplied(ctx, receipt.publishedWriteReceipt(), "verify instance delete event application")
	if err != nil {
		if latest, ok := eventReceiptObservationFromError(err); ok {
			receipt.applyObservation(latest)
			if persistErr := persistReceipt(receipt, 93, "instance deletion event unresolved"); persistErr != nil {
				return fmt.Errorf("persist unresolved instance delete receipt: %w", persistErr)
			}
		}
		return err
	}
	receipt.applyObservation(observation)
	if !receipt.AppliedDurably {
		return fmt.Errorf(
			"instance delete event did not reach applied_durably operation_id=%s event_id=%s published_root=%s",
			receipt.OperationID,
			receipt.EventID,
			receipt.PublishedRootHash,
		)
	}
	if err := persistReceipt(receipt, 94, "instance deletion event applied durably"); err != nil {
		return fmt.Errorf("persist applied instance delete receipt: %w", err)
	}
	checkpointCommitID := strings.TrimSpace(receipt.DurableCheckpointCommitID)
	if checkpointCommitID == "" {
		return fmt.Errorf(
			"instance delete applied_durably without a durable checkpoint commit operation_id=%s event_id=%s",
			receipt.OperationID,
			receipt.EventID,
		)
	}
	present, err := tracker.InstanceExistsAtCheckpoint(ctx, checkpointCommitID, receipt.ExpectedInvariant.InstanceID)
	if err != nil {
		return fmt.Errorf("query instance delete invariant at durable checkpoint %s: %w", checkpointCommitID, err)
	}
	if present {
		return fmt.Errorf(
			"%w: operation_id=%s instance_id=%s event_id=%s checkpoint_commit=%s durable_checkpoint_commit=%s content_coverage=%s",
			ErrInstanceDeleteInvariantConflict,
			receipt.OperationID,
			receipt.ExpectedInvariant.InstanceID,
			receipt.EventID,
			receipt.CheckpointCommitID,
			receipt.DurableCheckpointCommitID,
			receipt.ContentCoverage,
		)
	}
	return cm.assertInstancePeerRemoved(ctx, receipt.ExpectedInvariant.PeerID)
}

func (cm *Manager) markInstanceDeleting(ctx context.Context, instance InstanceInfo) error {
	if cm == nil || cm.db == nil || strings.TrimSpace(instance.ID) == "" || IsDeletingInstance(instance) {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instance.DesiredStatus = ServerStateDeleting
	im, _ := createInstanceUpdateMapper(instance)
	if _, err := db.UpdateWithAvailabilityContext(context.Background(), cm.db, im); err != nil {
		return fmt.Errorf("failed to mark instance '%s' as deleting: %w", instance.Name, err)
	}
	return nil
}

func (cm *Manager) deleteAppsForInstance(ctx context.Context, instanceID string) error {
	instanceID = strings.TrimSpace(instanceID)
	if cm == nil || cm.db == nil || instanceID == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := db.DeleteWithAvailabilityContext(ctx, cm.db, createAppDeleteByInstanceMapper(instanceID)); err != nil {
		return fmt.Errorf("failed to delete apps for instance '%s': %w", instanceID, err)
	}
	return nil
}

func (cm *Manager) assertInstancePeerRemoved(ctx context.Context, peerID string) error {
	peerID = strings.TrimSpace(peerID)
	if cm == nil || cm.db == nil || peerID == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	peerIDs, err := db.GetPeerIDs(cm.db)
	if err != nil {
		return err
	}
	if _, found := peerIDs[peerID]; found {
		return fmt.Errorf("peer table still contains %s", peerID)
	}
	return nil
}

func (cm *Manager) prepareInstancePeerDrain(ctx context.Context, instance InstanceInfo) error {
	if cm == nil || cm.db == nil {
		return fmt.Errorf("%w: provisioner database is unavailable", db.ErrReplicationPeerDrainUnavailable)
	}
	peerID, err := instance.GetPeerID()
	if err != nil {
		return fmt.Errorf("derive peer id for instance '%s': %w", instance.Name, err)
	}
	candidates, err := cm.replicationCandidatesExcluding(peerID)
	if err != nil {
		return fmt.Errorf("build remaining replication candidates for instance '%s': %w", instance.Name, err)
	}
	runtime := cm.replicationPeerDrainRuntime()
	if runtime == nil || !runtime.Available() {
		return fmt.Errorf("%w for instance '%s'", db.ErrReplicationPeerDrainUnavailable, instance.Name)
	}
	if err := runtime.Prepare(ctx, peerID, candidates); err != nil {
		return fmt.Errorf("prepare swarmion peer drain for instance '%s': %w", instance.Name, err)
	}
	return nil
}

func (cm *Manager) waitForInstancePeerDurableRemovalReady(ctx context.Context, instance InstanceInfo) error {
	return cm.withInstancePeerDurableRemovalReady(ctx, instance, nil)
}

const peerDrainFinalizeRetryDelay = 500 * time.Millisecond

func (cm *Manager) withInstancePeerDurableRemovalReady(
	ctx context.Context,
	instance InstanceInfo,
	afterFinalize func() error,
) error {
	if cm == nil {
		return fmt.Errorf("%w: provisioner manager is unavailable", db.ErrReplicationPeerDrainUnavailable)
	}
	peerID, err := instance.GetPeerID()
	if err != nil {
		return fmt.Errorf("derive peer id for instance '%s': %w", instance.Name, err)
	}
	runtime := cm.replicationPeerDrainRuntime()
	if runtime == nil || !runtime.Available() {
		return fmt.Errorf("%w for instance '%s'", db.ErrReplicationPeerDrainUnavailable, instance.Name)
	}
	routeFence := cm.replicationPeerRouteFence()
	if routeFence == nil {
		return fmt.Errorf("cannot drain swarmion peer %s without the application route fence", peerID)
	}
	drainCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		if err := drainCtx.Err(); err != nil {
			return fmt.Errorf("%w for instance '%s': %w", db.ErrReplicationPeerDrainPending, instance.Name, err)
		}
		fencedPeerID, generation, err := routeFence.FencePeer(instance)
		if err != nil {
			return fmt.Errorf("%w: fence peer for instance '%s': %w", db.ErrReplicationPeerDrainPending, instance.Name, err)
		}
		if fencedPeerID != peerID {
			return fmt.Errorf("fenced peer id %s does not match instance peer id %s", fencedPeerID, peerID)
		}

		status, err := runtime.Begin(drainCtx, peerID, generation)
		if err != nil {
			if !runtime.Available() {
				return fmt.Errorf("%w for instance '%s' while beginning generation %s: %w", db.ErrReplicationPeerDrainUnavailable, instance.Name, generation, err)
			}
			return fmt.Errorf("%w: begin generation %s for instance '%s': %w", db.ErrReplicationPeerDrainPending, generation, instance.Name, err)
		}
		restart, err := cm.observeInstancePeerDrainGeneration(
			drainCtx,
			instance,
			peerID,
			generation,
			status,
			runtime,
			routeFence,
			afterFinalize,
		)
		if err != nil {
			return err
		}
		if !restart {
			return nil
		}
	}
}

// observeInstancePeerDrainGeneration consumes only passive, generation-scoped
// status events. The application route remains fenced for the entire call. A
// true result means the watch was canceled and the caller must establish a new
// route generation before beginning another drain.
func (cm *Manager) observeInstancePeerDrainGeneration(
	ctx context.Context,
	instance InstanceInfo,
	peerID string,
	generation string,
	status swarmionapp.PeerDrainStatus,
	runtime replicationPeerDrainRuntime,
	routeFence replicationPeerRouteFence,
	afterFinalize func() error,
) (bool, error) {
	watchCtx, cancelWatch := context.WithCancel(ctx)
	events, err := runtime.Watch(watchCtx, peerID, generation)
	if err != nil {
		cancelWatch()
		if !runtime.Available() {
			return false, fmt.Errorf("%w for instance '%s' while watching generation %s: %w", db.ErrReplicationPeerDrainUnavailable, instance.Name, generation, err)
		}
		return false, fmt.Errorf("%w: watch generation %s for instance '%s': %w", db.ErrReplicationPeerDrainPending, generation, instance.Name, err)
	}
	if events == nil {
		cancelWatch()
		return false, fmt.Errorf("%w: watch generation %s for instance '%s' returned a nil event channel", db.ErrReplicationPeerDrainPending, generation, instance.Name)
	}
	defer cancelWatch()

	var (
		retryFinalize   bool
		lastFinalizeErr error
	)
	for {
		if !runtime.Available() {
			return false, fmt.Errorf("%w for instance '%s' after beginning generation %s", db.ErrReplicationPeerDrainUnavailable, instance.Name, generation)
		}
		if err := validatePeerDrainStatusIdentity(status, peerID, generation); err != nil {
			return false, fmt.Errorf("%w: observe generation %s for instance '%s': %w", db.ErrReplicationPeerDrainPending, generation, instance.Name, err)
		}
		log.Debugf("swarmion peer drain status for instance '%s': %s", instance.Name, db.PeerDrainStatusSummary(status))

		if peerDrainStatusInvalidatesGeneration(status) {
			log.Debugf(
				"invalidating swarmion peer drain generation %s for instance '%s' after status codes %v at heartbeat sequence %d",
				generation,
				instance.Name,
				status.BlockingReasonCodes,
				status.HeartbeatIngressFenceSequence,
			)
			return true, nil
		}
		if status.Finalized && (status.Active || !status.RouteGenerationMatches || !status.ReadyToFinalize ||
			status.PostFenceHeartbeatAccepted || len(status.BlockingReasonCodes) != 0) {
			return false, fmt.Errorf("%w: inconsistent finalized peer-drain status: %s", db.ErrReplicationPeerDrainPending, db.PeerDrainStatusSummary(status))
		}
		if status.ReadyToFinalize && (!status.PreFenceHeartbeatIngressObserved || len(status.BlockingReasonCodes) != 0) {
			return false, fmt.Errorf("%w: inconsistent ready peer-drain status: %s", db.ErrReplicationPeerDrainPending, db.PeerDrainStatusSummary(status))
		}

		if (status.ReadyToFinalize || status.Finalized) && !retryFinalize {
			var (
				finalized          swarmionapp.PeerDrainFinalizeResponse
				finalizeCalled     bool
				finalizeRuntimeErr error
				finalizeCompleted  bool
			)
			finalizeErr := routeFence.WithPeerFenceGeneration(ctx, peerID, generation, func() error {
				finalizeCalled = true
				finalized, finalizeRuntimeErr = runtime.Finalize(ctx, peerID, generation)
				if finalizeRuntimeErr != nil {
					return finalizeRuntimeErr
				}
				if !finalized.Finalized {
					return fmt.Errorf("%w: generation %s returned without finalization", db.ErrReplicationPeerDrainPending, generation)
				}
				if err := validatePeerDrainFinalizeResponse(finalized, peerID, generation); err != nil {
					return err
				}
				finalizeCompleted = true
				// The watch is no longer useful after successful Swarmion
				// completion. Stop it before potentially slow P/provider work while
				// retaining the route-generation lease held by this callback.
				cancelWatch()
				if afterFinalize != nil {
					return afterFinalize()
				}
				return nil
			})
			if finalizeErr == nil {
				return false, nil
			}
			// Finalize succeeded and the guarded phase continuation failed. Do
			// not re-finalize in this attempt; recovery resolves P first. If P
			// exhausted only explicitly-not-accepted retries, preserve this as a
			// deferred drain outcome. Its replicated instance snapshot remains
			// pre-delete, so a later attempt must establish a fresh route fence
			// and drain generation before trying P again.
			if finalizeCompleted {
				var noReceipt *instanceDeletePublicationWithoutReceiptError
				if errors.As(finalizeErr, &noReceipt) && db.IsRetryablePublishedWriteError(finalizeErr) {
					return false, fmt.Errorf(
						"%w: generation %s finalized but peer-drain authorization P was explicitly not accepted: %w",
						db.ErrReplicationPeerDrainPending,
						generation,
						finalizeErr,
					)
				}
				return false, finalizeErr
			}
			// A successful response with a malformed identity is not permission
			// to continue and must not be retried as though finalization failed.
			if finalizeRuntimeErr == nil && finalized.Finalized {
				return false, finalizeErr
			}
			// Losing the application generation lease before Finalize ran
			// requires a new fence; the old generation cannot be reused.
			if !finalizeCalled {
				return true, nil
			}

			finalizeStatusObserved := false
			if typedStatus, ok := peerDrainStatusFromTypedError(finalizeErr); ok {
				if err := validatePeerDrainStatusIdentity(typedStatus, peerID, generation); err != nil {
					return false, fmt.Errorf("%w: typed finalize status for generation %s: %w", db.ErrReplicationPeerDrainPending, generation, err)
				}
				status = typedStatus
				finalizeStatusObserved = true
			} else if finalized.Status.PeerID != "" || finalized.Status.RouteGeneration != "" {
				if err := validatePeerDrainStatusIdentity(finalized.Status, peerID, generation); err != nil {
					return false, fmt.Errorf("%w: finalize status for generation %s: %w", db.ErrReplicationPeerDrainPending, generation, err)
				}
				status = finalized.Status
				finalizeStatusObserved = true
			}

			if errors.Is(finalizeErr, swarmionapp.ErrPeerDrainGenerationInactive) || peerDrainStatusInvalidatesGeneration(status) {
				return true, nil
			}
			if errors.Is(finalizeErr, swarmionapp.ErrPeerDrainNotReady) {
				if !finalizeStatusObserved {
					return false, fmt.Errorf("%w: finalize reported not-ready without typed status for generation %s: %w", db.ErrReplicationPeerDrainPending, generation, finalizeErr)
				}
				if status.ReadyToFinalize || status.Finalized {
					return false, fmt.Errorf("%w: finalize reported not-ready with ready status for generation %s: %w", db.ErrReplicationPeerDrainPending, generation, finalizeErr)
				}
				lastFinalizeErr = finalizeErr
				continue
			}

			if !runtime.Available() {
				return false, fmt.Errorf("%w for instance '%s' while finalizing generation %s: %w", db.ErrReplicationPeerDrainUnavailable, instance.Name, generation, finalizeErr)
			}
			// Cache clearing can fail while the generation remains active and
			// ready. No status transition is emitted for that failure, so retry
			// Finalize explicitly after a bounded backoff while still consuming
			// invalidating watch events. This is not status polling.
			if finalizeStatusObserved && status.Active && status.RouteGenerationMatches && status.ReadyToFinalize {
				retryFinalize = true
				lastFinalizeErr = finalizeErr
			} else {
				return false, fmt.Errorf("%w: finalize generation %s for instance '%s': %w", db.ErrReplicationPeerDrainPending, generation, instance.Name, finalizeErr)
			}
		}

		var retryTimer *time.Timer
		var retry <-chan time.Time
		if retryFinalize {
			retryTimer = time.NewTimer(peerDrainFinalizeRetryDelay)
			retry = retryTimer.C
		}
		select {
		case <-ctx.Done():
			if retryTimer != nil {
				retryTimer.Stop()
			}
			if lastFinalizeErr != nil {
				return false, fmt.Errorf("%w for instance '%s': %s: last finalize error: %w: %w", db.ErrReplicationPeerDrainPending, instance.Name, db.PeerDrainStatusSummary(status), lastFinalizeErr, ctx.Err())
			}
			return false, fmt.Errorf("%w for instance '%s': %s: %w", db.ErrReplicationPeerDrainPending, instance.Name, db.PeerDrainStatusSummary(status), ctx.Err())
		case <-retry:
			retryFinalize = false
			continue
		case event, ok := <-events:
			wasWaitingToRetryFinalize := retryFinalize
			if retryTimer != nil {
				retryTimer.Stop()
			}
			retryFinalize = false
			if !ok {
				if ctx.Err() != nil {
					return false, fmt.Errorf("%w for instance '%s' while watching generation %s: %w", db.ErrReplicationPeerDrainPending, instance.Name, generation, ctx.Err())
				}
				if !runtime.Available() {
					return false, fmt.Errorf("%w for instance '%s' while watching generation %s", db.ErrReplicationPeerDrainUnavailable, instance.Name, generation)
				}
				return false, fmt.Errorf("%w: watch generation %s for instance '%s' closed before a terminal status", db.ErrReplicationPeerDrainPending, generation, instance.Name)
			}
			if event.Err != nil {
				if !runtime.Available() {
					return false, fmt.Errorf("%w for instance '%s' while watching generation %s: %w", db.ErrReplicationPeerDrainUnavailable, instance.Name, generation, event.Err)
				}
				return false, fmt.Errorf("%w: watch generation %s for instance '%s': %w", db.ErrReplicationPeerDrainPending, generation, instance.Name, event.Err)
			}
			status = event.Status
			// The initial watch snapshot can arrive after a cache-clear
			// failure. An unchanged ready snapshot is not a reason to bypass
			// the retry backoff; only an actual invalidating/not-ready state
			// cancels that scheduled Finalize retry.
			if wasWaitingToRetryFinalize &&
				status.Active && status.RouteGenerationMatches && status.ReadyToFinalize &&
				!peerDrainStatusInvalidatesGeneration(status) {
				retryFinalize = true
			}
		}
	}
}

func validatePeerDrainStatusIdentity(status swarmionapp.PeerDrainStatus, peerID, generation string) error {
	if status.PeerID != peerID || status.RouteGeneration != generation {
		return fmt.Errorf(
			"peer-drain status identity peer=%q generation=%q expected=%q/%q",
			status.PeerID,
			status.RouteGeneration,
			peerID,
			generation,
		)
	}
	return nil
}

func validatePeerDrainFinalizeResponse(response swarmionapp.PeerDrainFinalizeResponse, peerID, generation string) error {
	if response.PeerID != peerID || response.RouteGeneration != generation ||
		(response.Status.PeerID != "" && response.Status.PeerID != peerID) ||
		(response.Status.RouteGeneration != "" && response.Status.RouteGeneration != generation) {
		return fmt.Errorf(
			"%w: finalize response identity peer=%q generation=%q status_peer=%q status_generation=%q expected=%q/%q",
			db.ErrReplicationPeerDrainPending,
			response.PeerID,
			response.RouteGeneration,
			response.Status.PeerID,
			response.Status.RouteGeneration,
			peerID,
			generation,
		)
	}
	return nil
}

func peerDrainStatusHasReason(status swarmionapp.PeerDrainStatus, reason swarmionapp.PeerDrainBlockingReason) bool {
	for _, candidate := range status.BlockingReasonCodes {
		if candidate == reason {
			return true
		}
	}
	return false
}

func peerDrainStatusInvalidatesGeneration(status swarmionapp.PeerDrainStatus) bool {
	if status.Finalized {
		return false
	}
	return status.PostFenceHeartbeatAccepted ||
		peerDrainStatusHasReason(status, swarmionapp.PeerDrainBlockingReasonPostFenceHeartbeatAccepted) ||
		peerDrainStatusHasReason(status, swarmionapp.PeerDrainBlockingReasonNewerRouteGenerationActive) ||
		peerDrainStatusHasReason(status, swarmionapp.PeerDrainBlockingReasonNoActiveGeneration) ||
		!status.Active ||
		!status.RouteGenerationMatches
}

func peerDrainStatusFromTypedError(err error) (swarmionapp.PeerDrainStatus, bool) {
	var notReady *swarmionapp.PeerDrainNotReadyError
	if !errors.As(err, &notReady) || notReady == nil {
		return swarmionapp.PeerDrainStatus{}, false
	}
	return notReady.Status, true
}

func (cm *Manager) replicationCandidatesExcluding(peerID string) ([]db.ReplicationCandidate, error) {
	peerID = strings.TrimSpace(peerID)
	instances, err := cm.GetInstances(false)
	if err != nil {
		return nil, err
	}
	candidates := make([]db.ReplicationCandidate, 0, len(instances)+1)
	for _, instance := range instances {
		if strings.TrimSpace(instance.ID) == "" || strings.TrimSpace(instance.PublicKey) == "" || !IsActiveInstance(instance) {
			continue
		}
		instancePeerID, err := instance.GetPeerID()
		if err != nil || instancePeerID == peerID {
			continue
		}
		candidates = append(candidates, db.ReplicationCandidate{
			PeerID:      instancePeerID,
			DeviceClass: db.ReplicationDeviceClassForMachine(instance.Kind, instance.KindID),
			Priority:    instance.ReplicationPriority,
		})
	}
	if cm.um == nil {
		return candidates, nil
	}
	devices, err := cm.um.GetAllDevices(false)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		if strings.TrimSpace(device.ID) == "" || strings.TrimSpace(device.PublicKey) == "" {
			continue
		}
		devicePeerID, err := db.PeerIDFromPublicKeyString(device.PublicKey)
		if err != nil || devicePeerID == peerID {
			continue
		}
		candidates = append(candidates, db.ReplicationCandidate{
			PeerID:      devicePeerID,
			DeviceClass: db.ReplicationDeviceClassForUserDeviceName(device.Name),
			Priority:    device.ReplicationPriority,
		})
	}
	return candidates, nil
}

// StartInstance starts an instance
func (cm *Manager) StartInstance(id string) (tasks.Record, error) {
	return cm.QueueStartInstance(id)
}

// StopInstance stops an instance
func (cm *Manager) StopInstance(id string) (tasks.Record, error) {
	return cm.QueueStopInstance(id)
}

// TunnelInstance creates and SSH tunnel to the instance
func (cm *Manager) TunnelInstance(id string) error {
	instanceInfo, err := cm.GetInstance(id)
	if err != nil {
		return fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}

	localKey, err := cm.sm.GetLocalKey()
	if err != nil {
		return err
	}

	log.Infof("creating SSH tunnel to instance '%s', using ip '%s'", instanceInfo.Name, instanceInfo.PublicIP)
	tunnel := pcrypto.NewTunnel(instanceInfo.PublicIP+":22", "root", localKey.SSHAuth(), "localhost:8080")
	localPort, err := tunnel.Start()
	if err != nil {
		return fmt.Errorf("error while creating the SSH tunnel: %w", err)
	}

	quit := make(chan interface{}, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go catchSignals(sigs, quit)

	log.Infof("SSH tunnel ready. Use 'http://localhost:%d/' to access the instance dashboard. Once finished, press CTRL+C to terminate the SSH tunnel", localPort)

	// waiting for a SIGTERM or SIGINT
	<-quit

	log.Info("CTRL+C received. Terminating the SSH tunnel")
	err = tunnel.Close()
	if err != nil {
		return fmt.Errorf("error while terminating the SSH tunnel: %w", err)
	}
	log.Info("SSH tunnel terminated successfully")
	return nil
}

// LogsRemoteInstance retrieves the Protos logs from an instance.
func (cm *Manager) LogsRemoteInstance(id string) (string, error) {
	instanceInfo, err := cm.getInstanceRecord(id)
	if err != nil {
		return "", err
	}

	var providerLogsErr error
	if logs, handled, err := cm.logsRemoteInstanceViaProvisioner(instanceInfo); handled {
		if err == nil {
			return logs, nil
		}
		providerLogsErr = err
	}

	auth, err := cm.sshAuthForInstance(instanceInfo)
	if err != nil {
		if providerLogsErr != nil {
			return "", fmt.Errorf("provisioner log retrieval failed: %w; SSH auth failed: %w", providerLogsErr, err)
		}
		return "", err
	}

	sshCon, err := pcrypto.NewConnection(instanceInfo.PublicIP, "root", auth, 10)
	if err != nil {
		if providerLogsErr != nil {
			return "", fmt.Errorf("provisioner log retrieval failed: %w; SSH connection failed: %w", providerLogsErr, err)
		}
		return "", err
	}
	defer sshCon.Close()

	output, err := pcrypto.ExecuteCommand("cat /var/log/protos.log", sshCon)
	if err != nil {
		if providerLogsErr != nil {
			return "", fmt.Errorf("provisioner log retrieval failed: %w; SSH log retrieval failed: %w", providerLogsErr, err)
		}
		return "", err
	}
	return output, nil
}

func (cm *Manager) logsRemoteInstanceViaProvisioner(instanceInfo InstanceInfo) (string, bool, error) {
	if err := cm.assertInstanceLifecycleExecutor(instanceInfo, ""); err != nil {
		return "", true, err
	}
	provider, err := cm.GetProvider(instanceInfo.KindID)
	if err != nil && instanceInfo.Kind == KindLocalVM {
		provider, err = cm.GetProvisionerOrDefault(instanceInfo.KindID)
	}
	if err != nil {
		return "", false, err
	}
	logsProvider, ok := provider.(InstanceLogsProvider)
	if !ok {
		return "", false, nil
	}
	if err := provider.Init(); err != nil {
		return "", true, fmt.Errorf("failed to initialize provisioner '%s': %w", instanceInfo.KindID, err)
	}
	ref := firstNonEmptyString(instanceInfo.ProviderResourceID, instanceInfo.Name, instanceInfo.ID)
	if ref == "" {
		return "", true, fmt.Errorf("instance log reference is empty for '%s'", instanceInfo.Name)
	}
	logs, err := logsProvider.InstanceLogs(ref, instanceInfo.Location)
	if err != nil {
		return "", true, err
	}
	return logs, true, nil
}

func (cm *Manager) sshAuthForInstance(instance InstanceInfo) (ssh.AuthMethod, error) {
	keyPath, err := instanceSSHKeyPath(instance.ID)
	if err != nil {
		return nil, err
	}
	auth, err := cm.sm.NewAuthFromKeyFile(keyPath)
	if err == nil {
		return auth, nil
	}
	if !os.IsNotExist(err) {
		log.Warnf("failed to load SSH key for instance '%s': %s", instance.Name, err.Error())
	}

	localKey, localErr := cm.sm.GetLocalKey()
	if localErr != nil {
		return nil, fmt.Errorf("failed to load instance SSH key: %w; failed to load local SSH key: %w", err, localErr)
	}
	return localKey.SSHAuth(), nil
}

// GetInstance retrieves an instance from the db and returns it
func (cm *Manager) GetInstance(id string) (InstanceInfo, error) {

	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return instance, err
	}

	if IsDeletingInstance(instance) {
		instance.Status = ServerStateDeleting
		return instance, nil
	}

	if strings.TrimSpace(instance.PublicKey) == "" {
		instance.Status = cm.pendingInstanceStatus(instance)
		return instance, nil
	}

	status, err := cm.retrieveInstanceStatus(instance)
	if err != nil {
		log.Errorf("failed to retrieve status from instance '%s': %s", instance.Name, err.Error())
		instance.Status = "n/a"
	} else {
		instance.Status = status
	}
	return instance, nil
}

// GetDeclaredInstance retrieves an instance from declarative state without
// asking the provider for live runtime status.
func (cm *Manager) GetDeclaredInstance(id string) (InstanceInfo, error) {
	return cm.getInstanceRecord(id)
}

func (cm *Manager) getInstanceRecord(id string) (InstanceInfo, error) {
	iqm := createInstanceQueryMapper(id)
	instance, err := db.SelectOne(cm.db, iqm)
	if err == nil {
		return instance, nil
	}
	instanceByName, nameErr := db.SelectOne(cm.db, createInstanceQueryByNameMapper(id))
	if nameErr != nil {
		return instance, fmt.Errorf("failed to retrieve instance: %w", err)
	}
	return instanceByName, nil
}

// GetInstances returns all the instances from the db
func (cm *Manager) GetInstances(excludeLocalInstance bool) ([]InstanceInfo, error) {

	publicKey := ""
	if excludeLocalInstance {
		key, err := cm.sm.GetLocalKey()
		if err != nil {
			return nil, fmt.Errorf("could not retrieve local key: %w", err)
		}
		publicKey = key.PublicString()
	}

	instances, err := db.SelectMultiple(cm.db, createInstanceQueryAllMapper(publicKey))
	if err != nil {
		return instances, fmt.Errorf("failed to retrieve instances: %w", err)
	}

	return instances, nil
}

func (cm *Manager) retrieveInstanceStatus(instance InstanceInfo) (string, error) {
	if err := cm.assertInstanceLifecycleExecutor(instance, ""); err != nil {
		return "", err
	}
	provider, err := cm.GetProvider(instance.KindID)
	if err != nil && instance.Kind == KindLocalVM {
		provider, err = cm.GetProvisionerOrDefault(instance.KindID)
	}
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
	}
	computeProvider, err := requireComputeProvisioner(provider)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
	}
	if err := provider.Init(); err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())
	}
	instanceInfo, _, err := getProviderInstanceInfo(computeProvider, instance)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve status for instance '%s': %s", instance.Name, err.Error())

	}

	return instanceInfo.Status, nil
}

// GetInstances returns all the instances from the db
func (cm *Manager) GetInstancesWithUpdatedStatus() ([]InstanceInfo, error) {
	instances, err := db.SelectMultiple(cm.db, createInstanceQueryAllMapper(""))
	if err != nil {
		return instances, fmt.Errorf("failed to retrieve instances: %w", err)
	}

	for i, instance := range instances {
		if IsDeletingInstance(instance) {
			instances[i].Status = ServerStateDeleting
			continue
		}
		if strings.TrimSpace(instance.PublicKey) == "" {
			instances[i].Status = cm.pendingInstanceStatus(instance)
			continue
		}
		status, err := cm.retrieveInstanceStatus(instance)
		if err != nil {
			log.Warn(err.Error())
			instances[i].Status = "n/a"
		} else {
			instances[i].Status = status
		}
	}
	return instances, nil
}

func (cm *Manager) reconcileDesiredInstance(ctx context.Context, progress func(int, string, any) error, id string) (bool, InstanceInfo, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any) error { return nil }
	}
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return false, InstanceInfo{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if err := cm.assertInstanceLifecycleExecutor(instance, ""); err != nil {
		return false, instance, err
	}
	if IsDeletingInstance(instance) {
		return false, instance, nil
	}
	if authorized, authErr := cm.instancePeerDrainAuthorizationExists(ctx, instance.ID); authErr != nil {
		return false, instance, fmt.Errorf("inspect peer-drain authorization for instance %s: %w", instance.Name, authErr)
	} else if authorized {
		return false, instance, nil
	}
	desiredStatus := normalizeDesiredInstanceStatus(instance.DesiredStatus)
	if desiredStatus == "" || strings.TrimSpace(instance.PublicKey) == "" {
		return false, instance, nil
	}
	if err := ctx.Err(); err != nil {
		return false, instance, err
	}
	if err := progress(20, "loading provisioner", map[string]string{"instance_id": instance.ID, "provisioner": instance.KindID}); err != nil {
		return false, instance, err
	}
	provisioner, err := cm.GetProvisioner(instance.KindID)
	if err != nil {
		return false, instance, err
	}
	if err := provisioner.Init(); err != nil {
		return false, instance, fmt.Errorf("init provisioner %s: %w", instance.KindID, err)
	}
	if err := ctx.Err(); err != nil {
		return false, instance, err
	}
	instance.DesiredStatus = desiredStatus
	if err := progress(50, "applying desired instance state", map[string]string{
		"instance_id":    instance.ID,
		"desired_status": desiredStatus,
	}); err != nil {
		return false, instance, err
	}
	updated, err := reconcileProvisionerInstance(provisioner, instance)
	if err != nil {
		return false, instance, fmt.Errorf("reconcile instance %s: %w", instance.Name, err)
	}
	updated = mergedReconciledInstance(instance, updated)
	if persistentInstanceEqual(instance, updated) {
		return false, updated, nil
	}
	if err := progress(85, "saving observed instance state", map[string]string{"instance_id": instance.ID}); err != nil {
		return false, updated, err
	}
	im, cmm := createInstanceLifecycleUpdateMapper(updated)
	if _, err := db.UpdateWithAvailabilityContext(ctx, cm.db, im, cmm); err != nil {
		return false, updated, fmt.Errorf("save reconciled instance %s: %w", instance.Name, err)
	}
	if authorized, authErr := cm.instancePeerDrainAuthorizationExists(ctx, instance.ID); authErr != nil {
		return false, updated, fmt.Errorf("verify peer-drain authorization after reconciling instance %s: %w", instance.Name, authErr)
	} else if authorized {
		current, currentErr := cm.getInstanceRecord(instance.ID)
		if currentErr != nil {
			return false, updated, currentErr
		}
		return false, current, nil
	}
	return true, updated, nil
}

func (cm *Manager) pendingInstanceStatus(instance InstanceInfo) string {
	if cm == nil || cm.tasks == nil {
		return ServerStateChanging
	}
	task, found, err := cm.tasks.LatestForSubject(InstanceDeploymentTaskStream, taskSubjectInstance, instance.ID)
	if err != nil {
		log.Debugf("failed to load deployment task for pending instance '%s': %s", instance.Name, err.Error())
		return ServerStateChanging
	}
	if !found {
		return ServerStateChanging
	}
	status := string(task.Status)
	if task.Message != "" {
		status += ": " + task.Message
	}
	if task.ErrorMessage != "" {
		status += ": " + task.ErrorMessage
	}
	return status
}

func reconcileProvisionerInstance(provisioner Provisioner, instance InstanceInfo) (InstanceInfo, error) {
	if reconciler, ok := provisioner.(InstanceReconciler); ok {
		return reconciler.ReconcileInstance(instance)
	}
	computeProvider, err := requireComputeProvisioner(provisioner)
	if err != nil {
		return InstanceInfo{}, err
	}
	return reconcileComputeInstance(computeProvider, instance)
}

func reconcileComputeInstance(provider ComputeProvisioner, instance InstanceInfo) (InstanceInfo, error) {
	current, providerInstanceID, err := getProviderInstanceInfo(provider, instance)
	if err != nil {
		return InstanceInfo{}, err
	}
	if current.Status == ServerStateChanging {
		current.DesiredStatus = instance.DesiredStatus
		return current, nil
	}
	switch strings.ToLower(strings.TrimSpace(instance.DesiredStatus)) {
	case ServerStateRunning:
		if current.Status != ServerStateRunning {
			if err := provider.StartInstance(providerInstanceID, instance.Location); err != nil {
				return InstanceInfo{}, err
			}
			current, err = provider.GetInstanceInfo(providerInstanceID, instance.Location)
			if err != nil {
				return InstanceInfo{}, err
			}
		}
	case ServerStateStopped:
		if current.Status != ServerStateStopped {
			if err := provider.StopInstance(providerInstanceID, instance.Location); err != nil {
				return InstanceInfo{}, err
			}
			current, err = provider.GetInstanceInfo(providerInstanceID, instance.Location)
			if err != nil {
				return InstanceInfo{}, err
			}
		}
	}
	current.DesiredStatus = instance.DesiredStatus
	return current, nil
}

func normalizeDesiredInstanceStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case ServerStateRunning:
		return ServerStateRunning
	case ServerStateStopped:
		return ServerStateStopped
	default:
		return ""
	}
}

func lifecycleDesiredSignature(instance InstanceInfo, desiredStatus string) string {
	return strings.Join([]string{
		strings.TrimSpace(instance.ID),
		strings.TrimSpace(instance.Kind),
		strings.TrimSpace(instance.KindID),
		strings.TrimSpace(instance.ProviderResourceID),
		strings.TrimSpace(instance.LifecycleOwnerPeerID),
		strings.TrimSpace(instance.PublicKey),
		strings.TrimSpace(instance.Location),
		strings.TrimSpace(desiredStatus),
	}, "|")
}

func (cm *Manager) lifecycleSignatureCurrent(instanceID string, sig string) bool {
	if cm == nil {
		return false
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	if cm.lifecycleSig == nil {
		cm.lifecycleSig = map[string]string{}
	}
	return cm.lifecycleSig[instanceID] == sig
}

func (cm *Manager) setLifecycleSignature(instanceID string, sig string) {
	if cm == nil || strings.TrimSpace(instanceID) == "" || strings.TrimSpace(sig) == "" {
		return
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	if cm.lifecycleSig == nil {
		cm.lifecycleSig = map[string]string{}
	}
	cm.lifecycleSig[instanceID] = sig
}

func (cm *Manager) clearLifecycleSignature(instanceID string) {
	if cm == nil || strings.TrimSpace(instanceID) == "" {
		return
	}
	cm.lifecycleMu.Lock()
	defer cm.lifecycleMu.Unlock()
	delete(cm.lifecycleSig, instanceID)
}

func mergedReconciledInstance(current InstanceInfo, observed InstanceInfo) InstanceInfo {
	if observed.ProviderResourceID == "" && observed.ID != "" && observed.ID != current.ID {
		observed.ProviderResourceID = observed.ID
	}
	observed.ID = current.ID
	if observed.Name == "" {
		observed.Name = current.Name
	}
	if observed.PublicKey == "" {
		observed.PublicKey = current.PublicKey
	}
	if observed.Kind == "" {
		observed.Kind = current.Kind
	}
	if observed.KindID == "" {
		observed.KindID = current.KindID
	}
	if observed.ProviderResourceID == "" {
		observed.ProviderResourceID = current.ProviderResourceID
	}
	// Provider observations never own lifecycle authority. Preserve the
	// immutable application assignment from the persisted row.
	observed.LifecycleOwnerPeerID = current.LifecycleOwnerPeerID
	if observed.DesiredStatus == "" {
		observed.DesiredStatus = current.DesiredStatus
	}
	if observed.ReplicationPriority <= 0 {
		observed.ReplicationPriority = current.ReplicationPriority
	}
	if observed.Location == "" {
		observed.Location = current.Location
	}
	if observed.Architecture == "" {
		observed.Architecture = current.Architecture
	}
	return observed
}

func persistentInstanceEqual(a InstanceInfo, b InstanceInfo) bool {
	return a.ID == b.ID &&
		a.Name == b.Name &&
		a.Kind == b.Kind &&
		a.KindID == b.KindID &&
		a.ProviderResourceID == b.ProviderResourceID &&
		a.LifecycleOwnerPeerID == b.LifecycleOwnerPeerID &&
		a.DesiredStatus == b.DesiredStatus &&
		a.ReplicationPriority == b.ReplicationPriority &&
		a.PublicIP == b.PublicIP &&
		a.Location == b.Location &&
		a.Architecture == b.Architecture &&
		a.PublicKey == b.PublicKey
}

func (cm *Manager) uploadLocalImageImperative(ctx context.Context, progress func(int, string, any, bool) error, imagePath string, imageName string, cloudName string, cloudLocation string, timeout time.Duration) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if progress == nil {
		progress = func(int, string, any, bool) error { return nil }
	}
	errMsg := fmt.Sprintf("failed to upload local image '%s' to cloud '%s'", imagePath, cloudName)

	if err := progress(5, "validating image", map[string]string{
		"image_path":  imagePath,
		"image_name":  imageName,
		"provisioner": cloudName,
		"location":    cloudLocation,
	}, true); err != nil {
		return "", err
	}
	// check local image file
	finfo, err := os.Stat(imagePath)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if !finfo.IsDir() && finfo.Size() == 0 {
		return "", fmt.Errorf("%s: Image '%s' has 0 bytes", errMsg, imagePath)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := progress(15, "loading provisioner", map[string]string{
		"provisioner": cloudName,
		"location":    cloudLocation,
	}, true); err != nil {
		return "", err
	}
	provider, err := cm.GetProvider(cloudName)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}

	imageProvider, err := requireImageProvisioner(provider)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := provider.Init(); err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if err := progress(30, "checking existing images", map[string]string{
		"image_name": imageName,
		"location":   cloudLocation,
	}, true); err != nil {
		return "", err
	}
	// find image
	images, err := imageProvider.GetImages()
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	for _, img := range images {
		if img.Location == cloudLocation && img.Name == imageName {
			return "", fmt.Errorf("%s: Found an image with the same name", errMsg)
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	uploadDetails := map[string]string{
		"image_path":  imagePath,
		"image_name":  imageName,
		"provisioner": cloudName,
		"location":    cloudLocation,
	}
	if err := progress(60, "uploading image", uploadDetails, true); err != nil {
		return "", err
	}
	if err := progress(60, "upload in progress", uploadDetails, false); err != nil {
		return "", err
	}
	// upload image
	id, err := imageProvider.UploadLocalImage(ctx, imagePath, imageName, cloudLocation, timeout, uploadTaskProgressSink(progress, uploadDetails))
	if err != nil {
		return "", fmt.Errorf("%s: %w", errMsg, err)
	}
	if err := progress(95, "image uploaded", map[string]string{
		"image_id":    id,
		"image_name":  imageName,
		"provisioner": cloudName,
		"location":    cloudLocation,
	}, true); err != nil {
		return "", err
	}
	return id, nil
}

func uploadTaskProgressSink(progress func(int, string, any, bool) error, baseDetails map[string]string) UploadProgressFunc {
	lastProgress := -1
	lastAt := time.Time{}
	return func(update UploadProgress) error {
		total := update.TotalBytes
		transferred := update.BytesTransferred
		if transferred < 0 {
			transferred = 0
		}
		if total > 0 && transferred > total {
			transferred = total
		}
		taskProgress := 60
		percent := int64(0)
		if total > 0 {
			percent = transferred * 100 / total
			taskProgress = 60 + int(transferred*30/total)
			if taskProgress > 90 {
				taskProgress = 90
			}
		}
		if taskProgress == lastProgress && time.Since(lastAt) < 2*time.Second {
			return nil
		}
		lastProgress = taskProgress
		lastAt = time.Now()
		details := map[string]any{
			"bytes_uploaded":     transferred,
			"archive_size_bytes": total,
			"percent":            percent,
			"phase":              update.Phase,
		}
		for key, value := range baseDetails {
			details[key] = value
		}
		message := strings.TrimSpace(update.Message)
		if message == "" {
			message = "upload in progress"
		}
		return progress(taskProgress, message, details, false)
	}
}
