package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"math"

	swarmionruntime "github.com/nustiueudinastea/swarmion/runtime"
)

// openRuntimeReadDB adapts Swarmion's buffered, scoped QuerySQL result to the
// database/sql row cursor used throughout the backend. It is deliberately
// read-only: every mutation must enter DatabaseRuntime.Execute with a stable
// operation identity and a typed PublicationOutcome.
func openRuntimeReadDB(runtime *swarmionruntime.DatabaseRuntime) *sql.DB {
	if runtime == nil {
		return nil
	}
	return sql.OpenDB(runtimeSQLConnector{runtime: runtime})
}

type runtimeSQLConnector struct {
	runtime *swarmionruntime.DatabaseRuntime
}

func (c runtimeSQLConnector) Connect(context.Context) (driver.Conn, error) {
	if c.runtime == nil {
		return nil, fmt.Errorf("swarmion database runtime is not initialized")
	}
	return &runtimeSQLConn{runtime: c.runtime}, nil
}

func (c runtimeSQLConnector) Driver() driver.Driver {
	return runtimeSQLDriver{}
}

type runtimeSQLDriver struct{}

func (runtimeSQLDriver) Open(string) (driver.Conn, error) {
	return nil, fmt.Errorf("swarmion runtime SQL driver requires a scoped connector")
}

type runtimeSQLConn struct {
	runtime *swarmionruntime.DatabaseRuntime
}

func (c *runtimeSQLConn) Prepare(query string) (driver.Stmt, error) {
	if c == nil || c.runtime == nil {
		return nil, fmt.Errorf("swarmion database runtime is not initialized")
	}
	return &runtimeSQLStmt{runtime: c.runtime, query: query}, nil
}

func (*runtimeSQLConn) Close() error { return nil }

func (*runtimeSQLConn) Begin() (driver.Tx, error) {
	return nil, fmt.Errorf("direct SQL transactions are unavailable; use DatabaseRuntime.Execute")
}

func (c *runtimeSQLConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}

func (c *runtimeSQLConn) QueryContext(ctx context.Context, query string, values []driver.NamedValue) (driver.Rows, error) {
	if c == nil || c.runtime == nil {
		return nil, fmt.Errorf("swarmion database runtime is not initialized")
	}
	args := make([]any, len(values))
	for index := range values {
		args[index] = values[index].Value
	}
	rows, err := c.runtime.QuerySQL(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return newRuntimeSQLRows(rows), nil
}

func (*runtimeSQLConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, fmt.Errorf("direct SQL mutation is unavailable; use DatabaseRuntime.Execute")
}

func (*runtimeSQLConn) CheckNamedValue(*driver.NamedValue) error { return nil }

type runtimeSQLStmt struct {
	runtime *swarmionruntime.DatabaseRuntime
	query   string
}

func (*runtimeSQLStmt) Close() error  { return nil }
func (*runtimeSQLStmt) NumInput() int { return -1 }

func (*runtimeSQLStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, fmt.Errorf("direct SQL mutation is unavailable; use DatabaseRuntime.Execute")
}

func (s *runtimeSQLStmt) Query(values []driver.Value) (driver.Rows, error) {
	args := make([]any, len(values))
	for index := range values {
		args[index] = values[index]
	}
	rows, err := s.runtime.QuerySQL(context.Background(), s.query, args...)
	if err != nil {
		return nil, err
	}
	return newRuntimeSQLRows(rows), nil
}

func (s *runtimeSQLStmt) QueryContext(ctx context.Context, values []driver.NamedValue) (driver.Rows, error) {
	args := make([]any, len(values))
	for index := range values {
		args[index] = values[index].Value
	}
	rows, err := s.runtime.QuerySQL(ctx, s.query, args...)
	if err != nil {
		return nil, err
	}
	return newRuntimeSQLRows(rows), nil
}

type runtimeSQLRows struct {
	columns []string
	rows    [][]any
	next    int
}

func newRuntimeSQLRows(rows swarmionruntime.SQLRows) *runtimeSQLRows {
	cloned := &runtimeSQLRows{
		columns: append([]string(nil), rows.Columns...),
		rows:    make([][]any, len(rows.Rows)),
	}
	for index := range rows.Rows {
		cloned.rows[index] = append([]any(nil), rows.Rows[index]...)
	}
	return cloned
}

func (r *runtimeSQLRows) Columns() []string {
	if r == nil {
		return nil
	}
	return append([]string(nil), r.columns...)
}

func (*runtimeSQLRows) Close() error { return nil }

func (r *runtimeSQLRows) Next(destination []driver.Value) error {
	if r == nil || r.next >= len(r.rows) {
		return io.EOF
	}
	row := r.rows[r.next]
	r.next++
	if len(row) != len(r.columns) || len(destination) != len(r.columns) {
		return fmt.Errorf("swarmion SQL row width %d does not match columns %d", len(row), len(r.columns))
	}
	for index, value := range row {
		converted, err := runtimeSQLDriverValue(value)
		if err != nil {
			return fmt.Errorf("swarmion SQL column %q: %w", r.columns[index], err)
		}
		destination[index] = converted
	}
	return nil
}

func runtimeSQLDriverValue(value any) (driver.Value, error) {
	if driver.IsValue(value) {
		return value, nil
	}
	switch value := value.(type) {
	case int:
		return int64(value), nil
	case int8:
		return int64(value), nil
	case int16:
		return int64(value), nil
	case int32:
		return int64(value), nil
	case uint:
		if uint64(value) > math.MaxInt64 {
			return nil, fmt.Errorf("unsigned integer %d exceeds database/sql range", value)
		}
		return int64(value), nil
	case uint8:
		return int64(value), nil
	case uint16:
		return int64(value), nil
	case uint32:
		return int64(value), nil
	case uint64:
		if value > math.MaxInt64 {
			return nil, fmt.Errorf("unsigned integer %d exceeds database/sql range", value)
		}
		return int64(value), nil
	case float32:
		return float64(value), nil
	default:
		return nil, fmt.Errorf("unsupported database/sql value type %T", value)
	}
}

var (
	_ driver.Connector         = runtimeSQLConnector{}
	_ driver.Conn              = (*runtimeSQLConn)(nil)
	_ driver.ConnBeginTx       = (*runtimeSQLConn)(nil)
	_ driver.QueryerContext    = (*runtimeSQLConn)(nil)
	_ driver.ExecerContext     = (*runtimeSQLConn)(nil)
	_ driver.NamedValueChecker = (*runtimeSQLConn)(nil)
	_ driver.Stmt              = (*runtimeSQLStmt)(nil)
	_ driver.StmtQueryContext  = (*runtimeSQLStmt)(nil)
	_ driver.Rows              = (*runtimeSQLRows)(nil)
)
