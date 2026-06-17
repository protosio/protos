package tasks

import (
	"context"
	stdsql "database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bokwoon95/sq"
	"github.com/protosio/protos/internal/db"
)

const taskSelectColumns = "id, task_stream, subject_type, subject_id, status, title, message, progress, CAST(payload AS CHAR), CAST(result AS CHAR), error_message, attempts, max_attempts, created_at, updated_at, started_at, finished_at"
const taskSummarySelectColumns = "id, task_stream, subject_type, subject_id, status, title, message, progress, '{}', '{}', error_message, attempts, max_attempts, created_at, updated_at, started_at, finished_at"
const taskEventSelectColumns = "id, task_id, status, message, progress, CAST(details AS CHAR), created_at"

type taskQueryFilters struct {
	IDs         []string
	Stream      string
	SubjectType string
	SubjectID   string
	Status      Status
}

func createTaskInsertMapper(record Record) db.InsertMapper {
	return func() sq.InsertQuery {
		t := sq.New[db.TASK]("")
		return sq.InsertInto(t).ColumnValues(func(col *sq.Column) {
			setTaskColumns(col, t, record)
		})
	}
}

func createTaskUpdateMapper(record Record) db.UpdateMapper {
	return func() sq.UpdateQuery {
		t := sq.New[db.TASK]("")
		return sq.Update(t).SetFunc(func(col *sq.Column) {
			setTaskColumns(col, t, record)
		}).Where(db.UUIDEq(t.ID, record.ID))
	}
}

func createTaskEventInsertMapper(event Event) db.InsertMapper {
	return func() sq.InsertQuery {
		e := sq.New[db.TASK_EVENT]("")
		return sq.InsertInto(e).ColumnValues(func(col *sq.Column) {
			col.SetBytes(e.ID, db.MustUUIDBytes(event.ID))
			col.SetBytes(e.TASK_ID, db.MustUUIDBytes(event.TaskID))
			col.SetString(e.STATUS, string(event.Status))
			col.SetString(e.MESSAGE, event.Message)
			col.SetInt(e.PROGRESS, event.Progress)
			col.SetJSON(e.DETAILS, nullableJSON(event.Details))
			col.SetString(e.CREATED_AT, formatTime(event.CreatedAt))
		})
	}
}

func selectTaskRecords(database *db.DB, filters taskQueryFilters, includeDetails bool) ([]Record, error) {
	query := strings.Builder{}
	query.WriteString("SELECT ")
	if includeDetails {
		query.WriteString(taskSelectColumns)
	} else {
		query.WriteString(taskSummarySelectColumns)
	}
	query.WriteString(" FROM tasks")

	var args []any
	addPredicate := func(condition string, value any) {
		if len(args) == 0 {
			query.WriteString(" WHERE ")
		} else {
			query.WriteString(" AND ")
		}
		query.WriteString(condition)
		args = append(args, value)
	}

	for _, id := range filters.IDs {
		if id = strings.TrimSpace(id); id != "" {
			idBytes, err := db.UUIDBytes(id)
			if err != nil {
				addPredicate("1 = ?", 0)
			} else {
				addPredicate("id = ?", idBytes)
			}
			break
		}
	}
	if stream := strings.TrimSpace(filters.Stream); stream != "" {
		addPredicate("task_stream = ?", stream)
	}
	if subjectType := strings.TrimSpace(filters.SubjectType); subjectType != "" {
		addPredicate("subject_type = ?", subjectType)
	}
	if subjectID := strings.TrimSpace(filters.SubjectID); subjectID != "" {
		addPredicate("subject_id = ?", subjectID)
	}
	if status := Status(strings.TrimSpace(string(filters.Status))); status != "" {
		addPredicate("status = ?", string(status))
	}

	rows, err := database.QueryContext(context.Background(), query.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("select tasks: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		record, err := scanTaskRow(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tasks: %w", err)
	}
	return records, nil
}

func selectTaskEvents(database *db.DB, taskID string) ([]Event, error) {
	taskIDBytes, err := db.UUIDBytes(strings.TrimSpace(taskID))
	if err != nil {
		return nil, nil
	}
	rows, err := database.QueryContext(
		context.Background(),
		"SELECT "+taskEventSelectColumns+" FROM task_events WHERE task_id = ?",
		taskIDBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("select task events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		event, err := scanTaskEventRow(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read task events: %w", err)
	}
	return events, nil
}

func scanTaskRow(rows *stdsql.Rows) (Record, error) {
	var record Record
	var id []byte
	var payload, result stdsql.NullString
	var createdAt, updatedAt, startedAt, finishedAt string
	if err := rows.Scan(
		&id,
		&record.Stream,
		&record.SubjectType,
		&record.SubjectID,
		&record.Status,
		&record.Title,
		&record.Message,
		&record.Progress,
		&payload,
		&result,
		&record.ErrorMessage,
		&record.Attempts,
		&record.MaxAttempts,
		&createdAt,
		&updatedAt,
		&startedAt,
		&finishedAt,
	); err != nil {
		return Record{}, fmt.Errorf("scan task: %w", err)
	}
	record.ID = db.UUIDString(id)
	record.Payload = rawJSON(payload)
	record.Result = rawJSON(result)
	record.CreatedAt = parseTime(createdAt)
	record.UpdatedAt = parseTime(updatedAt)
	record.StartedAt = parseTime(startedAt)
	record.FinishedAt = parseTime(finishedAt)
	return record, nil
}

func scanTaskEventRow(rows *stdsql.Rows) (Event, error) {
	var event Event
	var id, taskID []byte
	var details stdsql.NullString
	var createdAt string
	if err := rows.Scan(
		&id,
		&taskID,
		&event.Status,
		&event.Message,
		&event.Progress,
		&details,
		&createdAt,
	); err != nil {
		return Event{}, fmt.Errorf("scan task event: %w", err)
	}
	event.ID = db.UUIDString(id)
	event.TaskID = db.UUIDString(taskID)
	event.Details = rawJSON(details)
	event.CreatedAt = parseTime(createdAt)
	return event, nil
}

func setTaskColumns(col *sq.Column, t db.TASK, record Record) {
	col.SetBytes(t.ID, db.MustUUIDBytes(record.ID))
	col.SetString(t.TASK_STREAM, record.Stream)
	col.SetString(t.SUBJECT_TYPE, record.SubjectType)
	col.SetString(t.SUBJECT_ID, record.SubjectID)
	col.SetString(t.STATUS, string(record.Status))
	col.SetString(t.TITLE, record.Title)
	col.SetString(t.MESSAGE, record.Message)
	col.SetInt(t.PROGRESS, record.Progress)
	col.SetJSON(t.PAYLOAD, nullableJSON(record.Payload))
	col.SetJSON(t.RESULT, nullableJSON(record.Result))
	col.SetString(t.ERROR_MESSAGE, record.ErrorMessage)
	col.SetInt(t.ATTEMPTS, record.Attempts)
	col.SetInt(t.MAX_ATTEMPTS, record.MaxAttempts)
	col.SetString(t.CREATED_AT, formatTime(record.CreatedAt))
	col.SetString(t.UPDATED_AT, formatTime(record.UpdatedAt))
	col.SetString(t.STARTED_AT, formatTime(record.StartedAt))
	col.SetString(t.FINISHED_AT, formatTime(record.FinishedAt))
}

func nullableJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

func rawJSON(value stdsql.NullString) json.RawMessage {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return json.RawMessage("{}")
	}
	return json.RawMessage(value.String)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}
