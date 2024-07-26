package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/ddl"
	"github.com/nustiueudinastea/doltswarm"
	"github.com/protosio/protos/internal/util"
)

var Instance *doltswarm.DB

//go:embed migrations/*.sql
var rootDir embed.FS

func Open(workDir string, dbName string, signer doltswarm.Signer) (*DB, error) {
	dbi, err := doltswarm.Open(workDir, dbName, util.GetLogger("swarm"), signer)
	if err != nil {
		return nil, fmt.Errorf("failed to create db: %v", err)
	}

	return &DB{DB: dbi, name: dbName}, nil
}

type DB struct {
	*doltswarm.DB
	name string
}

func (db *DB) Init() error {

	err := db.InitLocal()
	if err != nil {
		return fmt.Errorf("failed to init local: %v", err)
	}

	migrationsDir, err := fs.Sub(rootDir, "migrations")
	if err != nil {
		return fmt.Errorf("failed to get migrations dir: %v", err)
	}

	_, err = db.GetSqlDB().ExecContext(context.TODO(), fmt.Sprintf("USE %s;", db.name))
	if err != nil {
		return fmt.Errorf("failed to use db: %w", err)
	}

	migrateCmd := &ddl.MigrateCmd{
		Dialect: "mysql",
		DB:      db.GetSqlDB(),
		DirFS:   migrationsDir,
	}
	err = migrateCmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run migrations: %v", err)
	}

	_, err = db.Commit("init commit")
	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	return nil
}

//
// Read operations
//

func SelectOne[T any](db *DB, mc QueryMapper[T]) (T, error) {
	query, mapper := mc()
	res, err := sq.FetchOne(db, query.SetDialect(sq.DialectMySQL), mapper)
	if err != nil {
		return res, fmt.Errorf("failed to select one: %v", err)
	}
	return res, nil
}

func SelectMultiple[T any](db *DB, mc QueryMapper[T]) ([]T, error) {
	query, mapper := mc()
	res, err := sq.FetchAll(db, query.SetDialect(sq.DialectMySQL), mapper)
	if err != nil {
		return nil, fmt.Errorf("failed to select multiple: %v", err)
	}
	return res, nil
}

//
// Edit operations
//

// Insert inserts a new entry in the database using the sq query builder
func Insert(db *DB, mappers ...InsertMapper) error {
	execFunc := func(tx *sql.Tx) error {
		for _, mapper := range mappers {
			_, err := sq.Exec(tx, mapper().SetDialect(sq.DialectMySQL))
			if err != nil {
				return fmt.Errorf("failed to insert: %v", err)
			}
		}
		return nil
	}

	_, err := db.ExecAndCommit(execFunc, "insert")
	if err != nil {
		return fmt.Errorf("failed to inser: %v", err)
	}

	return nil
}

func Update(db *DB, mappers ...UpdateMapper) error {
	execFunc := func(tx *sql.Tx) error {
		for _, mapper := range mappers {
			_, err := sq.Exec(tx, mapper().SetDialect(sq.DialectMySQL))
			if err != nil {
				return fmt.Errorf("failed to update: %v", err)
			}
		}
		return nil
	}
	_, err := db.ExecAndCommit(execFunc, "update")
	if err != nil {
		return fmt.Errorf("failed to update: %v", err)
	}

	return nil
}

func Delete(db *DB, mappers ...DeleteMapper) error {
	execFunc := func(tx *sql.Tx) error {
		for _, mapper := range mappers {
			_, err := sq.Exec(tx, mapper().SetDialect(sq.DialectMySQL))
			if err != nil {
				return fmt.Errorf("failed to delete: %v", err)
			}
		}
		return nil
	}

	_, err := db.ExecAndCommit(execFunc, "delete")
	if err != nil {
		return fmt.Errorf("failed to delete: %v", err)
	}

	return nil
}
