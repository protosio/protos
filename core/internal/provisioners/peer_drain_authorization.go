package provisioners

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/tasks"
)

var errInstancePeerDrainAuthorizationFactAbsent = errors.New("peer-drain authorization fact absent from checkpoint")

func instancePeerDrainAuthorizationFromFact(fact tasks.OperationFact) (instancePeerDrainAuthorization, error) {
	var authorization instancePeerDrainAuthorization
	if err := decodeOperationFactPayload(fact.Payload, &authorization); err != nil {
		return instancePeerDrainAuthorization{}, fmt.Errorf("decode peer-drain authorization fact: %w", err)
	}
	if fact.Kind != instancePeerDrainAuthorizationFact ||
		strings.TrimSpace(fact.TaskID) != strings.TrimSpace(authorization.TaskID) ||
		strings.TrimSpace(fact.SubjectType) != taskSubjectInstance ||
		strings.TrimSpace(fact.SubjectID) != strings.TrimSpace(authorization.InstanceID) ||
		strings.TrimSpace(fact.OperationKey) != strings.TrimSpace(authorization.Operation.Key()) ||
		strings.TrimSpace(fact.IntentDigest) != strings.TrimSpace(authorization.Operation.IntentDigest()) ||
		strings.TrimSpace(fact.AuthorPeerID) != strings.TrimSpace(authorization.Operation.AuthorPeerID()) {
		return instancePeerDrainAuthorization{}, fmt.Errorf("%w: peer-drain authorization fact identity mismatch task=%s", tasks.ErrOperationFactConflict, fact.TaskID)
	}
	if err := validateInstancePeerDrainAuthorization(
		authorization,
		authorization.TaskID,
		authorization.DeleteOperation,
		authorization.InstanceID,
		authorization.LocalOnly,
	); err != nil {
		return instancePeerDrainAuthorization{}, fmt.Errorf("%w: %w", tasks.ErrOperationFactConflict, err)
	}
	return authorization, nil
}

func validateInstanceAgainstPeerDrainAuthorization(instance InstanceInfo, authorization instancePeerDrainAuthorization) error {
	expected := authorization.expectedInstance()
	if strings.TrimSpace(instance.ID) != strings.TrimSpace(expected.ID) ||
		instance.Name != expected.Name || instance.Kind != expected.Kind || instance.KindID != expected.KindID ||
		instance.ProviderResourceID != expected.ProviderResourceID || instance.ReplicationPriority != expected.ReplicationPriority ||
		instance.PublicIP != expected.PublicIP || instance.Location != expected.Location ||
		instance.Architecture != expected.Architecture || instance.PublicKey != expected.PublicKey ||
		strings.TrimSpace(instance.LifecycleOwnerPeerID) != strings.TrimSpace(expected.LifecycleOwnerPeerID) {
		return fmt.Errorf("%w: live instance %s does not match immutable peer-drain authorization", ErrInstanceDeleteInvariantConflict, instance.ID)
	}
	if instance.DesiredStatus != expected.DesiredStatus && instance.DesiredStatus != ServerStateDeleting {
		return fmt.Errorf("%w: live instance %s status %q is neither authorized pre-status %q nor deleting", ErrInstanceDeleteInvariantConflict, instance.ID, instance.DesiredStatus, expected.DesiredStatus)
	}
	peerID, err := instance.GetPeerID()
	if err != nil || peerID != authorization.PeerID {
		return fmt.Errorf("%w: live instance %s peer does not match immutable peer-drain authorization", ErrInstanceDeleteInvariantConflict, instance.ID)
	}
	return nil
}

func (cm *Manager) instancePeerDrainAuthorizationExists(ctx context.Context, instanceID string) (bool, error) {
	if cm == nil || cm.db == nil {
		return false, fmt.Errorf("peer-drain authorization store is not configured")
	}
	found := false
	err := cm.db.ReadRows(
		ctx,
		"SELECT id FROM task_operation_facts WHERE fact_kind = ? AND subject_type = ? AND subject_id = ? LIMIT 1",
		[]any{instancePeerDrainAuthorizationFact, taskSubjectInstance, strings.TrimSpace(instanceID)},
		func(rows *stdsql.Rows) error {
			found = rows.Next()
			if found {
				var ignored string
				if err := rows.Scan(&ignored); err != nil {
					return err
				}
			}
			return rows.Err()
		},
	)
	return found, err
}

func (cm *Manager) resolveInstancePeerDrainAuthorization(
	ctx context.Context,
	authorization instancePeerDrainAuthorization,
) (db.PublishedWriteReceipt, bool, error) {
	lookup := cm.db.LookupPublishedWriteOperation
	if cm.lookupPeerDrainAuthorization != nil {
		lookup = cm.lookupPeerDrainAuthorization
	}
	resolved, err := lookup(ctx, authorization.publishedWriteOperation())
	if err != nil {
		return db.PublishedWriteReceipt{}, false, fmt.Errorf("resolve peer-drain authorization P for task %s: %w", authorization.TaskID, err)
	}
	switch resolved.Disposition() {
	case swarmionapp.OperationAccepted:
		receipt, err := db.PublishedWriteReceiptFromResult(authorization.Operation, resolved)
		if err != nil {
			return db.PublishedWriteReceipt{}, false, fmt.Errorf("decode peer-drain authorization P receipt for task %s: %w", authorization.TaskID, err)
		}
		if err := validatePeerDrainAuthorizationReceipt(receipt, authorization); err != nil {
			return db.PublishedWriteReceipt{}, false, err
		}
		return receipt, true, nil
	case swarmionapp.OperationRecoveryRequired:
		return db.PublishedWriteReceipt{}, false, fmt.Errorf("%w: peer-drain authorization P task=%s", db.ErrOperationReceiptUnavailable, authorization.TaskID)
	case swarmionapp.OperationRetryPermitted:
		return db.PublishedWriteReceipt{}, false, nil
	case swarmionapp.OperationFailedClosed:
		return db.PublishedWriteReceipt{}, false, fmt.Errorf("peer-drain authorization P task=%s failed closed: %s", authorization.TaskID, operationDiagnosticText(resolved.Diagnostic()))
	default:
		return db.PublishedWriteReceipt{}, false, fmt.Errorf("peer-drain authorization P task=%s returned unknown disposition %q", authorization.TaskID, resolved.Disposition())
	}
}

func validatePeerDrainAuthorizationReceipt(
	receipt db.PublishedWriteReceipt,
	authorization instancePeerDrainAuthorization,
) error {
	if !receipt.HasExactEventIdentity() {
		return fmt.Errorf("peer-drain authorization P task=%s returned no exact event identity", authorization.TaskID)
	}
	if strings.TrimSpace(receipt.OperationIntentDigest) != strings.TrimSpace(authorization.Operation.IntentDigest()) ||
		strings.TrimSpace(receipt.AuthorPeerID) != strings.TrimSpace(authorization.Operation.AuthorPeerID()) {
		return fmt.Errorf("%w: peer-drain authorization P response identity does not match task %s", db.ErrPublishedWriteReceiptIdentityConflict, authorization.TaskID)
	}
	return nil
}

func samePeerDrainAuthorizationReceipt(left, right db.PublishedWriteReceipt) bool {
	return strings.TrimSpace(left.EventID) == strings.TrimSpace(right.EventID) &&
		strings.TrimSpace(left.PublishedRootHash) == strings.TrimSpace(right.PublishedRootHash) &&
		strings.TrimSpace(left.AuthorPeerID) == strings.TrimSpace(right.AuthorPeerID) &&
		strings.TrimSpace(left.OperationIntentDigest) == strings.TrimSpace(right.OperationIntentDigest)
}

// completeInstancePeerDrainAuthorization publishes P once except for outcomes
// explicitly proven not accepted and safe to retry, resolves its authoritative
// exact receipt, waits for AppliedDurably, then validates P's immutable fact and
// exact deleting instance snapshot at that event checkpoint.
func (cm *Manager) completeInstancePeerDrainAuthorization(
	ctx context.Context,
	authorization instancePeerDrainAuthorization,
	fact tasks.OperationFact,
	receipt db.PublishedWriteReceipt,
	alreadyAccepted bool,
	runtimeGeneration uint64,
) (db.PublishedWriteReceipt, error) {
	expected := authorization.expectedInstance()
	if err := cm.assertInstanceLifecycleExecutor(expected, authorization.Operation.AuthorPeerID()); err != nil {
		return db.PublishedWriteReceipt{}, err
	}
	if !alreadyAccepted {
		publisher := cm.publishPeerDrainAuthorization
		if publisher == nil {
			publisher = func(ctx context.Context, operation db.PublishedWriteOperation, instance InstanceInfo, fact tasks.OperationFact) (db.PublishedWriteReceipt, error) {
				if runtimeGeneration == 0 {
					return db.PublishedWriteReceipt{}, fmt.Errorf(
						"%w: peer-drain authorization P task=%s has no runtime generation",
						db.ErrOperationReceiptUnavailable,
						authorization.TaskID,
					)
				}
				return db.InsertAndUpdateWithOperationReceiptForRuntimeGenerationContext(
					ctx,
					cm.db,
					operation,
					runtimeGeneration,
					[]db.InsertMapper{createInstancePeerDrainAuthorizationFactCASMapper(fact, instance)},
					[]db.UpdateMapper{createInstancePeerDrainAuthorizationUpdateMapper(instance, fact)},
				)
			}
		}
		if err := ctx.Err(); err != nil {
			return db.PublishedWriteReceipt{}, instanceDeletePublicationWithoutReceipt(err)
		}
		published, publishErr := publisher(ctx, authorization.publishedWriteOperation(), expected, fact)
		if published.HasExactEventIdentity() {
			if err := validatePeerDrainAuthorizationReceipt(published, authorization); err != nil {
				return db.PublishedWriteReceipt{}, err
			}
			receipt = published
			if publishErr != nil {
				log.Warnf("tracking peer-drain authorization P after receipt-return error task_id=%s event_id=%s: %v", authorization.TaskID, receipt.EventID, publishErr)
			}
		} else if publishErr == nil {
			return db.PublishedWriteReceipt{}, instanceDeletePublicationWithoutReceipt(
				fmt.Errorf("peer-drain authorization P returned neither receipt nor error"),
			)
		} else {
			// The DB helper consumes direct OperationRetryPermitted results under
			// the same runtime lease. Errors here are diagnostic failures only and
			// never authorize another publication attempt in this layer.
			return db.PublishedWriteReceipt{}, instanceDeletePublicationWithoutReceipt(publishErr)
		}

		resolveCtx, cancelResolve := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		resolved, found, err := cm.resolveInstancePeerDrainAuthorization(resolveCtx, authorization)
		cancelResolve()
		if err != nil {
			return db.PublishedWriteReceipt{}, err
		}
		if !found || !samePeerDrainAuthorizationReceipt(receipt, resolved) {
			return db.PublishedWriteReceipt{}, fmt.Errorf("%w: peer-drain authorization P response disagrees with deterministic resolution task=%s", db.ErrPublishedWriteReceiptIdentityConflict, authorization.TaskID)
		}
		receipt = resolved
	}

	wait := cm.db.WaitForPublishedWriteApplied
	if cm.waitPeerDrainAuthorization != nil {
		wait = cm.waitPeerDrainAuthorization
	}
	observation, err := wait(ctx, receipt, "verify instance peer-drain authorization P")
	if err != nil {
		return db.PublishedWriteReceipt{}, err
	}
	if observation.State != db.EventReceiptStateAppliedDurably || !observation.Status.AppliedDurably {
		return db.PublishedWriteReceipt{}, fmt.Errorf("%w: peer-drain authorization P task=%s did not reach applied_durably", db.ErrEventReceiptPending, authorization.TaskID)
	}
	if !samePeerDrainAuthorizationReceipt(receipt, observation.Receipt) {
		return db.PublishedWriteReceipt{}, fmt.Errorf("%w: peer-drain authorization P wait returned a different exact receipt for task %s", db.ErrPublishedWriteReceiptIdentityConflict, authorization.TaskID)
	}
	if !observation.Status.Checkpointed {
		return db.PublishedWriteReceipt{}, fmt.Errorf("%w: peer-drain authorization P task=%s reached applied_durably without checkpointed state", db.ErrEventReceiptPending, authorization.TaskID)
	}
	eventCheckpointCommitID := strings.TrimSpace(observation.Status.CheckpointCommitID)
	if eventCheckpointCommitID == "" {
		return db.PublishedWriteReceipt{}, fmt.Errorf("peer-drain authorization P task=%s applied_durably without its exact event checkpoint", authorization.TaskID)
	}
	durableCheckpointCommitID := strings.TrimSpace(observation.Status.DurableCheckpointCommitID)
	if durableCheckpointCommitID == "" {
		return db.PublishedWriteReceipt{}, fmt.Errorf("peer-drain authorization P task=%s applied_durably without a durable checkpoint", authorization.TaskID)
	}
	verify := cm.verifyPeerDrainAuthorization
	if verify == nil {
		verify = cm.verifyInstancePeerDrainAuthorizationAtCheckpoint
	}
	// Check the immutable fact and deleting snapshot at this event's exact
	// checkpoint. DurableCheckpointCommitID is only lineage evidence that the
	// exact event checkpoint has reached durable state; it may be a later head
	// containing unrelated or superseding writes.
	if err := verify(ctx, eventCheckpointCommitID, authorization, fact); err != nil {
		if errors.Is(err, errInstancePeerDrainAuthorizationFactAbsent) {
			return db.PublishedWriteReceipt{}, fmt.Errorf("%w: peer-drain authorization P exact event checkpoint %s contains no authorization fact", ErrInstanceDeleteInvariantConflict, eventCheckpointCommitID)
		}
		return db.PublishedWriteReceipt{}, err
	}
	return observation.Receipt, nil
}

func (cm *Manager) verifyInstancePeerDrainAuthorizationAtCheckpoint(
	ctx context.Context,
	checkpointCommitID string,
	authorization instancePeerDrainAuthorization,
	fact tasks.OperationFact,
) error {
	if cm == nil || cm.db == nil || cm.tasks == nil {
		return fmt.Errorf("peer-drain authorization verifier is not configured")
	}
	matched, err := cm.tasks.OperationFactMatchesAtCheckpoint(ctx, checkpointCommitID, fact)
	if err != nil {
		return fmt.Errorf("%w: verify peer-drain authorization fact at checkpoint %s: %w", ErrInstanceDeleteInvariantConflict, checkpointCommitID, err)
	}
	if !matched {
		return fmt.Errorf("%w: %s", errInstancePeerDrainAuthorizationFactAbsent, checkpointCommitID)
	}
	expected := authorization.expectedInstance()
	var machineRows int
	if err := cm.db.ReadRowsAsOf(
		ctx,
		checkpointCommitID,
		"SELECT name, kind, desired_status, replication_priority FROM machines AS OF ? WHERE id = ?",
		[]any{db.MustUUIDBytes(expected.ID)},
		func(rows *stdsql.Rows) error {
			for rows.Next() {
				machineRows++
				var name, kind, desiredStatus string
				var priority int
				if err := rows.Scan(&name, &kind, &desiredStatus, &priority); err != nil {
					return err
				}
				if name != expected.Name || kind != expected.Kind || desiredStatus != ServerStateDeleting || priority != expected.ReplicationPriority {
					return fmt.Errorf("machine row does not match authorized deleting identity")
				}
			}
			return rows.Err()
		},
	); err != nil {
		return fmt.Errorf("%w: verify peer-drain machine row: %w", ErrInstanceDeleteInvariantConflict, err)
	}
	if machineRows != 1 {
		return fmt.Errorf("%w: peer-drain authorization checkpoint contains %d machine rows", ErrInstanceDeleteInvariantConflict, machineRows)
	}

	var metadataRows int
	if err := cm.db.ReadRowsAsOf(
		ctx,
		checkpointCommitID,
		"SELECT cloud_id, provider_resource_id, public_ip, location, architecture, public_key FROM cloud_machines_metadata AS OF ? WHERE id = ?",
		[]any{db.MustUUIDBytes(expected.ID)},
		func(rows *stdsql.Rows) error {
			for rows.Next() {
				metadataRows++
				var kindID, resourceID, publicIP, location, architecture, publicKey string
				if err := rows.Scan(&kindID, &resourceID, &publicIP, &location, &architecture, &publicKey); err != nil {
					return err
				}
				if kindID != expected.KindID || resourceID != expected.ProviderResourceID || publicIP != expected.PublicIP ||
					location != expected.Location || architecture != expected.Architecture || publicKey != expected.PublicKey {
					return fmt.Errorf("provider metadata row does not match authorization")
				}
			}
			return rows.Err()
		},
	); err != nil {
		return fmt.Errorf("%w: verify peer-drain provider row: %w", ErrInstanceDeleteInvariantConflict, err)
	}
	if metadataRows != 1 {
		return fmt.Errorf("%w: peer-drain authorization checkpoint contains %d provider rows", ErrInstanceDeleteInvariantConflict, metadataRows)
	}
	return nil
}
