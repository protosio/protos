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

func (db *DB) ExecuteSQL(ctx context.Context, statement string, maxRows int) (SQLResult, error) {
	if db == nil {
		return SQLResult{}, fmt.Errorf("db is not initialized")
	}
	statement = strings.TrimSpace(statement)
	if statement == "" {
		return SQLResult{}, fmt.Errorf("sql statement is empty")
	}
	if maxRows <= 0 {
		maxRows = defaultSQLConsoleMaxRows
	}
	if maxRows > hardSQLConsoleMaxRows {
		maxRows = hardSQLConsoleMaxRows
	}
	mayMutate := sqlMayMutate(statement)
	if mayMutate {
		db.opMu.Lock()
		defer db.opMu.Unlock()
	}

	var result SQLResult
	var err error
	if sqlReturnsRows(statement) {
		result, err = db.executeSQLQuery(ctx, statement, maxRows)
	} else {
		result, err = db.executeSQLStatement(ctx, statement)
	}
	if err != nil {
		return SQLResult{}, err
	}
	if mayMutate {
		db.NotifyChanges()
	}
	return result, nil
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

func (db *DB) executeSQLStatement(ctx context.Context, statement string) (SQLResult, error) {
	res, err := db.ExecContext(ctx, statement)
	if err != nil {
		return SQLResult{}, err
	}
	var rowsAffected int64
	if res != nil {
		rowsAffected, _ = res.RowsAffected()
	}
	return SQLResult{
		RowsAffected: rowsAffected,
		Message:      fmt.Sprintf("%d rows affected", rowsAffected),
	}, nil
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

func sqlReturnsRows(statement string) bool {
	switch sqlLeadKeyword(statement) {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH", "CALL", "VALUES", "TABLE":
		return true
	default:
		return false
	}
}

func sqlMayMutate(statement string) bool {
	switch sqlLeadKeyword(statement) {
	case "SELECT", "SHOW", "DESCRIBE", "DESC", "EXPLAIN", "WITH", "VALUES", "TABLE":
		return false
	default:
		return true
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
