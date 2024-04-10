package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/bokwoon95/sq"
	"github.com/bokwoon95/sqddl/ddl"
	"github.com/nustiueudinastea/doltswarm"
	"github.com/protosio/protos/internal/util"
)

var logger = util.GetLogger("db")
var Instance *doltswarm.DB

//go:embed migrations/*.sql
var rootDir embed.FS

func Open(workDir string, dbName string, signer doltswarm.Signer) (*DB, error) {
	dbi, err := doltswarm.Open(workDir, dbName, logger, signer)
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

	_, err = db.Commit("test")
	if err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	return nil
}

// Insert inserts a new entry in the database using the sq query builder
func Insert(db *DB, mc func() (sq.Table, func(*sq.Column))) error {
	t, mapper := mc()
	_, err := sq.Exec(db, sq.
		InsertInto(t).
		ColumnValues(mapper).
		SetDialect(sq.DialectMySQL),
	)
	return err
}

func Update(db *DB, mc func() (sq.Table, func(*sq.Column), []sq.Predicate)) error {
	t, mapper, predicates := mc()
	_, err := sq.Exec(db, sq.
		Update(t).
		SetFunc(mapper).
		Where(predicates...).
		SetDialect(sq.DialectMySQL),
	)
	return err
}

func SelectOne[T any](db *DB, mc func() (sq.Table, func(row *sq.Row) T, []sq.Predicate)) (T, error) {
	t, mapper, predicates := mc()
	res, err := sq.FetchOne(db, sq.
		From(t).
		Where(predicates...).
		SetDialect(sq.DialectMySQL),
		mapper,
	)
	if err != nil {
		return res, fmt.Errorf("failed to select one: %v", err)
	}
	return res, nil
}

func SelectMultiple[T any](db *DB, mc func() (sq.Table, func(row *sq.Row) T, []sq.Predicate)) ([]T, error) {
	t, mapper, predicates := mc()
	var res []T
	var err error
	if len(predicates) == 0 {
		res, err = sq.FetchAll(db, sq.
			From(t).
			SetDialect(sq.DialectMySQL),
			mapper,
		)
	} else {
		res, err = sq.FetchAll(db, sq.
			From(t).
			Where(predicates...).
			SetDialect(sq.DialectMySQL),
			mapper,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to select multiple: %v", err)
	}
	return res, nil
}

func Delete(db *DB, mc func() (sq.Table, []sq.Predicate)) error {
	t, predicates := mc()
	_, err := sq.Exec(db, sq.
		DeleteFrom(t).
		Where(predicates...).
		SetDialect(sq.DialectMySQL),
	)
	return err
}
