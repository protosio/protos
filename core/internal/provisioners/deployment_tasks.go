package provisioners

import (
	"bytes"
	"context"
	"crypto/sha256"
	stdsql "database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/release"
	"github.com/protosio/protos/internal/tasks"
)

const (
	InstanceDeploymentTaskStream     = "provisioners.instance.deploy"
	InstanceLifecycleTaskStream      = "provisioners.instance.lifecycle"
	ProvisionerImageUploadTaskStream = "provisioners.image.upload"

	taskSubjectInstance         = "instance"
	taskSubjectProvisionerImage = "provisioner_image"

	instanceLifecycleOperationReconcile = "reconcile"
	instanceLifecycleOperationDelete    = "delete"
	instanceDeleteInvariantAbsent       = "instance_absent"
	instanceDeleteRecoveryObserveLimit  = 5 * time.Second
	instanceDeleteRecoveryModel         = "immutable_operation_facts"
	instancePeerDrainAuthorizationFact  = "instance_peer_drain_authorization"

	// Instance delete is an idempotent multi-phase lifecycle (mark deleting,
	// remove apps, durable peer removal, provider stop, record deletion). Allowing
	// a bounded number of attempts lets a delete that fails on a transient
	// control-plane error (APIC deadline, host-agent network reconfigure) resume
	// itself on the next task tick instead of needing a fresh RemoveInstance call.
	instanceDeleteMaxAttempts = 3
)

type deployInstanceTaskPayload struct {
	PendingInstanceID string          `json:"pending_instance_id"`
	InstanceName      string          `json:"instance_name"`
	CloudName         string          `json:"cloud_name"`
	CloudLocation     string          `json:"cloud_location"`
	Release           release.Release `json:"release"`
	MachineType       string          `json:"machine_type"`
}

type deployInstanceTaskResult struct {
	PendingInstanceID  string `json:"pending_instance_id"`
	InstanceID         string `json:"instance_id"`
	ProviderResourceID string `json:"provider_resource_id"`
	PublicIP           string `json:"public_ip"`
	PublicKey          string `json:"public_key"`
}

type instanceLifecycleTaskPayload struct {
	InstanceID             string                           `json:"instance_id"`
	InstanceName           string                           `json:"instance_name"`
	Operation              string                           `json:"operation"`
	DesiredStatus          string                           `json:"desired_status"`
	LocalOnly              bool                             `json:"local_only"`
	DesiredSig             string                           `json:"desired_sig,omitempty"`
	RequestedByAPI         bool                             `json:"requested_by_api,omitempty"`
	RecoveryModel          string                           `json:"recovery_model,omitempty"`
	DeleteOperation        *instanceDeleteOperationIdentity `json:"delete_operation,omitempty"`
	PeerDrainAuthorization *instancePeerDrainAuthorization  `json:"peer_drain_authorization,omitempty"`
}

type instanceLifecycleTaskResult struct {
	InstanceID    string                          `json:"instance_id"`
	InstanceName  string                          `json:"instance_name"`
	Operation     string                          `json:"operation"`
	DesiredStatus string                          `json:"desired_status,omitempty"`
	Deleted       bool                            `json:"deleted,omitempty"`
	Changed       bool                            `json:"changed,omitempty"`
	DeleteReceipt *instanceDeleteOperationReceipt `json:"delete_receipt,omitempty"`
}

type instanceDeleteInvariant struct {
	Kind       string `json:"kind"`
	InstanceID string `json:"instance_id"`
	PeerID     string `json:"peer_id,omitempty"`
}

// instanceDeleteOperationIdentity is replicated before the final delete can
// publish. It is sufficient to resolve the original Swarmion event after a
// process restart even when DeleteReceipt was never checkpointed.
type instanceDeleteOperationIdentity struct {
	Key               string                  `json:"key"`
	IntentDigest      string                  `json:"intent_digest"`
	AuthorPeerID      string                  `json:"author_peer_id"`
	ExpectedInvariant instanceDeleteInvariant `json:"expected_invariant"`
}

// instancePeerDrainAuthorization is immutable, replicated phase authority for
// provider destruction. It is created with the delete task, before execution,
// so the exact operation P can always be resolved after a crash even though P
// changes the live instance status from its captured pre-status to deleting.
// It intentionally contains no process generation, process-local executor,
// attempt or time. The persisted lifecycle owner is immutable application
// authority and therefore is part of the snapshot and operation digest.
type instancePeerDrainAuthorization struct {
	Key             string                          `json:"key"`
	IntentDigest    string                          `json:"intent_digest"`
	AuthorPeerID    string                          `json:"author_peer_id"`
	DeleteOperation instanceDeleteOperationIdentity `json:"delete_operation"`
	TaskID          string                          `json:"task_id"`
	InstanceID      string                          `json:"instance_id"`
	PeerID          string                          `json:"peer_id"`
	LocalOnly       bool                            `json:"local_only"`
	Instance        instancePeerDrainInstance       `json:"instance"`
}

// instancePeerDrainInstance is the complete persisted machine/provider row
// captured before P. P's CAS covers every field and changes only DesiredStatus.
type instancePeerDrainInstance struct {
	Name                 string `json:"name"`
	Kind                 string `json:"kind"`
	KindID               string `json:"kind_id"`
	ProviderResourceID   string `json:"provider_resource_id"`
	PreDesiredStatus     string `json:"pre_desired_status"`
	ReplicationPriority  int    `json:"replication_priority"`
	PublicIP             string `json:"public_ip"`
	Location             string `json:"location"`
	Architecture         string `json:"architecture"`
	PublicKey            string `json:"public_key"`
	LifecycleOwnerPeerID string `json:"lifecycle_owner_peer_id"`
}

// instanceDeleteOperationReceipt is the durable recovery identity for one
// provisioner delete. EventID and PublishedRootHash are an inseparable pair;
// the historical root alone is never used as the operation identity.
type instanceDeleteOperationReceipt struct {
	OperationID       string                  `json:"operation_id"`
	Operation         string                  `json:"operation"`
	ExpectedInvariant instanceDeleteInvariant `json:"expected_invariant"`

	EventID               string `json:"event_id"`
	PublishedRootHash     string `json:"published_root_hash"`
	OperationIntentDigest string `json:"operation_intent_digest,omitempty"`
	OperationAuthorPeerID string `json:"operation_author_peer_id,omitempty"`
	OutcomeUncertain      bool   `json:"outcome_uncertain,omitempty"`

	CheckpointCommitID        string `json:"checkpoint_commit_id,omitempty"`
	CheckpointRootHash        string `json:"checkpoint_root_hash,omitempty"`
	DurableCheckpointCommitID string `json:"durable_checkpoint_commit_id,omitempty"`
	DurableCheckpointRootHash string `json:"durable_checkpoint_root_hash,omitempty"`
	QueryableRootHash         string `json:"queryable_root_hash,omitempty"`

	Checkpointed    bool                                           `json:"checkpointed,omitempty"`
	AppliedDurably  bool                                           `json:"applied_durably,omitempty"`
	ContentCoverage swarmionapp.BranchEventContentCoverage         `json:"content_coverage,omitempty"`
	ContentDurable  bool                                           `json:"content_durable,omitempty"`
	Proof           *swarmionapp.BranchRootDurableProofObservation `json:"proof,omitempty"`
}

// instanceDeleteEffectFactPayload is the immutable application meaning that
// is inserted in the same SQL transaction as the final record deletion. It
// deliberately excludes task progress and execution ownership.
type instanceDeleteEffectFactPayload struct {
	OperationID       string                  `json:"operation_id"`
	Operation         string                  `json:"operation"`
	ExpectedInvariant instanceDeleteInvariant `json:"expected_invariant"`
}

// instanceDeleteReceiptFactPayload contains only exact receipt identity
// available in the public runtime contract.
// Checkpoint and durability observations remain mutable projections and never
// participate in this fact's deterministic identity.
type instanceDeleteReceiptFactPayload struct {
	OperationID           string                  `json:"operation_id"`
	Operation             string                  `json:"operation"`
	ExpectedInvariant     instanceDeleteInvariant `json:"expected_invariant"`
	EventID               string                  `json:"event_id"`
	PublishedRootHash     string                  `json:"published_root_hash"`
	OperationIntentDigest string                  `json:"operation_intent_digest"`
	OperationAuthorPeerID string                  `json:"operation_author_peer_id"`
}

type uploadLocalImageTaskPayload struct {
	ImagePath       string `json:"image_path"`
	ImageName       string `json:"image_name"`
	ProvisionerName string `json:"provisioner_name"`
	Location        string `json:"location"`
	TimeoutSeconds  int64  `json:"timeout_seconds"`
}

type uploadLocalImageTaskResult struct {
	ImageID         string `json:"image_id"`
	ImageName       string `json:"image_name"`
	ProvisionerName string `json:"provisioner_name"`
	Location        string `json:"location"`
}

func (cm *Manager) registerTaskStreams() error {
	if err := tasks.Register(cm.tasks, tasks.Stream[deployInstanceTaskPayload, deployInstanceTaskResult]{
		Name:    InstanceDeploymentTaskStream,
		Recover: cm.recoverDeployInstanceTask,
		Run:     cm.runDeployInstanceTask,
	}); err != nil {
		return err
	}
	if err := tasks.Register(cm.tasks, tasks.Stream[instanceLifecycleTaskPayload, instanceLifecycleTaskResult]{
		Name:    InstanceLifecycleTaskStream,
		Recover: cm.recoverInstanceLifecycleTask,
		Run:     cm.runInstanceLifecycleTask,
	}); err != nil {
		return err
	}
	return tasks.Register(cm.tasks, tasks.Stream[uploadLocalImageTaskPayload, uploadLocalImageTaskResult]{
		Name: ProvisionerImageUploadTaskStream,
		Run:  cm.runUploadLocalImageTask,
	})
}

// recoverDeployInstanceTask prevents whole-operation replay once a provider
// resource or authenticated peer identity has been published. Those fields are
// replicated recovery authority: startup must preserve the running task until
// an explicit resume workflow continues that exact resource.
func (cm *Manager) recoverDeployInstanceTask(
	_ context.Context,
	recovery *tasks.RecoveryContext[deployInstanceTaskPayload],
) (tasks.StreamRecoveryDisposition, error) {
	payload := recovery.Payload()
	pendingID := strings.TrimSpace(payload.PendingInstanceID)
	if pendingID == "" {
		return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf("deployment task missing pending instance id"))
	}
	instance, err := db.SelectOne(cm.db, createInstanceQueryMapper(pendingID))
	if err != nil {
		if errors.Is(err, stdsql.ErrNoRows) {
			return tasks.StreamRecoveryReady, nil
		}
		return tasks.StreamRecoveryDeferred, fmt.Errorf("inspect interrupted deployment %s: %w", pendingID, err)
	}
	if deploymentInstanceRequiresRecovery(instance) {
		return tasks.StreamRecoveryDeferred, nil
	}
	return tasks.StreamRecoveryReady, nil
}

func deploymentInstanceRequiresRecovery(instance InstanceInfo) bool {
	return strings.TrimSpace(instance.ProviderResourceID) != "" || strings.TrimSpace(instance.PublicKey) != ""
}

func classifyDeploymentTaskError(err error) error {
	if errors.Is(err, ErrInstanceInitializationRecoveryRequired) {
		return tasks.MarkPermanent(err)
	}
	return err
}

// recoverInstanceLifecycleTask resolves immutable delete operation facts before
// generic recovery changes interrupted running-state bookkeeping.
func (cm *Manager) recoverInstanceLifecycleTask(
	ctx context.Context,
	recovery *tasks.RecoveryContext[instanceLifecycleTaskPayload],
) (tasks.StreamRecoveryDisposition, error) {
	payload := recovery.Payload()
	if strings.TrimSpace(payload.Operation) != instanceLifecycleOperationDelete {
		return tasks.StreamRecoveryReady, nil
	}
	if err := validateInstanceDeleteTaskPayloadModel(payload, recovery.Task().ID); err != nil {
		return tasks.StreamRecoveryReady, tasks.MarkPermanent(err)
	}
	return cm.recoverInstanceLifecycleTaskFromOperationFacts(ctx, recovery)
}

func validateInstanceDeleteTaskPayloadModel(payload instanceLifecycleTaskPayload, taskID string) error {
	model := strings.TrimSpace(payload.RecoveryModel)
	if model != instanceDeleteRecoveryModel {
		return fmt.Errorf(
			"instance delete task %s uses unsupported recovery model %q; immutable operation facts are required",
			strings.TrimSpace(taskID),
			model,
		)
	}
	if payload.DeleteOperation == nil {
		return fmt.Errorf("instance delete task %s is missing its immutable operation identity", strings.TrimSpace(taskID))
	}
	return nil
}

// recoverInstanceLifecycleTaskFromOperationFacts never resolves or claims a
// mutable post-delete task checkpoint. The delete operation itself contains an
// immutable effect fact, and any peer can derive the same exact receipt fact
// from Swarmion's author-scoped operation binding once that event arrives.
func (cm *Manager) recoverInstanceLifecycleTaskFromOperationFacts(
	ctx context.Context,
	recovery *tasks.RecoveryContext[instanceLifecycleTaskPayload],
) (tasks.StreamRecoveryDisposition, error) {
	if cm == nil || cm.db == nil || cm.tasks == nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("recover instance delete from operation facts: manager is not configured")
	}
	payload := recovery.Payload()
	operationID := strings.TrimSpace(recovery.Task().ID)
	instanceID := strings.TrimSpace(payload.InstanceID)
	identity := *payload.DeleteOperation
	if err := validateInstanceDeleteOperationIdentity(identity, operationID, instanceID, payload.LocalOnly); err != nil {
		return tasks.StreamRecoveryReady, tasks.MarkPermanent(err)
	}

	observeCtx, cancel := context.WithTimeout(ctx, instanceDeleteRecoveryObserveLimit)
	defer cancel()

	effectFact, effectFound, err := cm.tasks.OperationFact(observeCtx, operationID, tasks.OperationFactKindEffect)
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("read instance delete effect fact %s: %w", operationID, err)
	}
	if effectFound {
		factIdentity, factErr := instanceDeleteIdentityFromEffectFact(effectFact)
		if factErr != nil {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(factErr)
		}
		if factIdentity != identity {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: task payload and effect fact disagree task_id=%s",
				tasks.ErrOperationFactConflict,
				operationID,
			))
		}
	}

	receipt, receiptFound, err := cm.readInstanceDeleteReceiptFact(
		observeCtx,
		operationID,
		identity,
	)
	if err != nil {
		return tasks.StreamRecoveryReady, tasks.MarkPermanent(err)
	}
	if receiptFound {
		if !effectFound {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: receipt fact exists without its atomic effect fact task_id=%s",
				tasks.ErrOperationFactConflict,
				operationID,
			))
		}
		resolveOperation := cm.db.WaitPublishedWriteOperation
		if cm.lookupDeleteRecoveryOperation != nil {
			resolveOperation = cm.lookupDeleteRecoveryOperation
		}
		resolved, resolveErr := resolveOperation(observeCtx, identity.publishedWriteOperation())
		if resolveErr != nil {
			if errors.Is(resolveErr, context.DeadlineExceeded) || errors.Is(resolveErr, context.Canceled) {
				return tasks.StreamRecoveryDeferred, nil
			}
			return tasks.StreamRecoveryReady, fmt.Errorf("resolve immutable receipt fact %s: %w", operationID, resolveErr)
		}
		switch resolved.State {
		case swarmionapp.OperationResolvedUnavailable:
			return tasks.StreamRecoveryDeferred, nil
		case swarmionapp.OperationResolvedAccepted:
			if !receipt.matchesOperationResolution(resolved) {
				return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
					"%w: immutable receipt fact disagrees with Swarmion operation binding task_id=%s",
					tasks.ErrOperationFactConflict,
					operationID,
				))
			}
		default:
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: immutable receipt fact has no matching Swarmion operation binding task_id=%s resolution=%s safe_to_execute=%t",
				tasks.ErrOperationFactConflict,
				operationID,
				resolved.State,
				resolved.SafeToExecute,
			))
		}
	} else {
		lookupOperation := cm.db.LookupPublishedWriteOperation
		if cm.lookupDeleteRecoveryOperation != nil {
			lookupOperation = cm.lookupDeleteRecoveryOperation
		}
		resolved, lookupErr := lookupOperation(observeCtx, identity.publishedWriteOperation())
		if lookupErr != nil {
			if errors.Is(lookupErr, context.DeadlineExceeded) || errors.Is(lookupErr, context.Canceled) {
				return tasks.StreamRecoveryDeferred, nil
			}
			return tasks.StreamRecoveryReady, fmt.Errorf("resolve immutable instance delete operation %s: %w", operationID, lookupErr)
		}
		switch resolved.State {
		case swarmionapp.OperationResolvedAbsent:
			if effectFound {
				// SQL already proves that the operation event exists. Treat a lagging
				// protocol binding as not-ready, never as permission to republish.
				return tasks.StreamRecoveryDeferred, nil
			}
			if !resolved.SafeToExecute {
				return tasks.StreamRecoveryDeferred, nil
			}
			return tasks.StreamRecoveryReady, nil
		case swarmionapp.OperationResolvedUnavailable:
			return tasks.StreamRecoveryDeferred, nil
		case swarmionapp.OperationResolvedAccepted:
			published, receiptErr := db.PublishedWriteReceiptFromResolution(resolved)
			if receiptErr != nil {
				return tasks.StreamRecoveryReady, fmt.Errorf("recover immutable instance delete operation %s: %w", operationID, receiptErr)
			}
			receipt = instanceDeleteReceiptFromPublished(operationID, identity, published)
		default:
			return tasks.StreamRecoveryReady, fmt.Errorf(
				"resolve immutable instance delete operation %s returned unknown resolution %q",
				operationID,
				resolved.State,
			)
		}
	}

	observeReceipt := cm.db.ObservePublishedWriteReceipt
	if cm.observeDeleteRecoveryReceipt != nil {
		observeReceipt = cm.observeDeleteRecoveryReceipt
	}
	observation, err := observeReceipt(observeCtx, receipt.publishedWriteReceipt())
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("observe immutable instance delete operation %s: %w", operationID, err)
	}
	receipt.applyObservation(observation)
	if observation.State != db.EventReceiptStateAppliedDurably || !receipt.AppliedDurably {
		cm.logInstanceDeleteRecoveryDeferred(operationID, identity, receipt, observation.State, string(observation.Status.ParkedReason))
		return tasks.StreamRecoveryDeferred, nil
	}

	if !effectFound {
		// applied_durably makes D's SQL content queryable. The effect marker must
		// now exist because it was part of the same operation transaction.
		effectFact, effectFound, err = cm.tasks.OperationFact(observeCtx, operationID, tasks.OperationFactKindEffect)
		if err != nil {
			return tasks.StreamRecoveryReady, fmt.Errorf("re-read applied instance delete effect fact %s: %w", operationID, err)
		}
		if !effectFound {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: applied delete is missing its atomic effect fact task_id=%s event_id=%s",
				tasks.ErrOperationFactConflict,
				operationID,
				receipt.EventID,
			))
		}
		factIdentity, factErr := instanceDeleteIdentityFromEffectFact(effectFact)
		if factErr != nil || factIdentity != identity {
			if factErr == nil {
				factErr = fmt.Errorf("%w: applied effect fact identity mismatch task_id=%s", tasks.ErrOperationFactConflict, operationID)
			}
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(factErr)
		}
	}

	if !receiptFound {
		fact, factErr := newInstanceDeleteReceiptFact(receipt, identity)
		if factErr != nil {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(factErr)
		}
		if factErr = cm.tasks.EnsureOperationFact(context.WithoutCancel(observeCtx), fact); factErr != nil {
			return tasks.StreamRecoveryReady, fmt.Errorf("record recovered instance delete receipt fact %s: %w", operationID, factErr)
		}
		checkedReceipt, checkedFound, checkErr := cm.readInstanceDeleteReceiptFact(
			context.WithoutCancel(observeCtx),
			operationID,
			identity,
		)
		if checkErr != nil {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(checkErr)
		}
		if !checkedFound || !instanceDeleteReceiptFactsMatch(receipt, checkedReceipt) {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: persisted instance delete receipt fact disagrees with recovered receipt task_id=%s",
				tasks.ErrOperationFactConflict,
				operationID,
			))
		}
	}
	recovery.ReplacePayload(payload)
	return tasks.StreamRecoveryReady, nil
}

func (receipt instanceDeleteOperationReceipt) matchesOperationResolution(resolved swarmionapp.OperationResolution) bool {
	return resolved.State == swarmionapp.OperationResolvedAccepted &&
		resolved.Receipt != nil &&
		strings.TrimSpace(resolved.Receipt.EventID) == strings.TrimSpace(receipt.EventID) &&
		strings.TrimSpace(resolved.Receipt.PublishedRootHash) == strings.TrimSpace(receipt.PublishedRootHash) &&
		strings.TrimSpace(resolved.AuthorPeerID) == strings.TrimSpace(receipt.OperationAuthorPeerID) &&
		strings.TrimSpace(resolved.Identity.IntentDigest) == strings.TrimSpace(receipt.OperationIntentDigest)
}

// logInstanceDeleteRecoveryDeferred documents the deliberate parked/pending
// policy without retaining any local throttle or recovery state. Pending and
// unavailable observations are debug-level; parked protocol outcomes remain
// warnings on each bounded runner observation.
func (cm *Manager) logInstanceDeleteRecoveryDeferred(
	operationID string,
	identity instanceDeleteOperationIdentity,
	receipt instanceDeleteOperationReceipt,
	state db.EventReceiptState,
	reason string,
) {
	if cm == nil {
		return
	}
	proofResult := "not_available"
	if receipt.Proof != nil {
		proofResult = fmt.Sprintf(
			"ran:%t,covered:%t,conflict:%t,merged_root:%s",
			receipt.Proof.ProofRan,
			receipt.Proof.Covered,
			receipt.Proof.Conflict,
			receipt.Proof.MergedRootHash,
		)
	}
	format := "deferring interrupted instance delete recovery without task write or delete republication operation_id=%s operation_author=%s event_id=%s published_root=%s checkpoint_commit=%s checkpoint_root=%s durable_head_commit=%s durable_head_root=%s queryable_root=%s state=%s reason=%s proof_result=%s policy=bounded_background_reobserve"
	args := []any{
		operationID,
		identity.AuthorPeerID,
		receipt.EventID,
		receipt.PublishedRootHash,
		receipt.CheckpointCommitID,
		receipt.CheckpointRootHash,
		receipt.DurableCheckpointCommitID,
		receipt.DurableCheckpointRootHash,
		receipt.QueryableRootHash,
		state,
		strings.TrimSpace(reason),
		proofResult,
	}
	switch state {
	case db.EventReceiptStateParkedConflict, db.EventReceiptStateDependencyParked, db.EventReceiptStateStaleAnchor:
		log.Warnf(format, args...)
	default:
		log.Debugf(format, args...)
	}
}

func newPendingInstanceID() string {
	return db.MustNewUUIDv7()
}

func newInstanceDeleteOperationIdentity(
	operationID string,
	instance InstanceInfo,
	localOnly bool,
	authorPeerID string,
) (instanceDeleteOperationIdentity, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return instanceDeleteOperationIdentity{}, fmt.Errorf("instance delete operation ID is empty")
	}
	key, err := db.NewPublishedWriteOperationKey()
	if err != nil {
		return instanceDeleteOperationIdentity{}, err
	}
	peerID := ""
	if strings.TrimSpace(instance.PublicKey) != "" {
		peerID, _ = db.PeerIDFromPublicKeyString(instance.PublicKey)
	}
	invariant := instanceDeleteInvariant{
		Kind:       instanceDeleteInvariantAbsent,
		InstanceID: strings.TrimSpace(instance.ID),
		PeerID:     strings.TrimSpace(peerID),
	}
	operation, err := db.NewPublishedWriteOperation(
		key,
		"protos:instance-record-delete",
		operationID,
		invariant.InstanceID,
		strconv.FormatBool(localOnly),
		invariant.Kind,
		invariant.PeerID,
	)
	if err != nil {
		return instanceDeleteOperationIdentity{}, err
	}
	return instanceDeleteOperationIdentity{
		Key:               operation.Key,
		IntentDigest:      operation.IntentDigest,
		AuthorPeerID:      strings.TrimSpace(authorPeerID),
		ExpectedInvariant: invariant,
	}, nil
}

func (identity instanceDeleteOperationIdentity) publishedWriteOperation() db.PublishedWriteOperation {
	return db.PublishedWriteOperation{
		Key:          strings.TrimSpace(identity.Key),
		IntentDigest: strings.TrimSpace(identity.IntentDigest),
		AuthorPeerID: strings.TrimSpace(identity.AuthorPeerID),
	}
}

func newInstancePeerDrainAuthorization(
	taskID string,
	deleteOperation instanceDeleteOperationIdentity,
	instance InstanceInfo,
	localOnly bool,
) (instancePeerDrainAuthorization, error) {
	taskID = strings.TrimSpace(taskID)
	if err := validateInstanceDeleteOperationIdentity(deleteOperation, taskID, instance.ID, localOnly); err != nil {
		return instancePeerDrainAuthorization{}, err
	}
	if strings.TrimSpace(instance.PublicKey) == "" {
		return instancePeerDrainAuthorization{}, fmt.Errorf("instance peer-drain authorization requires a public key")
	}
	ownerPeerID, err := instanceLifecycleOwner(instance)
	if err != nil {
		return instancePeerDrainAuthorization{}, err
	}
	if strings.TrimSpace(deleteOperation.AuthorPeerID) != ownerPeerID {
		return instancePeerDrainAuthorization{}, fmt.Errorf(
			"%w: instance=%s persisted_owner=%s delete_author=%s",
			ErrInstanceLifecycleOwnerConflict,
			instance.ID,
			ownerPeerID,
			deleteOperation.AuthorPeerID,
		)
	}
	peerID, err := db.PeerIDFromPublicKeyString(instance.PublicKey)
	if err != nil {
		return instancePeerDrainAuthorization{}, fmt.Errorf("derive peer-drain authorization peer: %w", err)
	}
	keyDigest := sha256.Sum256([]byte(
		"protos:instance-peer-drain-authorization-key\x00" + strings.TrimSpace(deleteOperation.Key),
	))
	key := hex.EncodeToString(keyDigest[:])
	snapshot := instancePeerDrainInstance{
		Name:                 instance.Name,
		Kind:                 instance.Kind,
		KindID:               instance.KindID,
		ProviderResourceID:   instance.ProviderResourceID,
		PreDesiredStatus:     instance.DesiredStatus,
		ReplicationPriority:  instance.ReplicationPriority,
		PublicIP:             instance.PublicIP,
		Location:             instance.Location,
		Architecture:         instance.Architecture,
		PublicKey:            instance.PublicKey,
		LifecycleOwnerPeerID: ownerPeerID,
	}
	operation, err := db.NewPublishedWriteOperation(
		key,
		"protos:instance-peer-drain-authorization",
		instancePeerDrainAuthorizationFact,
		taskID,
		strings.TrimSpace(deleteOperation.Key),
		strings.TrimSpace(deleteOperation.IntentDigest),
		strings.TrimSpace(deleteOperation.AuthorPeerID),
		strings.TrimSpace(instance.ID),
		peerID,
		strconv.FormatBool(localOnly),
		snapshot.Name,
		snapshot.Kind,
		snapshot.KindID,
		snapshot.ProviderResourceID,
		snapshot.PreDesiredStatus,
		strconv.Itoa(snapshot.ReplicationPriority),
		snapshot.PublicIP,
		snapshot.Location,
		snapshot.Architecture,
		snapshot.PublicKey,
		snapshot.LifecycleOwnerPeerID,
	)
	if err != nil {
		return instancePeerDrainAuthorization{}, err
	}
	return instancePeerDrainAuthorization{
		Key:             operation.Key,
		IntentDigest:    operation.IntentDigest,
		AuthorPeerID:    strings.TrimSpace(deleteOperation.AuthorPeerID),
		DeleteOperation: deleteOperation,
		TaskID:          taskID,
		InstanceID:      strings.TrimSpace(instance.ID),
		PeerID:          peerID,
		LocalOnly:       localOnly,
		Instance:        snapshot,
	}, nil
}

func (authorization instancePeerDrainAuthorization) publishedWriteOperation() db.PublishedWriteOperation {
	return db.PublishedWriteOperation{
		Key:          strings.TrimSpace(authorization.Key),
		IntentDigest: strings.TrimSpace(authorization.IntentDigest),
		AuthorPeerID: strings.TrimSpace(authorization.AuthorPeerID),
	}
}

func (authorization instancePeerDrainAuthorization) expectedInstance() InstanceInfo {
	return InstanceInfo{
		ID:                   strings.TrimSpace(authorization.InstanceID),
		Name:                 authorization.Instance.Name,
		PublicKey:            authorization.Instance.PublicKey,
		PublicIP:             authorization.Instance.PublicIP,
		Kind:                 authorization.Instance.Kind,
		KindID:               authorization.Instance.KindID,
		ProviderResourceID:   authorization.Instance.ProviderResourceID,
		DesiredStatus:        authorization.Instance.PreDesiredStatus,
		ReplicationPriority:  authorization.Instance.ReplicationPriority,
		Location:             authorization.Instance.Location,
		Architecture:         authorization.Instance.Architecture,
		LifecycleOwnerPeerID: authorization.Instance.LifecycleOwnerPeerID,
	}
}

func validateInstancePeerDrainAuthorization(
	authorization instancePeerDrainAuthorization,
	taskID string,
	deleteOperation instanceDeleteOperationIdentity,
	instanceID string,
	localOnly bool,
) error {
	if strings.TrimSpace(authorization.TaskID) != strings.TrimSpace(taskID) ||
		strings.TrimSpace(authorization.InstanceID) != strings.TrimSpace(instanceID) ||
		authorization.LocalOnly != localOnly || authorization.DeleteOperation != deleteOperation {
		return fmt.Errorf("invalid immutable peer-drain authorization identity for task %s", taskID)
	}
	expected, err := newInstancePeerDrainAuthorization(
		taskID,
		deleteOperation,
		authorization.expectedInstance(),
		localOnly,
	)
	if err != nil {
		return err
	}
	if expected != authorization {
		return fmt.Errorf("immutable peer-drain authorization does not match its deterministic identity for task %s", taskID)
	}
	return nil
}

func newInstancePeerDrainAuthorizationFact(
	authorization instancePeerDrainAuthorization,
) (tasks.OperationFact, error) {
	return tasks.NewOperationFact(
		authorization.TaskID,
		instancePeerDrainAuthorizationFact,
		authorization.publishedWriteOperation(),
		taskSubjectInstance,
		authorization.InstanceID,
		authorization,
	)
}

func newInstanceDeleteEffectFact(
	operationID string,
	identity instanceDeleteOperationIdentity,
) (tasks.OperationFact, error) {
	return tasks.NewOperationFact(
		operationID,
		tasks.OperationFactKindEffect,
		identity.publishedWriteOperation(),
		taskSubjectInstance,
		identity.ExpectedInvariant.InstanceID,
		instanceDeleteEffectFactPayload{
			OperationID:       strings.TrimSpace(operationID),
			Operation:         instanceLifecycleOperationDelete,
			ExpectedInvariant: identity.ExpectedInvariant,
		},
	)
}

func newInstanceDeleteReceiptFact(
	receipt instanceDeleteOperationReceipt,
	identity instanceDeleteOperationIdentity,
) (tasks.OperationFact, error) {
	return tasks.NewOperationFact(
		receipt.OperationID,
		tasks.OperationFactKindReceipt,
		db.PublishedWriteOperation{
			Key:          identity.Key,
			IntentDigest: receipt.OperationIntentDigest,
			AuthorPeerID: receipt.OperationAuthorPeerID,
		},
		taskSubjectInstance,
		receipt.ExpectedInvariant.InstanceID,
		instanceDeleteReceiptFactPayload{
			OperationID:           strings.TrimSpace(receipt.OperationID),
			Operation:             instanceLifecycleOperationDelete,
			ExpectedInvariant:     receipt.ExpectedInvariant,
			EventID:               strings.TrimSpace(receipt.EventID),
			PublishedRootHash:     strings.TrimSpace(receipt.PublishedRootHash),
			OperationIntentDigest: strings.TrimSpace(receipt.OperationIntentDigest),
			OperationAuthorPeerID: strings.TrimSpace(receipt.OperationAuthorPeerID),
		},
	)
}

func instanceDeleteIdentityFromEffectFact(fact tasks.OperationFact) (instanceDeleteOperationIdentity, error) {
	var payload instanceDeleteEffectFactPayload
	if err := decodeOperationFactPayload(fact.Payload, &payload); err != nil {
		return instanceDeleteOperationIdentity{}, fmt.Errorf("decode instance delete effect fact: %w", err)
	}
	identity := instanceDeleteOperationIdentity{
		Key:               strings.TrimSpace(fact.OperationKey),
		IntentDigest:      strings.TrimSpace(fact.IntentDigest),
		AuthorPeerID:      strings.TrimSpace(fact.AuthorPeerID),
		ExpectedInvariant: payload.ExpectedInvariant,
	}
	if strings.TrimSpace(payload.OperationID) != strings.TrimSpace(fact.TaskID) ||
		strings.TrimSpace(payload.Operation) != instanceLifecycleOperationDelete ||
		strings.TrimSpace(fact.SubjectType) != taskSubjectInstance ||
		strings.TrimSpace(fact.SubjectID) != strings.TrimSpace(payload.ExpectedInvariant.InstanceID) {
		return instanceDeleteOperationIdentity{}, fmt.Errorf("%w: invalid instance delete effect fact task_id=%s", tasks.ErrOperationFactConflict, fact.TaskID)
	}
	return identity, nil
}

func (cm *Manager) readInstanceDeleteReceiptFact(
	ctx context.Context,
	operationID string,
	identity instanceDeleteOperationIdentity,
) (instanceDeleteOperationReceipt, bool, error) {
	if cm == nil || cm.tasks == nil {
		return instanceDeleteOperationReceipt{}, false, fmt.Errorf("read instance delete receipt fact: manager is not configured")
	}
	fact, found, err := cm.tasks.OperationFact(ctx, operationID, tasks.OperationFactKindReceipt)
	if err != nil {
		return instanceDeleteOperationReceipt{}, false, fmt.Errorf("read instance delete receipt fact %s: %w", operationID, err)
	}
	if !found {
		return instanceDeleteOperationReceipt{}, false, nil
	}
	receipt, err := instanceDeleteReceiptFromFact(fact, identity)
	if err != nil {
		return instanceDeleteOperationReceipt{}, false, err
	}
	if err := validateInstanceDeleteOperationReceipt(
		receipt,
		identity,
		operationID,
		identity.ExpectedInvariant.InstanceID,
	); err != nil {
		return instanceDeleteOperationReceipt{}, false, fmt.Errorf(
			"%w: invalid instance delete receipt fact task_id=%s: %s",
			tasks.ErrOperationFactConflict,
			operationID,
			err.Error(),
		)
	}
	return receipt, true, nil
}

func instanceDeleteReceiptFactsMatch(left, right instanceDeleteOperationReceipt) bool {
	return left.OperationID == right.OperationID &&
		left.Operation == right.Operation &&
		left.ExpectedInvariant == right.ExpectedInvariant &&
		left.EventID == right.EventID &&
		left.PublishedRootHash == right.PublishedRootHash &&
		left.OperationIntentDigest == right.OperationIntentDigest &&
		left.OperationAuthorPeerID == right.OperationAuthorPeerID
}

func instanceDeleteReceiptFromFact(
	fact tasks.OperationFact,
	identity instanceDeleteOperationIdentity,
) (instanceDeleteOperationReceipt, error) {
	if fact.Kind != tasks.OperationFactKindReceipt {
		return instanceDeleteOperationReceipt{}, fmt.Errorf(
			"%w: unsupported instance delete receipt fact kind %q task_id=%s",
			tasks.ErrOperationFactConflict,
			fact.Kind,
			fact.TaskID,
		)
	}
	var payload instanceDeleteReceiptFactPayload
	if err := decodeOperationFactPayload(fact.Payload, &payload); err != nil {
		return instanceDeleteOperationReceipt{}, fmt.Errorf("decode instance delete receipt fact: %w", err)
	}
	receipt := instanceDeleteOperationReceipt{
		OperationID:           strings.TrimSpace(payload.OperationID),
		Operation:             strings.TrimSpace(payload.Operation),
		ExpectedInvariant:     payload.ExpectedInvariant,
		EventID:               strings.TrimSpace(payload.EventID),
		PublishedRootHash:     strings.TrimSpace(payload.PublishedRootHash),
		OperationIntentDigest: strings.TrimSpace(payload.OperationIntentDigest),
		OperationAuthorPeerID: strings.TrimSpace(payload.OperationAuthorPeerID),
	}
	if strings.TrimSpace(fact.SubjectType) != taskSubjectInstance ||
		strings.TrimSpace(fact.SubjectID) != strings.TrimSpace(identity.ExpectedInvariant.InstanceID) ||
		fact.OperationKey != identity.Key ||
		fact.IntentDigest != identity.IntentDigest ||
		fact.AuthorPeerID != identity.AuthorPeerID ||
		receipt.OperationIntentDigest != identity.IntentDigest ||
		receipt.OperationAuthorPeerID != identity.AuthorPeerID {
		return instanceDeleteOperationReceipt{}, fmt.Errorf("%w: receipt/effect identity mismatch task_id=%s", tasks.ErrOperationFactConflict, fact.TaskID)
	}
	return receipt, nil
}

func decodeOperationFactPayload(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		if err == nil {
			return errors.New("operation fact payload contains trailing JSON value")
		}
		return fmt.Errorf("decode trailing operation fact payload: %w", err)
	}
	return nil
}

func validateInstanceDeleteOperationIdentity(
	identity instanceDeleteOperationIdentity,
	operationID string,
	instanceID string,
	localOnly bool,
) error {
	key := strings.TrimSpace(identity.Key)
	keyBytes, err := hex.DecodeString(key)
	if err != nil || len(keyBytes) < 16 {
		return fmt.Errorf("instance delete operation key must contain at least 128 bits")
	}
	if strings.TrimSpace(identity.AuthorPeerID) == "" {
		return fmt.Errorf("instance delete operation author peer ID is empty")
	}
	if identity.ExpectedInvariant.Kind != instanceDeleteInvariantAbsent ||
		strings.TrimSpace(identity.ExpectedInvariant.InstanceID) != strings.TrimSpace(instanceID) {
		return fmt.Errorf("instance delete operation has invalid expected invariant: %+v", identity.ExpectedInvariant)
	}
	expected, err := db.NewPublishedWriteOperation(
		key,
		"protos:instance-record-delete",
		strings.TrimSpace(operationID),
		strings.TrimSpace(instanceID),
		strconv.FormatBool(localOnly),
		identity.ExpectedInvariant.Kind,
		strings.TrimSpace(identity.ExpectedInvariant.PeerID),
	)
	if err != nil {
		return err
	}
	if strings.TrimSpace(identity.IntentDigest) != expected.IntentDigest {
		return fmt.Errorf("instance delete operation intent digest does not match its replicated invariant")
	}
	return nil
}

func (cm *Manager) runDeployInstanceTask(ctx context.Context, task *tasks.RunContext[deployInstanceTaskPayload]) (deployInstanceTaskResult, error) {
	payload := task.Payload()
	if payload.PendingInstanceID == "" {
		return deployInstanceTaskResult{}, fmt.Errorf("deployment task missing pending instance id")
	}
	pendingInstance, err := cm.getInstanceRecord(payload.PendingInstanceID)
	if err != nil {
		return deployInstanceTaskResult{}, tasks.MarkPermanent(fmt.Errorf("load deployment task authority: %w", err))
	}
	if err := cm.assertInstanceLifecycleExecutor(pendingInstance, instanceTaskOwner(task.Task())); err != nil {
		return deployInstanceTaskResult{}, tasks.MarkPermanent(err)
	}
	instance, err := cm.deployInstanceImperative(ctx, task.Update, payload.PendingInstanceID, payload.InstanceName, payload.CloudName, payload.CloudLocation, payload.Release, payload.MachineType)
	if err != nil {
		if errors.Is(err, ErrInstanceInitializationRecoveryRequired) {
			// Discovery/admission is irreversible without a coordinated drain.
			// Failing this attempt permanently prevents the task engine (or a
			// future MaxAttempts change) from replaying VM creation.
			return deployInstanceTaskResult{}, classifyDeploymentTaskError(err)
		}
		if persisted, lookupErr := db.SelectOne(cm.db, createInstanceQueryMapper(payload.PendingInstanceID)); lookupErr == nil && deploymentInstanceRequiresRecovery(persisted) {
			return deployInstanceTaskResult{}, classifyDeploymentTaskError(fmt.Errorf(
				"%w: instance=%s retained after deployment failure: %w",
				ErrInstanceInitializationRecoveryRequired,
				payload.PendingInstanceID,
				err,
			))
		}
		return deployInstanceTaskResult{}, err
	}
	return deployInstanceTaskResult{
		PendingInstanceID:  payload.PendingInstanceID,
		InstanceID:         instance.ID,
		ProviderResourceID: instance.ProviderResourceID,
		PublicIP:           instance.PublicIP,
		PublicKey:          instance.PublicKey,
	}, nil
}

func (cm *Manager) QueueDesiredInstanceReconciles(ctx context.Context) error {
	if cm == nil || cm.tasks == nil {
		return fmt.Errorf("task manager is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	instances, err := cm.GetInstancesContext(ctx, false)
	if err != nil {
		return err
	}
	var failures []string
	for _, instance := range instances {
		if IsDeletingInstance(instance) || strings.TrimSpace(instance.PublicKey) == "" {
			continue
		}
		ownerPeerID, ownerErr := instanceLifecycleOwner(instance)
		if ownerErr != nil || ownerPeerID != cm.localLifecycleExecutorPeerID() || cm.provisionerMutationDisabled {
			continue
		}
		desiredStatus := normalizeDesiredInstanceStatus(instance.DesiredStatus)
		if desiredStatus == "" {
			continue
		}
		sig := lifecycleDesiredSignature(instance, desiredStatus)
		if cm.lifecycleSignatureCurrent(instance.ID, sig) {
			continue
		}
		if _, err := cm.queueInstanceLifecycle(ctx, instance, instanceLifecycleOperationReconcile, desiredStatus, false, false, sig); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", instance.Name, err))
			continue
		}
		cm.setLifecycleSignature(instance.ID, sig)
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func (cm *Manager) QueueStartInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueSetInstanceDesiredStatus(ctx, id, ServerStateRunning)
}

func (cm *Manager) QueueStopInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueSetInstanceDesiredStatus(ctx, id, ServerStateStopped)
}

func (cm *Manager) QueueDeleteInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueDeleteInstance(ctx, id, false)
}

func (cm *Manager) QueueDeleteInstanceLocal(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueDeleteInstance(ctx, id, true)
}

func (cm *Manager) queueSetInstanceDesiredStatus(ctx context.Context, id string, desiredStatus string) (tasks.Record, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return tasks.Record{}, err
	}
	unlock, err := cm.acquireInstanceLifecycle(ctx, id)
	if err != nil {
		return tasks.Record{}, err
	}
	defer unlock()
	instance, err := cm.getInstanceRecordContext(ctx, id)
	if err != nil {
		return tasks.Record{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	desiredStatus = normalizeDesiredInstanceStatus(desiredStatus)
	if desiredStatus == "" {
		return tasks.Record{}, fmt.Errorf("invalid desired instance status")
	}
	if IsDeletingInstance(instance) {
		return tasks.Record{}, fmt.Errorf(
			"%w: instance '%s' has replicated desired status %q",
			ErrInstanceLifecycleConflict,
			instance.Name,
			instance.DesiredStatus,
		)
	}
	if _, err := cm.lifecycleTaskOwner(instance); err != nil {
		return tasks.Record{}, err
	}
	if err := cm.assertInstanceLifecycleExecutor(instance, ""); err != nil {
		// A foreign peer must not publish desired_status before the owner has
		// serialized it with P. Remote desired-state intents need their own
		// immutable ordering contract; until then this transition fails closed.
		return tasks.Record{}, err
	}
	if authorized, authErr := cm.instancePeerDrainAuthorizationExists(ctx, instance.ID); authErr != nil {
		return tasks.Record{}, fmt.Errorf("inspect peer-drain authorization for instance '%s': %w", instance.Name, authErr)
	} else if authorized {
		return tasks.Record{}, fmt.Errorf("%w: instance '%s' has immutable peer-drain authorization", ErrInstanceLifecycleConflict, instance.Name)
	}
	if cm.tasks != nil {
		deleteSubject := instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete)
		if task, found, lookupErr := cm.tasks.LatestForSubjectContext(ctx, InstanceLifecycleTaskStream, taskSubjectInstance, deleteSubject); lookupErr != nil {
			return tasks.Record{}, fmt.Errorf("inspect delete ownership for instance '%s': %w", instance.Name, lookupErr)
		} else if found && isActiveLifecycleTask(task) {
			return tasks.Record{}, fmt.Errorf(
				"%w: instance '%s' is owned by active delete task %s",
				ErrInstanceLifecycleConflict,
				instance.Name,
				task.ID,
			)
		}
	}
	instance.DesiredStatus = desiredStatus
	im, _ := createInstanceLifecycleUpdateMapper(instance)
	if _, err := db.UpdateWithAvailabilityContext(ctx, cm.db, im); err != nil {
		return tasks.Record{}, fmt.Errorf("failed to save instance '%s': %w", id, err)
	}
	if authorized, authErr := cm.instancePeerDrainAuthorizationExists(ctx, instance.ID); authErr != nil {
		return tasks.Record{}, fmt.Errorf("verify peer-drain authorization for instance '%s': %w", instance.Name, authErr)
	} else if authorized {
		return tasks.Record{}, fmt.Errorf("%w: instance '%s' became delete-authorized while changing desired status", ErrInstanceLifecycleConflict, instance.Name)
	}
	persisted, err := cm.getInstanceRecordContext(ctx, instance.ID)
	if err != nil || persisted.DesiredStatus != desiredStatus {
		return tasks.Record{}, fmt.Errorf("%w: instance '%s' desired status update was not accepted", ErrInstanceLifecycleConflict, instance.Name)
	}
	sig := lifecycleDesiredSignature(instance, desiredStatus)
	cm.clearLifecycleSignature(instance.ID)
	record, err := cm.queueInstanceLifecycle(ctx, instance, instanceLifecycleOperationReconcile, desiredStatus, false, true, sig)
	if err != nil {
		return tasks.Record{}, err
	}
	cm.setLifecycleSignature(instance.ID, sig)
	log.Infof("Queued desired status '%s' for instance '%s' as task '%s'", desiredStatus, instance.Name, record.ID)
	return record, nil
}

func (cm *Manager) queueDeleteInstance(ctx context.Context, id string, localOnly bool) (tasks.Record, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return tasks.Record{}, err
	}
	unlock, err := cm.acquireInstanceLifecycle(ctx, id)
	if err != nil {
		return tasks.Record{}, err
	}
	defer unlock()
	instance, err := cm.getInstanceRecordContext(ctx, id)
	if err != nil {
		return tasks.Record{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if _, err := cm.lifecycleTaskOwner(instance); err != nil {
		return tasks.Record{}, err
	}
	record, err := cm.queueInstanceLifecycle(ctx, instance, instanceLifecycleOperationDelete, ServerStateDeleting, localOnly, true, "")
	if err != nil {
		return tasks.Record{}, err
	}
	cm.clearLifecycleSignature(instance.ID)
	log.Infof("Queued delete for instance '%s' as task '%s'", instance.Name, record.ID)
	return record, nil
}

func (cm *Manager) queueInstanceLifecycle(ctx context.Context, instance InstanceInfo, operation string, desiredStatus string, localOnly bool, requestedByAPI bool, desiredSig string) (tasks.Record, error) {
	if cm == nil || cm.tasks == nil {
		return tasks.Record{}, fmt.Errorf("task manager is not configured")
	}
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return tasks.Record{}, fmt.Errorf("instance lifecycle operation is empty")
	}
	ownerPeerID, err := cm.lifecycleTaskOwner(instance)
	if err != nil {
		return tasks.Record{}, err
	}
	title := fmt.Sprintf("Reconcile instance %s", instance.Name)
	message := "queued"
	maxAttempts := 1
	taskID := ""
	var deleteOperation *instanceDeleteOperationIdentity
	var peerDrainAuthorization *instancePeerDrainAuthorization
	recoveryModel := ""
	if operation == instanceLifecycleOperationDelete {
		title = fmt.Sprintf("Delete instance %s", instance.Name)
		maxAttempts = instanceDeleteMaxAttempts
		var err error
		taskID, err = db.NewUUIDv7()
		if err != nil {
			return tasks.Record{}, err
		}
		identity, err := newInstanceDeleteOperationIdentity(taskID, instance, localOnly, ownerPeerID)
		if err != nil {
			return tasks.Record{}, err
		}
		deleteOperation = &identity
		if strings.TrimSpace(instance.PublicKey) != "" {
			authorization, err := newInstancePeerDrainAuthorization(taskID, identity, instance, localOnly)
			if err != nil {
				return tasks.Record{}, err
			}
			peerDrainAuthorization = &authorization
		}
		recoveryModel = instanceDeleteRecoveryModel
	}
	record, _, err := tasks.EnqueueUniqueContext(ctx, cm.tasks, tasks.EnqueueUniqueOptions[instanceLifecycleTaskPayload]{
		EnqueueOptions: tasks.EnqueueOptions[instanceLifecycleTaskPayload]{
			ID:          taskID,
			Stream:      InstanceLifecycleTaskStream,
			SubjectType: taskSubjectInstance,
			SubjectID:   instanceLifecycleSubjectID(instance.ID, operation),
			OwnerPeerID: ownerPeerID,
			Title:       title,
			Message:     message,
			Payload: instanceLifecycleTaskPayload{
				InstanceID:             instance.ID,
				InstanceName:           instance.Name,
				Operation:              operation,
				DesiredStatus:          desiredStatus,
				LocalOnly:              localOnly,
				DesiredSig:             desiredSig,
				RequestedByAPI:         requestedByAPI,
				RecoveryModel:          recoveryModel,
				DeleteOperation:        deleteOperation,
				PeerDrainAuthorization: peerDrainAuthorization,
			},
			MaxAttempts: maxAttempts,
		},
	})
	return record, err
}

func (cm *Manager) runInstanceLifecycleTask(ctx context.Context, task *tasks.RunContext[instanceLifecycleTaskPayload]) (instanceLifecycleTaskResult, error) {
	payload := task.Payload()
	instanceID := strings.TrimSpace(payload.InstanceID)
	if instanceID == "" {
		return instanceLifecycleTaskResult{}, fmt.Errorf("instance lifecycle task missing instance id")
	}
	operation := strings.TrimSpace(payload.Operation)
	switch operation {
	case instanceLifecycleOperationReconcile:
		instance, err := cm.getInstanceRecord(instanceID)
		if err != nil {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(fmt.Errorf("load reconcile authority: %w", err))
		}
		if err := cm.assertInstanceLifecycleExecutor(instance, instanceTaskOwner(task.Task())); err != nil {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(err)
		}
		if err := task.Update(10, "reconciling desired instance state", lifecycleTaskDetails(payload)); err != nil {
			return instanceLifecycleTaskResult{}, err
		}
		changed, instance, err := cm.reconcileDesiredInstance(ctx, task.Progress, instanceID)
		if err != nil {
			cm.clearLifecycleSignature(instanceID)
			return instanceLifecycleTaskResult{}, err
		}
		return instanceLifecycleTaskResult{
			InstanceID:    instance.ID,
			InstanceName:  instance.Name,
			Operation:     operation,
			DesiredStatus: instance.DesiredStatus,
			Changed:       changed,
		}, nil
	case instanceLifecycleOperationDelete:
		if err := validateInstanceDeleteTaskPayloadModel(payload, task.Task().ID); err != nil {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(err)
		}
		identity := payload.DeleteOperation
		persistedInstance, persistedErr := cm.getInstanceRecord(instanceID)
		if persistedErr != nil && !errors.Is(persistedErr, stdsql.ErrNoRows) {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(fmt.Errorf("load delete authority: %w", persistedErr))
		}
		authorityInstance := persistedInstance
		if persistedErr != nil {
			authorityInstance = InstanceInfo{ID: instanceID, LifecycleOwnerPeerID: identity.AuthorPeerID}
		}
		if err := cm.assertInstanceLifecycleExecutor(authorityInstance, instanceTaskOwner(task.Task())); err != nil {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(err)
		}
		if strings.TrimSpace(identity.AuthorPeerID) != strings.TrimSpace(authorityInstance.LifecycleOwnerPeerID) {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(fmt.Errorf(
				"%w: instance=%s persisted_owner=%s delete_author=%s",
				ErrInstanceLifecycleOwnerConflict,
				instanceID,
				authorityInstance.LifecycleOwnerPeerID,
				identity.AuthorPeerID,
			))
		}
		if err := validateInstanceDeleteOperationIdentity(*identity, task.Task().ID, instanceID, payload.LocalOnly); err != nil {
			return instanceLifecycleTaskResult{}, err
		}
		if payload.PeerDrainAuthorization != nil {
			if err := validateInstancePeerDrainAuthorization(
				*payload.PeerDrainAuthorization,
				task.Task().ID,
				*identity,
				instanceID,
				payload.LocalOnly,
			); err != nil {
				return instanceLifecycleTaskResult{}, tasks.MarkPermanent(err)
			}
		}
		if err := task.Update(5, "deleting instance", lifecycleTaskDetails(payload)); err != nil {
			return instanceLifecycleTaskResult{}, err
		}
		var completedDeleteReceipt *instanceDeleteOperationReceipt
		err := cm.deleteInstanceImperative(
			ctx,
			task.Update,
			instanceID,
			payload.LocalOnly,
			task.Task().ID,
			*identity,
			payload.PeerDrainAuthorization,
			func(next instanceDeleteOperationReceipt, _ int, _ string) error {
				fact, err := newInstanceDeleteReceiptFact(next, *identity)
				if err != nil {
					return err
				}
				if err := task.RecordOperationFact(ctx, fact); err != nil {
					return err
				}
				stored, found, err := cm.readInstanceDeleteReceiptFact(
					context.WithoutCancel(ctx),
					task.Task().ID,
					*identity,
				)
				if err != nil {
					return tasks.MarkPermanent(err)
				}
				if !found || !instanceDeleteReceiptFactsMatch(next, stored) {
					return tasks.MarkPermanent(fmt.Errorf(
						"%w: stored instance delete receipt fact disagrees with publication task_id=%s",
						tasks.ErrOperationFactConflict,
						task.Task().ID,
					))
				}
				completedDeleteReceipt = cloneInstanceDeleteOperationReceipt(&next)
				return nil
			},
		)
		if err != nil {
			if errors.Is(err, ErrInstanceDeleteInvariantConflict) {
				return instanceLifecycleTaskResult{}, tasks.MarkPermanent(err)
			}
			if errors.Is(err, db.ErrOperationReceiptUnavailable) ||
				errors.Is(err, db.ErrEventReceiptPending) ||
				errors.Is(err, db.ErrEventReceiptParked) ||
				errors.Is(err, db.ErrReplicationPeerDrainPending) ||
				errors.Is(err, db.ErrReplicationPeerDrainUnavailable) {
				return instanceLifecycleTaskResult{}, tasks.MarkDeferred(err)
			}
			var noReceipt *instanceDeletePublicationWithoutReceiptError
			if errors.As(err, &noReceipt) && !db.IsRetryablePublishedWriteError(err) {
				return instanceLifecycleTaskResult{}, tasks.MarkPermanent(err)
			}
			return instanceLifecycleTaskResult{}, err
		}
		return instanceLifecycleTaskResult{
			InstanceID:    instanceID,
			InstanceName:  payload.InstanceName,
			Operation:     operation,
			Deleted:       true,
			DeleteReceipt: cloneInstanceDeleteOperationReceipt(completedDeleteReceipt),
		}, nil
	default:
		return instanceLifecycleTaskResult{}, fmt.Errorf("unsupported instance lifecycle operation %q", operation)
	}
}

func instanceLifecycleSubjectID(instanceID string, operation string) string {
	return strings.TrimSpace(instanceID) + "/" + strings.TrimSpace(operation)
}

func isActiveLifecycleTask(record tasks.Record) bool {
	return record.Status == tasks.StatusPending || record.Status == tasks.StatusRunning
}

func lifecycleTaskDetails(payload instanceLifecycleTaskPayload) map[string]any {
	return map[string]any{
		"instance_id":      payload.InstanceID,
		"instance_name":    payload.InstanceName,
		"operation":        payload.Operation,
		"desired_status":   payload.DesiredStatus,
		"local_only":       payload.LocalOnly,
		"requested_by_api": payload.RequestedByAPI,
	}
}

func cloneInstanceDeleteOperationReceipt(receipt *instanceDeleteOperationReceipt) *instanceDeleteOperationReceipt {
	if receipt == nil {
		return nil
	}
	cloned := *receipt
	if receipt.Proof != nil {
		proof := *receipt.Proof
		cloned.Proof = &proof
	}
	return &cloned
}

func (cm *Manager) QueueUploadLocalImage(ctx context.Context, imagePath string, imageName string, provisionerName string, location string, timeout time.Duration) (tasks.Record, error) {
	if cm == nil || cm.tasks == nil {
		return tasks.Record{}, fmt.Errorf("task manager is not configured")
	}
	if imagePath == "" {
		return tasks.Record{}, fmt.Errorf("image path is empty")
	}
	if imageName == "" {
		return tasks.Record{}, fmt.Errorf("image name is empty")
	}
	if provisionerName == "" {
		return tasks.Record{}, fmt.Errorf("provisioner name is empty")
	}
	return tasks.EnqueueContext(ctx, cm.tasks, tasks.EnqueueOptions[uploadLocalImageTaskPayload]{
		Stream:      ProvisionerImageUploadTaskStream,
		SubjectType: taskSubjectProvisionerImage,
		SubjectID:   uploadLocalImageSubjectID(provisionerName, location, imageName),
		Title:       fmt.Sprintf("Upload image %s", imageName),
		Message:     "queued",
		Payload: uploadLocalImageTaskPayload{
			ImagePath:       imagePath,
			ImageName:       imageName,
			ProvisionerName: provisionerName,
			Location:        location,
			TimeoutSeconds:  int64(timeout / time.Second),
		},
	})
}

func (cm *Manager) runUploadLocalImageTask(ctx context.Context, task *tasks.RunContext[uploadLocalImageTaskPayload]) (uploadLocalImageTaskResult, error) {
	payload := task.Payload()
	progress := func(progress int, message string, details any, durable bool) error {
		if durable {
			return task.Update(progress, message, details)
		}
		return task.Progress(progress, message, details)
	}
	imageID, err := cm.uploadLocalImageImperative(
		ctx,
		progress,
		payload.ImagePath,
		payload.ImageName,
		payload.ProvisionerName,
		payload.Location,
		time.Duration(payload.TimeoutSeconds)*time.Second,
	)
	if err != nil {
		return uploadLocalImageTaskResult{}, err
	}
	return uploadLocalImageTaskResult{
		ImageID:         imageID,
		ImageName:       payload.ImageName,
		ProvisionerName: payload.ProvisionerName,
		Location:        payload.Location,
	}, nil
}

func uploadLocalImageSubjectID(provisionerName string, location string, imageName string) string {
	if location == "" {
		return fmt.Sprintf("%s/%s", provisionerName, imageName)
	}
	return fmt.Sprintf("%s/%s/%s", provisionerName, location, imageName)
}

func (cm *Manager) updateDeploymentPlaceholder(ctx context.Context, instance InstanceInfo) error {
	im, cmm := createInstanceUpdateMapper(instance)
	if _, err := db.UpdateWithAvailabilityContext(ctx, cm.db, im, cmm); err != nil {
		return fmt.Errorf("failed to update pending instance '%s': %w", instance.Name, err)
	}
	return nil
}

// persistDiscoveredDeploymentIdentity crosses the irreversible deployment
// boundary in one published SQL transaction. The machine identity and PEER row
// must become visible together before AddPeer or Init can run; a crash cannot
// leave an admitted identity represented only by provider-side state.
func (cm *Manager) persistDiscoveredDeploymentIdentity(ctx context.Context, pendingID string, instance InstanceInfo) error {
	pendingID = strings.TrimSpace(pendingID)
	if pendingID == "" {
		return fmt.Errorf("pending instance id is empty")
	}
	if strings.TrimSpace(instance.PublicKey) == "" {
		return fmt.Errorf("discovered peer public key is empty")
	}
	if _, err := db.SelectOne(cm.db, createInstanceQueryMapper(pendingID)); err != nil {
		return fmt.Errorf("pending instance '%s' no longer exists: %w", pendingID, err)
	}
	instance.ID = pendingID
	im, cmm := createInstanceUpdateMapper(instance)
	if _, err := db.UpdateAndInsertWithAvailabilityContext(
		ctx,
		cm.db,
		[]db.UpdateMapper{im, cmm},
		[]db.InsertMapper{db.CreatePeerInsertMapper(instance.PublicKey)},
	); err != nil {
		return fmt.Errorf("persist discovered identity for pending instance '%s': %w", pendingID, err)
	}
	return nil
}

func (cm *Manager) completeDeploymentInstance(ctx context.Context, pendingID string, instance InstanceInfo) error {
	if _, err := db.SelectOne(cm.db, createInstanceQueryMapper(pendingID)); err != nil {
		return fmt.Errorf("pending instance '%s' no longer exists: %w", pendingID, err)
	}
	instance.ID = pendingID
	im, cmm := createInstanceFinalizeMapper(pendingID, instance)
	if _, err := db.UpdateWithAvailabilityContext(ctx, cm.db, im, cmm); err != nil {
		return err
	}
	return nil
}
