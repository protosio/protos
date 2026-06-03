package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/protosio/protos/internal/db"
	"github.com/protosio/protos/internal/util"
	"github.com/rs/xid"
)

var log = util.GetLogger("tasks")

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

type EnqueueOptions[P any] struct {
	ID          string
	Stream      string
	SubjectType string
	SubjectID   string
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

type Stream[P any, R any] struct {
	Name string
	Run  func(context.Context, *RunContext[P]) (R, error)
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

func (ctx *RunContext[P]) Update(progress int, message string, details any) error {
	return ctx.manager.Update(ctx.record.ID, StatusRunning, progress, message, details)
}

type registeredStream interface {
	run(context.Context, *Manager, Record) error
}

type streamAdapter[P any, R any] struct {
	stream Stream[P, R]
}

func (s streamAdapter[P, R]) run(ctx context.Context, manager *Manager, record Record) error {
	if s.stream.Run == nil {
		err := fmt.Errorf("task stream %q has no run function", s.stream.Name)
		_ = manager.Fail(record.ID, err, nil)
		return err
	}

	var payload P
	if len(record.Payload) > 0 {
		if err := json.Unmarshal(record.Payload, &payload); err != nil {
			err = fmt.Errorf("decode task payload: %w", err)
			_ = manager.Fail(record.ID, err, nil)
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
		_ = manager.Fail(record.ID, err, nil)
		return err
	}
	if err := manager.Succeed(record.ID, result); err != nil {
		return err
	}
	return nil
}

type Manager struct {
	db      *db.DB
	mu      sync.RWMutex
	streams map[string]registeredStream
}

func NewManager(database *db.DB) *Manager {
	return &Manager{
		db:      database,
		streams: map[string]registeredStream{},
	}
}

func Register[P any, R any](manager *Manager, stream Stream[P, R]) error {
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
	if opts.Title == "" {
		opts.Title = opts.Stream
	}
	if opts.ID == "" {
		opts.ID = xid.New().String()
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
	if err := db.Insert(manager.db, createTaskInsertMapper(record), createTaskEventInsertMapper(taskEvent(record, nil))); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (m *Manager) Start(ctx context.Context, interval time.Duration) func() error {
	if m == nil {
		return func() error { return nil }
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	runCtx, cancel := context.WithCancel(ctx)
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			if err := m.RunPending(runCtx); err != nil {
				log.Errorf("task runner failed: %s", err.Error())
			}
			select {
			case <-ticker.C:
			case <-runCtx.Done():
				return
			}
		}
	}()
	return func() error {
		cancel()
		return nil
	}
}

func (m *Manager) RunPending(ctx context.Context) error {
	if m == nil || m.db == nil || !m.db.Initialized() {
		return nil
	}
	records, err := selectTaskRecords(m.db, taskQueryFilters{Status: StatusPending}, true)
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
		if err := m.markRunning(record); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", record.ID, err))
			continue
		}
		latestRecords, err := selectTaskRecords(m.db, taskQueryFilters{IDs: []string{record.ID}}, true)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", record.ID, err))
			continue
		}
		if len(latestRecords) == 0 {
			failures = append(failures, fmt.Sprintf("%s: task not found", record.ID))
			continue
		}
		latest := latestRecords[0]
		if err := stream.run(ctx, m, latest); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", record.ID, err))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("%s", strings.Join(failures, "; "))
	}
	return nil
}

func (m *Manager) Get(id string) (Record, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Record{}, fmt.Errorf("task id is empty")
	}
	records, err := selectTaskRecords(m.db, taskQueryFilters{IDs: []string{id}}, false)
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
	latest := records[0]
	for _, record := range records[1:] {
		if record.CreatedAt.After(latest.CreatedAt) {
			latest = record
		}
	}
	return latest, true, nil
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

func (m *Manager) Succeed(id string, result any) error {
	record, err := m.Get(id)
	if err != nil {
		return err
	}
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
	if record.Attempts >= record.MaxAttempts {
		return fmt.Errorf("task has reached max attempts")
	}
	now := time.Now().UTC()
	record.Status = StatusRunning
	record.Message = "running"
	record.Attempts++
	record.UpdatedAt = now
	if record.StartedAt.IsZero() {
		record.StartedAt = now
	}
	return m.saveTaskUpdate(record, taskEvent(record, nil))
}

func (m *Manager) saveTaskUpdate(record Record, event Event) error {
	if err := db.Update(m.db, createTaskUpdateMapper(record)); err != nil {
		return err
	}
	return db.Insert(m.db, createTaskEventInsertMapper(event))
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
		ID:        xid.New().String(),
		TaskID:    record.ID,
		Status:    record.Status,
		Message:   record.Message,
		Progress:  record.Progress,
		Details:   details,
		CreatedAt: time.Now().UTC(),
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
