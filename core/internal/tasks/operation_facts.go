package tasks

import (
	"context"
	"crypto/sha256"
	stdsql "database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

const (
	// OperationFactKindEffect binds a stable task operation and its intended
	// application effect to the business mutation that publishes it.
	OperationFactKindEffect = "operation_effect"
	// OperationFactKindReceipt retains the immutable exact event/root receipt.
	// Mutable lifecycle observations belong in separate facts or projections.
	OperationFactKindReceipt = "operation_receipt"
)

// ErrOperationFactConflict means peers produced different immutable content
// for the same task/fact-kind identity. Recovery must fail closed rather than
// selecting one mutable writer's version.
var ErrOperationFactConflict = errors.New("task operation fact conflicts with immutable content")

// OperationFact is replicated, append-only recovery authority. ID is derived
// only from TaskID and Kind so a second value for the same logical fact cannot
// silently coexist. Payload is canonical JSON and must contain no observation
// timestamp, executor identity, attempt count, or mutable progress field.
type OperationFact struct {
	ID           string          `json:"id"`
	TaskID       string          `json:"task_id"`
	Kind         string          `json:"kind"`
	OperationKey string          `json:"operation_key"`
	IntentDigest string          `json:"intent_digest"`
	AuthorPeerID string          `json:"author_peer_id"`
	SubjectType  string          `json:"subject_type"`
	SubjectID    string          `json:"subject_id"`
	Payload      json.RawMessage `json:"payload"`
}

// NewOperationFact canonicalizes payload and derives the deterministic fact
// identity. The raw high-entropy operation key is application data here: it is
// required so another peer can resolve the author-scoped Swarmion receipt.
func NewOperationFact(
	taskID string,
	kind string,
	operation db.PublishedWriteOperation,
	subjectType string,
	subjectID string,
	payload any,
) (OperationFact, error) {
	taskID = strings.TrimSpace(taskID)
	kind = strings.TrimSpace(kind)
	operation.Key = strings.TrimSpace(operation.Key)
	operation.IntentDigest = strings.TrimSpace(operation.IntentDigest)
	operation.AuthorPeerID = strings.TrimSpace(operation.AuthorPeerID)
	subjectType = strings.TrimSpace(subjectType)
	subjectID = strings.TrimSpace(subjectID)
	if _, err := db.UUIDBytes(taskID); err != nil {
		return OperationFact{}, fmt.Errorf("task operation fact task ID is invalid: %w", err)
	}
	if kind == "" {
		return OperationFact{}, fmt.Errorf("task operation fact kind is empty")
	}
	if operation.Key == "" {
		return OperationFact{}, fmt.Errorf("task operation fact key is empty")
	}
	if digest, err := hex.DecodeString(operation.IntentDigest); err != nil || len(digest) != sha256.Size {
		return OperationFact{}, fmt.Errorf("task operation fact intent digest must be a 32-byte hexadecimal digest")
	}
	if operation.AuthorPeerID == "" {
		return OperationFact{}, fmt.Errorf("task operation fact author peer ID is empty")
	}
	if subjectType == "" || subjectID == "" {
		return OperationFact{}, fmt.Errorf("task operation fact subject is incomplete")
	}
	canonicalPayload, err := canonicalOperationFactPayload(payload)
	if err != nil {
		return OperationFact{}, err
	}
	return OperationFact{
		ID:           operationFactID(taskID, kind),
		TaskID:       taskID,
		Kind:         kind,
		OperationKey: operation.Key,
		IntentDigest: operation.IntentDigest,
		AuthorPeerID: operation.AuthorPeerID,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		Payload:      canonicalPayload,
	}, nil
}

func operationFactID(taskID, kind string) string {
	digest := sha256.Sum256([]byte("protos:task-operation-fact:v1\x00" + strings.TrimSpace(taskID) + "\x00" + strings.TrimSpace(kind)))
	return hex.EncodeToString(digest[:])
}

func canonicalOperationFactPayload(payload any) (json.RawMessage, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal task operation fact payload: %w", err)
	}
	decoded, err := decodeTaskPayloadLosslessly(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode task operation fact payload: %w", err)
	}
	canonical, err := json.Marshal(decoded)
	if err != nil {
		return nil, fmt.Errorf("canonicalize task operation fact payload: %w", err)
	}
	return canonical, nil
}

// InsertOperationFactMapper returns the deterministic insert used both in a
// business mutation transaction and for later receipt projection.
func InsertOperationFactMapper(fact OperationFact) db.InsertMapper {
	return func() sq.InsertQuery {
		table := sq.New[db.TASK_OPERATION_FACT]("")
		return sq.InsertInto(table).ColumnValues(func(col *sq.Column) {
			col.SetString(table.ID, fact.ID)
			col.SetBytes(table.TASK_ID, db.MustUUIDBytes(fact.TaskID))
			col.SetString(table.FACT_KIND, fact.Kind)
			col.SetString(table.OPERATION_KEY, fact.OperationKey)
			col.SetString(table.INTENT_DIGEST, fact.IntentDigest)
			col.SetString(table.AUTHOR_PEER_ID, fact.AuthorPeerID)
			col.SetString(table.SUBJECT_TYPE, fact.SubjectType)
			col.SetString(table.SUBJECT_ID, fact.SubjectID)
			col.SetJSON(table.PAYLOAD, fact.Payload)
		})
	}
}

// OperationFact reads one immutable fact by its stable task/kind identity.
func (m *Manager) OperationFact(ctx context.Context, taskID, kind string) (OperationFact, bool, error) {
	if m == nil || m.db == nil {
		return OperationFact{}, false, fmt.Errorf("task manager is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	id := operationFactID(taskID, kind)
	var fact OperationFact
	found := false
	err := m.db.ReadRows(ctx,
		"SELECT id, task_id, fact_kind, operation_key, intent_digest, author_peer_id, subject_type, subject_id, CAST(payload AS CHAR) FROM task_operation_facts WHERE id = ?",
		[]any{id},
		func(rows *stdsql.Rows) error {
			if !rows.Next() {
				return nil
			}
			var taskIDBytes []byte
			var payload string
			if err := rows.Scan(
				&fact.ID,
				&taskIDBytes,
				&fact.Kind,
				&fact.OperationKey,
				&fact.IntentDigest,
				&fact.AuthorPeerID,
				&fact.SubjectType,
				&fact.SubjectID,
				&payload,
			); err != nil {
				return err
			}
			parsedTaskID := db.UUIDString(taskIDBytes)
			if parsedTaskID == "" {
				return fmt.Errorf("decode task operation fact task ID: expected %d bytes, got %d", db.UUIDBinaryLength, len(taskIDBytes))
			}
			fact.TaskID = parsedTaskID
			canonical, err := canonicalOperationFactPayload(json.RawMessage(payload))
			if err != nil {
				return err
			}
			fact.Payload = canonical
			found = true
			if rows.Next() {
				return fmt.Errorf("task operation fact %s returned multiple rows", id)
			}
			return rows.Err()
		},
	)
	if err != nil {
		return OperationFact{}, false, fmt.Errorf("read task operation fact %s/%s: %w", taskID, kind, err)
	}
	if !found {
		return OperationFact{}, false, nil
	}
	if fact.ID != id || fact.TaskID != strings.TrimSpace(taskID) || fact.Kind != strings.TrimSpace(kind) {
		return OperationFact{}, false, fmt.Errorf("%w: stored identity does not match lookup", ErrOperationFactConflict)
	}
	return fact, true, nil
}

// OperationFactMatchesAtCheckpoint verifies that expected is present unchanged
// in one immutable durable checkpoint. Absence is a normal negative result;
// malformed or divergent content for the same deterministic identity fails
// closed as ErrOperationFactConflict.
func (m *Manager) OperationFactMatchesAtCheckpoint(
	ctx context.Context,
	checkpointCommitID string,
	expected OperationFact,
) (bool, error) {
	if m == nil || m.db == nil {
		return false, fmt.Errorf("task manager is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOperationFactIdentity(expected); err != nil {
		return false, fmt.Errorf("validate expected task operation fact: %w", err)
	}

	var stored OperationFact
	found := false
	err := m.db.ReadRowsAsOf(
		ctx,
		checkpointCommitID,
		"SELECT id, task_id, fact_kind, operation_key, intent_digest, author_peer_id, subject_type, subject_id, CAST(payload AS CHAR) FROM task_operation_facts AS OF ? WHERE id = ?",
		[]any{expected.ID},
		func(rows *stdsql.Rows) error {
			if !rows.Next() {
				return nil
			}
			var taskIDBytes []byte
			var payload string
			if err := rows.Scan(
				&stored.ID,
				&taskIDBytes,
				&stored.Kind,
				&stored.OperationKey,
				&stored.IntentDigest,
				&stored.AuthorPeerID,
				&stored.SubjectType,
				&stored.SubjectID,
				&payload,
			); err != nil {
				return err
			}
			stored.TaskID = db.UUIDString(taskIDBytes)
			if stored.TaskID == "" {
				return fmt.Errorf("decode task operation fact task ID: expected %d bytes, got %d", db.UUIDBinaryLength, len(taskIDBytes))
			}
			canonical, err := canonicalOperationFactPayload(json.RawMessage(payload))
			if err != nil {
				return err
			}
			stored.Payload = canonical
			found = true
			if rows.Next() {
				return fmt.Errorf("task operation fact %s returned multiple rows", expected.ID)
			}
			return rows.Err()
		},
	)
	if err != nil {
		return false, fmt.Errorf(
			"read task operation fact %s/%s at durable checkpoint %s: %w",
			expected.TaskID,
			expected.Kind,
			checkpointCommitID,
			err,
		)
	}
	if !found {
		return false, nil
	}
	if err := validateOperationFactIdentity(stored); err != nil {
		return false, fmt.Errorf(
			"%w: stored task operation fact %s has invalid identity: %w",
			ErrOperationFactConflict,
			expected.ID,
			err,
		)
	}
	if err := compareOperationFacts(stored, expected); err != nil {
		return false, err
	}
	return true, nil
}

// EnsureOperationFact publishes fact once. A concurrent identical insert is
// success; a different value for the same logical fact fails closed.
func (m *Manager) EnsureOperationFact(ctx context.Context, fact OperationFact) error {
	if m == nil || m.db == nil {
		return fmt.Errorf("task manager is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateOperationFactIdentity(fact); err != nil {
		return err
	}
	if existing, found, err := m.OperationFact(ctx, fact.TaskID, fact.Kind); err != nil {
		return err
	} else if found {
		return compareOperationFacts(existing, fact)
	}
	_, insertErr := db.InsertWithAvailabilityContext(ctx, m.db, InsertOperationFactMapper(fact))
	existing, found, readErr := m.OperationFact(context.WithoutCancel(ctx), fact.TaskID, fact.Kind)
	if readErr == nil && found {
		if compareErr := compareOperationFacts(existing, fact); compareErr != nil {
			if insertErr != nil {
				return errors.Join(insertErr, compareErr)
			}
			return compareErr
		}
		return nil
	}
	if insertErr != nil && readErr != nil {
		return errors.Join(insertErr, readErr)
	}
	if insertErr != nil {
		return insertErr
	}
	if readErr != nil {
		return readErr
	}
	return fmt.Errorf("task operation fact %s was not visible after publication", fact.ID)
}

// RecordOperationFact is the run-context form used by task streams.
func (ctx *RunContext[P]) RecordOperationFact(callCtx context.Context, fact OperationFact) error {
	if ctx == nil || ctx.manager == nil {
		return fmt.Errorf("task run context is not configured")
	}
	if fact.TaskID != ctx.record.ID {
		return fmt.Errorf("task operation fact task ID %q does not match running task %q", fact.TaskID, ctx.record.ID)
	}
	return ctx.manager.EnsureOperationFact(callCtx, fact)
}

func validateOperationFactIdentity(fact OperationFact) error {
	if fact.ID != operationFactID(fact.TaskID, fact.Kind) {
		return fmt.Errorf("task operation fact ID does not match task/kind identity")
	}
	canonical, err := canonicalOperationFactPayload(fact.Payload)
	if err != nil {
		return err
	}
	fact.Payload = canonical
	_, err = NewOperationFact(
		fact.TaskID,
		fact.Kind,
		db.PublishedWriteOperation{Key: fact.OperationKey, IntentDigest: fact.IntentDigest, AuthorPeerID: fact.AuthorPeerID},
		fact.SubjectType,
		fact.SubjectID,
		fact.Payload,
	)
	return err
}

func compareOperationFacts(existing, candidate OperationFact) error {
	existingPayload, existingErr := canonicalOperationFactPayload(existing.Payload)
	candidatePayload, candidateErr := canonicalOperationFactPayload(candidate.Payload)
	if existingErr != nil || candidateErr != nil {
		return errors.Join(existingErr, candidateErr)
	}
	fields := []struct {
		name      string
		existing  string
		candidate string
	}{
		{"id", existing.ID, candidate.ID},
		{"task_id", existing.TaskID, candidate.TaskID},
		{"kind", existing.Kind, candidate.Kind},
		{"operation_key", existing.OperationKey, candidate.OperationKey},
		{"intent_digest", existing.IntentDigest, candidate.IntentDigest},
		{"author_peer_id", existing.AuthorPeerID, candidate.AuthorPeerID},
		{"subject_type", existing.SubjectType, candidate.SubjectType},
		{"subject_id", existing.SubjectID, candidate.SubjectID},
		{"payload", string(existingPayload), string(candidatePayload)},
	}
	for _, field := range fields {
		if field.existing != field.candidate {
			return fmt.Errorf(
				"%w: task_id=%s kind=%s field=%s stored=%q candidate=%q",
				ErrOperationFactConflict,
				candidate.TaskID,
				candidate.Kind,
				field.name,
				field.existing,
				field.candidate,
			)
		}
	}
	return nil
}
