package provisioners

import (
	"context"
	"crypto/sha256"
	stdsql "database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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
	instanceDeleteOperationFactsV1      = "immutable_operation_facts_v1"
	instancePeerDrainAuthorizedV1       = "instance_peer_drain_authorized_v1"

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
	CheckpointAuthorPeerID string                           `json:"checkpoint_author_peer_id,omitempty"`
	OperationStateModel    string                           `json:"operation_state_model,omitempty"`
	DeleteOperation        *instanceDeleteOperationIdentity `json:"delete_operation,omitempty"`
	PeerDrainAuthorization *instancePeerDrainAuthorization  `json:"peer_drain_authorization,omitempty"`
	DeleteReceipt          *instanceDeleteOperationReceipt  `json:"delete_receipt,omitempty"`
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
	Version         string                          `json:"version"`
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
	EventDigest           string `json:"event_digest,omitempty"`
	AuthorSeq             uint64 `json:"author_seq,omitempty"`
	OperationIntentDigest string `json:"operation_intent_digest,omitempty"`
	OperationAuthorPeerID string `json:"operation_author_peer_id,omitempty"`
	CommitHash            string `json:"commit_hash,omitempty"`
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

// instanceDeleteReceiptFactPayload contains only the exact, immutable
// protocol receipt. Checkpoint and durability observations are projections and
// therefore never participate in this fact's deterministic identity.
type instanceDeleteReceiptFactPayload struct {
	OperationID           string                  `json:"operation_id"`
	Operation             string                  `json:"operation"`
	ExpectedInvariant     instanceDeleteInvariant `json:"expected_invariant"`
	EventID               string                  `json:"event_id"`
	PublishedRootHash     string                  `json:"published_root_hash"`
	EventDigest           string                  `json:"event_digest,omitempty"`
	AuthorSeq             uint64                  `json:"author_seq"`
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

// recoverInstanceLifecycleTask resolves receipt-sensitive deletes before the
// generic task runner changes any interrupted running-state bookkeeping. A
// foreign pending receipt may be known locally while its event is not yet in a
// durable checkpoint; publishing a running->pending task update in that window
// can conflict with the pending author's transition. Keep the task untouched
// until the exact event is applied. The resumed stream then performs and
// persists the normal application-invariant outcome.
func (cm *Manager) recoverInstanceLifecycleTask(
	ctx context.Context,
	recovery *tasks.RecoveryContext[instanceLifecycleTaskPayload],
) (tasks.StreamRecoveryDisposition, error) {
	payload := recovery.Payload()
	if strings.TrimSpace(payload.Operation) != instanceLifecycleOperationDelete || payload.DeleteOperation == nil {
		return tasks.StreamRecoveryReady, nil
	}
	if strings.TrimSpace(payload.OperationStateModel) == instanceDeleteOperationFactsV1 {
		return cm.recoverInstanceLifecycleTaskFromOperationFacts(ctx, recovery)
	}
	identity := *payload.DeleteOperation
	if err := validateInstanceDeleteOperationIdentity(
		identity,
		recovery.Task().ID,
		strings.TrimSpace(payload.InstanceID),
		payload.LocalOnly,
	); err != nil {
		return tasks.StreamRecoveryReady, err
	}
	checkpointAuthorPeerID := strings.TrimSpace(payload.CheckpointAuthorPeerID)
	if checkpointAuthorPeerID == "" {
		// Payloads created before checkpoint ownership was explicit belong to the
		// immutable delete-operation author until a replicated recovery handoff.
		checkpointAuthorPeerID = identity.AuthorPeerID
	}
	// Recovery performs one local lookup/observation per runner tick. Bound the
	// call so a delayed receipt cannot block unrelated pending tasks forever.
	observeCtx, cancel := context.WithTimeout(ctx, instanceDeleteRecoveryObserveLimit)
	defer cancel()

	checkpointProgress := 4
	checkpointMessage := "prepared instance deletion operation"
	checkpointDetails := any(lifecycleTaskDetails(payload))
	if payload.DeleteReceipt != nil {
		checkpointProgress = recovery.Task().Progress
		checkpointMessage = recovery.Task().Message
		checkpointDetails = instanceDeleteReceiptDetails(*payload.DeleteReceipt)
		// If the exact receipt checkpoint was attempted but not installed in the
		// SQL view, its deterministic next state is still recoverable from receipt
		// status. Applied receipts always checkpoint at the protocol-complete phase.
		if payload.DeleteReceipt.AppliedDurably && checkpointProgress < 92 {
			checkpointProgress = 94
			checkpointMessage = "instance deletion event applied durably"
		}
	}
	expectedCheckpointRecord, err := recovery.PayloadCheckpointRecord(
		payload,
		checkpointProgress,
		checkpointMessage,
	)
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("reconstruct interrupted instance delete task checkpoint row %s: %w", recovery.Task().ID, err)
	}
	checkpointOperation, err := recovery.PayloadCheckpointOperation(
		identity.Key,
		checkpointAuthorPeerID,
		payload,
		checkpointProgress,
		checkpointMessage,
		checkpointDetails,
	)
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("reconstruct interrupted instance delete task checkpoint %s: %w", recovery.Task().ID, err)
	}
	lookupCheckpoint := cm.db.LookupPublishedWriteOperation
	if cm.lookupTaskCheckpointRecoveryOperation != nil {
		lookupCheckpoint = cm.lookupTaskCheckpointRecoveryOperation
	}
	resolvedCheckpoint, err := lookupCheckpoint(observeCtx, checkpointOperation)
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("resolve interrupted instance delete task checkpoint %s: %w", recovery.Task().ID, err)
	}
	switch resolvedCheckpoint.Resolution {
	case swarmionapp.BranchOperationReceiptUnavailable:
		log.Debugf(
			"deferring interrupted instance delete before task checkpoint recovery task_id=%s checkpoint_author=%s resolution=%s policy=bounded_background_reobserve",
			recovery.Task().ID,
			checkpointAuthorPeerID,
			resolvedCheckpoint.Resolution,
		)
		return tasks.StreamRecoveryDeferred, nil
	case swarmionapp.BranchOperationReceiptFound:
		checkpointReceipt, receiptErr := db.PublishedWriteReceiptFromOperation(resolvedCheckpoint)
		if receiptErr != nil {
			return tasks.StreamRecoveryReady, fmt.Errorf("recover interrupted instance delete task checkpoint %s: %w", recovery.Task().ID, receiptErr)
		}
		observeCheckpoint := cm.db.ObservePublishedWriteReceipt
		if cm.observeTaskCheckpointRecoveryReceipt != nil {
			observeCheckpoint = cm.observeTaskCheckpointRecoveryReceipt
		}
		checkpointObservation, observeErr := observeCheckpoint(observeCtx, checkpointReceipt)
		if observeErr != nil {
			return tasks.StreamRecoveryReady, fmt.Errorf("observe interrupted instance delete task checkpoint %s: %w", recovery.Task().ID, observeErr)
		}
		if checkpointObservation.State != db.EventReceiptStateAppliedDurably || !checkpointObservation.Status.AppliedDurably {
			log.Debugf(
				"deferring interrupted instance delete on exact task checkpoint task_id=%s checkpoint_author=%s event_id=%s published_root=%s state=%s policy=bounded_background_reobserve",
				recovery.Task().ID,
				checkpointAuthorPeerID,
				checkpointReceipt.EventID,
				checkpointReceipt.PublishedRootHash,
				checkpointObservation.State,
			)
			return tasks.StreamRecoveryDeferred, nil
		}
		eventCheckpointCommitID := strings.TrimSpace(checkpointObservation.Status.CheckpointCommitID)
		if eventCheckpointCommitID == "" {
			log.Debugf(
				"deferring interrupted instance delete task checkpoint invariant task_id=%s checkpoint_author=%s event_id=%s published_root=%s reason=checkpoint_commit_unavailable policy=bounded_background_reobserve",
				recovery.Task().ID,
				checkpointAuthorPeerID,
				checkpointReceipt.EventID,
				checkpointReceipt.PublishedRootHash,
			)
			return tasks.StreamRecoveryDeferred, nil
		}
		if cm.tasks == nil {
			return tasks.StreamRecoveryReady, fmt.Errorf("verify interrupted instance delete task checkpoint %s: task manager is not configured", recovery.Task().ID)
		}
		verifyCheckpoint := cm.tasks.VerifyCheckpointRecord
		if cm.verifyTaskCheckpointRecoveryInvariant != nil {
			verifyCheckpoint = cm.verifyTaskCheckpointRecoveryInvariant
		}
		invariantErr := verifyCheckpoint(observeCtx, eventCheckpointCommitID, expectedCheckpointRecord)
		if invariantErr != nil {
			if errors.Is(invariantErr, tasks.ErrCheckpointInvariantConflict) {
				return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
					"%w: task_id=%s event_id=%s published_root=%s event_checkpoint=%s durable_head=%s content_coverage=%s: %w",
					tasks.ErrCheckpointInvariantConflict,
					recovery.Task().ID,
					checkpointReceipt.EventID,
					checkpointReceipt.PublishedRootHash,
					eventCheckpointCommitID,
					strings.TrimSpace(checkpointObservation.Status.DurableCheckpointCommitID),
					checkpointObservation.Status.ContentCoverage,
					invariantErr,
				))
			}
			log.Debugf(
				"deferring interrupted instance delete task checkpoint invariant task_id=%s checkpoint_author=%s event_id=%s published_root=%s event_checkpoint=%s content_coverage=%s error=%s policy=bounded_background_reobserve",
				recovery.Task().ID,
				checkpointAuthorPeerID,
				checkpointReceipt.EventID,
				checkpointReceipt.PublishedRootHash,
				eventCheckpointCommitID,
				checkpointObservation.Status.ContentCoverage,
				invariantErr.Error(),
			)
			return tasks.StreamRecoveryDeferred, nil
		}
	case swarmionapp.BranchOperationReceiptAbsent:
		// The checkpoint body was not accepted, so generic recovery may return
		// this task to pending and retry its first safe attempt.
	default:
		return tasks.StreamRecoveryReady, fmt.Errorf(
			"resolve interrupted instance delete task checkpoint %s returned unknown resolution %q",
			recovery.Task().ID,
			resolvedCheckpoint.Resolution,
		)
	}

	var receipt instanceDeleteOperationReceipt
	if payload.DeleteReceipt != nil {
		receipt = *cloneInstanceDeleteOperationReceipt(payload.DeleteReceipt)
		if err := validateInstanceDeleteOperationReceipt(
			receipt,
			identity,
			recovery.Task().ID,
			payload.InstanceID,
		); err != nil {
			return tasks.StreamRecoveryReady, err
		}
	} else {
		lookupOperation := cm.db.WaitPublishedWriteOperation
		if cm.lookupDeleteRecoveryOperation != nil {
			lookupOperation = cm.lookupDeleteRecoveryOperation
		}
		resolved, err := lookupOperation(observeCtx, identity.publishedWriteOperation())
		if err != nil {
			return tasks.StreamRecoveryReady, fmt.Errorf("resolve interrupted instance delete operation %s: %w", recovery.Task().ID, err)
		}
		switch resolved.Resolution {
		case swarmionapp.BranchOperationReceiptAbsent:
			// No event was accepted for this identity, so normal task recovery can
			// safely resume the operation's first and only execution attempt.
			return tasks.StreamRecoveryReady, nil
		case swarmionapp.BranchOperationReceiptUnavailable:
			// A foreign miss is never authoritative absence. Leave the running
			// task and its replicated identity unchanged while peer state arrives.
			// The next quiescent runner tick observes again; it does not publish
			// task bookkeeping or a replacement delete.
			cm.logInstanceDeleteRecoveryDeferred(
				recovery.Task().ID,
				identity,
				instanceDeleteOperationReceipt{},
				db.EventReceiptStatePending,
				"foreign receipt unavailable",
			)
			return tasks.StreamRecoveryDeferred, nil
		case swarmionapp.BranchOperationReceiptFound:
			published, err := db.PublishedWriteReceiptFromOperation(resolved)
			if err != nil {
				return tasks.StreamRecoveryReady, fmt.Errorf("recover interrupted instance delete operation %s: %w", recovery.Task().ID, err)
			}
			receipt = instanceDeleteReceiptFromPublished(recovery.Task().ID, identity, published)
			if err := validateInstanceDeleteOperationReceipt(
				receipt,
				identity,
				recovery.Task().ID,
				payload.InstanceID,
			); err != nil {
				return tasks.StreamRecoveryReady, err
			}
		default:
			return tasks.StreamRecoveryReady, fmt.Errorf(
				"resolve interrupted instance delete operation %s returned unknown resolution %q",
				recovery.Task().ID,
				resolved.Resolution,
			)
		}
	}

	// This is deliberately one read, not a wait that performs task writes or a
	// publication retry. Startup recovery observes again on its next tick.
	observeReceipt := cm.db.ObservePublishedWriteReceipt
	if cm.observeDeleteRecoveryReceipt != nil {
		observeReceipt = cm.observeDeleteRecoveryReceipt
	}
	observation, err := observeReceipt(observeCtx, receipt.publishedWriteReceipt())
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("observe interrupted instance delete operation %s: %w", recovery.Task().ID, err)
	}
	receipt.applyObservation(observation)
	if observation.State != db.EventReceiptStateAppliedDurably || !receipt.AppliedDurably {
		cm.logInstanceDeleteRecoveryDeferred(
			recovery.Task().ID,
			identity,
			receipt,
			observation.State,
			string(observation.Status.ParkedReason),
		)
		return tasks.StreamRecoveryDeferred, nil
	}

	if strings.TrimSpace(receipt.DurableCheckpointCommitID) == "" {
		return tasks.StreamRecoveryReady, fmt.Errorf(
			"interrupted instance delete applied_durably without a durable checkpoint commit operation_id=%s event_id=%s",
			recovery.Task().ID,
			receipt.EventID,
		)
	}
	status, ok := cm.db.SwarmionStatus()
	if !ok || strings.TrimSpace(status.PeerID) == "" {
		return tasks.StreamRecoveryReady, fmt.Errorf(
			"interrupted instance delete cannot claim task checkpoint authorship: local Swarmion peer is unavailable",
		)
	}
	// Persisting the exact receipt is safe only after protocol completion. The
	// stream deliberately re-runs the durable-checkpoint invariant query so a
	// later recreation becomes its normal application-level conflict outcome,
	// rather than trapping startup recovery on a running task forever. The
	// ordinary running->pending recovery write also publishes the recovering
	// peer's checkpoint-author claim. Every later checkpoint by that peer is
	// causally descended from this replicated handoff; the immutable delete
	// operation continues to identify the original author's exact event.
	payload.DeleteReceipt = cloneInstanceDeleteOperationReceipt(&receipt)
	payload.CheckpointAuthorPeerID = strings.TrimSpace(status.PeerID)
	recovery.ReplacePayload(payload)
	return tasks.StreamRecoveryReady, nil
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

	var receipt instanceDeleteOperationReceipt
	receiptFact, receiptFound, err := cm.tasks.OperationFact(observeCtx, operationID, tasks.OperationFactKindReceipt)
	if err != nil {
		return tasks.StreamRecoveryReady, fmt.Errorf("read instance delete receipt fact %s: %w", operationID, err)
	}
	if receiptFound {
		if !effectFound {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: receipt fact exists without its atomic effect fact task_id=%s",
				tasks.ErrOperationFactConflict,
				operationID,
			))
		}
		receipt, err = instanceDeleteReceiptFromFact(receiptFact, identity)
		if err != nil {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(err)
		}
		if err := validateInstanceDeleteOperationReceipt(receipt, identity, operationID, instanceID); err != nil {
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(err)
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
		switch resolved.Resolution {
		case swarmionapp.BranchOperationReceiptUnavailable:
			return tasks.StreamRecoveryDeferred, nil
		case swarmionapp.BranchOperationReceiptFound:
			if !swarmionapp.SameBranchOperationReceiptIdentity(receipt.branchOperationReceipt(), resolved) {
				return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
					"%w: immutable receipt fact disagrees with Swarmion operation binding task_id=%s",
					tasks.ErrOperationFactConflict,
					operationID,
				))
			}
		default:
			return tasks.StreamRecoveryReady, tasks.MarkPermanent(fmt.Errorf(
				"%w: immutable receipt fact has no matching Swarmion operation binding task_id=%s resolution=%s safe_to_publish=%t",
				tasks.ErrOperationFactConflict,
				operationID,
				resolved.Resolution,
				resolved.SafeToPublish,
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
		switch resolved.Resolution {
		case swarmionapp.BranchOperationReceiptAbsent:
			if effectFound {
				// SQL already proves that the operation event exists. Treat a lagging
				// protocol binding as not-ready, never as permission to republish.
				return tasks.StreamRecoveryDeferred, nil
			}
			if !resolved.SafeToPublish {
				return tasks.StreamRecoveryDeferred, nil
			}
			return tasks.StreamRecoveryReady, nil
		case swarmionapp.BranchOperationReceiptUnavailable:
			return tasks.StreamRecoveryDeferred, nil
		case swarmionapp.BranchOperationReceiptFound:
			published, receiptErr := db.PublishedWriteReceiptFromOperation(resolved)
			if receiptErr != nil {
				return tasks.StreamRecoveryReady, fmt.Errorf("recover immutable instance delete operation %s: %w", operationID, receiptErr)
			}
			receipt = instanceDeleteReceiptFromPublished(operationID, identity, published)
		default:
			return tasks.StreamRecoveryReady, fmt.Errorf(
				"resolve immutable instance delete operation %s returned unknown resolution %q",
				operationID,
				resolved.Resolution,
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
	}
	payload.DeleteReceipt = cloneInstanceDeleteOperationReceipt(&receipt)
	recovery.ReplacePayload(payload)
	return tasks.StreamRecoveryReady, nil
}

func (receipt instanceDeleteOperationReceipt) branchOperationReceipt() swarmionapp.BranchOperationReceipt {
	return swarmionapp.BranchOperationReceipt{
		Resolution:        swarmionapp.BranchOperationReceiptFound,
		EventID:           strings.TrimSpace(receipt.EventID),
		PublishedRootHash: strings.TrimSpace(receipt.PublishedRootHash),
		EventDigest:       strings.TrimSpace(receipt.EventDigest),
		AuthorPeerID:      strings.TrimSpace(receipt.OperationAuthorPeerID),
		AuthorSeq:         receipt.AuthorSeq,
		IntentDigest:      strings.TrimSpace(receipt.OperationIntentDigest),
	}
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
		"protos:instance-record-delete:v1",
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
		"protos:instance-peer-drain-authorization-key:v1\x00" + strings.TrimSpace(deleteOperation.Key),
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
		"protos:instance-peer-drain-authorization:v1",
		instancePeerDrainAuthorizedV1,
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
		Version:         instancePeerDrainAuthorizedV1,
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
	if authorization.Version != instancePeerDrainAuthorizedV1 ||
		strings.TrimSpace(authorization.TaskID) != strings.TrimSpace(taskID) ||
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
		instancePeerDrainAuthorizedV1,
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
	if receipt.AuthorSeq == 0 {
		return tasks.OperationFact{}, fmt.Errorf("instance delete immutable receipt fact is missing its author sequence")
	}
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
			EventDigest:           strings.TrimSpace(receipt.EventDigest),
			AuthorSeq:             receipt.AuthorSeq,
			OperationIntentDigest: strings.TrimSpace(receipt.OperationIntentDigest),
			OperationAuthorPeerID: strings.TrimSpace(receipt.OperationAuthorPeerID),
		},
	)
}

func instanceDeleteIdentityFromEffectFact(fact tasks.OperationFact) (instanceDeleteOperationIdentity, error) {
	var payload instanceDeleteEffectFactPayload
	if err := json.Unmarshal(fact.Payload, &payload); err != nil {
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

func instanceDeleteReceiptFromFact(
	fact tasks.OperationFact,
	identity instanceDeleteOperationIdentity,
) (instanceDeleteOperationReceipt, error) {
	var payload instanceDeleteReceiptFactPayload
	if err := json.Unmarshal(fact.Payload, &payload); err != nil {
		return instanceDeleteOperationReceipt{}, fmt.Errorf("decode instance delete receipt fact: %w", err)
	}
	if fact.Kind != tasks.OperationFactKindReceipt ||
		strings.TrimSpace(fact.SubjectType) != taskSubjectInstance ||
		strings.TrimSpace(fact.SubjectID) != strings.TrimSpace(identity.ExpectedInvariant.InstanceID) ||
		fact.OperationKey != identity.Key ||
		fact.IntentDigest != identity.IntentDigest ||
		fact.AuthorPeerID != identity.AuthorPeerID ||
		strings.TrimSpace(payload.OperationIntentDigest) != identity.IntentDigest ||
		strings.TrimSpace(payload.OperationAuthorPeerID) != identity.AuthorPeerID {
		return instanceDeleteOperationReceipt{}, fmt.Errorf("%w: receipt/effect identity mismatch task_id=%s", tasks.ErrOperationFactConflict, fact.TaskID)
	}
	if payload.AuthorSeq == 0 {
		return instanceDeleteOperationReceipt{}, fmt.Errorf("%w: receipt fact is missing its immutable author sequence task_id=%s", tasks.ErrOperationFactConflict, fact.TaskID)
	}
	receipt := instanceDeleteOperationReceipt{
		OperationID:           strings.TrimSpace(payload.OperationID),
		Operation:             strings.TrimSpace(payload.Operation),
		ExpectedInvariant:     payload.ExpectedInvariant,
		EventID:               strings.TrimSpace(payload.EventID),
		PublishedRootHash:     strings.TrimSpace(payload.PublishedRootHash),
		EventDigest:           strings.TrimSpace(payload.EventDigest),
		AuthorSeq:             payload.AuthorSeq,
		OperationIntentDigest: strings.TrimSpace(payload.OperationIntentDigest),
		OperationAuthorPeerID: strings.TrimSpace(payload.OperationAuthorPeerID),
	}
	return receipt, nil
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
		"protos:instance-record-delete:v1",
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

func (cm *Manager) QueueDesiredInstanceReconciles() error {
	if cm == nil || cm.tasks == nil {
		return fmt.Errorf("task manager is not configured")
	}
	instances, err := cm.GetInstances(false)
	if err != nil {
		return err
	}
	var failures []string
	for _, instance := range instances {
		if IsDeletingInstance(instance) || strings.TrimSpace(instance.PublicKey) == "" {
			continue
		}
		ownerPeerID, ownerErr := instanceLifecycleOwner(instance)
		if ownerErr != nil || ownerPeerID != cm.localLifecycleExecutorPeerID() || cm.providerMutationDisabled {
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
		if _, err := cm.queueInstanceLifecycle(instance, instanceLifecycleOperationReconcile, desiredStatus, false, false, sig); err != nil {
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

func (cm *Manager) QueueStartInstance(id string) (tasks.Record, error) {
	return cm.queueSetInstanceDesiredStatus(id, ServerStateRunning)
}

func (cm *Manager) QueueStopInstance(id string) (tasks.Record, error) {
	return cm.queueSetInstanceDesiredStatus(id, ServerStateStopped)
}

func (cm *Manager) QueueDeleteInstance(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueDeleteInstance(ctx, id, false)
}

func (cm *Manager) QueueDeleteInstanceLocal(ctx context.Context, id string) (tasks.Record, error) {
	return cm.queueDeleteInstance(ctx, id, true)
}

func (cm *Manager) queueSetInstanceDesiredStatus(id string, desiredStatus string) (tasks.Record, error) {
	unlock := cm.lockInstanceLifecycle(id)
	defer unlock()
	instance, err := cm.getInstanceRecord(id)
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
	if authorized, authErr := cm.instancePeerDrainAuthorizationExists(context.Background(), instance.ID); authErr != nil {
		return tasks.Record{}, fmt.Errorf("inspect peer-drain authorization for instance '%s': %w", instance.Name, authErr)
	} else if authorized {
		return tasks.Record{}, fmt.Errorf("%w: instance '%s' has immutable peer-drain authorization", ErrInstanceLifecycleConflict, instance.Name)
	}
	if cm.tasks != nil {
		deleteSubject := instanceLifecycleSubjectID(instance.ID, instanceLifecycleOperationDelete)
		if task, found, lookupErr := cm.tasks.LatestForSubject(InstanceLifecycleTaskStream, taskSubjectInstance, deleteSubject); lookupErr != nil {
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
	if err := db.Update(cm.db, im); err != nil {
		return tasks.Record{}, fmt.Errorf("failed to save instance '%s': %w", id, err)
	}
	if authorized, authErr := cm.instancePeerDrainAuthorizationExists(context.Background(), instance.ID); authErr != nil {
		return tasks.Record{}, fmt.Errorf("verify peer-drain authorization for instance '%s': %w", instance.Name, authErr)
	} else if authorized {
		return tasks.Record{}, fmt.Errorf("%w: instance '%s' became delete-authorized while changing desired status", ErrInstanceLifecycleConflict, instance.Name)
	}
	persisted, err := cm.getInstanceRecord(instance.ID)
	if err != nil || persisted.DesiredStatus != desiredStatus {
		return tasks.Record{}, fmt.Errorf("%w: instance '%s' desired status update was not accepted", ErrInstanceLifecycleConflict, instance.Name)
	}
	sig := lifecycleDesiredSignature(instance, desiredStatus)
	cm.clearLifecycleSignature(instance.ID)
	record, err := cm.queueInstanceLifecycle(instance, instanceLifecycleOperationReconcile, desiredStatus, false, true, sig)
	if err != nil {
		return tasks.Record{}, err
	}
	cm.setLifecycleSignature(instance.ID, sig)
	log.Infof("Queued desired status '%s' for instance '%s' as task '%s'", desiredStatus, instance.Name, record.ID)
	return record, nil
}

func (cm *Manager) queueDeleteInstance(ctx context.Context, id string, localOnly bool) (tasks.Record, error) {
	unlock := cm.lockInstanceLifecycle(id)
	defer unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return tasks.Record{}, err
	}
	instance, err := cm.getInstanceRecord(id)
	if err != nil {
		return tasks.Record{}, fmt.Errorf("could not retrieve instance '%s': %w", id, err)
	}
	if _, err := cm.lifecycleTaskOwner(instance); err != nil {
		return tasks.Record{}, err
	}
	record, err := cm.queueInstanceLifecycle(instance, instanceLifecycleOperationDelete, ServerStateDeleting, localOnly, true, "")
	if err != nil {
		return tasks.Record{}, err
	}
	cm.clearLifecycleSignature(instance.ID)
	log.Infof("Queued delete for instance '%s' as task '%s'", instance.Name, record.ID)
	return record, nil
}

func (cm *Manager) queueInstanceLifecycle(instance InstanceInfo, operation string, desiredStatus string, localOnly bool, requestedByAPI bool, desiredSig string) (tasks.Record, error) {
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
	operationStateModel := ""
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
		operationStateModel = instanceDeleteOperationFactsV1
	}
	record, _, err := tasks.EnqueueUnique(cm.tasks, tasks.EnqueueUniqueOptions[instanceLifecycleTaskPayload]{
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
				OperationStateModel:    operationStateModel,
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
		useOperationFacts := strings.TrimSpace(payload.OperationStateModel) == instanceDeleteOperationFactsV1
		identity := payload.DeleteOperation
		persistedInstance, persistedErr := cm.getInstanceRecord(instanceID)
		if persistedErr != nil && !errors.Is(persistedErr, stdsql.ErrNoRows) {
			return instanceLifecycleTaskResult{}, tasks.MarkPermanent(fmt.Errorf("load delete authority: %w", persistedErr))
		}
		if identity == nil {
			if persistedErr != nil {
				return instanceLifecycleTaskResult{}, fmt.Errorf("prepare instance delete operation: %w", persistedErr)
			}
			ownerPeerID, ownerErr := instanceLifecycleOwner(persistedInstance)
			if ownerErr != nil {
				return instanceLifecycleTaskResult{}, tasks.MarkPermanent(ownerErr)
			}
			prepared, err := newInstanceDeleteOperationIdentity(task.Task().ID, persistedInstance, payload.LocalOnly, ownerPeerID)
			if err != nil {
				return instanceLifecycleTaskResult{}, err
			}
			identity = &prepared
			payload.DeleteOperation = identity
			if !useOperationFacts {
				payload.CheckpointAuthorPeerID = ownerPeerID
			}
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
		checkpointAuthorPeerID := strings.TrimSpace(payload.CheckpointAuthorPeerID)
		if checkpointAuthorPeerID == "" {
			checkpointAuthorPeerID = strings.TrimSpace(identity.AuthorPeerID)
			if !useOperationFacts {
				payload.CheckpointAuthorPeerID = checkpointAuthorPeerID
			}
		}
		if payload.DeleteReceipt == nil {
			if !useOperationFacts {
				// Legacy tasks keep their original checkpoint contract. New tasks put
				// the immutable effect marker in D itself and never create T92.
				if err := task.CheckpointPayloadWithOperationAuthor(
					identity.Key,
					checkpointAuthorPeerID,
					payload,
					4,
					"prepared instance deletion operation",
					lifecycleTaskDetails(payload),
				); err != nil {
					return instanceLifecycleTaskResult{}, err
				}
			}
			if err := task.Update(5, "deleting instance", lifecycleTaskDetails(payload)); err != nil {
				return instanceLifecycleTaskResult{}, err
			}
		}
		err := cm.deleteInstanceImperative(
			ctx,
			task.Update,
			instanceID,
			payload.LocalOnly,
			task.Task().ID,
			*identity,
			payload.PeerDrainAuthorization,
			payload.DeleteReceipt,
			func(next instanceDeleteOperationReceipt, progress int, message string) error {
				payload.DeleteReceipt = cloneInstanceDeleteOperationReceipt(&next)
				if useOperationFacts {
					fact, err := newInstanceDeleteReceiptFact(next, *identity)
					if err != nil {
						return err
					}
					return task.RecordOperationFact(ctx, fact)
				}
				return task.CheckpointPayloadWithOperationAuthor(
					identity.Key,
					checkpointAuthorPeerID,
					payload,
					progress,
					message,
					instanceDeleteReceiptDetails(next),
				)
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
			DeleteReceipt: cloneInstanceDeleteOperationReceipt(payload.DeleteReceipt),
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

func instanceDeleteReceiptDetails(receipt instanceDeleteOperationReceipt) map[string]any {
	return map[string]any{
		"operation_id":                 receipt.OperationID,
		"operation":                    receipt.Operation,
		"instance_id":                  receipt.ExpectedInvariant.InstanceID,
		"event_id":                     receipt.EventID,
		"published_root_hash":          receipt.PublishedRootHash,
		"event_digest":                 receipt.EventDigest,
		"operation_intent_digest":      receipt.OperationIntentDigest,
		"operation_author_peer_id":     receipt.OperationAuthorPeerID,
		"checkpoint_commit_id":         receipt.CheckpointCommitID,
		"checkpoint_root_hash":         receipt.CheckpointRootHash,
		"durable_checkpoint_commit_id": receipt.DurableCheckpointCommitID,
		"durable_checkpoint_root_hash": receipt.DurableCheckpointRootHash,
		"queryable_root_hash":          receipt.QueryableRootHash,
		"checkpointed":                 receipt.Checkpointed,
		"applied_durably":              receipt.AppliedDurably,
		"content_coverage":             receipt.ContentCoverage,
		"content_durable":              receipt.ContentDurable,
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

func (cm *Manager) QueueUploadLocalImage(imagePath string, imageName string, provisionerName string, location string, timeout time.Duration) (tasks.Record, error) {
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
	return tasks.Enqueue(cm.tasks, tasks.EnqueueOptions[uploadLocalImageTaskPayload]{
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

func (cm *Manager) updateDeploymentPlaceholder(instance InstanceInfo) error {
	im, cmm := createInstanceUpdateMapper(instance)
	if err := db.Update(cm.db, im, cmm); err != nil {
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
	if err := db.UpdateAndInsertPublishedContext(
		ctx,
		cm.db,
		[]db.UpdateMapper{im, cmm},
		[]db.InsertMapper{db.CreatePeerInsertMapper(instance.PublicKey)},
	); err != nil {
		return fmt.Errorf("persist discovered identity for pending instance '%s': %w", pendingID, err)
	}
	return nil
}

func (cm *Manager) completeDeploymentInstance(pendingID string, instance InstanceInfo) error {
	if _, err := db.SelectOne(cm.db, createInstanceQueryMapper(pendingID)); err != nil {
		return fmt.Errorf("pending instance '%s' no longer exists: %w", pendingID, err)
	}
	instance.ID = pendingID
	im, cmm := createInstanceFinalizeMapper(pendingID, instance)
	if err := db.Update(cm.db, im, cmm); err != nil {
		return err
	}
	return nil
}
