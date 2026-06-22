package db

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/nustiueudinastea/swarmion/runtime/schema"
	protoscontracts "github.com/protosio/protos/internal/db/contracts/sql/protos"
)

const commitDiffMaxRowsPerTable = 200

type CommitDiffValue struct {
	Value string
	Null  bool
}

type CommitDiffField struct {
	Name      string
	Before    CommitDiffValue
	After     CommitDiffValue
	BeforeCUE string
	AfterCUE  string
	Changed   bool
}

type CommitDiffRow struct {
	ChangeType   string
	Key          string
	Fields       []CommitDiffField
	BeforeValues map[string]CommitDiffValue
	AfterValues  map[string]CommitDiffValue
	BeforeCUE    string
	AfterCUE     string
	CUE          string
}

type CommitDiffTable struct {
	Name      string
	Rows      []CommitDiffRow
	CUE       string
	SQL       string
	Truncated bool
}

type CommitDiffTaskContext struct {
	ID            string
	Stream        string
	SubjectType   string
	SubjectID     string
	OwnerPeerID   string
	Status        string
	Title         string
	Message       string
	Progress      int
	ChangeSources []string
	EventCount    int
	Summary       string
}

type CommitDiff struct {
	BaseHash     string
	TargetHash   string
	Tables       []CommitDiffTable
	RelatedTasks []CommitDiffTaskContext
	CUE          string
	SQL          string
	UnifiedDiff  string
	Truncated    bool
	Message      string
}

type commitDiffTableSchema struct {
	Name       string
	Columns    []schema.SQLColumn
	PrimaryKey []schema.SQLColumn
}

func (db *DB) GetCommit(ref string) (Commit, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return Commit{}, fmt.Errorf("commit ref is empty")
	}
	query := fmt.Sprintf("SELECT commit_hash, committer, email, date, message, parents, refs FROM dolt_log('%s', '--parents', '--decorate=short') LIMIT 1;", escapeSQL(ref))
	commits, err := db.getCommits(query)
	if err != nil {
		return Commit{}, err
	}
	if len(commits) == 0 {
		return Commit{}, fmt.Errorf("commit %q not found", ref)
	}
	return commits[0], nil
}

func (db *DB) GetCommitDiff(ctx context.Context, targetHash string, baseHash string) (CommitDiff, error) {
	if db == nil {
		return CommitDiff{}, fmt.Errorf("db is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	targetHash = strings.TrimSpace(targetHash)
	baseHash = strings.TrimSpace(baseHash)
	if targetHash == "" {
		return CommitDiff{}, fmt.Errorf("commit hash is empty")
	}

	target, err := db.GetCommit(targetHash)
	if err != nil {
		return CommitDiff{}, err
	}
	if baseHash == "" && len(target.ParentHashes) > 0 {
		baseHash = target.ParentHashes[0]
	}

	diff := CommitDiff{
		BaseHash:   baseHash,
		TargetHash: target.Hash,
	}
	if baseHash == "" {
		diff.Message = "commit has no parent to diff against"
		diff.CUE = renderCommitDiffCUE(diff)
		diff.SQL = renderCommitDiffSQL(diff)
		diff.UnifiedDiff = renderCommitDiffUnified(diff)
		return diff, nil
	}

	for _, tableSchema := range commitDiffTableSchemas() {
		tableDiff, err := db.getCommitTableDiff(ctx, tableSchema, baseHash, target.Hash)
		if err != nil {
			return CommitDiff{}, fmt.Errorf("diff %s: %w", tableSchema.Name, err)
		}
		if len(tableDiff.Rows) == 0 {
			continue
		}
		tableDiff.CUE = renderCommitDiffTableCUE(tableDiff)
		tableDiff.SQL = renderCommitDiffTableSQL(tableSchema, tableDiff)
		diff.Tables = append(diff.Tables, tableDiff)
		if tableDiff.Truncated {
			diff.Truncated = true
		}
	}
	if len(diff.Tables) == 0 {
		diff.Message = "no contract table changes"
	} else if diff.Truncated {
		diff.Message = "diff truncated"
	}
	diff.RelatedTasks = commitDiffRelatedTaskContexts(diff.Tables)
	diff.CUE = renderCommitDiffCUE(diff)
	diff.SQL = renderCommitDiffSQL(diff)
	diff.UnifiedDiff = renderCommitDiffUnified(diff)
	return diff, nil
}

func (db *DB) getCommitTableDiff(ctx context.Context, tableSchema commitDiffTableSchema, baseHash string, targetHash string) (CommitDiffTable, error) {
	query := fmt.Sprintf(
		"SELECT * FROM dolt_diff('%s', '%s', '%s') LIMIT %d;",
		escapeSQL(baseHash),
		escapeSQL(targetHash),
		escapeSQL(tableSchema.Name),
		commitDiffMaxRowsPerTable+1,
	)
	tableDiff := CommitDiffTable{Name: tableSchema.Name}
	if err := db.ReadRows(ctx, query, nil, func(rows *sql.Rows) error {
		columns, err := rows.Columns()
		if err != nil {
			return err
		}
		values := make([]any, len(columns))
		scan := make([]any, len(columns))
		for i := range values {
			scan[i] = &values[i]
		}
		for rows.Next() {
			for i := range values {
				values[i] = nil
			}
			if err := rows.Scan(scan...); err != nil {
				return err
			}
			if len(tableDiff.Rows) >= commitDiffMaxRowsPerTable {
				tableDiff.Truncated = true
				break
			}
			row, ok := buildCommitDiffRow(tableSchema, columns, values, len(tableDiff.Rows))
			if ok {
				tableDiff.Rows = append(tableDiff.Rows, row)
			}
		}
		return nil
	}); err != nil {
		return CommitDiffTable{}, err
	}
	return tableDiff, nil
}

func buildCommitDiffRow(tableSchema commitDiffTableSchema, columns []string, values []any, fallbackIndex int) (CommitDiffRow, bool) {
	columnValues := commitDiffColumnValues(columns, values)
	before := make(map[string]CommitDiffValue, len(tableSchema.Columns))
	after := make(map[string]CommitDiffValue, len(tableSchema.Columns))
	rawType := commitDiffRawString(columnValues["diff_type"])
	changeType := normalizeCommitDiffType(rawType)

	var fields []CommitDiffField
	for _, column := range tableSchema.Columns {
		beforeRaw, hasBefore := columnValues["from_"+strings.ToLower(column.Name)]
		afterRaw, hasAfter := columnValues["to_"+strings.ToLower(column.Name)]
		if !hasBefore && !hasAfter {
			continue
		}
		beforeValue := formatCommitDiffSQLValue(beforeRaw, column)
		afterValue := formatCommitDiffSQLValue(afterRaw, column)
		if hasBefore {
			before[column.Name] = beforeValue
		}
		if hasAfter {
			after[column.Name] = afterValue
		}

		changed := !commitDiffValuesEqual(beforeValue, afterValue)
		include := changed
		switch changeType {
		case "added":
			include = hasAfter
		case "deleted":
			include = hasBefore
		}
		if !include {
			continue
		}
		fields = append(fields, CommitDiffField{
			Name:      column.Name,
			Before:    beforeValue,
			After:     afterValue,
			BeforeCUE: commitDiffCUELiteral(beforeValue, column),
			AfterCUE:  commitDiffCUELiteral(afterValue, column),
			Changed:   true,
		})
	}
	if len(fields) == 0 {
		return CommitDiffRow{}, false
	}
	if changeType == "" {
		changeType = inferCommitDiffType(before, after)
	}
	if changeType == "" {
		changeType = "modified"
	}

	row := CommitDiffRow{
		ChangeType:   changeType,
		Key:          commitDiffRowKey(tableSchema, before, after, fallbackIndex),
		Fields:       fields,
		BeforeValues: before,
		AfterValues:  after,
	}
	if changeType != "added" {
		row.BeforeCUE = renderCommitDiffRowValuesCUE(tableSchema.Columns, before)
	}
	if changeType != "deleted" {
		row.AfterCUE = renderCommitDiffRowValuesCUE(tableSchema.Columns, after)
	}
	row.CUE = renderCommitDiffRowCUE(row)
	return row, true
}

func commitDiffColumnValues(columns []string, values []any) map[string]any {
	out := make(map[string]any, len(columns))
	for i, column := range columns {
		if i >= len(values) {
			break
		}
		out[strings.ToLower(column)] = values[i]
	}
	return out
}

func normalizeCommitDiffType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "added", "add", "insert", "inserted":
		return "added"
	case "removed", "remove", "delete", "deleted":
		return "deleted"
	case "modified", "modify", "updated", "update", "changed":
		return "modified"
	default:
		return ""
	}
}

func inferCommitDiffType(before map[string]CommitDiffValue, after map[string]CommitDiffValue) string {
	hasBefore := commitDiffMapHasValue(before)
	hasAfter := commitDiffMapHasValue(after)
	switch {
	case hasBefore && hasAfter:
		return "modified"
	case hasAfter:
		return "added"
	case hasBefore:
		return "deleted"
	default:
		return ""
	}
}

func commitDiffMapHasValue(values map[string]CommitDiffValue) bool {
	for _, value := range values {
		if !value.Null || value.Value != "" {
			return true
		}
	}
	return false
}

func formatCommitDiffSQLValue(value any, column schema.SQLColumn) CommitDiffValue {
	if value == nil {
		return CommitDiffValue{Null: true}
	}
	switch typed := value.(type) {
	case []byte:
		if strings.EqualFold(column.Type, "BINARY(16)") && len(typed) == UUIDBinaryLength {
			if id := UUIDString(typed); id != "" {
				return CommitDiffValue{Value: id}
			}
		}
		return CommitDiffValue{Value: string(typed)}
	case string:
		if strings.EqualFold(column.Type, "BINARY(16)") {
			bytes := []byte(typed)
			if len(bytes) == UUIDBinaryLength {
				if id := UUIDString(bytes); id != "" {
					return CommitDiffValue{Value: id}
				}
			}
		}
		return CommitDiffValue{Value: typed}
	case time.Time:
		return CommitDiffValue{Value: typed.Format(time.RFC3339Nano)}
	default:
		return CommitDiffValue{Value: fmt.Sprint(typed)}
	}
}

func commitDiffRawString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case []byte:
		return string(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func commitDiffValuesEqual(left CommitDiffValue, right CommitDiffValue) bool {
	return left.Null == right.Null && left.Value == right.Value
}

func commitDiffRowKey(tableSchema commitDiffTableSchema, before map[string]CommitDiffValue, after map[string]CommitDiffValue, fallbackIndex int) string {
	keyColumns := tableSchema.PrimaryKey
	if len(keyColumns) == 0 && len(tableSchema.Columns) > 0 {
		keyColumns = tableSchema.Columns[:1]
	}
	if len(keyColumns) == 0 {
		return fmt.Sprintf("row-%d", fallbackIndex+1)
	}
	parts := make([]string, 0, len(keyColumns))
	for _, column := range keyColumns {
		value, ok := after[column.Name]
		if !ok || value.Null {
			value = before[column.Name]
		}
		if value.Null {
			parts = append(parts, "null")
			continue
		}
		parts = append(parts, value.Value)
	}
	if len(parts) == 1 {
		if parts[0] != "" {
			return parts[0]
		}
		return fmt.Sprintf("row-%d", fallbackIndex+1)
	}
	return strings.Join(parts, "|")
}

func commitDiffRelatedTaskContexts(tables []CommitDiffTable) []CommitDiffTaskContext {
	byID := map[string]*CommitDiffTaskContext{}
	ensureTask := func(id string) *CommitDiffTaskContext {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil
		}
		if existing := byID[id]; existing != nil {
			return existing
		}
		ctx := &CommitDiffTaskContext{ID: id}
		byID[id] = ctx
		return ctx
	}

	for _, table := range tables {
		switch table.Name {
		case "tasks":
			for _, row := range table.Rows {
				id := commitDiffRowString(row, "id")
				if id == "" {
					id = row.Key
				}
				task := ensureTask(id)
				if task == nil {
					continue
				}
				task.addSource("tasks")
				task.Stream = firstNonEmpty(commitDiffRowString(row, "task_stream"), task.Stream)
				task.SubjectType = firstNonEmpty(commitDiffRowString(row, "subject_type"), task.SubjectType)
				task.SubjectID = firstNonEmpty(commitDiffRowString(row, "subject_id"), task.SubjectID)
				task.OwnerPeerID = firstNonEmpty(commitDiffRowString(row, "owner_peer_id"), task.OwnerPeerID)
				task.Status = firstNonEmpty(commitDiffRowString(row, "status"), task.Status)
				task.Title = firstNonEmpty(commitDiffRowString(row, "title"), task.Title)
				task.Message = firstNonEmpty(commitDiffRowString(row, "message"), task.Message)
				if progress, ok := commitDiffRowInt(row, "progress"); ok {
					task.Progress = progress
				}
			}
		case "task_events":
			for _, row := range table.Rows {
				task := ensureTask(commitDiffRowString(row, "task_id"))
				if task == nil {
					continue
				}
				task.addSource("task_events")
				task.EventCount++
				task.Status = firstNonEmpty(task.Status, commitDiffRowString(row, "status"))
				task.Message = firstNonEmpty(task.Message, commitDiffRowString(row, "message"))
				if progress, ok := commitDiffRowInt(row, "progress"); ok && task.Progress == 0 {
					task.Progress = progress
				}
			}
		}
	}

	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]CommitDiffTaskContext, 0, len(ids))
	for _, id := range ids {
		task := *byID[id]
		sort.Strings(task.ChangeSources)
		task.Summary = commitDiffTaskSummary(task)
		out = append(out, task)
	}
	return out
}

func (task *CommitDiffTaskContext) addSource(source string) {
	if task == nil || source == "" {
		return
	}
	for _, existing := range task.ChangeSources {
		if existing == source {
			return
		}
	}
	task.ChangeSources = append(task.ChangeSources, source)
}

func commitDiffRowString(row CommitDiffRow, column string) string {
	value, ok := commitDiffRowValue(row, column)
	if !ok || value.Null {
		return ""
	}
	return strings.TrimSpace(value.Value)
}

func commitDiffRowInt(row CommitDiffRow, column string) (int, bool) {
	raw := commitDiffRowString(row, column)
	if raw == "" {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return value, true
}

func commitDiffRowValue(row CommitDiffRow, column string) (CommitDiffValue, bool) {
	switch row.ChangeType {
	case "deleted":
		if value, ok := row.BeforeValues[column]; ok {
			return value, true
		}
		value, ok := row.AfterValues[column]
		return value, ok
	default:
		if value, ok := row.AfterValues[column]; ok {
			return value, true
		}
		value, ok := row.BeforeValues[column]
		return value, ok
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func commitDiffTaskSummary(task CommitDiffTaskContext) string {
	var parts []string
	if task.Title != "" {
		parts = append(parts, task.Title)
	} else if task.Stream != "" {
		parts = append(parts, task.Stream)
	} else {
		parts = append(parts, "task "+task.ID)
	}
	if task.Status != "" {
		status := task.Status
		if task.Progress > 0 {
			status = fmt.Sprintf("%s %d%%", status, task.Progress)
		}
		parts = append(parts, status)
	}
	if task.Message != "" {
		parts = append(parts, task.Message)
	}
	return strings.Join(parts, " - ")
}

func commitDiffCUELiteral(value CommitDiffValue, column schema.SQLColumn) string {
	if value.Null {
		return "null"
	}
	columnType := strings.ToUpper(strings.TrimSpace(column.Type))
	raw := strings.TrimSpace(value.Value)
	if columnType == "TINYINT(1)" {
		switch strings.ToLower(raw) {
		case "1", "true":
			return "true"
		case "0", "false":
			return "false"
		}
	}
	if columnType == "JSON" && json.Valid([]byte(raw)) {
		var compact bytes.Buffer
		if err := json.Compact(&compact, []byte(raw)); err == nil {
			return compact.String()
		}
	}
	if strings.Contains(columnType, "INT") {
		if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return raw
		}
	}
	if strings.Contains(columnType, "DOUBLE") || strings.Contains(columnType, "FLOAT") || strings.Contains(columnType, "DECIMAL") {
		if _, err := strconv.ParseFloat(raw, 64); err == nil {
			return raw
		}
	}
	return strconv.Quote(value.Value)
}

func renderCommitDiffCUE(diff CommitDiff) string {
	var b strings.Builder
	b.WriteString("{\n")
	if diff.BaseHash != "" {
		b.WriteString("\t_base: ")
		b.WriteString(strconv.Quote(diff.BaseHash))
		b.WriteByte('\n')
	}
	if diff.TargetHash != "" {
		b.WriteString("\t_target: ")
		b.WriteString(strconv.Quote(diff.TargetHash))
		b.WriteByte('\n')
	}
	if diff.Message != "" {
		b.WriteString("\t_message: ")
		b.WriteString(strconv.Quote(diff.Message))
		b.WriteByte('\n')
	}
	if len(diff.RelatedTasks) > 0 {
		b.WriteString("\t_related_tasks: [\n")
		for _, task := range diff.RelatedTasks {
			appendIndentedCUE(&b, renderCommitDiffTaskContextCUE(task), 2)
		}
		b.WriteString("\t]\n")
	}
	for _, table := range diff.Tables {
		appendIndentedCUE(&b, table.CUE, 1)
	}
	b.WriteString("}\n")
	return b.String()
}

func renderCommitDiffUnified(diff CommitDiff) string {
	var b strings.Builder
	if len(diff.Tables) == 0 {
		if diff.Message != "" {
			b.WriteString("# ")
			b.WriteString(diff.Message)
			b.WriteByte('\n')
		}
		return b.String()
	}
	for _, table := range diff.Tables {
		for _, row := range table.Rows {
			renderCommitDiffRowUnified(&b, table.Name, row)
		}
	}
	if diff.Truncated {
		b.WriteString("# diff truncated\n")
	}
	return b.String()
}

func renderCommitDiffSQL(diff CommitDiff) string {
	var b strings.Builder
	if len(diff.Tables) == 0 {
		if diff.Message != "" {
			b.WriteString("-- ")
			b.WriteString(diff.Message)
			b.WriteByte('\n')
		}
		return b.String()
	}
	for _, table := range diff.Tables {
		if strings.TrimSpace(table.SQL) == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(table.SQL)
		if !strings.HasSuffix(table.SQL, "\n") {
			b.WriteByte('\n')
		}
	}
	if diff.Truncated {
		b.WriteString("-- diff truncated\n")
	}
	return b.String()
}

func renderCommitDiffTableSQL(tableSchema commitDiffTableSchema, table CommitDiffTable) string {
	var b strings.Builder
	for _, row := range table.Rows {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		renderCommitDiffRowSQL(&b, tableSchema, row)
	}
	if table.Truncated {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("-- table diff truncated\n")
	}
	return b.String()
}

func renderCommitDiffRowSQL(b *strings.Builder, tableSchema commitDiffTableSchema, row CommitDiffRow) {
	b.WriteString("-- ")
	b.WriteString(tableSchema.Name)
	b.WriteByte('/')
	b.WriteString(row.Key)
	b.WriteByte(' ')
	b.WriteString(row.ChangeType)
	b.WriteByte('\n')

	switch row.ChangeType {
	case "added":
		renderCommitDiffInsertSQL(b, tableSchema, row)
	case "deleted":
		renderCommitDiffDeleteSQL(b, tableSchema, row)
	default:
		renderCommitDiffUpdateSQL(b, tableSchema, row)
	}
}

func renderCommitDiffInsertSQL(b *strings.Builder, tableSchema commitDiffTableSchema, row CommitDiffRow) {
	columns := commitDiffColumnsWithValues(tableSchema.Columns, row.AfterValues)
	if len(columns) == 0 {
		b.WriteString("-- no insertable values\n")
		return
	}

	b.WriteString("INSERT INTO ")
	b.WriteString(sqlIdent(tableSchema.Name))
	b.WriteString(" (")
	for index, column := range columns {
		if index > 0 {
			b.WriteString(", ")
		}
		b.WriteString(sqlIdent(column.Name))
	}
	b.WriteString(") VALUES (")
	for index, column := range columns {
		if index > 0 {
			b.WriteString(", ")
		}
		b.WriteString(commitDiffSQLLiteral(row.AfterValues[column.Name], column))
	}
	b.WriteString(");\n")
}

func renderCommitDiffDeleteSQL(b *strings.Builder, tableSchema commitDiffTableSchema, row CommitDiffRow) {
	b.WriteString("DELETE FROM ")
	b.WriteString(sqlIdent(tableSchema.Name))
	if where := commitDiffWhereClause(tableSchema, row.BeforeValues, row.AfterValues); where != "" {
		b.WriteString(" WHERE ")
		b.WriteString(where)
	}
	b.WriteString(";\n")
}

func renderCommitDiffUpdateSQL(b *strings.Builder, tableSchema commitDiffTableSchema, row CommitDiffRow) {
	setFields := commitDiffChangedSQLFields(tableSchema, row)
	if len(setFields) == 0 {
		b.WriteString("-- no changed SQL fields\n")
		return
	}

	b.WriteString("UPDATE ")
	b.WriteString(sqlIdent(tableSchema.Name))
	b.WriteString(" SET ")
	for index, field := range setFields {
		if index > 0 {
			b.WriteString(", ")
		}
		b.WriteString(sqlIdent(field.column.Name))
		b.WriteString(" = ")
		b.WriteString(commitDiffSQLLiteral(field.value, field.column))
	}
	if where := commitDiffWhereClause(tableSchema, row.BeforeValues, row.AfterValues); where != "" {
		b.WriteString(" WHERE ")
		b.WriteString(where)
	}
	b.WriteString(";\n")
}

type commitDiffSQLField struct {
	column schema.SQLColumn
	value  CommitDiffValue
}

func commitDiffChangedSQLFields(tableSchema commitDiffTableSchema, row CommitDiffRow) []commitDiffSQLField {
	changed := make(map[string]struct{}, len(row.Fields))
	for _, field := range row.Fields {
		changed[field.Name] = struct{}{}
	}
	fields := make([]commitDiffSQLField, 0, len(changed))
	for _, column := range tableSchema.Columns {
		if _, ok := changed[column.Name]; !ok {
			continue
		}
		value, ok := row.AfterValues[column.Name]
		if !ok {
			continue
		}
		fields = append(fields, commitDiffSQLField{column: column, value: value})
	}
	return fields
}

func commitDiffColumnsWithValues(columns []schema.SQLColumn, values map[string]CommitDiffValue) []schema.SQLColumn {
	out := make([]schema.SQLColumn, 0, len(columns))
	for _, column := range columns {
		if _, ok := values[column.Name]; ok {
			out = append(out, column)
		}
	}
	return out
}

func commitDiffWhereClause(tableSchema commitDiffTableSchema, preferred map[string]CommitDiffValue, fallback map[string]CommitDiffValue) string {
	keyColumns := tableSchema.PrimaryKey
	if len(keyColumns) == 0 {
		keyColumns = tableSchema.Columns
	}
	var parts []string
	for _, column := range keyColumns {
		value, ok := preferred[column.Name]
		if !ok {
			value, ok = fallback[column.Name]
		}
		if !ok {
			continue
		}
		parts = append(parts, commitDiffSQLPredicate(column, value))
	}
	return strings.Join(parts, " AND ")
}

func commitDiffSQLPredicate(column schema.SQLColumn, value CommitDiffValue) string {
	name := sqlIdent(column.Name)
	if value.Null {
		return name + " IS NULL"
	}
	return name + " = " + commitDiffSQLLiteral(value, column)
}

func commitDiffSQLLiteral(value CommitDiffValue, column schema.SQLColumn) string {
	if value.Null {
		return "NULL"
	}
	columnType := strings.ToUpper(strings.TrimSpace(column.Type))
	raw := strings.TrimSpace(value.Value)
	if columnType == "BINARY(16)" && validUUIDString(raw) {
		return "UNHEX(REPLACE(" + sqlString(raw) + ", '-', ''))"
	}
	if columnType == "TINYINT(1)" {
		switch strings.ToLower(raw) {
		case "1", "true":
			return "TRUE"
		case "0", "false":
			return "FALSE"
		}
	}
	if columnType == "JSON" && json.Valid([]byte(raw)) {
		return "CAST(" + sqlString(compactJSONString(raw)) + " AS JSON)"
	}
	if strings.Contains(columnType, "INT") {
		if _, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return raw
		}
	}
	if strings.Contains(columnType, "DOUBLE") || strings.Contains(columnType, "FLOAT") || strings.Contains(columnType, "DECIMAL") {
		if _, err := strconv.ParseFloat(raw, 64); err == nil {
			return raw
		}
	}
	return sqlString(value.Value)
}

func compactJSONString(raw string) string {
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(raw)); err != nil {
		return raw
	}
	return compact.String()
}

func validUUIDString(value string) bool {
	_, err := UUIDBytes(value)
	return err == nil
}

func sqlIdent(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

func sqlString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func renderCommitDiffRowUnified(b *strings.Builder, tableName string, row CommitDiffRow) {
	path := tableName + "/" + row.Key + ".cue"
	b.WriteString("diff --cue a/")
	b.WriteString(path)
	b.WriteString(" b/")
	b.WriteString(path)
	b.WriteByte('\n')
	switch row.ChangeType {
	case "added":
		b.WriteString("--- /dev/null\n")
		b.WriteString("+++ b/")
		b.WriteString(path)
		b.WriteByte('\n')
		b.WriteString("@@\n")
		for _, line := range splitDiffLines(row.AfterCUE) {
			b.WriteByte('+')
			b.WriteString(line)
			b.WriteByte('\n')
		}
	case "deleted":
		b.WriteString("--- a/")
		b.WriteString(path)
		b.WriteByte('\n')
		b.WriteString("+++ /dev/null\n")
		b.WriteString("@@\n")
		for _, line := range splitDiffLines(row.BeforeCUE) {
			b.WriteByte('-')
			b.WriteString(line)
			b.WriteByte('\n')
		}
	default:
		b.WriteString("--- a/")
		b.WriteString(path)
		b.WriteByte('\n')
		b.WriteString("+++ b/")
		b.WriteString(path)
		b.WriteByte('\n')
		b.WriteString("@@\n")
		for _, line := range unifiedLineDiff(splitDiffLines(row.BeforeCUE), splitDiffLines(row.AfterCUE)) {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
}

func splitDiffLines(text string) []string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

func unifiedLineDiff(before []string, after []string) []string {
	if len(before) == 0 && len(after) == 0 {
		return nil
	}
	lcs := make([][]int, len(before)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(after)+1)
	}
	for i := len(before) - 1; i >= 0; i-- {
		for j := len(after) - 1; j >= 0; j-- {
			if before[i] == after[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var out []string
	i, j := 0, 0
	for i < len(before) && j < len(after) {
		switch {
		case before[i] == after[j]:
			out = append(out, " "+before[i])
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, "-"+before[i])
			i++
		default:
			out = append(out, "+"+after[j])
			j++
		}
	}
	for ; i < len(before); i++ {
		out = append(out, "-"+before[i])
	}
	for ; j < len(after); j++ {
		out = append(out, "+"+after[j])
	}
	return out
}

func renderCommitDiffTableCUE(table CommitDiffTable) string {
	var b strings.Builder
	b.WriteString(cueLabel(table.Name))
	b.WriteString(": {\n")
	for _, row := range table.Rows {
		b.WriteByte('\t')
		b.WriteString(strconv.Quote(row.Key))
		b.WriteString(": ")
		appendInlineIndentedCUE(&b, row.CUE, 1)
	}
	if table.Truncated {
		b.WriteString("\t_truncated: true\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func renderCommitDiffTaskContextCUE(task CommitDiffTaskContext) string {
	var b strings.Builder
	b.WriteString("{\n")
	writeOptionalCUEString(&b, 1, "id", task.ID)
	writeOptionalCUEString(&b, 1, "stream", task.Stream)
	writeOptionalCUEString(&b, 1, "subject_type", task.SubjectType)
	writeOptionalCUEString(&b, 1, "subject_id", task.SubjectID)
	writeOptionalCUEString(&b, 1, "owner_peer_id", task.OwnerPeerID)
	writeOptionalCUEString(&b, 1, "status", task.Status)
	writeOptionalCUEString(&b, 1, "title", task.Title)
	writeOptionalCUEString(&b, 1, "message", task.Message)
	if task.Progress > 0 {
		b.WriteString("\tprogress: ")
		b.WriteString(strconv.Itoa(task.Progress))
		b.WriteByte('\n')
	}
	if len(task.ChangeSources) > 0 {
		b.WriteString("\tchange_sources: [")
		for index, source := range task.ChangeSources {
			if index > 0 {
				b.WriteString(", ")
			}
			b.WriteString(strconv.Quote(source))
		}
		b.WriteString("]\n")
	}
	if task.EventCount > 0 {
		b.WriteString("\tevent_count: ")
		b.WriteString(strconv.Itoa(task.EventCount))
		b.WriteByte('\n')
	}
	writeOptionalCUEString(&b, 1, "summary", task.Summary)
	b.WriteString("}")
	return b.String()
}

func writeOptionalCUEString(b *strings.Builder, indent int, label string, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	for i := 0; i < indent; i++ {
		b.WriteByte('\t')
	}
	b.WriteString(cueLabel(label))
	b.WriteString(": ")
	b.WriteString(strconv.Quote(value))
	b.WriteByte('\n')
}

func renderCommitDiffRowCUE(row CommitDiffRow) string {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("\t_op: ")
	b.WriteString(strconv.Quote(row.ChangeType))
	b.WriteByte('\n')
	b.WriteString("\t_key: ")
	b.WriteString(strconv.Quote(row.Key))
	b.WriteByte('\n')
	for _, field := range row.Fields {
		b.WriteByte('\t')
		b.WriteString(cueLabel(field.Name))
		b.WriteString(": {\n")
		b.WriteString("\t\tbefore: ")
		b.WriteString(field.BeforeCUE)
		b.WriteByte('\n')
		b.WriteString("\t\tafter: ")
		b.WriteString(field.AfterCUE)
		b.WriteByte('\n')
		b.WriteString("\t}\n")
	}
	b.WriteString("}")
	return b.String()
}

func renderCommitDiffRowValuesCUE(columns []schema.SQLColumn, values map[string]CommitDiffValue) string {
	var b strings.Builder
	b.WriteString("{\n")
	for _, column := range columns {
		value, ok := values[column.Name]
		if !ok {
			continue
		}
		b.WriteByte('\t')
		b.WriteString(cueLabel(column.Name))
		b.WriteString(": ")
		b.WriteString(commitDiffCUELiteral(value, column))
		b.WriteByte('\n')
	}
	b.WriteString("}")
	return b.String()
}

func appendInlineIndentedCUE(b *strings.Builder, text string, extraIndent int) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 0 {
		b.WriteString("{}\n")
		return
	}
	b.WriteString(lines[0])
	b.WriteByte('\n')
	for _, line := range lines[1:] {
		for i := 0; i < extraIndent; i++ {
			b.WriteByte('\t')
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func appendIndentedCUE(b *strings.Builder, text string, indent int) {
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		for i := 0; i < indent; i++ {
			b.WriteByte('\t')
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func cueLabel(label string) string {
	if cueIdentifier(label) {
		return label
	}
	return strconv.Quote(label)
}

func cueIdentifier(label string) bool {
	if label == "" {
		return false
	}
	for i, r := range label {
		if i == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func commitDiffTableSchemas() []commitDiffTableSchema {
	for _, surface := range protoscontracts.Catalog.Surfaces {
		if surface.ID != "protos.db" {
			continue
		}
		if len(surface.Versions) == 0 {
			return nil
		}
		version := surface.Versions[len(surface.Versions)-1]
		tables := make([]commitDiffTableSchema, 0, len(version.SQL.Tables))
		for _, table := range version.SQL.Tables {
			tableSchema := commitDiffTableSchema{
				Name:    table.Name,
				Columns: append([]schema.SQLColumn(nil), table.Columns...),
			}
			for _, column := range table.Columns {
				if column.PrimaryKey {
					tableSchema.PrimaryKey = append(tableSchema.PrimaryKey, column)
				}
			}
			tables = append(tables, tableSchema)
		}
		return tables
	}
	return nil
}
