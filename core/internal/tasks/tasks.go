package tasks

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	swarmionapp "github.com/nustiueudinastea/swarmion/runtime"
	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/util"
)

var log = util.GetLogger("tasks")

// ErrCheckpointInvariantConflict means an exact task-state event was applied
// durably, but the task state at that durable checkpoint does not match the
// operation receipt the runner attempted to save.
var ErrCheckpointInvariantConflict = errors.New("task checkpoint invariant conflict")

// PermanentTaskError marks a stream failure that must not consume another
// task attempt. It is intended for application-level terminal outcomes (for
// example, a durable delete invariant conflict), not transient transport or
// provider failures.
type PermanentTaskError struct {
	Cause error
}

// DeferredTaskError means the runner must leave the owned task running exactly
// as it is. Receipt-sensitive recovery uses this for pending/parked exact
// receipts and foreign receipt unavailability; a later recovery tick observes
// the same stable identity without publishing retry/failure bookkeeping.
type DeferredTaskError struct {
	Cause error
}

func (err *DeferredTaskError) Error() string {
	if err == nil || err.Cause == nil {
		return "task recovery deferred"
	}
	return err.Cause.Error()
}

func (err *DeferredTaskError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// MarkDeferred preserves a running task without requeue or failure writes.
func MarkDeferred(err error) error {
	if err == nil {
		return nil
	}
	var deferred *DeferredTaskError
	if errors.As(err, &deferred) {
		return err
	}
	return &DeferredTaskError{Cause: err}
}

// IsDeferred reports whether a task must stay running for later observation.
func IsDeferred(err error) bool {
	var deferred *DeferredTaskError
	return errors.As(err, &deferred)
}

func (err *PermanentTaskError) Error() string {
	if err == nil || err.Cause == nil {
		return "permanent task failure"
	}
	return err.Cause.Error()
}

func (err *PermanentTaskError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// MarkPermanent prevents the task runner from requeueing err. Wrapping is
// idempotent so stream layers can preserve a permanent classification.
func MarkPermanent(err error) error {
	if err == nil {
		return nil
	}
	var permanent *PermanentTaskError
	if errors.As(err, &permanent) {
		return err
	}
	return &PermanentTaskError{Cause: err}
}

// IsPermanent reports whether a stream failure is terminal for task retries.
func IsPermanent(err error) bool {
	var permanent *PermanentTaskError
	return errors.As(err, &permanent)
}

const (
	taskCheckpointDurabilityTimeout      = 45 * time.Second
	taskCheckpointPublicationMaxAttempts = 20
	taskRunnerStopTimeout                = taskCheckpointDurabilityTimeout + 15*time.Second
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Record struct {
	ID           string
	Stream       string
	SubjectType  string
	SubjectID    string
	OwnerPeerID  string
	Status       Status
	Title        string
	Message      string
	Progress     int
	Payload      json.RawMessage
	Result       json.RawMessage
	ErrorMessage string
	Attempts     int
	MaxAttempts  int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	StartedAt    time.Time
	FinishedAt   time.Time
}

type Event struct {
	ID        string
	TaskID    string
	Status    Status
	Message   string
	Progress  int
	Details   json.RawMessage
	CreatedAt time.Time
}

type ProgressUpdate struct {
	Sequence  uint64
	TaskID    string
	Status    Status
	Message   string
	Progress  int
	Details   json.RawMessage
	CreatedAt time.Time
	// Durable means the task state was saved and its local root published. It
	// does not report Swarmion event applied_durably or content durability.
	Durable bool
}

type taskCheckpointTracker interface {
	WaitForPublishedWriteApplied(context.Context, db.PublishedWriteReceipt, string) (db.EventReceiptObservation, error)
	RecordAtCheckpoint(context.Context, string, string) (Record, bool, error)
}

type swarmionTaskCheckpointTracker struct {
	database *db.DB
}

func (tracker swarmionTaskCheckpointTracker) WaitForPublishedWriteApplied(
	ctx context.Context,
	receipt db.PublishedWriteReceipt,
	reason string,
) (db.EventReceiptObservation, error) {
	return tracker.database.WaitForPublishedWriteApplied(ctx, receipt, reason)
}

func (tracker swarmionTaskCheckpointTracker) RecordAtCheckpoint(
	ctx context.Context,
	checkpointCommitID string,
	taskID string,
) (Record, bool, error) {
	taskIDBytes, err := db.UUIDBytes(strings.TrimSpace(taskID))
	if err != nil {
		return Record{}, false, fmt.Errorf("invalid task id %q: %w", taskID, err)
	}
	var record Record
	found := false
	if err := tracker.database.ReadRowsAsOf(
		ctx,
		checkpointCommitID,
		"SELECT "+taskSelectColumns+" FROM tasks AS OF ? WHERE id = ?",
		[]any{taskIDBytes},
		func(rows *stdsql.Rows) error {
			if !rows.Next() {
				return nil
			}
			found = true
			var scanErr error
			record, scanErr = scanTaskRow(rows)
			return scanErr
		},
	); err != nil {
		return Record{}, false, fmt.Errorf("read task state at durable checkpoint: %w", err)
	}
	return record, found, nil
}

type EnqueueOptions[P any] struct {
	ID          string
	Stream      string
	SubjectType string
	SubjectID   string
	OwnerPeerID string
	Title       string
	Message     string
	Payload     P
	MaxAttempts int
}

type ListOptions struct {
	Stream      string
	SubjectType string
	SubjectID   string
	Status      Status
	MaxResults  int
}

type EnqueueUniqueOptions[P any] struct {
	EnqueueOptions[P]
	ReuseStatuses []Status
}

// StreamRecoveryDisposition tells startup recovery whether an interrupted
// running task can be returned to the pending queue. A deferred task is left
// byte-for-byte unchanged so a receipt-sensitive stream can observe replicated
// operation state before any local task-bookkeeping write.
type StreamRecoveryDisposition uint8

const (
	StreamRecoveryReady StreamRecoveryDisposition = iota
	StreamRecoveryDeferred
)

// RecoveryContext is a read-only startup-recovery view of an interrupted task.
// ReplacePayload only changes the in-memory record that will be published if
// recovery returns StreamRecoveryReady; it never writes by itself.
type RecoveryContext[P any] struct {
	record         Record
	payload        P
	payloadChanged bool
}

func (ctx *RecoveryContext[P]) Task() Record {
	return ctx.record
}

func (ctx *RecoveryContext[P]) Payload() P {
	return ctx.payload
}

func (ctx *RecoveryContext[P]) ReplacePayload(payload P) {
	ctx.payload = payload
	ctx.payloadChanged = true
}

// PayloadCheckpointRecord reconstructs the exact task row that a payload
// checkpoint would publish. Recovery hooks use it to validate the application
// invariant at the event checkpoint before publishing any recovery handoff.
func (ctx *RecoveryContext[P]) PayloadCheckpointRecord(
	payload P,
	progress int,
	message string,
) (Record, error) {
	if ctx == nil {
		return Record{}, fmt.Errorf("task recovery context is not configured")
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return Record{}, fmt.Errorf("marshal task recovery checkpoint payload: %w", err)
	}
	record := ctx.record
	record.Payload = payloadJSON
	record.Status = StatusRunning
	record.Progress = normalizeProgress(progress)
	record.Message = strings.TrimSpace(message)
	if record.Message == "" {
		record.Message = string(record.Status)
	}
	return record, nil
}

// PayloadCheckpointOperation reconstructs the exact stable operation identity
// that CheckpointPayloadWithOperationAuthor would use for the supplied logical
// checkpoint. Recovery hooks call it before changing a running task so an
// accepted pending/parked checkpoint is observed rather than overwritten.
func (ctx *RecoveryContext[P]) PayloadCheckpointOperation(
	operationKey string,
	authorPeerID string,
	payload P,
	progress int,
	message string,
	details any,
) (db.PublishedWriteOperation, error) {
	if ctx == nil {
		return db.PublishedWriteOperation{}, fmt.Errorf("task recovery context is not configured")
	}
	record, err := ctx.PayloadCheckpointRecord(payload, progress, message)
	if err != nil {
		return db.PublishedWriteOperation{}, err
	}
	eventDetails, err := marshalOptional(details)
	if err != nil {
		return db.PublishedWriteOperation{}, err
	}
	return taskCheckpointPublishedWriteOperationForAuthor(
		record,
		taskEvent(record, eventDetails),
		operationKey,
		authorPeerID,
	)
}

type Stream[P any, R any] struct {
	Name string
	// Recover runs before startup recovery changes a running task back to
	// pending. It must not perform task-state writes. Receipt-sensitive streams
	// use it to resolve and inspect the exact accepted operation first, deferring
	// recovery while that operation is pending or unavailable.
	Recover func(context.Context, *RecoveryContext[P]) (StreamRecoveryDisposition, error)
	Run     func(context.Context, *RunContext[P]) (R, error)
}

type RunContext[P any] struct {
	manager *Manager
	record  Record
	payload P
}

func (ctx *RunContext[P]) Task() Record {
	return ctx.record
}

func (ctx *RunContext[P]) Payload() P {
	return ctx.payload
}

// CheckpointPayload replaces the persisted task payload while recording a
// saved running-state update. The in-memory run context adopts the payload
// before publication so an error path can requeue the exact known operation
// receipt instead of reverting to an older payload. Unlike ordinary task
// updates, this persistence-sensitive boundary waits for its exact Swarmion
// event to reach applied_durably before returning. Historical full-root
// content coverage is diagnostic and is not required.
func (ctx *RunContext[P]) CheckpointPayload(payload P, progress int, message string, details any) error {
	return ctx.persistPayload(payload, &progress, &message, details, true, "", "")
}

// CheckpointPayloadWithOperationKey uses a replicated, high-entropy domain
// operation key as the basis for restart-safe task-checkpoint correlation.
func (ctx *RunContext[P]) CheckpointPayloadWithOperationKey(operationKey string, payload P, progress int, message string, details any) error {
	operationKey = strings.TrimSpace(operationKey)
	if operationKey == "" {
		return fmt.Errorf("task checkpoint operation key is empty")
	}
	return ctx.persistPayload(payload, &progress, &message, details, true, operationKey, "")
}

// CheckpointPayloadWithOperationAuthor retains the original operation author
// when a different peer resumes a persistence-sensitive checkpoint. A foreign
// miss remains unavailable; the resuming peer must never author a duplicate
// checkpoint event under its own identity.
func (ctx *RunContext[P]) CheckpointPayloadWithOperationAuthor(
	operationKey string,
	authorPeerID string,
	payload P,
	progress int,
	message string,
	details any,
) error {
	operationKey = strings.TrimSpace(operationKey)
	if operationKey == "" {
		return fmt.Errorf("task checkpoint operation key is empty")
	}
	authorPeerID = strings.TrimSpace(authorPeerID)
	if authorPeerID == "" {
		return fmt.Errorf("task checkpoint operation author peer ID is empty")
	}
	return ctx.persistPayload(payload, &progress, &message, details, true, operationKey, authorPeerID)
}

// ReplacePayload replaces the persisted task payload without changing the
// task's current status, progress, or message.
func (ctx *RunContext[P]) ReplacePayload(payload P) error {
	return ctx.persistPayload(payload, nil, nil, nil, false, "", "")
}

func (ctx *RunContext[P]) persistPayload(
	payload P,
	progress *int,
	message *string,
	details any,
	waitAppliedDurably bool,
	operationKey string,
	operationAuthorPeerID string,
) error {
	if ctx == nil || ctx.manager == nil {
		return fmt.Errorf("task run context is not configured")
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal task payload: %w", err)
	}
	// The run context owns the exact record published by markRunning and every
	// subsequent task update in this run. Do not re-read here: ordinary Swarmion
	// writes return after local-root publication, so a live read can temporarily
	// observe an older durable/queryable root and roll back attempts or a receipt.
	record := ctx.record
	record.Payload = payloadJSON
	if progress != nil {
		record.Status = StatusRunning
		record.Progress = normalizeProgress(*progress)
		record.Message = strings.TrimSpace(*message)
		if record.Message == "" {
			record.Message = string(record.Status)
		}
	}
	record.UpdatedAt = time.Now().UTC()
	eventDetails, err := marshalOptional(details)
	if err != nil {
		return err
	}
	// Adopt the known logical state before publication. If this first save
	// fails, streamAdapter's retry/fail path must still carry the receipt in its
	// owned record. A process loss before any save succeeds remains an
	// unavoidable pre-persistence window, but an in-process retry must never
	// discard an already returned operation identity.
	ctx.record = record
	ctx.payload = payload
	event := taskEvent(record, eventDetails)
	if waitAppliedDurably {
		if err := ctx.manager.saveTaskCheckpoint(record, event, operationKey, operationAuthorPeerID); err != nil {
			return err
		}
		return nil
	}
	if err := ctx.manager.saveTaskUpdate(record, event); err != nil {
		return err
	}
	return nil
}

func (ctx *RunContext[P]) Update(progress int, message string, details any) error {
	if ctx == nil || ctx.manager == nil {
		return fmt.Errorf("task run context is not configured")
	}
	record := ctx.record
	record.Status = StatusRunning
	record.Progress = normalizeProgress(progress)
	record.Message = strings.TrimSpace(message)
	if record.Message == "" {
		record.Message = string(record.Status)
	}
	record.UpdatedAt = time.Now().UTC()
	eventDetails, err := marshalOptional(details)
	if err != nil {
		return err
	}
	if err := ctx.manager.saveTaskUpdate(record, taskEvent(record, eventDetails)); err != nil {
		return err
	}
	ctx.record = record
	return nil
}

func (ctx *RunContext[P]) Progress(progress int, message string, details any) error {
	return ctx.manager.Progress(ctx.record.ID, StatusRunning, progress, message, details)
}

type registeredStream interface {
	recover(context.Context, Record) (Record, StreamRecoveryDisposition, error)
	run(context.Context, *Manager, Record) error
}

type streamAdapter[P any, R any] struct {
	stream Stream[P, R]
}

func (s streamAdapter[P, R]) recover(ctx context.Context, record Record) (Record, StreamRecoveryDisposition, error) {
	if s.stream.Recover == nil {
		return record, StreamRecoveryReady, nil
	}
	var payload P
	if len(record.Payload) > 0 {
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			return record, StreamRecoveryReady, fmt.Errorf("decode task payload for recovery: %w", err)
		}
	}
	recoveryCtx := &RecoveryContext[P]{record: record, payload: payload}
	disposition, err := s.stream.Recover(ctx, recoveryCtx)
	if err != nil {
		return record, disposition, err
	}
	switch disposition {
	case StreamRecoveryReady, StreamRecoveryDeferred:
	default:
		return record, disposition, fmt.Errorf("task stream %q returned unknown recovery disposition %d", s.stream.Name, disposition)
	}
	if disposition == StreamRecoveryDeferred || !recoveryCtx.payloadChanged {
		return record, disposition, nil
	}
	payloadJSON, err := json.Marshal(recoveryCtx.payload)
	if err != nil {
		return record, disposition, fmt.Errorf("marshal recovered task payload: %w", err)
	}
	record.Payload = payloadJSON
	return record, disposition, nil
}

func (s streamAdapter[P, R]) run(ctx context.Context, manager *Manager, record Record) error {
	if s.stream.Run == nil {
		err := fmt.Errorf("task stream %q has no run function", s.stream.Name)
		_ = manager.failRecord(record, err, nil)
		return err
	}

	var payload P
	if len(record.Payload) > 0 {
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			err = fmt.Errorf("decode task payload: %w", err)
			_ = manager.failRecord(record, err, nil)
			return err
		}
	}

	runCtx := &RunContext[P]{
		manager: manager,
		record:  record,
		payload: payload,
	}
	result, err := s.stream.Run(ctx, runCtx)
	if err != nil {
		// Runner cancellation models an interrupted process/task loop, not a
		// business failure. Preserve the exact owned running record (including a
		// checkpointed operation receipt) so startup recovery can return it to the
		// pending queue without losing identity or consuming an attempt.
		if ctx.Err() != nil {
			return err
		}
		// Exact-receipt pending/parked work and foreign receipt unavailability
		// stay running. Publishing pending/failed bookkeeping here could conflict
		// with the foreign pending root or derive a different checkpoint operation.
		if IsDeferred(err) {
			return err
		}
		// Bounded auto-retry: if the stream opted into multiple attempts and the
		// run was not cancelled, requeue the task so the next RunPending tick
		// resumes it. This makes idempotent lifecycle work (e.g. instance delete)
		// self-resumable instead of terminally failing on a transient
		// control-plane error. Streams that keep MaxAttempts=1 fail terminally,
		// exactly as before.
		if !IsPermanent(err) && runCtx.record.Attempts < runCtx.record.MaxAttempts {
			if requeueErr := manager.requeueForRetry(runCtx.record, err); requeueErr == nil {
				return err
			} else {
				log.Errorf("failed to requeue task %s for retry: %s", record.ID, requeueErr.Error())
			}
		}
		_ = manager.failRecord(runCtx.record, err, nil)
		return err
	}
	if err := manager.succeedRecord(runCtx.record, result); err != nil {
		return err
	}
	return nil
}

type Manager struct {
	db             *db.DB
	mu             sync.RWMutex
	streams        map[string]registeredStream
	executorPeerID string
	// runnerMu serializes recovery and task execution for this Manager. Startup
	// recovery runs only at the quiescent boundary immediately before a
	// RunPending scan, so it cannot mistake this runner's own live task for an
	// interrupted one. It also makes multiple concurrent Start calls safe.
	runnerMu          sync.Mutex
	progressMu        sync.Mutex
	progressSeq       uint64
	nextWatcherID     uint64
	progressWatchers  map[string]map[uint64]chan ProgressUpdate
	latestProgress    map[string]ProgressUpdate
	checkpointTracker taskCheckpointTracker
	// publishTaskCheckpoint is an internal seam for exercising the ambiguous
	// post-publication error boundary without republishing a logical write.
	publishTaskCheckpoint func(context.Context, Record, Event) (db.PublishedWriteReceipt, error)
	// waitTaskCheckpointPublicationRetry is an internal seam for keeping
	// bounded publication-retry tests deterministic and fast.
	waitTaskCheckpointPublicationRetry func(context.Context, int) error
	// beforeSaveTaskUpdate is an internal fault-injection seam used to prove
	// receipt ownership across task-state publication failures.
	beforeSaveTaskUpdate func(Record, Event) error
}

func NewManager(database *db.DB) *Manager {
	return &Manager{
		db:                database,
		streams:           map[string]registeredStream{},
		executorPeerID:    "local",
		progressWatchers:  map[string]map[uint64]chan ProgressUpdate{},
		latestProgress:    map[string]ProgressUpdate{},
		checkpointTracker: swarmionTaskCheckpointTracker{database: database},
	}
}

func (m *Manager) SetExecutorPeerID(peerID string) {
	if m == nil {
		return
	}
	peerID = strings.TrimSpace(peerID)
	if peerID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executorPeerID = peerID
}

func (m *Manager) ExecutorPeerID() string {
	if m == nil {
		return ""
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return strings.TrimSpace(m.executorPeerID)
}

// VerifyCheckpointRecord reads the task row from an exact Swarmion checkpoint
// and verifies the same logical fields used by task checkpoint operation
// identity. Read failures remain transient; absence or a semantic mismatch is
// an application-level checkpoint invariant conflict.
func (m *Manager) VerifyCheckpointRecord(ctx context.Context, checkpointCommitID string, expected Record) error {
	if m == nil || m.checkpointTracker == nil {
		return fmt.Errorf("task checkpoint tracker is not configured")
	}
	checkpointCommitID = strings.TrimSpace(checkpointCommitID)
	if checkpointCommitID == "" {
		return fmt.Errorf("task checkpoint commit ID is empty")
	}
	recordAtCheckpoint, found, err := m.checkpointTracker.RecordAtCheckpoint(
		ctx,
		checkpointCommitID,
		expected.ID,
	)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("%w: task_id=%s checkpoint=%s outcome=task is absent", ErrCheckpointInvariantConflict, expected.ID, checkpointCommitID)
	}
	matches, err := taskCheckpointRecordsSemanticallyEqual(recordAtCheckpoint, expected)
	if err != nil {
		return fmt.Errorf("compare task checkpoint state: %w", err)
	}
	if !matches {
		return fmt.Errorf("%w: task_id=%s checkpoint=%s outcome=task state differs", ErrCheckpointInvariantConflict, expected.ID, checkpointCommitID)
	}
	return nil
}

func Register[P any, R any](manager *Manager, stream Stream[P, R]) error {
	return register(manager, stream, false)
}

func RegisterIfAbsent[P any, R any](manager *Manager, stream Stream[P, R]) error {
	return register(manager, stream, true)
}

func register[P any, R any](manager *Manager, stream Stream[P, R], allowExisting bool) error {
	if manager == nil {
		return fmt.Errorf("task manager is nil")
	}
	stream.Name = strings.TrimSpace(stream.Name)
	if stream.Name == "" {
		return fmt.Errorf("task stream name is empty")
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if _, found := manager.streams[stream.Name]; found {
		if allowExisting {
			return nil
		}
		return fmt.Errorf("task stream %q is already registered", stream.Name)
	}
	manager.streams[stream.Name] = streamAdapter[P, R]{stream: stream}
	return nil
}

func Enqueue[P any](manager *Manager, opts EnqueueOptions[P]) (Record, error) {
	if manager == nil {
		return Record{}, fmt.Errorf("task manager is nil")
	}
	opts.Stream = strings.TrimSpace(opts.Stream)
	opts.SubjectType = strings.TrimSpace(opts.SubjectType)
	opts.SubjectID = strings.TrimSpace(opts.SubjectID)
	opts.OwnerPeerID = strings.TrimSpace(opts.OwnerPeerID)
	if opts.OwnerPeerID == "" {
		opts.OwnerPeerID = manager.ExecutorPeerID()
	}
	opts.Title = strings.TrimSpace(opts.Title)
	if opts.Stream == "" {
		return Record{}, fmt.Errorf("task stream is empty")
	}
	if opts.SubjectType == "" {
		return Record{}, fmt.Errorf("task subject type is empty")
	}
	if opts.SubjectID == "" {
		return Record{}, fmt.Errorf("task subject id is empty")
	}
	if opts.OwnerPeerID == "" {
		return Record{}, fmt.Errorf("task owner peer id is empty")
	}
	if opts.Title == "" {
		opts.Title = opts.Stream
	}
	if opts.ID == "" {
		id, err := db.NewUUIDv7()
		if err != nil {
			return Record{}, err
		}
		opts.ID = id
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}

	payload, err := json.Marshal(opts.Payload)
	if err != nil {
		return Record{}, fmt.Errorf("marshal task payload: %w", err)
	}
	now := time.Now().UTC()
	record := Record{
		ID:          opts.ID,
		Stream:      opts.Stream,
		SubjectType: opts.SubjectType,
		SubjectID:   opts.SubjectID,
		OwnerPeerID: opts.OwnerPeerID,
		Status:      StatusPending,
		Title:       opts.Title,
		Message:     strings.TrimSpace(opts.Message),
		Progress:    0,
		Payload:     payload,
		Result:      json.RawMessage("{}"),
		MaxAttempts: opts.MaxAttempts,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if record.Message == "" {
		record.Message = "queued"
	}
	event := taskEvent(record, nil)
	if err := db.InsertPublished(manager.db, createTaskInsertMapper(record), createTaskEventInsertMapper(event)); err != nil {
		return Record{}, err
	}
	manager.publishProgress(recordProgressUpdate(record, event.Details, true))
	return record, nil
}

func EnqueueUnique[P any](manager *Manager, opts EnqueueUniqueOptions[P]) (Record, bool, error) {
	if manager == nil {
		return Record{}, false, fmt.Errorf("task manager is nil")
	}
	opts.OwnerPeerID = strings.TrimSpace(opts.OwnerPeerID)
	if opts.OwnerPeerID == "" {
		opts.OwnerPeerID = manager.ExecutorPeerID()
	}
	reuseStatuses := opts.ReuseStatuses
	if len(reuseStatuses) == 0 {
		reuseStatuses = []Status{StatusPending, StatusRunning}
	}
	for _, status := range reuseStatuses {
		if !validStatus(status) {
			continue
		}
		records, err := selectTaskRecords(manager.db, taskQueryFilters{
			Stream:      opts.Stream,
			SubjectType: opts.SubjectType,
			SubjectID:   opts.SubjectID,
			OwnerPeerID: opts.OwnerPeerID,
			Status:      status,
		}, true)
		if err != nil {
			return Record{}, false, err
		}
		latest, found, err := latestRecord(records)
		if err != nil {
			return Record{}, false, err
		}
		if found {
			return latest, false, nil
		}
	}
	record, err := Enqueue(manager, opts.EnqueueOptions)
	return record, err == nil, err
}

func (m *Manager) Start(ctx context.Context, interval time.Duration) func() error {
	if m == nil {
		return func() error { return nil }
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	runCtx, cancel := context.WithCancel(ctx)
	runnerDone := make(chan struct{})
	go func() {
		defer close(runnerDone)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			m.runRunnerTick(runCtx)
			select {
			case <-ticker.C:
			case <-runCtx.Done():
				return
			}
		}
	}()
	var (
		stopOnce sync.Once
		stopErr  error
	)
	return func() error {
		stopOnce.Do(func() {
			cancel()
			timer := time.NewTimer(taskRunnerStopTimeout)
			defer timer.Stop()
			select {
			case <-runnerDone:
			case <-timer.C:
				stopErr = fmt.Errorf("task runner did not stop within %s", taskRunnerStopTimeout)
			}
		})
		return stopErr
	}
}

func (m *Manager) runRunnerTick(ctx context.Context) {
	m.runnerMu.Lock()
	defer m.runnerMu.Unlock()

	recoveryReady, err := m.recoverOwnedRunningAtStartup(ctx)
	if err != nil {
		// A malformed or temporarily unobservable interrupted task remains
		// running, but it must not starve unrelated pending work on this tick.
		log.Errorf("task runner recovery scan failed: %s", err.Error())
	}
	if !recoveryReady {
		return
	}
	if err := m.runPending(ctx); err != nil {
		log.Errorf("task runner failed: %s", err.Error())
	}
}

func (m *Manager) recoverOwnedRunningAtStartup(ctx context.Context) (bool, error) {
	if m == nil || m.db == nil || !m.db.Initialized() {
		return false, nil
	}
	recovered, _, err := m.recoverOwnedRunning(ctx)
	if err != nil {
		return true, err
	}
	if recovered > 0 {
		log.Infof("recovered %d interrupted task(s) for executor %s", recovered, m.ExecutorPeerID())
	}
	return true, nil
}

func (m *Manager) RunPending(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.runnerMu.Lock()
	defer m.runnerMu.Unlock()
	return m.runPending(ctx)
}

func (m *Manager) runPending(ctx context.Context) error {
	if m == nil || m.db == nil || !m.db.Initialized() {
		return nil
	}
	ownerPeerID := m.ExecutorPeerID()
	if ownerPeerID == "" {
		return fmt.Errorf("task executor peer id is empty")
	}
	records, err := selectTaskRecords(m.db, taskQueryFilters{Status: StatusPending, OwnerPeerID: ownerPeerID}, true)
	if err != nil {
		return err
	}
	var failures []string
	for _, record := range records {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		stream, found := m.stream(record.Stream)
		if !found {
			failures = append(failures, fmt.Sprintf("no task stream registered for %q", record.Stream))
			continue
		}
		marked, err := m.markRunningRecord(record)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", record.ID, err))
			continue
		}
		if err := stream.run(ctx, m, marked); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", record.ID, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

// RecoverOwnedRunning returns tasks interrupted while running to the current
// executor's pending queue. The interrupted attempt is rolled back so recovery
// resumes the same logical attempt, including when MaxAttempts is one. Payload
// and result data are preserved exactly as last published by the task. A
// stream recovery hook may defer a receipt-sensitive task; deferred tasks are
// not included in the returned count and remain unchanged.
func (m *Manager) RecoverOwnedRunning() (int, error) {
	if m == nil {
		return 0, fmt.Errorf("task manager is not configured")
	}
	m.runnerMu.Lock()
	defer m.runnerMu.Unlock()
	recovered, _, err := m.recoverOwnedRunning(context.Background())
	return recovered, err
}

func (m *Manager) recoverOwnedRunning(ctx context.Context) (int, int, error) {
	if m == nil || m.db == nil {
		return 0, 0, fmt.Errorf("task manager is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ownerPeerID := m.ExecutorPeerID()
	if ownerPeerID == "" {
		return 0, 0, fmt.Errorf("task executor peer id is empty")
	}
	records, err := selectTaskRecords(m.db, taskQueryFilters{
		Status:      StatusRunning,
		OwnerPeerID: ownerPeerID,
	}, true)
	if err != nil {
		return 0, 0, err
	}

	recovered, deferred := 0, 0
	var failures []error
	for _, record := range records {
		if err := ctx.Err(); err != nil {
			return recovered, deferred, err
		}
		if stream, found := m.stream(record.Stream); found {
			prepared, disposition, recoverErr := stream.recover(ctx, record)
			if recoverErr != nil {
				failures = append(failures, fmt.Errorf("%s: %w", record.ID, recoverErr))
				continue
			}
			if disposition == StreamRecoveryDeferred {
				deferred++
				continue
			}
			record = prepared
		}
		expected := record
		record.Status = StatusPending
		record.Message = "recovering interrupted task"
		if record.Attempts > 0 {
			record.Attempts--
		}
		record.UpdatedAt = time.Now().UTC()
		updated, err := m.saveRecoveredTaskUpdate(expected, record, taskEvent(record, nil))
		if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", record.ID, err))
			continue
		}
		if !updated {
			log.Infof("skipped recovery for task %s because its running state changed during recovery", record.ID)
			continue
		}
		recovered++
	}
	if len(failures) > 0 {
		return recovered, deferred, fmt.Errorf("recover running tasks: %w", errors.Join(failures...))
	}
	return recovered, deferred, nil
}

func (m *Manager) Get(id string) (Record, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Record{}, fmt.Errorf("task id is empty")
	}
	records, err := selectTaskRecords(m.db, taskQueryFilters{IDs: []string{id}}, true)
	if err != nil {
		return Record{}, err
	}
	if len(records) == 0 {
		return Record{}, fmt.Errorf("task %q not found", id)
	}
	return records[0], nil
}

func (m *Manager) List(opts ListOptions) ([]Record, bool, error) {
	if m == nil {
		return nil, false, fmt.Errorf("task manager is nil")
	}
	status := Status(strings.TrimSpace(string(opts.Status)))
	if opts.Status != "" {
		if !validStatus(status) {
			return []Record{}, false, nil
		}
	}
	records, err := selectTaskRecords(m.db, taskQueryFilters{
		Stream:      opts.Stream,
		SubjectType: opts.SubjectType,
		SubjectID:   opts.SubjectID,
		Status:      status,
	}, false)
	if err != nil {
		return nil, false, err
	}
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].UpdatedAt.After(records[j].UpdatedAt)
	})
	if opts.MaxResults > 0 && len(records) > opts.MaxResults {
		return records[:opts.MaxResults], true, nil
	}
	return records, false, nil
}

func (m *Manager) Events(taskID string) ([]Event, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("task id is empty")
	}
	events, err := selectTaskEvents(m.db, taskID)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})
	return events, nil
}

func (m *Manager) Subscribe(taskID string) (<-chan ProgressUpdate, func(), error) {
	if m == nil {
		return nil, nil, fmt.Errorf("task manager is nil")
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, nil, fmt.Errorf("task id is empty")
	}
	ch := make(chan ProgressUpdate, 32)
	m.progressMu.Lock()
	m.nextWatcherID++
	watcherID := m.nextWatcherID
	if m.progressWatchers[taskID] == nil {
		m.progressWatchers[taskID] = map[uint64]chan ProgressUpdate{}
	}
	m.progressWatchers[taskID][watcherID] = ch
	m.progressMu.Unlock()

	cancel := func() {
		m.progressMu.Lock()
		defer m.progressMu.Unlock()
		watchers := m.progressWatchers[taskID]
		if watchers == nil {
			return
		}
		if existing, found := watchers[watcherID]; found {
			delete(watchers, watcherID)
			close(existing)
		}
		if len(watchers) == 0 {
			delete(m.progressWatchers, taskID)
		}
	}
	return ch, cancel, nil
}

func (m *Manager) LatestProgress(taskID string) (ProgressUpdate, bool) {
	if m == nil {
		return ProgressUpdate{}, false
	}
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return ProgressUpdate{}, false
	}
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	update, found := m.latestProgress[taskID]
	if !found {
		return ProgressUpdate{}, false
	}
	update.Details = append(json.RawMessage(nil), update.Details...)
	return update, true
}

func (m *Manager) LatestForSubject(stream string, subjectType string, subjectID string) (Record, bool, error) {
	records, err := selectTaskRecords(m.db, taskQueryFilters{
		Stream:      stream,
		SubjectType: subjectType,
		SubjectID:   subjectID,
	}, false)
	if err != nil {
		return Record{}, false, err
	}
	if len(records) == 0 {
		return Record{}, false, nil
	}
	return latestRecord(records)
}

func (m *Manager) Update(id string, status Status, progress int, message string, details any) error {
	record, err := m.Get(id)
	if err != nil {
		return err
	}
	record.Status = normalizeStatus(status)
	record.Progress = normalizeProgress(progress)
	record.Message = strings.TrimSpace(message)
	if record.Message == "" {
		record.Message = string(record.Status)
	}
	record.UpdatedAt = time.Now().UTC()
	eventDetails, err := marshalOptional(details)
	if err != nil {
		return err
	}
	return m.saveTaskUpdate(record, taskEvent(record, eventDetails))
}

func (m *Manager) Progress(id string, status Status, progress int, message string, details any) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("task id is empty")
	}
	eventDetails, err := marshalOptional(details)
	if err != nil {
		return err
	}
	status = normalizeStatus(status)
	message = strings.TrimSpace(message)
	if message == "" {
		message = string(status)
	}
	m.publishProgress(ProgressUpdate{
		TaskID:    id,
		Status:    status,
		Message:   message,
		Progress:  normalizeProgress(progress),
		Details:   eventDetails,
		CreatedAt: time.Now().UTC(),
		Durable:   false,
	})
	return nil
}

func (m *Manager) Succeed(id string, result any) error {
	record, err := m.Get(id)
	if err != nil {
		return err
	}
	return m.succeedRecord(record, result)
}

func (m *Manager) succeedRecord(record Record, result any) error {
	resultJSON, err := marshalOptional(result)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	record.Status = StatusSucceeded
	record.Progress = 100
	record.Message = "completed"
	record.Result = resultJSON
	record.ErrorMessage = ""
	record.UpdatedAt = now
	record.FinishedAt = now
	return m.saveTaskUpdate(record, taskEvent(record, resultJSON))
}

func (m *Manager) Fail(id string, cause error, details any) error {
	record, err := m.Get(id)
	if err != nil {
		return err
	}
	return m.failRecord(record, cause, details)
}

func (m *Manager) failRecord(record Record, cause error, details any) error {
	if cause == nil {
		cause = fmt.Errorf("task failed")
	}
	eventDetails, err := marshalOptional(details)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	record.Status = StatusFailed
	record.Message = "failed"
	record.ErrorMessage = cause.Error()
	record.UpdatedAt = now
	record.FinishedAt = now
	return m.saveTaskUpdate(record, taskEvent(record, eventDetails))
}

// requeueForRetry returns a task to the pending queue after a retryable failure
// so the next RunPending tick resumes it. Attempts is not incremented here; it
// is incremented by markRunning on the next run, which also enforces the
// MaxAttempts ceiling. This is the engine primitive behind self-resumable
// idempotent lifecycle work.
func (m *Manager) requeueForRetry(record Record, cause error) error {
	now := time.Now().UTC()
	record.Status = StatusPending
	record.Message = fmt.Sprintf("retrying after error (attempt %d/%d)", record.Attempts, record.MaxAttempts)
	if cause != nil {
		record.ErrorMessage = cause.Error()
	}
	record.UpdatedAt = now
	return m.saveTaskUpdate(record, taskEvent(record, nil))
}

func (m *Manager) Cancel(id string, message string) error {
	record, err := m.Get(id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	record.Status = StatusCancelled
	record.Message = strings.TrimSpace(message)
	if record.Message == "" {
		record.Message = "cancelled"
	}
	record.UpdatedAt = now
	record.FinishedAt = now
	return m.saveTaskUpdate(record, taskEvent(record, nil))
}

func (m *Manager) markRunning(record Record) error {
	_, err := m.markRunningRecord(record)
	return err
}

func (m *Manager) markRunningRecord(record Record) (Record, error) {
	if record.Attempts >= record.MaxAttempts {
		return record, fmt.Errorf("task has reached max attempts")
	}
	now := time.Now().UTC()
	record.Status = StatusRunning
	record.Message = "running"
	record.Attempts++
	record.UpdatedAt = now
	if record.StartedAt.IsZero() {
		record.StartedAt = now
	}
	if err := m.saveTaskUpdate(record, taskEvent(record, nil)); err != nil {
		return record, err
	}
	return record, nil
}

func (m *Manager) saveTaskUpdate(record Record, event Event) error {
	if m.beforeSaveTaskUpdate != nil {
		if err := m.beforeSaveTaskUpdate(record, event); err != nil {
			return err
		}
	}
	if err := db.UpdateAndInsertPublished(m.db, []db.UpdateMapper{createTaskUpdateMapper(record)}, []db.InsertMapper{createTaskEventInsertMapper(event)}); err != nil {
		return err
	}
	m.publishProgress(recordProgressUpdate(record, event.Details, true))
	return nil
}

func (m *Manager) saveRecoveredTaskUpdate(expected Record, record Record, event Event) (bool, error) {
	if m.beforeSaveTaskUpdate != nil {
		if err := m.beforeSaveTaskUpdate(record, event); err != nil {
			return false, err
		}
	}
	if err := db.UpdateAndInsertPublished(
		m.db,
		[]db.UpdateMapper{createTaskRecoveryUpdateMapper(expected, record)},
		[]db.InsertMapper{createTaskRecoveryEventInsertMapper(record, event)},
	); err != nil {
		return false, err
	}
	current, err := m.Get(record.ID)
	if err != nil {
		return false, fmt.Errorf("verify recovered task state: %w", err)
	}
	updated := current.Status == record.Status &&
		current.OwnerPeerID == record.OwnerPeerID &&
		current.Attempts == record.Attempts &&
		current.UpdatedAt.Equal(record.UpdatedAt)
	if !updated {
		return false, nil
	}
	m.publishProgress(recordProgressUpdate(record, event.Details, true))
	return true, nil
}

func hasExactPublishedWriteReceipt(receipt db.PublishedWriteReceipt) bool {
	return receipt.HasExactEventIdentity()
}

func (m *Manager) publishTaskCheckpointWithRetry(
	ctx context.Context,
	record Record,
	event Event,
	publisher func(context.Context, Record, Event) (db.PublishedWriteReceipt, error),
) (db.PublishedWriteReceipt, error) {
	waitRetry := m.waitTaskCheckpointPublicationRetry
	if waitRetry == nil {
		waitRetry = waitBeforeTaskCheckpointPublicationRetry
	}
	for attempt := 1; attempt <= taskCheckpointPublicationMaxAttempts; attempt++ {
		receipt, err := publisher(ctx, record, event)
		// A complete receipt is the operation identity, even when Commit also
		// returned an ancillary error. Status tracking must resume from that
		// exact receipt and must never publish the logical write again.
		if hasExactPublishedWriteReceipt(receipt) || err == nil {
			return receipt, err
		}
		// Retry only when Swarmion's typed outcome proves that no event or
		// operation receipt was accepted and that the transaction workspace was
		// restored. The same stable key and intent are retained for every attempt.
		if errors.Is(err, db.ErrOperationReceiptUnavailable) || !db.IsRetryablePublishedWriteError(err) {
			return receipt, err
		}
		if attempt == taskCheckpointPublicationMaxAttempts {
			return db.PublishedWriteReceipt{}, fmt.Errorf(
				"publish task checkpoint %s remained unresolved after %d retryable attempts: %w",
				record.ID,
				attempt,
				err,
			)
		}
		log.Debugf(
			"retrying task checkpoint publication task_id=%s attempt=%d/%d: %s",
			record.ID,
			attempt,
			taskCheckpointPublicationMaxAttempts,
			err.Error(),
		)
		if waitErr := waitRetry(ctx, attempt); waitErr != nil {
			return db.PublishedWriteReceipt{}, errors.Join(
				err,
				fmt.Errorf("wait to retry task checkpoint publication %s: %w", record.ID, waitErr),
			)
		}
	}
	return db.PublishedWriteReceipt{}, fmt.Errorf("publish task checkpoint %s exhausted retry attempts", record.ID)
}

func waitBeforeTaskCheckpointPublicationRetry(ctx context.Context, attempt int) error {
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
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) saveTaskCheckpoint(record Record, event Event, operationKey string, operationAuthorPeerID string) error {
	if m.beforeSaveTaskUpdate != nil {
		if err := m.beforeSaveTaskUpdate(record, event); err != nil {
			return err
		}
	}
	// The domain operation receipt already exists when CheckpointPayload reaches
	// this method. Shield its entire task-state publication and proof from runner
	// cancellation, while keeping the operation bounded.
	persistenceCtx, cancel := context.WithTimeout(context.Background(), taskCheckpointDurabilityTimeout)
	defer cancel()
	operation, err := taskCheckpointPublishedWriteOperationForAuthor(record, event, operationKey, operationAuthorPeerID)
	if err != nil {
		return err
	}
	resolved, err := m.db.LookupPublishedWriteOperation(persistenceCtx, operation)
	if err != nil {
		return MarkDeferred(fmt.Errorf("resolve task checkpoint operation %s: %w", record.ID, err))
	}
	var (
		receipt    db.PublishedWriteReceipt
		publishErr error
	)
	switch resolved.Resolution {
	case swarmionapp.BranchOperationReceiptFound:
		receipt, err = db.PublishedWriteReceiptFromOperation(resolved)
		if err != nil {
			return fmt.Errorf("recover task checkpoint operation %s: %w", record.ID, err)
		}
	case swarmionapp.BranchOperationReceiptUnavailable:
		return MarkDeferred(fmt.Errorf("%w: task checkpoint %s", db.ErrOperationReceiptUnavailable, record.ID))
	case swarmionapp.BranchOperationReceiptAbsent:
		// Publish below.
	default:
		return fmt.Errorf("resolve task checkpoint operation %s returned unknown resolution %q", record.ID, resolved.Resolution)
	}
	publisher := m.publishTaskCheckpoint
	if resolved.Resolution == swarmionapp.BranchOperationReceiptAbsent {
		if publisher == nil {
			publisher = func(ctx context.Context, record Record, event Event) (db.PublishedWriteReceipt, error) {
				return db.UpdateAndInsertWithOperationReceiptContext(
					ctx,
					m.db,
					operation,
					[]db.UpdateMapper{createTaskUpdateMapper(record)},
					[]db.InsertMapper{createTaskEventInsertMapper(event)},
				)
			}
		}
		receipt, publishErr = m.publishTaskCheckpointWithRetry(persistenceCtx, record, event, publisher)
	}
	if errors.Is(publishErr, db.ErrPublishedWriteReceiptIdentityConflict) {
		return MarkPermanent(publishErr)
	}
	hasReceipt := hasExactPublishedWriteReceipt(receipt)
	if !hasReceipt {
		if publishErr != nil {
			// Re-entering the stream would execute the same checkpoint SQL again.
			// Only Swarmion's explicit typed not-accepted outcome proves that is
			// safe. Dirty workspaces and ambiguous apply/commit/rollback failures
			// fail the task without replaying its operation body.
			if db.IsRetryablePublishedWriteError(publishErr) {
				return publishErr
			}
			return MarkPermanent(publishErr)
		}
		return MarkPermanent(fmt.Errorf("persist task operation receipt checkpoint %s returned no exact event receipt", record.ID))
	}

	tracker := m.checkpointTracker
	if tracker == nil {
		tracker = swarmionTaskCheckpointTracker{database: m.db}
	}
	observation, waitErr := tracker.WaitForPublishedWriteApplied(
		persistenceCtx,
		receipt,
		"persist task operation receipt checkpoint "+record.ID,
	)
	if waitErr != nil {
		if publishErr != nil {
			return MarkDeferred(errors.Join(publishErr, waitErr))
		}
		return MarkDeferred(waitErr)
	}
	if observation.State != db.EventReceiptStateAppliedDurably || !observation.Status.AppliedDurably {
		statusErr := fmt.Errorf(
			"persist task operation receipt checkpoint %s event %s/%s returned state=%s applied_durably=%t",
			record.ID,
			receipt.EventID,
			receipt.PublishedRootHash,
			observation.State,
			observation.Status.AppliedDurably,
		)
		if publishErr != nil {
			return MarkDeferred(errors.Join(publishErr, statusErr))
		}
		return MarkDeferred(statusErr)
	}

	// Read the invariant at the exact checkpoint selected for this event. The
	// durable checkpoint head may legitimately contain later task writes which
	// supersede this record; AppliedDurably proves the selected event checkpoint
	// is already contained in that durable lineage.
	eventCheckpointCommitID := strings.TrimSpace(observation.Status.CheckpointCommitID)
	durableCheckpointCommitID := strings.TrimSpace(observation.Status.DurableCheckpointCommitID)
	recordAtCheckpoint, found, invariantErr := tracker.RecordAtCheckpoint(
		persistenceCtx,
		eventCheckpointCommitID,
		record.ID,
	)
	if invariantErr == nil && found {
		matches, compareErr := taskCheckpointRecordsSemanticallyEqual(recordAtCheckpoint, record)
		if compareErr != nil {
			invariantErr = fmt.Errorf("compare task checkpoint state: %w", compareErr)
		} else if !matches {
			invariantErr = fmt.Errorf(
				"%w: task_id=%s event_id=%s published_root=%s event_checkpoint=%s durable_head=%s content_coverage=%s outcome=task state differs",
				ErrCheckpointInvariantConflict,
				record.ID,
				receipt.EventID,
				receipt.PublishedRootHash,
				eventCheckpointCommitID,
				durableCheckpointCommitID,
				observation.Status.ContentCoverage,
			)
		}
	}
	if invariantErr == nil && !found {
		invariantErr = fmt.Errorf(
			"%w: task_id=%s event_id=%s published_root=%s event_checkpoint=%s durable_head=%s content_coverage=%s outcome=task is absent",
			ErrCheckpointInvariantConflict,
			record.ID,
			receipt.EventID,
			receipt.PublishedRootHash,
			eventCheckpointCommitID,
			durableCheckpointCommitID,
			observation.Status.ContentCoverage,
		)
	}
	if invariantErr != nil {
		if errors.Is(invariantErr, ErrCheckpointInvariantConflict) {
			if publishErr != nil {
				return MarkPermanent(errors.Join(publishErr, invariantErr))
			}
			return MarkPermanent(invariantErr)
		}
		if publishErr != nil {
			return MarkDeferred(errors.Join(publishErr, invariantErr))
		}
		return MarkDeferred(invariantErr)
	}
	if publishErr != nil {
		log.Warnf(
			"task checkpoint recovered published event after write error task_id=%s event_id=%s published_root=%s event_checkpoint=%s durable_head=%s error=%s",
			record.ID,
			receipt.EventID,
			receipt.PublishedRootHash,
			eventCheckpointCommitID,
			durableCheckpointCommitID,
			publishErr.Error(),
		)
	}
	m.publishProgress(recordProgressUpdate(record, event.Details, true))
	return nil
}

type taskCheckpointLogicalRecord struct {
	ID           string
	Stream       string
	SubjectType  string
	SubjectID    string
	OwnerPeerID  string
	Status       Status
	Title        string
	Message      string
	Progress     int
	Payload      any
	Result       any
	ErrorMessage string
	Attempts     int
	MaxAttempts  int
}

type taskCheckpointLogicalEvent struct {
	TaskID   string
	Status   Status
	Message  string
	Progress int
	Details  any
}

type taskCheckpointLogicalState struct {
	Record taskCheckpointLogicalRecord
	Event  taskCheckpointLogicalEvent
}

// taskCheckpointPublishedWriteOperation binds every stable logical value written
// by the checkpoint transaction. Generated row/event timestamps and the event
// UUID are intentionally excluded: a retry must reconstruct the same operation
// after a process restart, while a changed progress update or event must not be
// mistaken for an already persisted checkpoint.
func taskCheckpointPublishedWriteOperation(record Record, event Event, baseKey string) (db.PublishedWriteOperation, error) {
	return taskCheckpointPublishedWriteOperationForAuthor(record, event, baseKey, "")
}

func taskCheckpointPublishedWriteOperationForAuthor(
	record Record,
	event Event,
	baseKey string,
	authorPeerID string,
) (db.PublishedWriteOperation, error) {
	baseKey = strings.TrimSpace(baseKey)
	if baseKey == "" {
		// Existing generic callers predate explicit domain operation keys. The
		// task ID remains stable across their retries; persistence-sensitive
		// production callers should use CheckpointPayloadWithOperationKey.
		baseKey = strings.TrimSpace(record.ID)
	}
	if baseKey == "" {
		return db.PublishedWriteOperation{}, fmt.Errorf("task checkpoint base operation key is empty")
	}
	state, err := newTaskCheckpointLogicalState(record, event)
	if err != nil {
		return db.PublishedWriteOperation{}, err
	}
	canonicalState, err := json.Marshal(state)
	if err != nil {
		return db.PublishedWriteOperation{}, fmt.Errorf("canonicalize task checkpoint state: %w", err)
	}
	intentParts := []string{
		"protos:task-payload-checkpoint:v2",
		string(canonicalState),
	}
	intent, err := db.NewPublishedWriteOperation("pending", intentParts...)
	if err != nil {
		return db.PublishedWriteOperation{}, err
	}
	keySeed, err := db.NewPublishedWriteOperation(
		"pending",
		"protos:task-payload-checkpoint-key:v2",
		baseKey,
		intent.IntentDigest,
	)
	if err != nil {
		return db.PublishedWriteOperation{}, err
	}
	operation, err := db.NewPublishedWriteOperation(keySeed.IntentDigest, intentParts...)
	if err != nil {
		return db.PublishedWriteOperation{}, err
	}
	operation.AuthorPeerID = strings.TrimSpace(authorPeerID)
	return operation, nil
}

func newTaskCheckpointLogicalState(record Record, event Event) (taskCheckpointLogicalState, error) {
	payload, err := decodeTaskCheckpointJSON(record.Payload)
	if err != nil {
		return taskCheckpointLogicalState{}, fmt.Errorf("decode task checkpoint payload: %w", err)
	}
	result, err := decodeTaskCheckpointJSON(record.Result)
	if err != nil {
		return taskCheckpointLogicalState{}, fmt.Errorf("decode task checkpoint result: %w", err)
	}
	details, err := decodeTaskCheckpointJSON(event.Details)
	if err != nil {
		return taskCheckpointLogicalState{}, fmt.Errorf("decode task checkpoint event details: %w", err)
	}
	return taskCheckpointLogicalState{
		Record: taskCheckpointLogicalRecord{
			ID:           strings.TrimSpace(record.ID),
			Stream:       record.Stream,
			SubjectType:  record.SubjectType,
			SubjectID:    record.SubjectID,
			OwnerPeerID:  strings.TrimSpace(record.OwnerPeerID),
			Status:       record.Status,
			Title:        record.Title,
			Message:      record.Message,
			Progress:     record.Progress,
			Payload:      payload,
			Result:       result,
			ErrorMessage: record.ErrorMessage,
			Attempts:     record.Attempts,
			MaxAttempts:  record.MaxAttempts,
		},
		Event: taskCheckpointLogicalEvent{
			TaskID:   strings.TrimSpace(event.TaskID),
			Status:   event.Status,
			Message:  event.Message,
			Progress: event.Progress,
			Details:  details,
		},
	}, nil
}

func taskCheckpointRecordsSemanticallyEqual(left Record, right Record) (bool, error) {
	leftState, err := newTaskCheckpointLogicalState(left, Event{})
	if err != nil {
		return false, err
	}
	rightState, err := newTaskCheckpointLogicalState(right, Event{})
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(leftState.Record, rightState.Record), nil
}

func decodeTaskCheckpointJSON(raw json.RawMessage) (any, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
		raw = json.RawMessage("{}")
	}
	return decodeTaskPayloadLosslessly(raw)
}

func taskPayloadsSemanticallyEqual(left json.RawMessage, right json.RawMessage) bool {
	leftValue, err := decodeTaskPayloadLosslessly(left)
	if err != nil {
		return false
	}
	rightValue, err := decodeTaskPayloadLosslessly(right)
	if err != nil {
		return false
	}
	return reflect.DeepEqual(leftValue, rightValue)
}

func decodeTaskPayloadLosslessly(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, fmt.Errorf("task payload contains multiple JSON values")
		}
		return nil, err
	}
	return value, nil
}

func (m *Manager) stream(name string) (registeredStream, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stream, found := m.streams[name]
	return stream, found
}

func taskEvent(record Record, details json.RawMessage) Event {
	if len(details) == 0 {
		details = json.RawMessage("{}")
	}
	return Event{
		ID:        db.MustNewUUIDv7(),
		TaskID:    record.ID,
		Status:    record.Status,
		Message:   record.Message,
		Progress:  record.Progress,
		Details:   details,
		CreatedAt: time.Now().UTC(),
	}
}

func recordProgressUpdate(record Record, details json.RawMessage, durable bool) ProgressUpdate {
	if len(details) == 0 {
		details = json.RawMessage("{}")
	}
	return ProgressUpdate{
		TaskID:    record.ID,
		Status:    record.Status,
		Message:   record.Message,
		Progress:  record.Progress,
		Details:   details,
		CreatedAt: time.Now().UTC(),
		Durable:   durable,
	}
}

func (m *Manager) publishProgress(update ProgressUpdate) {
	if m == nil {
		return
	}
	update.TaskID = strings.TrimSpace(update.TaskID)
	if update.TaskID == "" {
		return
	}
	update.Status = normalizeStatus(update.Status)
	update.Progress = normalizeProgress(update.Progress)
	update.Message = strings.TrimSpace(update.Message)
	if update.Message == "" {
		update.Message = string(update.Status)
	}
	if update.CreatedAt.IsZero() {
		update.CreatedAt = time.Now().UTC()
	}
	if len(update.Details) == 0 {
		update.Details = json.RawMessage("{}")
	} else {
		update.Details = append(json.RawMessage(nil), update.Details...)
	}

	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	m.progressSeq++
	update.Sequence = m.progressSeq
	m.latestProgress[update.TaskID] = update
	for _, watcher := range m.progressWatchers[update.TaskID] {
		select {
		case watcher <- update:
		default:
			select {
			case <-watcher:
			default:
			}
			select {
			case watcher <- update:
			default:
			}
		}
	}
}

func normalizeStatus(status Status) Status {
	if validStatus(status) {
		return status
	}
	return StatusRunning
}

func validStatus(status Status) bool {
	switch status {
	case StatusPending, StatusRunning, StatusSucceeded, StatusFailed, StatusCancelled:
		return true
	default:
		return false
	}
}

func latestRecord(records []Record) (Record, bool, error) {
	if len(records) == 0 {
		return Record{}, false, nil
	}
	latest := records[0]
	for _, record := range records[1:] {
		if record.CreatedAt.After(latest.CreatedAt) {
			latest = record
		}
	}
	return latest, true, nil
}

func normalizeProgress(progress int) int {
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func marshalOptional(value any) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage("{}"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal task details: %w", err)
	}
	return raw, nil
}
