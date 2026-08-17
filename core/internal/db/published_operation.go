package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	swarmion "github.com/nustiueudinastea/swarmion/runtime"
)

const (
	// These are immutable application-level intent schemas. They are deliberately
	// independent of Swarmion's private digest-framing revision.
	OperationSchemaMigrationBatch     = "io.protos.migrations.batch/v1"
	OperationSchemaOrdinarySQLWrite   = "io.protos.sql.write/v1"
	OperationSchemaInstanceDelete     = "io.protos.instances.delete/v1"
	OperationSchemaPeerDrainAuthorize = "io.protos.instances.peer-drain-authorize/v1"
	operationRecoveryJournalVersion   = 1
	operationRecoveryJournalDir       = "protos-operation-recovery"
	migrationOperationRecordSuffix    = ".protos-migration-operation.json"
)

// PublishedWriteOperation is the complete restart-safe address of one exact
// mutation. Both opaque Swarmion values must be persisted before Execute; a
// key/digest/author tuple is intentionally insufficient because it cannot
// reconstruct the route-scoped recovery record after a crash.
type PublishedWriteOperation struct {
	Identity swarmion.OperationIdentity `json:"identity"`
	Recovery swarmion.OperationRecovery `json:"recovery"`
}

func (operation PublishedWriteOperation) Validate() error {
	if err := operation.Identity.Validate(); err != nil {
		return fmt.Errorf("operation identity: %w", err)
	}
	if err := operation.Recovery.Validate(); err != nil {
		return fmt.Errorf("operation recovery: %w", err)
	}
	if !operationIdentitiesEqual(operation.Recovery.Identity(), operation.Identity) {
		return fmt.Errorf("operation recovery identity does not match operation identity")
	}
	if _, branch := operation.Recovery.BranchID(); branch {
		return fmt.Errorf("published database write unexpectedly carries branch recovery")
	}
	return nil
}

func (operation PublishedWriteOperation) Key() string {
	return operation.Identity.Key()
}

func (operation PublishedWriteOperation) IntentDigest() string {
	return operation.Identity.IntentDigest()
}

func (operation PublishedWriteOperation) AuthorPeerID() string {
	return operation.Recovery.AuthorPeerID()
}

func (operation PublishedWriteOperation) Equal(other PublishedWriteOperation) bool {
	if operation.Validate() != nil || other.Validate() != nil || !operationIdentitiesEqual(operation.Identity, other.Identity) {
		return false
	}
	if operation.Recovery.Namespace() != other.Recovery.Namespace() ||
		operation.Recovery.AuthorPeerID() != other.Recovery.AuthorPeerID() {
		return false
	}
	leftBranch, leftIsBranch := operation.Recovery.BranchID()
	rightBranch, rightIsBranch := other.Recovery.BranchID()
	if leftIsBranch != rightIsBranch || leftBranch != rightBranch {
		return false
	}
	leftReceipt, leftHasReceipt := operation.Recovery.ExpectedReceipt()
	rightReceipt, rightHasReceipt := other.Recovery.ExpectedReceipt()
	return leftHasReceipt == rightHasReceipt && (!leftHasReceipt || leftReceipt == rightReceipt)
}

func operationIdentitiesEqual(left, right swarmion.OperationIdentity) bool {
	return left.Validate() == nil && right.Validate() == nil &&
		left.Key() == right.Key() && left.IntentDigest() == right.IntentDigest()
}

// NewPublishedWriteOperation creates and durably records a random, opaque
// operation identity and its exact database-line recovery address. The caller
// must additionally persist the returned value in its own business workflow
// before invoking a mutation helper.
func (db *DB) NewPublishedWriteOperation(schema string, parts ...[]byte) (PublishedWriteOperation, error) {
	identity, err := swarmion.NewRandomOperationIdentity(schema, parts...)
	if err != nil {
		return PublishedWriteOperation{}, err
	}
	return db.newPublishedWriteOperation(identity)
}

func (db *DB) newPublishedWriteOperation(identity swarmion.OperationIdentity) (PublishedWriteOperation, error) {
	if db == nil {
		return PublishedWriteOperation{}, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	runtime := db.runtime
	db.mu.Unlock()
	if runtime == nil {
		return PublishedWriteOperation{}, fmt.Errorf("db is not initialized")
	}
	recovery, err := runtime.OperationRecovery(identity)
	if err != nil {
		return PublishedWriteOperation{}, fmt.Errorf("construct operation recovery: %w", err)
	}
	operation := PublishedWriteOperation{Identity: identity, Recovery: recovery}
	if err := operation.Validate(); err != nil {
		return PublishedWriteOperation{}, err
	}
	if err := db.persistPublishedWriteOperation(operation); err != nil {
		return PublishedWriteOperation{}, err
	}
	return operation, nil
}

func (db *DB) migrationPublishedWriteOperation(parts ...[]byte) (PublishedWriteOperation, error) {
	if db == nil {
		return PublishedWriteOperation{}, fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	workingDir := db.workingDir
	databaseName := db.name
	db.mu.Unlock()
	stateDir := filepath.Join(workingDir, swarmionStateDirName)
	path := filepath.Join(stateDir, databaseName+migrationOperationRecordSuffix)
	if record, err := readPersistedPublishedWriteOperation(path); err == nil {
		expected, identityErr := swarmion.NewOperationIdentity(record.Operation.Key(), OperationSchemaMigrationBatch, parts...)
		if identityErr != nil {
			return PublishedWriteOperation{}, identityErr
		}
		if operationIdentitiesEqual(expected, record.Operation.Identity) {
			if err := db.persistPublishedWriteOperation(record.Operation); err != nil {
				return PublishedWriteOperation{}, err
			}
			return record.Operation, nil
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return PublishedWriteOperation{}, fmt.Errorf("read persisted migration operation: %w", err)
	}
	operation, err := db.NewPublishedWriteOperation(OperationSchemaMigrationBatch, parts...)
	if err != nil {
		return PublishedWriteOperation{}, err
	}
	if err := ensureCrashDurableDirectory(workingDir, stateDir); err != nil {
		return PublishedWriteOperation{}, fmt.Errorf("create migration operation directory: %w", err)
	}
	encoded, err := json.Marshal(persistedPublishedWriteOperation{
		Version: operationRecoveryJournalVersion, Operation: operation,
	})
	if err != nil {
		return PublishedWriteOperation{}, err
	}
	if err := replaceCrashDurableFile(path, encoded, 0o600); err != nil {
		return PublishedWriteOperation{}, fmt.Errorf("persist migration operation: %w", err)
	}
	return operation, nil
}

type persistedPublishedWriteOperation struct {
	Version   int                     `json:"version"`
	Operation PublishedWriteOperation `json:"operation"`
}

func (db *DB) persistPublishedWriteOperation(operation PublishedWriteOperation) error {
	return db.persistPublishedWriteOperationMode(operation, false)
}

// enrichPersistedPublishedWriteOperation is used by passive Resolve/Wait. It
// may monotonically add an expected receipt to an existing crash record, but
// it never recreates a record already retired by durable cleanup. Execute uses
// persistPublishedWriteOperation because it must establish a record before
// admission and schedules its own cleanup if it discovers prior acceptance.
func (db *DB) enrichPersistedPublishedWriteOperation(operation PublishedWriteOperation) error {
	return db.persistPublishedWriteOperationMode(operation, true)
}

func (db *DB) persistPublishedWriteOperationMode(operation PublishedWriteOperation, existingOnly bool) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if err := operation.Validate(); err != nil {
		return err
	}
	db.operationRecoveryMu.Lock()
	defer db.operationRecoveryMu.Unlock()
	filename, err := operationRecoveryFilename(operation.Identity)
	if err != nil {
		return err
	}
	var journalDir string
	if existingOnly {
		_, journalDir, err = db.operationRecoveryJournalPaths()
	} else {
		journalDir, err = db.ensureOperationRecoveryJournalLocked()
	}
	if err != nil {
		return err
	}
	destination := filepath.Join(journalDir, filename)
	if existing, readErr := readPersistedPublishedWriteOperation(destination); readErr == nil {
		merged, changed, mergeErr := mergePublishedWriteOperation(existing.Operation, operation)
		if mergeErr != nil {
			return fmt.Errorf("merge operation recovery record: %w", mergeErr)
		}
		if !changed {
			return nil
		}
		operation = merged
	} else if errors.Is(readErr, os.ErrNotExist) && existingOnly {
		return nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("read existing operation recovery record: %w", readErr)
	}
	encoded, err := json.Marshal(persistedPublishedWriteOperation{
		Version: operationRecoveryJournalVersion, Operation: operation,
	})
	if err != nil {
		return fmt.Errorf("encode operation recovery: %w", err)
	}
	if beforeReplace := db.beforeOperationRecoveryReplaceForTest; beforeReplace != nil {
		if err := beforeReplace(operation); err != nil {
			return fmt.Errorf("replace operation recovery record: %w", err)
		}
	}

	temporary, err := os.CreateTemp(journalDir, ".operation-recovery-*.tmp")
	if err != nil {
		return fmt.Errorf("create operation recovery temporary file: %w", err)
	}
	temporaryName := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := temporary.Chmod(0o600); err != nil {
		return fmt.Errorf("protect operation recovery temporary file: %w", err)
	}
	if _, err := temporary.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write operation recovery temporary file: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync operation recovery temporary file: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close operation recovery temporary file: %w", err)
	}
	if err := os.Rename(temporaryName, destination); err != nil {
		return fmt.Errorf("replace operation recovery record: %w", err)
	}
	removeTemporary = false
	if err := syncDirectory(journalDir); err != nil {
		return fmt.Errorf("sync operation recovery journal: %w", err)
	}
	return nil
}

func (db *DB) operationRecoveryJournalPaths() (string, string, error) {
	if db == nil {
		return "", "", fmt.Errorf("db is nil")
	}
	db.mu.Lock()
	workingDir := db.workingDir
	databaseName := db.name
	db.mu.Unlock()
	if strings.TrimSpace(workingDir) == "" || strings.TrimSpace(databaseName) == "" {
		return "", "", fmt.Errorf("operation recovery store is not initialized")
	}
	return workingDir, filepath.Join(workingDir, swarmionStateDirName, operationRecoveryJournalDir, databaseName), nil
}

func (db *DB) ensureOperationRecoveryJournalLocked() (string, error) {
	anchor, journalDir, err := db.operationRecoveryJournalPaths()
	if err != nil {
		return "", err
	}
	if err := ensureCrashDurableDirectory(anchor, journalDir); err != nil {
		return "", fmt.Errorf("create operation recovery journal: %w", err)
	}
	return journalDir, nil
}

// ensureCrashDurableDirectory creates each missing directory one level at a
// time and syncs its parent immediately. This covers first creation of
// .swarmion as well as both journal subdirectories.
func ensureCrashDurableDirectory(anchor, target string) error {
	anchor, err := filepath.Abs(anchor)
	if err != nil {
		return err
	}
	target, err = filepath.Abs(target)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(anchor, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("journal directory escapes database work directory")
	}
	info, err := os.Stat(anchor)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("database work directory is not a directory")
	}
	current := anchor
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		if component == "" || component == "." {
			continue
		}
		next := filepath.Join(current, component)
		info, statErr := os.Stat(next)
		switch {
		case statErr == nil && !info.IsDir():
			return fmt.Errorf("journal path %q is not a directory", next)
		case statErr == nil:
			current = next
			continue
		case !errors.Is(statErr, os.ErrNotExist):
			return statErr
		}
		if err := os.Mkdir(next, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		if err := syncDirectory(current); err != nil {
			return err
		}
		current = next
	}
	return nil
}

func readPersistedPublishedWriteOperation(path string) (persistedPublishedWriteOperation, error) {
	file, err := os.Open(path)
	if err != nil {
		return persistedPublishedWriteOperation{}, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var record persistedPublishedWriteOperation
	if err := decoder.Decode(&record); err != nil {
		return persistedPublishedWriteOperation{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return persistedPublishedWriteOperation{}, fmt.Errorf("operation recovery record contains trailing JSON")
		}
		return persistedPublishedWriteOperation{}, err
	}
	if record.Version != operationRecoveryJournalVersion {
		return persistedPublishedWriteOperation{}, fmt.Errorf("unsupported operation recovery journal version %d", record.Version)
	}
	if err := record.Operation.Validate(); err != nil {
		return persistedPublishedWriteOperation{}, err
	}
	return record, nil
}

func operationRecoveryFilename(identity swarmion.OperationIdentity) (string, error) {
	identityJSON, err := identity.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("encode operation recovery identity: %w", err)
	}
	digest := sha256.Sum256(identityJSON)
	return hex.EncodeToString(digest[:]) + ".json", nil
}

func mergePublishedWriteOperation(existing, incoming PublishedWriteOperation) (PublishedWriteOperation, bool, error) {
	if err := existing.Validate(); err != nil {
		return PublishedWriteOperation{}, false, err
	}
	if err := incoming.Validate(); err != nil {
		return PublishedWriteOperation{}, false, err
	}
	if !operationIdentitiesEqual(existing.Identity, incoming.Identity) ||
		existing.Recovery.Namespace() != incoming.Recovery.Namespace() ||
		existing.Recovery.AuthorPeerID() != incoming.Recovery.AuthorPeerID() {
		return PublishedWriteOperation{}, false, fmt.Errorf("operation identity, namespace, or author changed")
	}
	existingBranch, existingIsBranch := existing.Recovery.BranchID()
	incomingBranch, incomingIsBranch := incoming.Recovery.BranchID()
	if existingIsBranch != incomingIsBranch || existingBranch != incomingBranch {
		return PublishedWriteOperation{}, false, fmt.Errorf("operation recovery line changed")
	}
	existingReceipt, existingHasReceipt := existing.Recovery.ExpectedReceipt()
	incomingReceipt, incomingHasReceipt := incoming.Recovery.ExpectedReceipt()
	switch {
	case existingHasReceipt && !incomingHasReceipt:
		return existing, false, nil
	case existingHasReceipt && incomingHasReceipt && existingReceipt != incomingReceipt:
		return PublishedWriteOperation{}, false, fmt.Errorf("operation expected receipt changed")
	case existingHasReceipt && incomingHasReceipt:
		return existing, false, nil
	case !existingHasReceipt && !incomingHasReceipt:
		return existing, false, nil
	default:
		return incoming, true, nil
	}
}

func (db *DB) removePublishedWriteOperation(operation PublishedWriteOperation) error {
	if err := operation.Validate(); err != nil {
		return err
	}
	db.operationRecoveryMu.Lock()
	defer db.operationRecoveryMu.Unlock()
	_, journalDir, err := db.operationRecoveryJournalPaths()
	if err != nil {
		return err
	}
	filename, err := operationRecoveryFilename(operation.Identity)
	if err != nil {
		return err
	}
	path := filepath.Join(journalDir, filename)
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if _, err := os.Stat(journalDir); err == nil {
		return syncDirectory(journalDir)
	}
	return nil
}

func (db *DB) loadPublishedWriteOperations() ([]PublishedWriteOperation, error) {
	db.operationRecoveryMu.Lock()
	defer db.operationRecoveryMu.Unlock()
	_, journalDir, err := db.operationRecoveryJournalPaths()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(journalDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	operations := make([]PublishedWriteOperation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		record, err := readPersistedPublishedWriteOperation(filepath.Join(journalDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read operation recovery %q: %w", entry.Name(), err)
		}
		expectedName, err := operationRecoveryFilename(record.Operation.Identity)
		if err != nil || expectedName != entry.Name() {
			return nil, fmt.Errorf("operation recovery %q does not match its exact identity", entry.Name())
		}
		operations = append(operations, record.Operation)
	}
	return operations, nil
}

// recoverPublishedWriteOperations runs before the DB becomes ready. It makes
// every process-local pre-execution record actionable after a crash:
// authoritative-absence records are retired, accepted records are correlated
// to their exact receipts and handed to lifecycle-owned asynchronous cleanup,
// and ambiguous or malformed records keep startup fail-closed for a later
// bootstrap retry. Accepted-but-pending publication never blocks database
// readiness; replay is already fenced by the exact accepted identity.
func (db *DB) recoverPublishedWriteOperations(ctx context.Context) error {
	operations, err := db.loadPublishedWriteOperations()
	if err != nil {
		return fmt.Errorf("load published write recovery journal: %w", err)
	}
	for _, operation := range operations {
		result, err := db.resolvePublishedWriteOperation(ctx, operation, false)
		if err != nil {
			return fmt.Errorf("resolve persisted published write %s: %w", operation.Identity, err)
		}
		switch result.Disposition() {
		case swarmion.OperationAccepted:
			receipt, err := publishedWriteReceiptFromAcceptedResult(operation, result)
			if err != nil {
				return fmt.Errorf("validate recovered accepted write %s: %w", operation.Identity, err)
			}
			db.schedulePublishedWriteOperationCleanup(operation, receipt)
		case swarmion.OperationRetryPermitted:
			reason, ok := result.RetryReason()
			if !ok || reason != swarmion.RetryResolvedAbsent {
				return fmt.Errorf(
					"persisted operation %s returned retry disposition without exact resolved-absence proof",
					operation.Identity,
				)
			}
			// The private runtime boundary proved this exact identity absent. The
			// interrupted caller is gone, so discard the local attempt instead of
			// replaying its unavailable SQL body.
			if err := db.removePublishedWriteOperation(operation); err != nil {
				return fmt.Errorf("retire recovered absent write %s: %w", operation.Identity, err)
			}
		case swarmion.OperationRecoveryRequired:
			return fmt.Errorf("%w: persisted operation %s remains ambiguous: %s", ErrOperationReceiptUnavailable, operation.Identity, operationDiagnosticText(result.Diagnostic()))
		case swarmion.OperationFailedClosed:
			return fmt.Errorf("persisted operation %s failed closed: %s", operation.Identity, operationDiagnosticText(result.Diagnostic()))
		default:
			return fmt.Errorf("persisted operation %s returned unknown disposition %q", operation.Identity, result.Disposition())
		}
	}
	return nil
}

// schedulePublishedWriteOperationCleanup retains the recovery record across
// the caller-return crash window, then retires it only after the exact event is
// durably applied. A timeout or close leaves the record for the next restart
// scan rather than dropping recovery evidence.
func (db *DB) schedulePublishedWriteOperationCleanup(operation PublishedWriteOperation, receipt PublishedWriteReceipt) {
	if db == nil || !receipt.HasExactEventIdentity() {
		return
	}
	if db.beforeOperationCleanupAdmissionForTest != nil {
		db.beforeOperationCleanupAdmissionForTest()
	}
	done, admitted := db.backgroundWork.begin()
	if !admitted {
		return
	}
	go func() {
		defer done()
		ctx, cancel := context.WithTimeout(context.Background(), committedWriteCheckpointTimeout)
		defer cancel()
		observation, err := db.WaitForPublishedWriteApplied(ctx, receipt, "retire operation recovery")
		if err != nil || !publishedWriteObservationProvesAppliedDurably(operation, receipt, observation) {
			return
		}
		if err := db.removePublishedWriteOperation(operation); err != nil {
			notifyLog.Warnf("failed to retire durable published write recovery %s: %v", operation.Identity, err)
		}
	}()
}

func publishedWriteObservationProvesAppliedDurably(
	operation PublishedWriteOperation,
	receipt PublishedWriteReceipt,
	observation EventReceiptObservation,
) bool {
	if operation.Validate() != nil || !receipt.HasExactEventIdentity() {
		return false
	}
	if strings.TrimSpace(receipt.AuthorPeerID) != strings.TrimSpace(operation.AuthorPeerID()) ||
		strings.TrimSpace(receipt.OperationIntentDigest) != strings.TrimSpace(operation.IntentDigest()) {
		return false
	}
	if observation.State != EventReceiptStateAppliedDurably ||
		!observation.Status.AppliedDurably ||
		!observation.Status.Checkpointed {
		return false
	}
	return strings.TrimSpace(observation.Receipt.EventID) == strings.TrimSpace(receipt.EventID) &&
		strings.TrimSpace(observation.Receipt.PublishedRootHash) == strings.TrimSpace(receipt.PublishedRootHash) &&
		strings.TrimSpace(observation.Status.EventID) == strings.TrimSpace(receipt.EventID) &&
		strings.TrimSpace(observation.Status.ExpectedPublishedRootHash) == strings.TrimSpace(receipt.PublishedRootHash)
}

func syncDirectory(path string) error {
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func replaceCrashDurableFile(path string, content []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".protos-operation-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	removeTemporary := true
	defer func() {
		_ = temporary.Close()
		if removeTemporary {
			_ = os.Remove(temporaryName)
		}
	}()
	if err := temporary.Chmod(mode); err != nil {
		return err
	}
	if _, err := temporary.Write(append(content, '\n')); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return err
	}
	removeTemporary = false
	return syncDirectory(directory)
}

func recoveryFromOperationResult(operation PublishedWriteOperation, result swarmion.OperationResult) (swarmion.OperationRecovery, error) {
	recovery, ok := result.Recovery()
	if !ok {
		return swarmion.OperationRecovery{}, fmt.Errorf("operation result has no exact recovery record")
	}
	if err := recovery.Validate(); err != nil {
		return swarmion.OperationRecovery{}, fmt.Errorf("operation result recovery: %w", err)
	}
	if !operationIdentitiesEqual(recovery.Identity(), operation.Identity) ||
		recovery.Namespace() != operation.Recovery.Namespace() ||
		recovery.AuthorPeerID() != operation.Recovery.AuthorPeerID() {
		return swarmion.OperationRecovery{}, fmt.Errorf("operation result recovery is not correlated to the persisted operation")
	}
	resultBranch, resultIsBranch := recovery.BranchID()
	persistedBranch, persistedIsBranch := operation.Recovery.BranchID()
	if resultIsBranch != persistedIsBranch || resultBranch != persistedBranch {
		return swarmion.OperationRecovery{}, fmt.Errorf("operation result recovery line is not correlated to the persisted operation")
	}
	if expected, hadExpected := operation.Recovery.ExpectedReceipt(); hadExpected {
		returned, hasReturned := recovery.ExpectedReceipt()
		if !hasReturned || returned != expected {
			return swarmion.OperationRecovery{}, fmt.Errorf("operation result regressed or contradicted the persisted expected receipt")
		}
	}
	return recovery, nil
}

func publishedWriteReceiptFromAcceptedResult(operation PublishedWriteOperation, result swarmion.OperationResult) (PublishedWriteReceipt, error) {
	if result.Disposition() != swarmion.OperationAccepted {
		return PublishedWriteReceipt{}, fmt.Errorf("operation disposition is %q, want accepted", result.Disposition())
	}
	if _, ok := result.Acceptance(); !ok {
		return PublishedWriteReceipt{}, fmt.Errorf("accepted operation result has no acceptance history")
	}
	receipt, ok := result.Receipt()
	if !ok {
		return PublishedWriteReceipt{}, fmt.Errorf("accepted operation result has no exact receipt")
	}
	recovery, err := recoveryFromOperationResult(operation, result)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	return PublishedWriteReceipt{
		Committed: true, EventID: receipt.EventID, PublishedRootHash: receipt.PublishedRootHash,
		AuthorPeerID: recovery.AuthorPeerID(), OperationIntentDigest: operation.Identity.IntentDigest(),
	}, nil
}

func publishedWriteReceiptFromRecoveryRequired(operation PublishedWriteOperation, result swarmion.OperationResult) (PublishedWriteReceipt, error) {
	if result.Disposition() != swarmion.OperationRecoveryRequired {
		return PublishedWriteReceipt{}, fmt.Errorf("operation disposition is %q, want recovery required", result.Disposition())
	}
	recovery, err := recoveryFromOperationResult(operation, result)
	if err != nil {
		return PublishedWriteReceipt{}, err
	}
	receipt, ok := recovery.ExpectedReceipt()
	if !ok {
		return PublishedWriteReceipt{}, nil
	}
	return PublishedWriteReceipt{
		OutcomeUncertain: true, EventID: receipt.EventID, PublishedRootHash: receipt.PublishedRootHash,
		AuthorPeerID: recovery.AuthorPeerID(), OperationIntentDigest: operation.Identity.IntentDigest(),
	}, nil
}

func (db *DB) resolvePublishedWriteOperation(ctx context.Context, operation PublishedWriteOperation, wait bool) (swarmion.OperationResult, error) {
	if db == nil {
		return swarmion.OperationResult{}, fmt.Errorf("db is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := operation.Validate(); err != nil {
		return swarmion.OperationResult{}, err
	}
	db.mu.Lock()
	runtime := db.runtime
	db.mu.Unlock()
	if runtime == nil {
		return swarmion.OperationResult{}, ErrOperationReceiptUnavailable
	}
	var result swarmion.OperationResult
	if wait {
		result = runtime.WaitOperation(ctx, operation.Recovery)
	} else {
		result = runtime.ResolveOperation(ctx, operation.Recovery)
	}
	recovery, err := recoveryFromOperationResult(operation, result)
	if err != nil {
		if diagnostic := result.Diagnostic(); diagnostic != nil {
			notifyLog.Warnf("passive operation resolution returned malformed recovery operation=%s diagnostic=%s", operation.Key(), diagnostic.Error())
		}
		return swarmion.OperationResult{}, err
	}
	if err := db.enrichOperationRecoveryFromResult(operation, result, recovery, "resolved"); err != nil {
		return result, err
	}
	return result, nil
}

func (db *DB) enrichOperationRecoveryFromResult(
	operation PublishedWriteOperation,
	result swarmion.OperationResult,
	recovery swarmion.OperationRecovery,
	source string,
) error {
	if err := db.enrichPersistedPublishedWriteOperation(PublishedWriteOperation{Identity: operation.Identity, Recovery: recovery}); err != nil {
		if result.Disposition() == swarmion.OperationAccepted {
			// The complete receipt-less recovery address was already synced before
			// execution. Exact accepted evidence remains authoritative even when
			// enriching that local optimization record fails; callers must not lose
			// the receipt and accidentally mint a second operation identity.
			db.transactionMetrics.operationRecoveryPersistenceFailures.Add(1)
			receipt, _ := result.Receipt()
			notifyLog.Warnf(
				"%s accepted write retained its pre-execute recovery record after enrichment failure operation=%s event_id=%s error=%s",
				source,
				operation.Key(),
				receipt.EventID,
				err.Error(),
			)
			return nil
		}
		return fmt.Errorf("persist %s operation recovery: %w", source, err)
	}
	return nil
}
