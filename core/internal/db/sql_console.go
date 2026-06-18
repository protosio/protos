package db

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const (
	defaultSQLConsoleMaxRows = 200
	hardSQLConsoleMaxRows    = 1000
)

type SQLCell struct {
	Value string
	Null  bool
}

type SQLRow struct {
	Cells []SQLCell
}

type SQLResult struct {
	Columns      []string
	Rows         []SQLRow
	RowsAffected int64
	Truncated    bool
	Message      string
}

// ExecuteSQL exists only for the user-facing SQL console. It is intentionally
// read-only and bounded; application code must use typed domain methods.
func (db *DB) ExecuteSQL(ctx context.Context, statement string, maxRows int) (SQLResult, error) {
	if db == nil {
		return SQLResult{}, fmt.Errorf("db is not initialized")
	}
	statement, err := normalizeReadOnlySQL(statement)
	if err != nil {
		return SQLResult{}, err
	}
	if maxRows <= 0 {
		maxRows = defaultSQLConsoleMaxRows
	}
	if maxRows > hardSQLConsoleMaxRows {
		maxRows = hardSQLConsoleMaxRows
	}
	return db.executeSQLQuery(ctx, statement, maxRows)
}

func (db *DB) executeSQLQuery(ctx context.Context, statement string, maxRows int) (SQLResult, error) {
	rows, err := db.QueryContext(ctx, statement)
	if err != nil {
		return SQLResult{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return SQLResult{}, err
	}
	values := make([]any, len(columns))
	scan := make([]any, len(columns))
	for i := range values {
		scan[i] = &values[i]
	}

	result := SQLResult{Columns: columns}
	for rows.Next() {
		if len(result.Rows) >= maxRows {
			result.Truncated = true
			break
		}
		for i := range values {
			values[i] = nil
		}
		if err := rows.Scan(scan...); err != nil {
			return SQLResult{}, err
		}
		resultRow := SQLRow{Cells: make([]SQLCell, len(columns))}
		for i, value := range values {
			resultRow.Cells[i] = formatSQLCell(value)
		}
		result.Rows = append(result.Rows, resultRow)
	}
	if err := rows.Err(); err != nil {
		return SQLResult{}, err
	}
	result.Message = fmt.Sprintf("%d rows", len(result.Rows))
	if result.Truncated {
		result.Message = fmt.Sprintf("%s shown, result truncated", result.Message)
	}
	return result, nil
}

func (db *DB) NotifyChanges(tableNames ...string) {
	if db == nil {
		return
	}
	db.triggerTableChangeCallbacks(tableNames...)
}

func formatSQLCell(value any) SQLCell {
	switch typed := value.(type) {
	case nil:
		return SQLCell{Null: true}
	case []byte:
		return SQLCell{Value: string(typed)}
	case time.Time:
		return SQLCell{Value: typed.Format(time.RFC3339Nano)}
	default:
		return SQLCell{Value: fmt.Sprint(typed)}
	}
}

func normalizeReadOnlySQL(statement string) (string, error) {
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return "", fmt.Errorf("sql statement is empty")
	}
	statement = strings.TrimSuffix(statement, ";")
	statement = strings.TrimSpace(statement)
	if strings.Contains(statement, ";") {
		return "", fmt.Errorf("only one SQL statement is allowed")
	}
	if !sqlReadOnly(statement) {
		return "", fmt.Errorf("only read-only SQL statements are allowed")
	}
	return statement, nil
}

func sqlReadOnly(statement string) bool {
	switch sqlLeadKeyword(statement) {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN":
		return true
	default:
		return false
	}
}

func sqlLeadKeyword(statement string) string {
	statement = strings.TrimSpace(statement)
	for {
		switch {
		case strings.HasPrefix(statement, "--") || strings.HasPrefix(statement, "#"):
			idx := strings.IndexByte(statement, '\n')
			if idx < 0 {
				return ""
			}
			statement = strings.TrimSpace(statement[idx+1:])
		case strings.HasPrefix(statement, "/*"):
			idx := strings.Index(statement, "*/")
			if idx < 0 {
				return ""
			}
			statement = strings.TrimSpace(statement[idx+2:])
		default:
			fields := strings.Fields(statement)
			if len(fields) == 0 {
				return ""
			}
			return strings.ToUpper(strings.Trim(fields[0], "();"))
		}
	}
}
